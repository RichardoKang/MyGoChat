package relation

import (
	"context"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) JoinGroupRelation(ctx context.Context, userUUID string, groupUUID string) error {
	return s.repo.JoinGroupRelation(ctx, userUUID, groupUUID)
}

func (s *Service) CreateFriendRelation(ctx context.Context, userUUID string, friendUUID string) error {
	return s.repo.CreateFriendRelation(ctx, userUUID, friendUUID)
}

func (s *Service) ListUserRelation(ctx context.Context, userUUID string) ([]*Relation, error) {
	return s.repo.ListUserRelation(ctx, userUUID)
}
