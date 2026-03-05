package group

import (
	"MyGoChat/chat/internal/user"
	"MyGoChat/pkg/common/response"
	"MyGoChat/pkg/log"
	"fmt"
	"time"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"golang.org/x/net/context"
)

type MemberAdder interface {
	JoinGroupRelation(ctx context.Context, userUUID, groupUUID string) error
}

type Service struct {
	repo     Repository
	userRepo user.Repository
	redis    *redis.Client
	relRepo  MemberAdder
}

func NewService(repo Repository, userRepo user.Repository, relRepo MemberAdder) *Service {
	return &Service{repo: repo, userRepo: userRepo, relRepo: relRepo}
}

func (s *Service) CreateGroup(group *Group, adminUserUuid string) error {

	adminUser, err := s.userRepo.GetUserByUuid(adminUserUuid)
	if err != nil {
		logger.Error("CreateGroup: admin user not found")
		return err
	}

	group.Uuid = uuid.New().String()
	group.GroupNumber, err = s.generateGroupNumber(context.Background())
	if err != nil {
		logger.Error("CreateGroup: failed to generate group number")
		return err
	}
	group.CreatedAt = time.Now()

	// 创建群组记录
	if err := s.repo.CreateGroup(group, adminUser); err != nil {
		logger.Error("CreateGroup: failed to create group")
		return err
	}

	// 将创建者添加为群成员
	if err := s.relRepo.JoinGroupRelation(context.Background(), adminUser.Uuid, group.Uuid); err != nil {
		logger.Error("CreateGroup: failed to add admin user to group members")
		return err
	}

	logger.Info("Group created successfully", log.Any("group", group), log.Any("adminUser", adminUser))
	return nil
}

func (s *Service) GetMyGroups(userUuid string) ([]response.GroupResponse, error) {
	user, err := s.userRepo.GetUserByUuid(userUuid)
	if err != nil {
		logger.Error("GetUserGroups: user not found")
		return nil, err
	}

	groups, err := s.repo.GetMyGroups(user.ID)
	if err != nil {
		logger.Error("GetUserGroups: failed to get user groups")
		return nil, err
	}

	logger.Info("GetUserGroups: fetched user groups successfully", log.Any("groups", groups))
	return groups, nil
}

func (s *Service) GetGroupByGroupNumber(groupNumber string) (response.GroupResponse, error) {
	group, err := s.repo.GetGroupByGroupNumber(groupNumber)
	if err != nil {
		logger.Error("GetGroupByGroupNumber: group not found")
		return response.GroupResponse{}, err
	}

	groupResponse := response.GroupResponse{
		Uuid:      group.Uuid,
		Name:      group.Name,
		CreatedAt: group.CreatedAt,
	}

	return groupResponse, nil
}

func (s *Service) generateGroupNumber(ctx context.Context) (string, error) {
	// 1. 利用 Redis 获取自增 ID (基数)
	seqId, err := s.redis.Incr(ctx, "sys:group_seq").Result()
	if err != nil {
		return "", err
	}

	// 2. 基于自增 ID 生成群号
	baseNum := int64(114514)
	finalNum := baseNum + seqId

	return fmt.Sprintf("%d", finalNum), nil
}
