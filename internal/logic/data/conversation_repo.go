package data

import (
	"MyGoChat/internal/model"
	"context"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type IConversationRepo interface {
	GetConversationsByUserID(ctx context.Context, userID string, limit int64) ([]*model.Conversation, error)
	GetOrCreateConversation(ctx context.Context, participants []string, conversationType int, groupNumber ...string) (*model.Conversation, error)
	UpdateLastMessage(ctx context.Context, conversationID string, message *model.Message) error
	CreateConversation(ctx context.Context, conv *model.Conversation) error
	GetConversationByGroupNumber(ctx context.Context, groupNumber string) (*model.Conversation, error)
}

type conversationRepo struct {
	coll *mongo.Collection
}

func NewConversationRepo(data *Data) IConversationRepo {
	coll := data.Mdb.Collection("conversations")

	// 创建索引
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "participants", Value: 1},
				{Key: "lastMessageTimestamp", Value: -1},
			},
		},
	}

	// 创建复合索引, 提高查询效率
	_, err := coll.Indexes().CreateMany(context.Background(), indexes)
	if err != nil {
		panic(err)
	}

	return &conversationRepo{
		coll: coll,
	}
}

func (r *conversationRepo) GetConversationsByUserID(ctx context.Context, userID string, limit int64) ([]*model.Conversation, error) {

	var conversations []*model.Conversation
	filter := bson.M{"participants": userID}

	opts := options.Find().
		SetLimit(limit).
		SetSort(bson.D{{"lastMessageTimestamp", -1}})

	// 使用 r.coll (仓储的集合) 和传入的 ctx
	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	if err = cursor.All(ctx, &conversations); err != nil {
		return nil, err
	}

	return conversations, nil
}

// GetOrCreateConversation 获取或创建会话
func (r *conversationRepo) GetOrCreateConversation(ctx context.Context, participants []string, conversationType int, groupNumber ...string) (*model.Conversation, error) {
	// 对参与者排序，确保一致性
	sort.Strings(participants)

	// 构建查询过滤器
	filter := bson.M{
		"participants": bson.M{"$all": participants, "$size": len(participants)},
		"type":         conversationType,
	}

	// 如果是群聊且提供了群号，添加群号过滤条件
	if conversationType == 2 && len(groupNumber) > 0 && groupNumber[0] != "" {
		filter["groupNumber"] = groupNumber[0]
	}

	var conv model.Conversation
	err := r.coll.FindOne(ctx, filter).Decode(&conv)
	if err == nil {
		return &conv, nil // 找到现有会话
	}

	if err != mongo.ErrNoDocuments {
		return nil, err // 查询出错
	}

	// 不存在，创建新会话
	newConv := &model.Conversation{
		ID:                   primitive.NewObjectID(),
		Participants:         participants,
		Type:                 conversationType,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
		LastMessageTimestamp: time.Now().Unix(),
	}

	// 如果是群聊，设置群号
	if conversationType == 2 && len(groupNumber) > 0 && groupNumber[0] != "" {
		newConv.GroupNumber = groupNumber[0]
	}

	_, err = r.coll.InsertOne(ctx, newConv)
	if err != nil {
		return nil, err
	}

	return newConv, nil
}

// UpdateLastMessage 更新会话的最后一条消息
func (r *conversationRepo) UpdateLastMessage(ctx context.Context, conversationID string, message *model.Message) error {
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

	_, err = r.coll.UpdateByID(ctx, objID, update)
	return err
}

// CreateConversation 创建新会话
func (r *conversationRepo) CreateConversation(ctx context.Context, conv *model.Conversation) error {
	_, err := r.coll.InsertOne(ctx, conv)
	return err
}

// GetConversationByGroupNumber 通过群号获取会话
func (r *conversationRepo) GetConversationByGroupNumber(ctx context.Context, groupNumber string) (*model.Conversation, error) {
	filter := bson.M{
		"groupNumber": groupNumber,
		"type":        2, // 确保是群聊类型
	}

	var conv model.Conversation
	err := r.coll.FindOne(ctx, filter).Decode(&conv)
	if err != nil {
		return nil, err
	}

	return &conv, nil
}
