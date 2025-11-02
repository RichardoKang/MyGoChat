package service

import (
	"MyGoChat/internal/logic/data"
)

type MessageService struct {
	data *data.Data
}

func NewMessageService(data *data.Data) *MessageService {
	return &MessageService{data: data}
}
