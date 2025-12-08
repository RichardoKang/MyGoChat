package group

import (
	"MyGoChat/internal/platform"
	"MyGoChat/internal/user"
	"MyGoChat/pkg/common/response"

	"gorm.io/gorm"
)

type Repository interface {
	CreateGroup(group *Group, adminUser *user.User) error
	GetMyGroups(userID uint) ([]response.GroupResponse, error)
	GetGroupByGroupNumber(groupNumber string) (*Group, error)
	JoinGroup(user *user.User, group *Group, nickname string) error
	GetGroupMembers(groupNumber string) ([]response.GroupMemberResponse, error)
}

type repository struct {
	db *gorm.DB
}

func NewGroupRepo(data *platform.Data) Repository {
	return &repository{db: data.Db}
}

func (r *repository) CreateGroup(group *Group, adminUser *user.User) error {
	if err := r.db.AutoMigrate(&Group{}, &GroupMember{}); err != nil {
		return err
	}

	group.AdminUserID = adminUser.ID
	if err := r.db.Save(group).Error; err != nil {
		return err
	}

	groupMember := GroupMember{
		UserID:   adminUser.ID,
		GroupID:  group.ID,
		Nickname: adminUser.Nickname,
		Mute:     false,
	}
	return r.db.Save(&groupMember).Error
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

func (r *repository) JoinGroup(user *user.User, group *Group, nickname string) error {
	var existingMember GroupMember
	result := r.db.Where("user_id = ? AND group_id = ?", user.ID, group.ID).First(&existingMember)
	if result.Error == nil {
		return nil // User is already a member
	}

	groupMember := GroupMember{
		UserID:   user.ID,
		GroupID:  group.ID,
		Nickname: nickname,
		Mute:     false,
	}
	return r.db.Save(&groupMember).Error
}

func (r *repository) GetGroupMembers(groupNumber string) ([]response.GroupMemberResponse, error) {

	groupID := r.db.Table("groups").Select("id").Where("group_number = ?", groupNumber).Scan(new(uint)).RowsAffected
	if groupID == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	var members []response.GroupMemberResponse
	result := r.db.Table("group_members").
		Select("group_members.user_id, users.username, group_members.nickname, group_members.mute").
		Joins("join users on group_members.user_id = users.id").
		Where("group_members.group_id = ?", groupID).
		Scan(&members)
	return members, result.Error
}
