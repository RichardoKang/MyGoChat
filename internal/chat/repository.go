package chat

import (
	"MyGoChat/internal/platform"
	"MyGoChat/pkg/log"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type Repository interface {
	CreateMsg(ctx context.Context, msg *Message) error
	GetByConversation(ctx context.Context, convID string, limit, offset int) ([]*Message, error)
	MarkAsRead(ctx context.Context, msgIDs []string) error

	CreateConversation(ctx context.Context, conv *Conversation) error
	GetConversationByGroupNumber(ctx context.Context, groupNumber string) (*Conversation, error)
	GetConversationsByUserID(ctx context.Context, userID string, limit int64) ([]*Conversation, error)
	GetConversationByID(ctx context.Context, conversationID string) (*Conversation, error)
	UpdateLastMessage(ctx context.Context, conversationID string, message *Message) error
}

type repository struct {
	msgColl  *mongo.Collection
	convColl *mongo.Collection
}

func NewChatRepo(data *platform.Data) Repository {
	r := &repository{
		msgColl:  data.Mdb.Collection("messages"),
		convColl: data.Mdb.Collection("conversations"),
	}

	r.initConversationIndexes()

	return r
}

func (r *repository) initConversationIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "participants", Value: 1},
				{Key: "lastMessageTimestamp", Value: -1},
			},
		},
		// 如果有其他索引也加在这里
	}

	// 对 convColl 创建索引
	if _, err := r.convColl.Indexes().CreateMany(ctx, indexes); err != nil {
		log.Logger.Error("Failed to create conversation indexes", zap.Error(err))
	}
}

func (r *repository) CreateMsg(ctx context.Context, msg *Message) error {
	_, err := r.msgColl.InsertOne(ctx, msg)
	return err
}

func (r *repository) GetByConversation(ctx context.Context, convID string, limit, offset int) ([]*Message, error) {

	// TODO: 实现分页查询和排序
	cursor, err := r.msgColl.Find(ctx, map[string]interface{}{
		"conversationID": convID,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []*Message
	for cursor.Next(ctx) {
		var msg Message
		if err := cursor.Decode(&msg); err != nil {
			continue
		}
		messages = append(messages, &msg)
	}

	return messages, nil
}

func (r *repository) MarkAsRead(ctx context.Context, msgIDs []string) error {
	// TODO: 实现消息已读标记
	return nil
}

func (r *repository) GetConversationsByUserID(ctx context.Context, userID string, limit int64) ([]*Conversation, error) {

	var conversations []*Conversation
	filter := bson.M{"participants": userID}

	opts := options.Find().
		SetLimit(limit).
		SetSort(bson.D{{"lastMessageTimestamp", -1}})

	// 使用 r.coll (仓储的集合) 和传入的 ctx
	cursor, err := r.convColl.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	if err = cursor.All(ctx, &conversations); err != nil {
		return nil, err
	}

	return conversations, nil
}

// UpdateLastMessage 更新会话的最后一条消息
func (r *repository) UpdateLastMessage(ctx context.Context, conversationID string, message *Message) error {
	objID, err := primitive.ObjectIDFromHex(conversationID)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"lastMessage":          message.Body,
			"lastMessageTimestamp": message.SendAt,
			"updatedAt":            time.Now(),
		},
	}

	_, err = r.convColl.UpdateByID(ctx, objID, update)
	return err
}

// CreateConversation 创建新会话
func (r *repository) CreateConversation(ctx context.Context, conv *Conversation) error {
	_, err := r.convColl.InsertOne(ctx, conv)
	return err
}

// GetConversationByGroupNumber 通过群号获取会话
func (r *repository) GetConversationByGroupNumber(ctx context.Context, groupNumber string) (*Conversation, error) {
	filter := bson.M{
		"groupNumber": groupNumber,
		"type":        2, // 确保是群聊类型
	}

	var conv Conversation
	err := r.convColl.FindOne(ctx, filter).Decode(&conv)
	if err != nil {
		return nil, err
	}

	return &conv, nil
}

func (r *repository) GetConversationByID(ctx context.Context, conversationID string) (*Conversation, error) {
	objID, err := primitive.ObjectIDFromHex(conversationID)
	if err != nil {
		return nil, err
	}

	var conv Conversation
	// r.convColl 是 conversations 表的集合
	err = r.convColl.FindOne(ctx, bson.M{"_id": objID}).Decode(&conv)
	if err != nil {
		return nil, err
	}
	return &conv, nil
}
