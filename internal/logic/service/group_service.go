package service

import (
	"MyGoChat/internal/logic/data"
	"MyGoChat/internal/model"
	"MyGoChat/pkg/common/response"
	"MyGoChat/pkg/log"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

type GroupService struct {
	repo     data.IGroupRepo
	userRepo data.IUserRepo
}

func NewGroupService(repo data.IGroupRepo, userRepo data.IUserRepo) *GroupService {
	return &GroupService{repo: repo, userRepo: userRepo}
}

func (s *GroupService) CreateGroup(group *model.Group, adminUserUuid string) error {
	logger := log.Logger

	adminUser, err := s.userRepo.GetUserByUuid(adminUserUuid)
	if err != nil {
		logger.Error("CreateGroup: admin user not found")
		return err
	}

	group.Uuid = uuid.New().String()
	// 生成一个唯一的6位群号
	maxAttempts := 100 // 防止无限循环，最多尝试100次
	var attempts int
	for attempts = 0; attempts < maxAttempts; attempts++ {
		group.GroupNumber = fmt.Sprintf("%06d", rand.New(rand.NewSource(time.Now().UnixNano())).Int31n(1000000))
		_, err := s.repo.GetGroupByGroupNumber(group.GroupNumber)
		if err != nil {
			// 找不到记录说明群号可用，跳出循环
			logger.Debug("Generated unique group number",
				log.String("groupNumber", group.GroupNumber),
				log.Int("attempts", attempts+1))
			break
		}
		// 找到了记录说明群号已存在，继续生成新的群号
		logger.Debug("Group number already exists, generating new one",
			log.String("groupNumber", group.GroupNumber),
			log.Int("attempt", attempts+1))
	}

	if attempts >= maxAttempts {
		logger.Error("Failed to generate unique group number after max attempts",
			log.Int("maxAttempts", maxAttempts))
		return fmt.Errorf("failed to generate unique group number after %d attempts", maxAttempts)
	}
	group.CreatedAt = time.Now()

	if err := s.repo.CreateGroup(group, adminUser); err != nil {
		logger.Error("CreateGroup: failed to create group")
		return err
	}

	logger.Info("Group created successfully", log.Any("group", group), log.Any("adminUser", adminUser))
	return nil
}

func (s *GroupService) GetMyGroups(userUuid string) ([]response.GroupResponse, error) {
	logger := log.Logger

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

func (s *GroupService) GetGroupByGroupNumber(groupNumber string) (response.GroupResponse, error) {
	logger := log.Logger

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

func (s *GroupService) JoinGroup(userUuid string, groupNumber string, nickname string) error {
	logger := log.Logger

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

func (s *GroupService) GetGroupMembers(groupId uint) ([]response.GroupMemberResponse, error) {
	logger := log.Logger

	members, err := s.repo.GetGroupMembers(groupId)
	if err != nil {
		logger.Error("GetGroupMembers: failed to get group members")
		return nil, err
	}

	logger.Info("GetGroupMembers: fetched group members successfully", log.Any("members", members))
	return members, nil
}

// GetGroupMemberUUIDs 获取群组成员的UUID列表
func (s *GroupService) GetGroupMemberUUIDs(groupNumber string) ([]string, error) {
	logger := log.Logger

	uuids, err := s.repo.GetGroupMemberUUIDs(groupNumber)
	if err != nil {
		logger.Error("GetGroupMemberUUIDs: failed to get group member UUIDs", log.String("groupNumber", groupNumber))
		return nil, err
	}

	logger.Info("GetGroupMemberUUIDs: fetched group member UUIDs successfully",
		log.String("groupNumber", groupNumber),
		log.Int("memberCount", len(uuids)))
	return uuids, nil
}

// GetGroupMembersByGroupNumber 通过群号获取群成员详细信息
func (s *GroupService) GetGroupMembersByGroupNumber(groupNumber string) ([]response.GroupMemberResponse, error) {
	logger := log.Logger

	// 1. 先通过群号获取群组信息
	group, err := s.repo.GetGroupByGroupNumber(groupNumber)
	if err != nil {
		logger.Error("GetGroupMembersByGroupNumber: group not found", log.String("groupNumber", groupNumber))
		return nil, err
	}

	// 2. 通过群组ID获取成员列表
	members, err := s.repo.GetGroupMembers(group.ID)
	if err != nil {
		logger.Error("GetGroupMembersByGroupNumber: failed to get group members", log.Any("groupId", group.ID))
		return nil, err
	}

	logger.Info("GetGroupMembersByGroupNumber: fetched group members successfully",
		log.String("groupNumber", groupNumber),
		log.Int("memberCount", len(members)))
	return members, nil
}
