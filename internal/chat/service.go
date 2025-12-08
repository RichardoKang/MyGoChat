package chat

import (
	pb "MyGoChat/api/v1"
	"MyGoChat/internal/group"
	"MyGoChat/internal/util"
	"MyGoChat/pkg/config"
	myKafka "MyGoChat/pkg/kafka"
	"MyGoChat/pkg/log"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type Service struct {
	repo      Repository
	groupRepo group.Repository
	redis     *redis.Client
	producer  *myKafka.Producer
}

func NewService(
	repo Repository,
	groupRepo group.Repository,
	rdb *redis.Client,
	producer *myKafka.Producer,
) *Service {
	return &Service{
		repo:      repo,
		groupRepo: groupRepo,
		redis:     rdb,
		producer:  producer,
	}
}

// GetProducer 返回 Kafka Producer 实例（供 Handler 使用）
func (s *Service) GetProducer() *myKafka.Producer {
	return s.producer
}

func (s *Service) EnqueueMessage(ctx context.Context, msg *pb.Message) error {
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	// 发送到 Ingest Topic
	return s.producer.SendMessage(config.GetConfig().Kafka.Topics.Ingest, msgBytes)
}

func (s *Service) ProcessMessage(ctx context.Context, kafkaMsg kafka.Message) error {
	// 1. 反序列化 Protobuf
	var msg pb.Message
	if err := proto.Unmarshal(kafkaMsg.Value, &msg); err != nil {
		log.Logger.Sugar().Errorf("Failed to unmarshal message: %v", err)
		return err
	}

	// 2. 验证消息
	if err := s.validateMessage(&msg); err != nil {
		log.Logger.Sugar().Errorf("Invalid message: %v", err)
		return err
	}

	// 3. 解包消息体
	body, err := s.unpackProtoBody(msg.ContentType, msg.Body)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to unpack body: %v", err)
		return err
	}

	// 4. 构建数据模型
	convObjID, _ := primitive.ObjectIDFromHex(msg.ConversationID)
	message := &Message{
		ConversationID: convObjID,
		SenderUUID:     msg.SenderUUID,
		ContentType:    int16(msg.ContentType),
		Body:           body,
		SendAt:         time.Now().Unix(),
	}

	// 5. 存储消息
	if err := s.repo.CreateMsg(ctx, message); err != nil {
		log.Logger.Sugar().Errorf("Failed to save message: %v", err)
		return err
	}

	// 6. 更新会话最后消息
	if err := s.repo.UpdateLastMessage(ctx, msg.ConversationID, message); err != nil {
		log.Logger.Sugar().Warnf("Failed to update conversation: %v", err)
		return err
	}

	// 7. 推送消息给目标用户（在线和离线）
	s.deliverMessage(ctx, &msg)
	return nil
}

// validateMessage 验证消息的有效性
func (s *Service) validateMessage(msg *pb.Message) error {
	if msg.SenderUUID == "" {
		return errors.New("sender UUID is required")
	}

	if msg.ConversationID == "" {
		return errors.New("conversation ID is required")
	}

	if msg.MessageType == 1 && msg.RecipientUUID == "" {
		return errors.New("recipient UUID is required for private chat")
	}

	return nil
}

