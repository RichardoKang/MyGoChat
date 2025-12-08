package group

import (
	"MyGoChat/internal/user"
	"MyGoChat/pkg/common/response"
	"MyGoChat/pkg/log"
	"fmt"
	"time"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"golang.org/x/net/context"
)

type IChatService interface {
	CreateGroupConversation(ctx context.Context, groupUUID string) error
}

type Service struct {
	repo        Repository
	userRepo    user.Repository
	chatService IChatService
	redis       *redis.Client
}

func NewService(repo Repository, userRepo user.Repository, chatService IChatService) *Service {
	return &Service{repo: repo, userRepo: userRepo, chatService: chatService}
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
	// 创建群聊会话
	if err := s.chatService.CreateGroupConversation(context.Background(), group.Uuid); err != nil {
		logger.Error("CreateGroup: failed to create group conversation")
		return fmt.Errorf("failed to create group conversation: %w", err)
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

func (s *Service) JoinGroup(userUuid string, groupNumber string, nickname string) error {
	user, err := s.userRepo.GetUserByUuid(userUuid)
	if err != nil {
		logger.Error("JoinGroup: user not found")
		return err
	}

	group, err := s.repo.GetGroupByGroupNumber(groupNumber)
	if err != nil {
		logger.Error("JoinGroup: group not found")
		return err
	}

	if err := s.repo.JoinGroup(user, group, nickname); err != nil {
		logger.Error("JoinGroup: failed to join group")
		return err
	}

	logger.Info("JoinGroup: user joined group successfully", log.Any("userUuid", userUuid), log.Any("groupNumber", groupNumber))
	return nil
}

// GetGroupMembersByGroupNumber 通过群号获取群成员详细信息
func (s *Service) GetGroupMembersByGroupNumber(groupNumber string) ([]response.GroupMemberResponse, error) {

	group, err := s.repo.GetGroupByGroupNumber(groupNumber)
	if err != nil {
		logger.Error("GetGroupMembersByGroupNumber: group not found", log.String("groupNumber", groupNumber))
		return nil, err
	}

	members, err := s.repo.GetGroupMembers(groupNumber)
	if err != nil {
		logger.Error("GetGroupMembersByGroupNumber: failed to get group members", log.Any("groupId", group.ID))
		return nil, err
	}

	logger.Info("GetGroupMembersByGroupNumber: fetched group members successfully",
		log.String("groupNumber", groupNumber),
		log.Int("memberCount", len(members)))
	return members, nil
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
