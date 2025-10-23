package model

import (
	"time"

	"gorm.io/plugin/soft_delete"
)

type GroupMember struct {
	ID        int32                 `json:"id" gorm:"primarykey"`
	CreatedAt time.Time             `json:"createAt"`
	UpdatedAt time.Time             `json:"updatedAt"`
	DeletedAt soft_delete.DeletedAt `json:"deletedAt"`
	UserId    int32                 `json:"userId" gorm:"index;comment:'用户ID'"`
	User      User                  `gorm:"foreignKey:UserId;references:Id"`
	GroupId   int32                 `json:"groupId" gorm:"index;comment:'群组ID'"`
	Group     Group                 `gorm:"foreignKey:GroupId"`
	Nickname  string                `json:"nickname" gorm:"type:varchar(350);comment:'昵称"`
	Mute      bool                  `json:"mute" gorm:"comment:'是否禁言'"`
}
