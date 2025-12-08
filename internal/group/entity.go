package group

import (
	"time"

	"gorm.io/plugin/soft_delete"
)

type Group struct {
	ID          uint      `json:"id" gorm:"primarykey"`
	Uuid        string    `json:"uuid" gorm:"index;unique;not null"`
	GroupNumber string    `json:"groupNumber" gorm:"index;unique;not null;comment:'群号'"`
	Name        string    `json:"name" gorm:"type:varchar(150);not null;comment:'群名称'"`
	AdminUserID uint      `json:"adminUserId" gorm:"not null;comment:'群主ID'"`
	CreatedAt   time.Time `json:"createdAt"`
}

type GroupMember struct {
	ID        uint                  `json:"id" gorm:"primarykey"`
	CreatedAt time.Time             `json:"createAt"`
	UpdatedAt time.Time             `json:"updatedAt"`
	DeletedAt soft_delete.DeletedAt `json:"deletedAt"`
	UserID    uint                  `json:"userId" gorm:"index;comment:'用户ID'"`
	GroupID   uint                  `json:"groupId" gorm:"index;comment:'群组ID'"`
	Nickname  string                `json:"nickname" gorm:"type:varchar(255);comment:'昵称'"`
	Mute      bool                  `json:"mute" gorm:"comment:'是否禁言'"`
}