// deliverMessage 将消息推送给目标用户（在线推送到Gateway，离线存储到Redis）
func (s *Service) deliverMessage(ctx context.Context, msg *pb.Message) {
	// 根据消息类型决定推送目标
	var targetUsers []string

	if msg.MessageType == 1 { // 私聊
		targetUsers = []string{msg.RecipientUUID}
		log.Logger.Sugar().Infof("Delivering private message from %s to %s", msg.SenderUUID, msg.RecipientUUID)
	} else if msg.MessageType == 2 { // 群聊
		// 查询群成员列表
		conv, err := s.repo.GetConversationByID(ctx, msg.ConversationID)
		if err != nil {
			log.Logger.Sugar().Errorf("deliverMessage: failed to find conversation %s: %v", msg.ConversationID, err)
			return
		}

		// 校验数据一致性
		if conv.GroupNumber == "" {
			log.Logger.Sugar().Errorf("deliverMessage: conversation %s has no group number", msg.ConversationID)
			return
		}

		// Step B: [Group域] 用 GroupNumber 查成员列表
		// s.groupRepo 是注入进来的 group.Repository
		memberUUIDs, err := s.groupRepo.GetMemberUUIDs(conv.GroupNumber)
		if err != nil {
			log.Logger.Sugar().Errorf("deliverMessage: failed to get group members for %s: %v", conv.GroupNumber, err)
			return
		}

		targetUsers = memberUUIDs
	}

	// 为每个目标用户推送消息
	for _, userUUID := range targetUsers {
		if userUUID == msg.SenderUUID { // 不推送给发送者自己
			continue
		}

		// 构造推送消息
		pushMsg := &pb.Message{
			Id:             msg.Id,
			ConversationID: msg.ConversationID,
			SenderUUID:     msg.SenderUUID,
			SendAt:         msg.SendAt,
			ContentType:    msg.ContentType,
			Body:           msg.Body,
			Metadata:       msg.Metadata,
			SenderName:     msg.SenderName,
			Avatar:         msg.Avatar,
			MessageType:    msg.MessageType,
			RecipientUUID:  userUUID,
		}

		// 据用户在线状态推送
		gatewayID := s.getUserGateway(userUUID)
		if gatewayID != "" {
			// 在线用户：推送到Gateway
			cfg := config.GetConfig()
			topic := cfg.Kafka.Topics.Delivery + gatewayID
			s.publishToKafka(topic, pushMsg)
			log.Logger.Sugar().Infof("Delivered message to online user %s via gateway %s", userUUID, gatewayID)
		} else {
			// 离线用户：存储到Redis
			s.storeOfflineMessage(userUUID, pushMsg)
			log.Logger.Sugar().Infof("Stored offline message for user %s", userUUID)
		}
	}
}

// getUserGateway 获取用户当前连接的网关ID
func (s *Service) getUserGateway(userUUID string) string {
	ctx := context.Background()
	gatewayID, err := s.redis.Get(ctx, "user_gateway:"+userUUID).Result()
	if err != nil {
		return ""
	}
	return gatewayID
}

// publishToKafka 发布消息到 Kafka
func (s *Service) publishToKafka(topic string, msg *pb.Message) {
	msgData, err := proto.Marshal(msg)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to marshal message: %v", err)
		return
	}

	err = s.producer.SendMessage(topic, msgData)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to publish to kafka: %v", err)
	}
}

// storeOfflineMessage 存储离线消息
func (s *Service) storeOfflineMessage(userUUID string, msg *pb.Message) {
	ctx := context.Background()
	msgData, err := proto.Marshal(msg)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to marshal offline message: %v", err)
		return
	}

	// redis 列表存储离线消息，key格式为 "offline_msg:{userUUID}"
	key := "offline_msg:" + userUUID
	err = s.redis.LPush(ctx, key, string(msgData)).Err()
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to store offline message: %v", err)
	}

	// 设置过期时间（7天）
	s.redis.Expire(ctx, key, time.Hour*24*7)
}

// SyncOfflineMessages 用户上线时同步离线消息
func (s *Service) SyncOfflineMessages(userUUID string) error {
	ctx := context.Background()
	key := "offline_msg:" + userUUID

	// 获取所有离线消息
	messages, err := s.redis.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return err
	}

	if len(messages) == 0 {
		return nil // 没有离线消息
	}

	// 获取用户当前连接的网关
	gatewayID := s.getUserGateway(userUUID)
	if gatewayID == "" {
		return nil // 用户不在线，跳过
	}

	// 推送离线消息到网关
	cfg := config.GetConfig()
	topic := cfg.Kafka.Topics.Delivery + gatewayID
	for _, msgData := range messages {
		// 直接发送原始数据
		err = s.producer.SendMessage(topic, []byte(msgData))
		if err != nil {
			log.Logger.Sugar().Errorf("Failed to send offline message: %v", err)
		}
	}

	// 清除已推送的离线消息
	s.redis.Del(ctx, key)

	log.Logger.Sugar().Infof("Synced %d offline messages for user: %s", len(messages), userUUID)
	return nil
}

