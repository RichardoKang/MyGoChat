package service

import (
	"MyGoChat/internal/model"
	"MyGoChat/pkg/common/response"
	"MyGoChat/pkg/db"
	"MyGoChat/pkg/log"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

type groupService struct {
}

var GroupService = new(groupService)

func (g *groupService) CreateGroup(group *model.Group, adminUserUuid string) error {
	logger := log.Logger
	d := db.GetDB()

	// 确保表存在
	if err := d.AutoMigrate(&model.Group{}, &model.GroupMember{}); err != nil {
		logger.Error("CreateGroup: failed to migrate tables")
		return err
	}

	var adminUser model.User
	result := d.Find(&adminUser, "uuid = ?", adminUserUuid)
	if result.Error != nil {
		logger.Error("CreateGroup: admin user not found")
		return result.Error
	}

	group.Uuid = uuid.New().String()
	// 生成一个唯一的6位群号
	for {
		group.GroupNumber = fmt.Sprintf("%06v", rand.New(rand.NewSource(time.Now().UnixNano())).Int31n(1000000))
		var existingGroup model.Group
		if err := d.Where("group_number = ?", group.GroupNumber).First(&existingGroup).Error; err != nil {
			break // 未找到重复的群号，跳出循环
		}
	}
	group.AdminUserID = adminUser.ID
	group.CreatedAt = time.Now()
	d.Save(&group)

	groupMember := model.GroupMember{
		UserID:   adminUser.ID,
		GroupID:  group.ID,
		Nickname: adminUser.Nickname,
		Mute:     false,
	}
	d.Save(&groupMember)

	logger.Info("Group created successfully", log.Any("group", group), log.Any("adminUser", adminUser))
	return nil

}

func (g *groupService) GetMyGroups(userUuid string) ([]response.GroupResponse, error) {
	logger := log.Logger

	d := db.GetDB()

	var user model.User
	result := d.Find(&user, "uuid = ?", userUuid)
	if result.Error != nil {
		logger.Error("GetUserGroups: user not found")
		return []response.GroupResponse{}, result.Error
	}

	var queryGroups []response.GroupResponse
	// 根据用户ID查询所属群组
	result = d.Table("groups").
		Select("groups.uuid,groups.group_number, groups.created_at, groups.name").
		Joins("join group_members on groups.id = group_members.group_id").
		Where("group_members.user_id = ?", user.ID).
		Scan(&queryGroups) //Scan方法将结果映射到切片中

	if result.Error != nil {
		logger.Error("GetUserGroups: failed to get user groups")
		return []response.GroupResponse{}, result.Error
	}

	logger.Info("GetUserGroups: fetched user groups successfully", log.Any("groups", queryGroups))
	return queryGroups, nil

}

func (g *groupService) GetGroupByGroupNumber(groupNumber string) (response.GroupResponse, error) {
	logger := log.Logger
	d := db.GetDB()

	var group model.Group
	result := d.Find(&group, "group_number = ?", groupNumber)
	if result.Error != nil {
		logger.Error("GetGroupByGroupNumber: group not found")
		return response.GroupResponse{}, result.Error
	}

	groupResponse := response.GroupResponse{
		Uuid:      group.Uuid,
		Name:      group.Name,
		CreatedAt: group.CreatedAt,
	}

	return groupResponse, nil
}

func (g *groupService) JoinGroup(userUuid string, groupNumber string, nickname string) error {
	logger := log.Logger
	d := db.GetDB()

	var user model.User
	result := d.Find(&user, "uuid = ?", userUuid)
	if result.Error != nil {
		logger.Error("JoinGroup: user not found")
		return result.Error
	}

	var group model.Group
	result = d.Find(&group, "group_number = ?", groupNumber)
	if result.Error != nil {
		logger.Error("JoinGroup: group not found")
		return result.Error
	}

	// 检查用户是否已经是群成员
	var existingMember model.GroupMember
	result = d.Where("user_id = ? AND group_id = ?", user.ID, group.ID).First(&existingMember)
	if result.Error == nil {
		logger.Info("JoinGroup: user is already a member of the group", log.Any("userUuid", userUuid), log.Any("groupNumber", groupNumber))
		return nil // 用户已经是群成员，直接返回
	}

	groupMember := model.GroupMember{
		UserID:   user.ID,
		GroupID:  group.ID,
		Nickname: nickname,
		Mute:     false,
	}
	d.Save(&groupMember)

	logger.Info("JoinGroup: user joined group successfully", log.Any("userUuid", userUuid), log.Any("groupNumber", groupNumber))
	return nil
}

func (g *groupService) GetGroupMembers(GroupNumber string) ([]response.GroupMemberResponse, error) {
	logger := log.Logger
	d := db.GetDB()

	// get group by group number
	group := model.Group{}
	result := d.Find(&group, "group_number = ?", GroupNumber)
	if result.Error != nil {
		logger.Error("GetGroupMembers: group not found")
		return []response.GroupMemberResponse{}, result.Error
	}

	var members []response.GroupMemberResponse
	result = d.Table("group_members").
		Select("group_members.user_id, users.username, group_members.nickname, group_members.mute").
		Joins("join users on group_members.user_id = users.id").
		Where("group_members.group_id = ?", group.ID).
		Scan(&members)

	if result.Error != nil {
		logger.Error("GetGroupMembers: failed to get group members")
		return []response.GroupMemberResponse{}, result.Error
	}

	logger.Info("GetGroupMembers: fetched group members successfully", log.Any("members", members))
	return members, nil
}
