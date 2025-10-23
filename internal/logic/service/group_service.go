package service

import (
	"MyGoChat/internal/model"
	"MyGoChat/pkg/common/response"
	"MyGoChat/pkg/db"
	"MyGoChat/pkg/log"
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
		Select("groups.uuid, groups.name, groups.created_at").
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
