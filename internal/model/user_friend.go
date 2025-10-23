package model

type UserFriend struct {
	ID       uint `json:"id" gorm:"primarykey"`
	UserId   uint `json:"userId" gorm:"index;comment:'用户ID'"`
	User     User `gorm:"foreignKey:UserId;references:Id"`
	FriendId uint `json:"friendId" gorm:"index;comment:'好友ID'"`
	Friend   User `gorm:"foreignKey:FriendId;references:Id"`
}
