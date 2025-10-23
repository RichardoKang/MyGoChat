package model

import (
	"time"

	"gorm.io/plugin/soft_delete"
)

type GroupMember struct {
	ID        uint                  `json:"id" gorm:"primarykey"`
	CreatedAt time.Time             `json:"createAt"`
	UpdatedAt time.Time             `json:"updatedAt"`
	DeletedAt soft_delete.DeletedAt `json:"deletedAt"`
	UserID    uint                  `json:"userId" gorm:"index;comment:'用户ID'"`
	User      User                  `gorm:"foreignKey:UserID"`
	GroupID   uint                  `json:"groupId" gorm:"index;comment:'群组ID'"`
	Group     Group                 `gorm:"foreignKey:GroupID"`
	Nickname  string                `json:"nickname" gorm:"type:varchar(255);comment:'昵称'"`
	Mute      bool                  `json:"mute" gorm:"comment:'是否禁言'"`
}
