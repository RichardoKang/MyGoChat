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

// GetConversationsByUserID 获取用户的会话列表
func (s *ConversationService) GetConversationsByUserID(ctx context.Context, userID string, limit int64) ([]*model.Conversation, error) {
	return s.repo.GetConversationsByUserID(ctx, userID, limit)
}

// GetOrCreatePrivateConversation 获取或创建私聊会话
func (s *ConversationService) GetOrCreatePrivateConversation(ctx context.Context, userID1, userID2 string) (*model.Conversation, error) {
	participants := []string{userID1, userID2}
	return s.repo.GetOrCreateConversation(ctx, participants, 1) // 1 = 私聊
}

// GetOrCreateGroupConversation 获取或创建群聊会话
func (s *ConversationService) GetOrCreateGroupConversation(ctx context.Context, participants []string, groupNumber string) (*model.Conversation, error) {
	// 先尝试通过群号查找现有会话
	existing, err := s.repo.GetConversationByGroupNumber(ctx, groupNumber)
	if err == nil && existing != nil {
		return existing, nil
	}

	// 如果不存在，创建新的群聊会话
	return s.repo.GetOrCreateConversation(ctx, participants, 2, groupNumber) // 2 = 群聊
}

// CreateConversation 创建新会话
func (s *ConversationService) CreateConversation(ctx context.Context, conv *model.Conversation) error {
	return s.repo.CreateConversation(ctx, conv)
}
