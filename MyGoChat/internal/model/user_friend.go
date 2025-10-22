package model

import (
	"time"

	"gorm.io/plugin/soft_delete"
)

type UserFriend struct {
	ID        int32                 `json:"id" gorm:"primarykey"`
	CreatedAt time.Time             `json:"createAt"`
	UpdatedAt time.Time             `json:"updatedAt"`
	DeletedAt soft_delete.DeletedAt `json:"deletedAt"`
	UserId    int32                 `json:"userId" gorm:"index;comment:'用户ID'"`
	User      User                  `gorm:"foreignKey:UserId;references:Id"`
	FriendId  int32                 `json:"friendId" gorm:"index;comment:'好友ID'"`
	Friend    User                  `gorm:"foreignKey:FriendId;references:Id"`
}
