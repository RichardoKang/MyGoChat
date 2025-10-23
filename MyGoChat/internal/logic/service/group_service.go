package service

import (
	"MyGoChat/internal/model"
	"MyGoChat/pkg/db"
	"MyGoChat/pkg/log"

	"github.com/google/uuid"
)

type groupService struct {
}

var GroupService = new(groupService)

func (g *groupService) CreateGroup(group *model.Group, adminUserUuid string) error {
	logger := log.Logger
	d := db.GetDB()

	var adminUser model.User
	result := d.Find(&adminUser, "uuid = ?", adminUserUuid)
	if result.Error != nil {
		logger.Error("CreateGroup: admin user not found")
		return result.Error
	}

	group.AdminUserUuid = adminUserUuid
	group.Uuid = uuid.New().String()
	d.Save(&group)

	groupMember := model.GroupMember{
		UserId:   adminUser.ID,
		GroupId:  group.ID,
		Nickname: adminUser.Nickname,
		Mute:     false,
	}
	d.Save(&groupMember)

	logger.Info("Group created successfully", log.Any("group", group), log.Any("adminUser", adminUser))
	return nil

}
