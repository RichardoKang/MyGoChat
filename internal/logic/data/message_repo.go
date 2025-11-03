package data

import (
	"MyGoChat/internal/model"
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

type IMessageRepo interface {
	Create(msg *model.Message) error
	GetByConversation(convID string, limit, offset int) ([]*model.Message, error)
	MarkAsRead(msgIDs []string) error
}

type messageRepo struct {
	coll *mongo.Collection
}

func NewMessageRepo(data *Data) IMessageRepo {
	coll := data.Mdb.Collection("messages")

	return &messageRepo{
		coll: coll,
	}
}

func (r *messageRepo) Create(msg *model.Message) error {
	ctx := context.Background()
	_, err := r.coll.InsertOne(ctx, msg)
	return err
}

func (r *messageRepo) GetByConversation(convID string, limit, offset int) ([]*model.Message, error) {
	ctx := context.Background()

	// TODO: 实现分页查询和排序
	cursor, err := r.coll.Find(ctx, map[string]interface{}{
		"conversationID": convID,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []*model.Message
	for cursor.Next(ctx) {
		var msg model.Message
		if err := cursor.Decode(&msg); err != nil {
			continue
		}
		messages = append(messages, &msg)
	}

	return messages, nil
}

func (r *messageRepo) MarkAsRead(msgIDs []string) error {
	// TODO: 实现消息已读标记
	return nil
}
