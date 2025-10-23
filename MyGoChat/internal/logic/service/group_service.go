package service

import (
	"MyGoChat/internal/model"
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
