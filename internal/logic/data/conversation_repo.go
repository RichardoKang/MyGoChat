package data

import (
	"MyGoChat/internal/model"
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// 1. (创建) 定义你的仓储接口
type IConversationRepo interface {
	GetConversationsByUserID(ctx context.Context, userID string, limit int64) ([]*model.Conversation, error)
	// ... (其他方法，例如 CreateConversation)
}

type conversationRepo struct {
	coll *mongo.Collection
}

// NewConversationRepo
// 3. (创建) 仓储的构造函数
// (在 data.go 的 NewData 之后，会调用这个)
func NewConversationRepo(data *Data) IConversationRepo {
	coll := data.mdb.Collection("conversations")

	// 4. 创建索引
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

	// 5. (修复) 这里使用 r.coll (仓储的集合) 和传入的 ctx
	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	if err = cursor.All(ctx, &conversations); err != nil {
		return nil, err
	}

	return conversations, nil
}
