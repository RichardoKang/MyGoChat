package group

import (
	"MyGoChat/internal/platform"
	"MyGoChat/internal/user"
	"MyGoChat/pkg/common/response"
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	CreateGroup(group *Group, adminUser *user.User) error
	GetMyGroups(userID uint) ([]response.GroupResponse, error)
	GetGroupByGroupNumber(groupNumber string) (*Group, error)
	GetUUIDByNumber(ctx context.Context, groupNumber string) (string, error)
	GetNameByUUID(ctx context.Context, uuid string) (string, error)
}

type repository struct {
	db *gorm.DB
}

func NewGroupRepo(data *platform.Data) Repository {
	return &repository{db: data.Db}
}

func (r *repository) CreateGroup(group *Group, adminUser *user.User) error {
	if err := r.db.AutoMigrate(&Group{}); err != nil {
		return err
	}

	group.AdminUserID = adminUser.ID
	if err := r.db.Save(group).Error; err != nil {
		return err
	}

	return nil
}

func (r *repository) GetMyGroups(userID uint) ([]response.GroupResponse, error) {
	var queryGroups []response.GroupResponse
	result := r.db.Table("groups").
		Select("groups.uuid,groups.group_number, groups.created_at, groups.name").
		Joins("join group_members on groups.id = group_members.group_id").
		Where("group_members.user_id = ?", userID).
		Scan(&queryGroups)
	return queryGroups, result.Error
}

func (r *repository) GetGroupByGroupNumber(groupNumber string) (*Group, error) {
	var group Group
	result := r.db.First(&group, "group_number = ?", groupNumber)
	if result.Error != nil {
		return nil, result.Error // 找不到记录时返回 gorm.ErrRecordNotFound
	}
	return &group, nil
}

func (r *repository) GetUUIDByNumber(ctx context.Context, number string) (string, error) {
	var group Group
	err := r.db.WithContext(ctx).
		Select("uuid").
		Where("group_number = ?", number).
		First(&group).Error

	if err != nil {
		return "", err
	}
	return group.Uuid, nil
}

// GetNameByUUID 根据群组 UUID 获取群名
func (r *repository) GetNameByUUID(ctx context.Context, uuid string) (string, error) {
	var group Group
	err := r.db.WithContext(ctx).
		Select("name").
		Where("uuid = ?", uuid).
		First(&group).Error

	if err != nil {
		return "", err
	}
	return group.Name, nil
}
