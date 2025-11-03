package data

import (
	pb "MyGoChat/api/v1"
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

type IMessageRepo interface {
	SaveMessage(msg *pb.Message) error
	CreateMessage(ctx context.Context, msg *pb.Message) error
}

type messageRepo struct {
	coll *mongo.Collection
}

func NewMessageRepo(data *Data) IMessageRepo {
	coll := data.mdb.Collection("messages")

	return &messageRepo{
		coll: coll,
	}
}
func (r *messageRepo) CreateMessage(ctx context.Context, msg *pb.Message) error {
	//TODO implement me
	panic("implement me")
}

func (r *messageRepo) SaveMessage(msg *pb.Message) error {
	_, err := r.coll.InsertOne(nil, msg)
	return err
}
