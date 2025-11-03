package service

import (
	pb "MyGoChat/api/v1"
	"MyGoChat/internal/logic/data"
	"MyGoChat/internal/model"
	myKafka "MyGoChat/pkg/kafka"
	"MyGoChat/pkg/log"
	"context"
	"errors"

	"github.com/go-redis/redis/v8"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type MessageService struct {
	msgRepo  data.IMessageRepo
	convRepo data.IConversationRepo
	redis    *redis.Client
	producer *myKafka.Producer
}

func NewMessageService(msgRepo data.IMessageRepo, convRepo data.IConversationRepo, rdb *redis.Client, producer *myKafka.Producer) *MessageService {
	return &MessageService{
		msgRepo:  msgRepo,
		convRepo: convRepo,
		redis:    rdb,
		producer: producer,
	}
}

func (s *MessageService) ProcessMessage(kafkaMsg kafka.Message) {
	ctx := context.Background()

	// 1. 反序列化
	var msg pb.Message
	if err := proto.Unmarshal(kafkaMsg.Value, &msg); err != nil {
		log.Logger.Sugar().Errorf("ProcessMessage: 反序列化失败: %v", err)
		return
	}

	// 2. 转换 (PB -> Mongo Model)
	// 你需要一个辅助函数来确定 ConversationID
	// convID, err := s.convRepo.FindOrCreateConversation(msg.SenderID, msg.RecipientID, msg.MessageType)

	// (我们先用 pb 里的 ConversationID)
	convObjID, _ := primitive.ObjectIDFromHex(msg.ConversationID)

	// 转换 Body (Any -> interface{})
	body, err := s.unpackProtoBody(msg.ContentType, msg.Body)
	if err != nil {
		log.Logger.Sugar().Errorf("ProcessMessage: 解包 Body 失败: %v", err)
		return
	}

	mongoMsg := &model.Message{
		ConversationID: convObjID,
		SenderID:       msg.SenderID, // 你 model 里的 SenderID 是 string
		Timestamp:      msg.Timestamp,
		ContentType:    int16(msg.ContentType),
		Body:           body,
		// ...
	}

	// 3. 存储到 MongoDB
	if err := s.msgRepo.CreateMessage(ctx, mongoMsg); err != nil {
		log.Logger.Sugar().Errorf("ProcessMessage: 存入 MongoDB 失败: %v", err)
		return
	}

	// 4. 更新会话的 'lastMessageTimestamp'
	// err = s.convRepo.UpdateLastMessageTimestamp(ctx, convObjID, msg.Timestamp)

	// 5. 扇出 (Fan-out)
	// participants, err := s.convRepo.GetParticipants(ctx, convObjID)
	// (现在，我们暂时假设 participants 列表就是 SenderID 和 RecipientID)

	// 你的 pb.Message 结构是旧的，它有 RecipientID
	// 我们需要根据 messageType 分发
	var recipientIDs []string

	if msg.MessageType == 1 { // 私聊
		recipientIDs = []string{msg.SenderID, msg.RecipientID} // 发送者和接收者
	} else if msg.MessageType == 2 { // 群聊
		// ... 你需要实现 GetGroupMembers ...
		// groupMembers, err := s.groupRepo.GetGroupMemberIDs(msg.RecipientID)
		// recipientIDs = groupMembers
	}

	// 6. 查询 Redis 并推送到 Kafka 投递队列
	for _, userID := range recipientIDs {
		// (你的 user.go 里的 ID 是 uint，但 pb 里是 string/int32，这里需要统一)
		// 我们假设 userID 是 string
		gatewayID, err := s.redis.HGet(ctx, "online_users", userID).Result()
		if err == redis.Nil {
			// 用户不在线，跳过
			continue
		} else if err != nil {
			log.Logger.Sugar().Errorf("ProcessMessage: 查询 Redis 失败: %v", err)
			continue
		}

		// 用户在线，推送到该网关的专属 topic
		deliveryTopic := "im_delivery_" + gatewayID
		s.producer.SendMessage(deliveryTopic, kafkaMsg.Value)
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
