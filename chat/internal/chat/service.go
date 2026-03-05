package chat

import (
	pb "MyGoChat/pkg/api/v1"
	"MyGoChat/chat/internal/group"
	"MyGoChat/chat/internal/relation"
	"MyGoChat/chat/internal/user"
	"MyGoChat/chat/internal/util"
	"MyGoChat/pkg/common/request"
	"MyGoChat/pkg/config"
	myKafka "MyGoChat/pkg/kafka"
	"MyGoChat/pkg/log"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type Service struct {
	repo      Repository
	relRepo   relation.Repository
	groupRepo group.Repository
	userRepo  user.Repository
	redis     *redis.Client
	producer  *myKafka.Producer
}

// ConversationCreatorAdapter 适配器，实现 relation.ConversationCreator 接口
type ConversationCreatorAdapter struct {
	repo Repository
}

// NewConversationCreator 创建会话创建器
func NewConversationCreator(repo Repository) *ConversationCreatorAdapter {
	return &ConversationCreatorAdapter{repo: repo}
}

// CreateConversation 实现 relation.ConversationCreator 接口
func (a *ConversationCreatorAdapter) CreateConversation(ctx context.Context, id string, convType int) error {
	conv := &Conversation{
		ID:        id,
		Type:      convType,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return a.repo.CreateConversation(ctx, conv)
}

func NewService(
	repo Repository,
	relRepo relation.Repository,
	groupRepo group.Repository, // 【新增】
	userRepo user.Repository,
	rdb *redis.Client,
	producer *myKafka.Producer,
) *Service {
	return &Service{
		repo:      repo,
		relRepo:   relRepo,
		groupRepo: groupRepo, // 【赋值】
		userRepo:  userRepo,  // 【赋值】
		redis:     rdb,
		producer:  producer,
	}
}

// GetProducer 返回 Kafka Producer 实例（供 Handler 使用）
func (s *Service) GetProducer() *myKafka.Producer {
	return s.producer
}

// EnqueueMessage 将消息发送到 Kafka Ingest Topic
func (s *Service) EnqueueMessage(ctx context.Context, msg *pb.Message) error {
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	// 发送到 Ingest Topic
	key := []byte(msg.ConversationID)
	return s.producer.SendMessageWithKey(config.GetConfig().Kafka.Topics.Ingest, key, msgBytes)
}

// SendMessage 是发送消息的唯一入口
// 它负责：1. 号码转UUID  2. 生成确定性会话ID  3. 构造消息  4. 发送Kafka
func (s *Service) SendMessage(ctx context.Context, senderUUID string, req *request.SendMessageRequest) (*pb.Message, error) {
	var (
		targetUUID     string
		conversationID string
		err            error
	)

	// 1. 核心逻辑：号码转 UUID + 计算会话 ID
	if req.MessageType == 1 {
		// --- 私聊 ---
		// 前端传的是 Username，我们需要 UserUUID
		targetUUID, err = s.userRepo.GetUUIDByUsername(ctx, req.TargetName)
		if err != nil {
			return nil, errors.New("用户不存在")
		}
		// 算法生成确定性 ID: Hash(Sort(A, B))
		conversationID = util.GetPrivateConversationID(senderUUID, targetUUID)

	} else if req.MessageType == 2 {
		// --- 群聊 ---
		// 前端传的是 GroupNumber，我们需要 GroupUUID
		targetUUID, err = s.groupRepo.GetUUIDByNumber(ctx, req.TargetName)
		if err != nil {
			return nil, errors.New("群组不存在")
		}
		// 确定性 ID: GroupUUID 本身就是 会话ID
		conversationID = targetUUID
	} else {
		return nil, errors.New("不支持的消息类型")
	}

	// 2. 获取发送者用户名
	senderName, _ := s.userRepo.GetUsernameByUUID(ctx, senderUUID)

	// 3. 构造 Protobuf 消息
	// 注意：这里全是 UUID，没有任何 Number/Username
	msg := &pb.Message{
		Id:             primitive.NewObjectID().Hex(), // 生成消息 ID
		ConversationID: conversationID,
		SenderUUID:     senderUUID,
		SenderName:     senderName, // 发送者用户名
		RecipientUUID:  targetUUID, // 私聊是对方UUID，群聊是群UUID
		MessageType:    req.MessageType,
		ContentType:    req.ContentType,
		SendAt:         time.Now().Unix(),
	}

	// 4. 处理消息体 (Any)
	var errPack error
	msg.Body, errPack = s.packProtoBody(req.ContentType, req.Body)
	if errPack != nil {
		return nil, errPack
	}
	// 5. 发送到 Kafka (Ingest Topic)
	// 消费者收到后会负责：存 Mongo(Upsert) + 推送给用户
	if err := s.EnqueueMessage(ctx, msg); err != nil {
		return nil, err
	}

	// 5. 返回结果
	return msg, nil
}

// ProcessMessage 是 Logic 服务的核心消息处理方法
// 该方法由 Kafka Consumer 调用，处理从 Gateway 发送过来的消息
//
// 消息处理流程：
// 1. 反序列化：将 Kafka 消息体解析为 Protobuf Message 结构
// 2. 验证消息：检查必要字段（SenderUUID, ConversationID 等）
// 3. 解包消息体：将 google.protobuf.Any 解包为具体类型
// 4. 持久化：将消息存储到 MongoDB
// 5. 更新会话：更新对应会话的最后消息信息
// 6. 消息投递：
//   - 在线用户：查询 Redis 路由表，投递到对应 Gateway 的 Delivery Topic
//   - 离线用户：存储到 Redis 离线消息队列，等待用户上线时同步
func (s *Service) ProcessMessage(ctx context.Context, kafkaMsg kafka.Message) error {
	// Step 1: 反序列化 Protobuf 消息
	// Gateway 发送过来的是序列化后的 pb.Message 字节数组
	var msg pb.Message
	if err := proto.Unmarshal(kafkaMsg.Value, &msg); err != nil {
		log.Logger.Sugar().Errorf("Failed to unmarshal message: %v", err)
		return err
	}

	log.Logger.Sugar().Infof("Processing message: sender=%s, recipient=%s, type=%d",
		msg.SenderUUID, msg.RecipientUUID, msg.MessageType)

	// Step 2: 如果 SenderName 为空，从数据库获取
	if msg.SenderName == "" && msg.SenderUUID != "" {
		if senderName, err := s.userRepo.GetUsernameByUUID(ctx, msg.SenderUUID); err == nil {
			msg.SenderName = senderName
		}
	}

	// Step 3: 验证消息有效性
	// 确保必要字段存在，防止无效消息进入后续处理流程
	if err := s.validateMessage(&msg); err != nil {
		log.Logger.Sugar().Errorf("Invalid message: %v", err)
		return err
	}

	// Step 4: 解包消息体 (google.protobuf.Any -> 具体类型)
	// 根据 ContentType 将 Any 类型解包为 TextBody/FileAttachment 等具体类型
	body, err := s.unpackProtoBody(msg.ContentType, msg.Body)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to unpack body: %v", err)
		return err
	}

	// Step 5: 构建 MongoDB 数据模型
	message := &Message{
		ConversationID: msg.ConversationID, // 直接使用字符串
		SenderUUID:     msg.SenderUUID,
		SenderName:     msg.SenderName,
		ContentType:    int16(msg.ContentType),
		Body:           body,
		SendAt:         time.Now().Unix(),
	}

	// Step 6: 持久化消息到 MongoDB
	// 消息必须先入库，确保数据不丢失，即使后续投递失败也可以通过离线消息恢复
	if err := s.repo.CreateMsg(ctx, message); err != nil {
		log.Logger.Sugar().Errorf("Failed to save message: %v", err)
		return err
	}

	// Step 6: 更新会话的最后消息（用于聊天列表展示）
	if err := s.repo.UpdateLastMessage(ctx, msg.ConversationID, message); err != nil {
		log.Logger.Sugar().Warnf("Failed to update conversation: %v", err)
		// 这里不返回错误，因为消息已经存储成功，会话更新失败不影响消息投递
	}

	// Step 7: 投递消息给目标用户
	// 根据用户在线状态决定是实时推送还是存储为离线消息
	s.deliverMessage(ctx, &msg)

	log.Logger.Sugar().Infof("Message processed successfully: id=%s", msg.Id)
	return nil
}

// validateMessage 验证消息的有效性
// 确保核心字段存在，防止无效消息进入后续处理流程
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

// deliverMessage 将消息推送给目标用户
// 这是消息投递的核心方法，负责实现"消息到达"的最后一公里
//
// 投递策略：
// 1. 私聊 (MessageType=1): 目标用户就是 RecipientUUID
// 2. 群聊 (MessageType=2): 从关系表查询所有群成员，逐一投递（排除发送者）
//
// 路由机制：
// - 查询 Redis 键 "user_gateway:{userUUID}" 获取用户当前连接的 Gateway ID
// - 如果存在：用户在线，发送到 Kafka Topic "im_message_delivery_{gatewayID}"
// - 如果不存在：用户离线，存储到 Redis 列表 "offline_msg:{userUUID}"
func (s *Service) deliverMessage(ctx context.Context, msg *pb.Message) {
	// 根据消息类型确定推送目标列表
	var targetUsers []string

	if msg.MessageType == 1 { // 私聊：直接推送给接收者
		targetUsers = []string{msg.RecipientUUID}
		log.Logger.Sugar().Infof("Delivering private message from %s to %s", msg.SenderUUID, msg.RecipientUUID)
	} else if msg.MessageType == 2 { // 群聊：查询群成员列表
		memberUUIDs, err := s.relRepo.GetGroupMemberUUIDs(ctx, msg.ConversationID)
		if err != nil {
			log.Logger.Sugar().Errorf("deliverMessage: failed to get group members for %s: %v", msg.ConversationID, err)
			return
		}
		targetUsers = memberUUIDs
		log.Logger.Sugar().Infof("Delivering group message to %d members", len(memberUUIDs))
	}

	// 遍历目标用户列表，逐一投递
	for _, userUUID := range targetUsers {
		// 跳过发送者自己，避免收到自己发的消息
		if userUUID == msg.SenderUUID {
			continue
		}

		// 构造推送消息（复制原消息，更新接收者字段）
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
			RecipientUUID:  userUUID, // 设置当前推送的目标用户
		}

		// 【核心路由逻辑】根据用户在线状态决定投递方式
		// Redis 中存储了 user_gateway:{uuid} -> gatewayID 的映射关系
		// 这个映射由 Gateway 的 Hub 在用户连接时写入，断开时删除
		gatewayID := s.getUserGateway(userUUID)
		if gatewayID != "" {
			// 用户在线：通过 Kafka 投递到对应 Gateway
			// Topic 格式: im_message_delivery_{gatewayID}
			// Gateway 会订阅自己的 Delivery Topic，收到消息后通过 WebSocket 推送给客户端
			cfg := config.GetConfig()
			topic := cfg.Kafka.Topics.Delivery + gatewayID
			s.publishToKafka(topic, pushMsg)
			log.Logger.Sugar().Infof("Delivered message to online user %s via gateway %s", userUUID, gatewayID)
		} else {
			// 用户离线：存储到 Redis 离线消息队列
			// 当用户重新上线时，Gateway 会发送 sync_offline 请求
			// Logic 服务收到请求后会调用 SyncOfflineMessages 将消息推送给用户
			s.storeOfflineMessage(userUUID, pushMsg)
			log.Logger.Sugar().Infof("Stored offline message for user %s", userUUID)
		}
	}
}

