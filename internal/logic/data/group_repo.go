package data

import (
	"MyGoChat/internal/model"
	"MyGoChat/pkg/common/response"

	"gorm.io/gorm"
)

type IGroupRepo interface {
	CreateGroup(group *model.Group, adminUser *model.User) error
	GetMyGroups(userID uint) ([]response.GroupResponse, error)
	GetGroupByGroupNumber(groupNumber string) (*model.Group, error)
	JoinGroup(user *model.User, group *model.Group, nickname string) error
	GetGroupMembers(groupID uint) ([]response.GroupMemberResponse, error)
	GetGroupMemberUUIDs(conversationID string) ([]string, error) // 新增：获取群成员UUID列表
}

type groupRepo struct {
	db *gorm.DB
}

func NewGroupRepo(data *Data) IGroupRepo {
	return &groupRepo{db: data.Db}
}

func (r *groupRepo) CreateGroup(group *model.Group, adminUser *model.User) error {
	if err := r.db.AutoMigrate(&model.Group{}, &model.GroupMember{}); err != nil {
		return err
	}

	group.AdminUserID = adminUser.ID
	if err := r.db.Save(group).Error; err != nil {
		return err
	}

	groupMember := model.GroupMember{
		UserID:   adminUser.ID,
		GroupID:  group.ID,
		Nickname: adminUser.Nickname,
		Mute:     false,
	}
	return r.db.Save(&groupMember).Error
}

func (r *groupRepo) GetMyGroups(userID uint) ([]response.GroupResponse, error) {
	var queryGroups []response.GroupResponse
	result := r.db.Table("groups").
		Select("groups.uuid,groups.group_number, groups.created_at, groups.name").
		Joins("join group_members on groups.id = group_members.group_id").
		Where("group_members.user_id = ?", userID).
		Scan(&queryGroups)
	return queryGroups, result.Error
}

func (r *groupRepo) GetGroupByGroupNumber(groupNumber string) (*model.Group, error) {
	var group model.Group
	result := r.db.First(&group, "group_number = ?", groupNumber)
	if result.Error != nil {
		return nil, result.Error // 找不到记录时返回 gorm.ErrRecordNotFound
	}
	return &group, nil
}

func (r *groupRepo) JoinGroup(user *model.User, group *model.Group, nickname string) error {
	var existingMember model.GroupMember
	result := r.db.Where("user_id = ? AND group_id = ?", user.ID, group.ID).First(&existingMember)
	if result.Error == nil {
		return nil // User is already a member
	}

	groupMember := model.GroupMember{
		UserID:   user.ID,
		GroupID:  group.ID,
		Nickname: nickname,
		Mute:     false,
	}
	return r.db.Save(&groupMember).Error
}

func (r *groupRepo) GetGroupMembers(groupID uint) ([]response.GroupMemberResponse, error) {
	var members []response.GroupMemberResponse
	result := r.db.Table("group_members").
		Select("group_members.user_id, users.username, group_members.nickname, group_members.mute").
		Joins("join users on group_members.user_id = users.id").
		Where("group_members.group_id = ?", groupID).
		Scan(&members)
	return members, result.Error
}

// GetGroupMemberUUIDs 通过会话ID（群号）获取群成员的UUID列表
func (r *groupRepo) GetGroupMemberUUIDs(conversationID string) ([]string, error) {
	// conversationID 实际上是群号或者需要从 conversation 中解析出群号
	// 这里假设 conversationID 就是群号
	var userUUIDs []string

	result := r.db.Table("group_members").
		Select("users.uuid").
		Joins("join users on group_members.user_id = users.id").
		Joins("join groups on group_members.group_id = groups.id").
		Where("groups.group_number = ?", conversationID).
		Pluck("users.uuid", &userUUIDs)

	return userUUIDs, result.Error
}
