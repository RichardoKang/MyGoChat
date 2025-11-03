package service

import (
	"MyGoChat/internal/logic/data"
	"MyGoChat/internal/model"
	"context"
)

type ConversationService struct {
	repo data.IConversationRepo
}

func NewConversationService(repo data.IConversationRepo) *ConversationService {
	return &ConversationService{repo: repo}
}

func (s *ConversationService) GetConversationsByUserID(ctx context.Context, userID string, limit int64) ([]*model.Conversation, error) {
	return s.repo.GetConversationsByUserID(ctx, userID, limit)
}
