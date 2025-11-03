package service

import (
	pb "MyGoChat/api/v1"
	"MyGoChat/internal/logic/data"
)

type MessageService struct {
	data *data.Data
}

func NewMessageService(data *data.Data) *MessageService {
	return &MessageService{data: data}
}

func (s *MessageService) SaveMessage(msg *pb.Message) error {

	mdb := s.data.GetMongoDB()
	collection := mdb.Collection("messages")

	// TODO: Convert pb.Message to a MongoDB document format

	return nil
}
func (s *MessageService) ProcessMessage(msg *pb.Message) string {
	return "processed"
}
