package service

import (
	pb "MyGoChat/api/v1"
	"MyGoChat/internal/logic/data"
	"MyGoChat/internal/model"
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

type MessageService struct {
	msgRepo   data.IMessageRepo
	convRepo  data.IConversationRepo
	groupRepo data.IGroupRepo
	redis     *redis.Client
	producer  *myKafka.Producer
}

func NewMessageService(msgRepo data.IMessageRepo, convRepo data.IConversationRepo, groupRepo data.IGroupRepo, rdb *redis.Client, producer *myKafka.Producer) *MessageService {
	return &MessageService{
		msgRepo:   msgRepo,
		convRepo:  convRepo,
		groupRepo: groupRepo,
		redis:     rdb,
		producer:  producer,
	}
}

func (s *MessageService) ProcessMessage(kafkaMsg kafka.Message) {
	// 1. 反序列化 Protobuf
	var msg pb.Message
	if err := proto.Unmarshal(kafkaMsg.Value, &msg); err != nil {
		//	log.Logger
		return
	}

	// 2. 解包消息体
	body, err := s.unpackProtoBody(msg.ContentType, msg.Body)
	if err != nil {
		//log.Errorf("Failed to unpack body: %v", err)
		return
	}

	// 3. 构建数据模型
	convObjID, _ := primitive.ObjectIDFromHex(msg.ConversationID)
	message := &model.Message{
		ConversationID: convObjID,
		SenderUUID:     msg.SenderUUID,
		ContentType:    int16(msg.ContentType),
		Body:           body, // 需要 model.Message 支持 interface{} 或 JSON
		SendAt:         time.Now().Unix(),
	}

	// 4. 存储消息
	if err := s.msgRepo.Create(message); err != nil {
		log.Logger.Sugar().Errorf("Failed to save message: %v", err)
		return
	}

	// 5. 更新会话最后消息
	ctx := context.Background()
	if err := s.convRepo.UpdateLastMessage(ctx, msg.ConversationID, message); err != nil {
		log.Logger.Sugar().Warnf("Failed to update conversation: %v", err)
	}

	// 6. 推送离线消息（通过 Kafka 发给 Gateway）
	s.pushToOfflineUsers(&msg)
}

// pushToOfflineUsers 将消息推送给离线用户
func (s *MessageService) pushToOfflineUsers(msg *pb.Message) {
	// 根据消息类型决定推送目标
	var targetUsers []string

	if msg.MessageType == 1 { // 私聊
		targetUsers = []string{msg.RecipientUUID}
	} else if msg.MessageType == 2 { // 群聊
		// TODO: 查询群成员列表
		groupMembers, err := s.getGroupMembers(msg.ConversationID)
		if err != nil {
			log.Logger.Sugar().Errorf("Failed to get group members: %v", err)
			return
		}
		targetUsers = groupMembers
	}

	// 为每个目标用户发送推送消息
	for _, userUUID := range targetUsers {
		if userUUID == msg.SenderUUID { // 不推送给自己
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

		// 根据用户在线状态选择推送策略
		gatewayID := s.getUserGateway(userUUID)
		if gatewayID != "" {
			// 用户在线，推送到对应的 Gateway
			topic := "im_delivery_" + gatewayID
			s.publishToKafka(topic, pushMsg)
		} else {
			// 用户离线，存储离线消息（使用 Redis）
			s.storeOfflineMessage(userUUID, pushMsg)
		}
	}
}

// getUserGateway 获取用户当前连接的网关ID
func (s *MessageService) getUserGateway(userUUID string) string {
	ctx := context.Background()
	gatewayID, err := s.redis.Get(ctx, "user_gateway:"+userUUID).Result()
	if err != nil {
		return ""
	}
	return gatewayID
}

// publishToKafka 发布消息到 Kafka
func (s *MessageService) publishToKafka(topic string, msg *pb.Message) {
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
func (s *MessageService) storeOfflineMessage(userUUID string, msg *pb.Message) {
	ctx := context.Background()
	msgData, err := proto.Marshal(msg)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to marshal offline message: %v", err)
		return
	}

	key := "offline_msg:" + userUUID
	err = s.redis.LPush(ctx, key, string(msgData)).Err()
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to store offline message: %v", err)
	}

	// 设置过期时间（7天）
	s.redis.Expire(ctx, key, time.Hour*24*7)
}

// getGroupMembers 获取群成员列表
func (s *MessageService) getGroupMembers(conversationID string) ([]string, error) {
	// 通过会话ID（这里可能是群号）获取群成员UUID列表
	userUUIDs, err := s.groupRepo.GetGroupMemberUUIDs(conversationID)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to get group member UUIDs: %v", err)
		return []string{}, err
	}

	log.Logger.Sugar().Debugf("Found %d group members for conversation: %s", len(userUUIDs), conversationID)
	return userUUIDs, nil
}

// SyncOfflineMessages 用户上线时同步离线消息
func (s *MessageService) SyncOfflineMessages(userUUID string) error {
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
	topic := "im_delivery_" + gatewayID
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
func (s *MessageService) GetMessageHistory(conversationID string, limit, offset int) ([]*model.Message, error) {
	return s.msgRepo.GetByConversation(conversationID, limit, offset)
}

// MarkMessagesAsRead 标记消息为已读
func (s *MessageService) MarkMessagesAsRead(msgIDs []string) error {
	return s.msgRepo.MarkAsRead(msgIDs)
}

// ProcessSyncRequest 处理离线消息同步请求
func (s *MessageService) ProcessSyncRequest(kafkaMsg kafka.Message) {
	var syncRequest map[string]interface{}
	if err := json.Unmarshal(kafkaMsg.Value, &syncRequest); err != nil {
		log.Logger.Sugar().Errorf("Failed to unmarshal sync request: %v", err)
		return
	}

	action, ok := syncRequest["action"].(string)
	if !ok || action != "sync_offline" {
		log.Logger.Sugar().Warnf("Unknown sync request action: %v", action)
		return
	}

	userUUID, ok := syncRequest["user_uuid"].(string)
	if !ok {
		log.Logger.Sugar().Errorf("Invalid user_uuid in sync request")
		return
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
func (s *MessageService) unpackProtoBody(contentType int32, anyBody *anypb.Any) (any, error) {
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
		return model.FileAttachment{
			URL:      fileBody.Url,
			FileName: fileBody.FileName,
			Size:     fileBody.Size,
			MimeType: fileBody.MimeType,
		}, nil
	default:
		return nil, errors.New("unknown content type")
	}
}
