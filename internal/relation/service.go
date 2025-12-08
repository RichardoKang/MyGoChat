package relation

import (
	"MyGoChat/internal/group"
	"MyGoChat/internal/user"
	"context"
)

type Service struct {
	repo      Repository
	userRepo  user.Repository // 注入 User 能力
	groupRepo group.Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
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

	return s.repo.CreateFriendRelation(ctx, userAUUID, userBUUID)
}

func (s *Service) ListUserRelation(ctx context.Context, userName string) ([]*Relation, error) {
	userUUID, err := s.userRepo.GetUUIDByUsername(ctx, userName)
	if err != nil {
		return nil, err
	}

	return s.repo.ListUserRelation(ctx, userUUID)
}