// getUserGateway 获取用户当前连接的网关ID
// 这是消息路由的核心：通过 Redis 查询用户当前在哪个 Gateway 上在线
//
// 路由表结构：
// - Key: "user_gateway:{userUUID}"
// - Value: gatewayID (如 "gateway-default", "gateway-1" 等)
// - 由 Gateway 的 Hub 在用户连接时 SET，断开时 DEL
//
// 返回值：
// - 非空字符串：用户在线，返回 gatewayID
// - 空字符串：用户离线（Redis 中没有该 key）
func (s *Service) getUserGateway(userUUID string) string {
	ctx := context.Background()
	gatewayID, err := s.redis.Get(ctx, "user_gateway:"+userUUID).Result()
	if err != nil {
		// redis.Nil 表示 key 不存在，即用户离线
		return ""
	}
	return gatewayID
}

// publishToKafka 发布消息到 Kafka
// 使用 ConversationID 作为 Key 保证同一会话的消息有序
func (s *Service) publishToKafka(topic string, msg *pb.Message) {
	msgData, err := proto.Marshal(msg)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to marshal message: %v", err)
		return
	}

	// 使用 ConversationID 作为 Key，确保同一会话的消息发送到同一分区
	key := []byte(msg.ConversationID)
	err = s.producer.SendMessageWithKey(topic, key, msgData)
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
	ctx := context.Background()
	return s.repo.GetByConversation(ctx, conversationID, limit, offset)
}

