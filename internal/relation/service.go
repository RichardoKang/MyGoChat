package relation

import (
	"MyGoChat/internal/group"
	"MyGoChat/internal/user"
	"MyGoChat/internal/util"
	"context"
)

// ConversationCreator 会话创建接口，避免循环依赖
type ConversationCreator interface {
	CreateConversation(ctx context.Context, id string, convType int) error
}

type Service struct {
	repo        Repository
	userRepo    user.Repository     // 注入 User 能力
	groupRepo   group.Repository    // 注入 Group 能力
	convCreator ConversationCreator // 会话创建能力
}

func NewService(repo Repository, userRepo user.Repository, groupRepo group.Repository, convCreator ConversationCreator) *Service {
	return &Service{
		repo:        repo,
		userRepo:    userRepo,
		groupRepo:   groupRepo,
		convCreator: convCreator,
	}
}

func (s *Service) JoinGroupRelation(ctx context.Context, userName string, groupNumber string) error {
	userUUID, err := s.userRepo.GetUUIDByUsername(ctx, userName)
	if err != nil {
		return err
	}

	groupUUID, err := s.groupRepo.GetUUIDByNumber(ctx, groupNumber)
	if err != nil {
		return err
	}

	return s.repo.JoinGroupRelation(ctx, userUUID, groupUUID)
}

func (s *Service) CreateFriendRelation(ctx context.Context, userA, userB string) error {
	userAUUID, err := s.userRepo.GetUUIDByUsername(ctx, userA)
	if err != nil {
		return err
	}

	userBUUID, err := s.userRepo.GetUUIDByUsername(ctx, userB)
	if err != nil {
		return err
	}

	// 1. 创建好友关系
	if err := s.repo.CreateFriendRelation(ctx, userAUUID, userBUUID); err != nil {
		return err
	}

	// 2. 自动创建私聊会话（如果会话创建器存在）
	if s.convCreator != nil {
		convID := util.GetPrivateConversationID(userAUUID, userBUUID)
		// 忽略错误（会话可能已存在）
		_ = s.convCreator.CreateConversation(ctx, convID, 1) // 1 = 私聊
	}

	return nil
}

func (s *Service) ListUserRelation(ctx context.Context, userName string) ([]*Relation, error) {
	userUUID, err := s.userRepo.GetUUIDByUsername(ctx, userName)
	if err != nil {
		return nil, err
	}

	return s.repo.ListUserRelation(ctx, userUUID)
}
