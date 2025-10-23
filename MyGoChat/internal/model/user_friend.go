package model

type UserFriend struct {
	ID       int32 `json:"id" gorm:"primarykey"`
	UserId   int32 `json:"userId" gorm:"index;comment:'用户ID'"`
	User     User  `gorm:"foreignKey:UserId;references:Id"`
	FriendId int32 `json:"friendId" gorm:"index;comment:'好友ID'"`
	Friend   User  `gorm:"foreignKey:FriendId;references:Id"`
}