// GetMessageHistory 获取消息历史记录
func (s *Service) GetMessageHistory(conversationID string, limit, offset int) ([]*Message, error) {
	return s.repo.GetByConversation(conversationID, limit, offset)
}

// MarkMessagesAsRead 标记消息为已读
func (s *Service) MarkMessagesAsRead(msgIDs []string) error {
	return s.repo.MarkAsRead(msgIDs)
}

// ProcessSyncRequest 处理离线消息同步请求
func (s *Service) ProcessSyncRequest(ctx context.Context, kafkaMsg kafka.Message) error {
	var syncRequest map[string]interface{}
	if err := json.Unmarshal(kafkaMsg.Value, &syncRequest); err != nil {
		log.Logger.Sugar().Errorf("Failed to unmarshal sync request: %v", err)
		return err
	}

	action, ok := syncRequest["action"].(string)
	if !ok || action != "sync_offline" {
		return errors.New("invalid sync action")
	}

	userUUID, ok := syncRequest["useruuid"].(string)
	if !ok {
		log.Logger.Sugar().Errorf("Invalid user_uuid in sync request")
		return errors.New("invalid user_uuid in sync request")
	}

	// 同步离线消息
	err := s.SyncOfflineMessages(userUUID)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to sync offline messages for user %s: %v", userUUID, err)
	} else {
		log.Logger.Sugar().Infof("Successfully synced offline messages for user: %s", userUUID)
	}
}

// 辅助函数：解包 google.protobuf.Any
func (s *Service) unpackProtoBody(contentType int32, anyBody *anypb.Any) (any, error) {
	switch contentType {
	case 1: // Text
		var textBody pb.TextBody
		if err := anyBody.UnmarshalTo(&textBody); err != nil {
			return nil, err
		}
		return textBody.Content, nil // 只存 string
	case 2, 3, 4: // Image, File, Voice
		var fileBody pb.FileAttachment
		if err := anyBody.UnmarshalTo(&fileBody); err != nil {
			return nil, err
		}
		// 存为 Mongo model 里的 struct
		return FileAttachment{
			URL:      fileBody.Url,
			FileName: fileBody.FileName,
			Size:     fileBody.Size,
			MimeType: fileBody.MimeType,
		}, nil
	default:
		return nil, errors.New("unknown content type")
	}
}

/***************************************************************************************************************
会话相关服务
****************************************************************************************************************/

// GetConversationsByUserID 获取用户的会话列表
func (s *Service) GetConversationsByUserID(ctx context.Context, userID string, limit int64) ([]*Conversation, error) {
	return s.repo.GetConversationsByUserID(ctx, userID, limit)
}

// GetPrivateConversation 获取私聊会话
func (s *Service) GetPrivateConversation(ctx context.Context, userID1, userID2 string) (*Conversation, error) {
	convID := util.GetPrivateConversationID(userID1, userID2)

	return s.repo.GetConversationByID(ctx, convID)
}

// CreateGroupConversation 创建群聊会话
func (s *Service) CreateGroupConversation(ctx context.Context, groupUUID string) error {

	conv := &Conversation{
		ID:        groupUUID,
		Type:      2,
		CreatedAt: time.Now(),
	}

	return s.repo.CreateConversation(ctx, conv)
}

// GetGroupConversation 获取群聊会话
func (s *Service) GetGroupConversation(ctx context.Context, groupUUID string) (*Conversation, error) {
	return s.repo.GetConversationByID(ctx, groupUUID)
}