// MarkMessagesAsRead 标记消息为已读
func (s *Service) MarkMessagesAsRead(msgIDs []string) error {
	return s.repo.MarkAsRead(context.Background(), msgIDs)
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
	return nil
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

// convertToProtobufAny 将 JSON body 转换为 google.protobuf.Any 类型
func convertToProtobufAny(contentType int32, body interface{}) (*anypb.Any, error) {
	switch contentType {
	case 1: // 文本消息
		if bodyMap, ok := body.(map[string]interface{}); ok {
			if content, exists := bodyMap["content"]; exists {
				textBody := &pb.TextBody{
					Content: content.(string),
				}
				return anypb.New(textBody)
			}
		}
		return nil, fmt.Errorf("invalid text body format, expected {\"content\": \"...\"}")

	case 2, 3, 4: // 图片、文件、语音消息
		if bodyMap, ok := body.(map[string]interface{}); ok {
			fileBody := &pb.FileAttachment{
				Url:      getStringValue(bodyMap, "url"),
				FileName: getStringValue(bodyMap, "fileName"),
				Size:     getInt64Value(bodyMap, "size"),
				MimeType: getStringValue(bodyMap, "mimeType"),
			}
			return anypb.New(fileBody)
		}
		return nil, fmt.Errorf("invalid file body format")

	default:
		return nil, fmt.Errorf("unsupported content type: %d", contentType)
	}
}

// packProtoBody 将前端传来的 JSON Body (interface{}) 打包成 Protobuf Any 类型
func (s *Service) packProtoBody(contentType int32, body interface{}) (*anypb.Any, error) {
	switch contentType {
	case 1: // 文本消息
		// JSON 解析后的 body 通常是 map[string]interface{}
		if bodyMap, ok := body.(map[string]interface{}); ok {
			if content, exists := bodyMap["content"]; exists {
				if strContent, ok := content.(string); ok {
					textBody := &pb.TextBody{
						Content: strContent,
					}
					return anypb.New(textBody)
				}
			}
		}
		return nil, fmt.Errorf("invalid text body format, expected {\"content\": \"...\"}")

	case 2, 3, 4: // 图片、文件、语音
		if bodyMap, ok := body.(map[string]interface{}); ok {
			fileBody := &pb.FileAttachment{
				Url:      s.getStringValue(bodyMap, "url"),
				FileName: s.getStringValue(bodyMap, "fileName"),
				Size:     s.getInt64Value(bodyMap, "size"),
				MimeType: s.getStringValue(bodyMap, "mimeType"),
			}
			return anypb.New(fileBody)
		}
		return nil, fmt.Errorf("invalid file body format")

	default:
		return nil, fmt.Errorf("unsupported content type: %d", contentType)
	}
}

// --- 私有辅助工具函数 ---

func (s *Service) getStringValue(m map[string]interface{}, key string) string {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func (s *Service) getInt64Value(m map[string]interface{}, key string) int64 {
	if val, exists := m[key]; exists {
		switch v := val.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case float64: // JSON 数字默认解析为 float64
			return int64(v)
		}
	}
	return 0
}

// 辅助函数：安全地从 map 中获取字符串值
func getStringValue(m map[string]interface{}, key string) string {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// 辅助函数：安全地从 map 中获取 int64 值
func getInt64Value(m map[string]interface{}, key string) int64 {
	if val, exists := m[key]; exists {
		switch v := val.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case float64:
			return int64(v)
		}
	}
	return 0
}

/***************************************************************************************************************
会话相关服务
****************************************************************************************************************/

// GetConversationsByUserID 获取用户的会话列表
// 通过 relation 表获取用户的会话ID列表，然后从 MongoDB 获取会话详情
func (s *Service) GetConversationsByUserID(ctx context.Context, userID string, limit int64) ([]*Conversation, error) {
	// 1. 从 relation 表获取用户的关系列表（包含 conversationID）
	relations, err := s.relRepo.GetUserRelationsWithConversation(ctx, userID, int(limit))
	if err != nil {
		return nil, err
	}

	if len(relations) == 0 {
		return []*Conversation{}, nil
	}

	// 2. 根据 conversationID 列表从 MongoDB 获取会话详情
	var conversations []*Conversation
	for _, rel := range relations {
		conv, err := s.repo.GetConversationByID(ctx, rel.ConversationID)
		if err != nil {
			// 会话可能还不存在（只有关系但没发过消息），创建一个空的会话对象
			conv = &Conversation{
				ID:   rel.ConversationID,
				Type: rel.Type,
			}
		}
		// 填充 TargetUUID，用于前端 WebSocket 发送消息
		conv.TargetUUID = rel.TargetUUID

		// 获取目标用户名/群名，用于前端显示
		if rel.Type == 1 { // 私聊
			if targetName, err := s.userRepo.GetUsernameByUUID(ctx, rel.TargetUUID); err == nil {
				conv.TargetName = targetName
			} else {
				conv.TargetName = rel.TargetUUID // 获取失败则显示 UUID
			}
		} else { // 群聊
			if groupName, err := s.groupRepo.GetNameByUUID(ctx, rel.TargetUUID); err == nil {
				conv.TargetName = groupName
			} else {
				conv.TargetName = rel.TargetUUID
			}
		}

		conversations = append(conversations, conv)
	}

	return conversations, nil
}

// GetPrivateConversation 获取或创建私聊会话
// 如果会话不存在，则自动创建
func (s *Service) GetPrivateConversation(ctx context.Context, userID1, userID2 string) (*Conversation, error) {
	convID := util.GetPrivateConversationID(userID1, userID2)

	// 尝试获取现有会话
	conv, err := s.repo.GetConversationByID(ctx, convID)
	if err == nil {
		return conv, nil
	}

	// 会话不存在，创建新会话
	newConv := &Conversation{
		ID:        convID,
		Type:      1, // 私聊
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateConversation(ctx, newConv); err != nil {
		// 如果是重复键错误（并发创建），尝试重新获取
		conv, getErr := s.repo.GetConversationByID(ctx, convID)
		if getErr == nil {
			return conv, nil
		}
		return nil, err
	}

	return newConv, nil
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
