package model

import (
	"time"

	"gorm.io/plugin/soft_delete"
)

type Message struct {
	ID          uint                  `json:"id" gorm:"primarykey"`
	CreatedAt   time.Time             `json:"createAt"`
	UpdatedAt   time.Time             `json:"updatedAt"`
	DeletedAt   soft_delete.DeletedAt `json:"deletedAt"`
	SenderId    uint                  `json:"senderId" gorm:"index"`
	Sender      User                  `gorm:"foreignKey:FromUserId;references:Id"`
	RecipientId uint                  `json:"recipientId" gorm:"index;comment:'发送给端的id，可为用户id或者群id'"`
	Content     string                `json:"content" gorm:"type:varchar(2500)"`
	MessageType int16                 `json:"messageType" gorm:"comment:'消息类型：1单聊，2群聊'"`
	ContentType int16                 `json:"contentType" gorm:"comment:'消息内容类型：1文字 2.普通文件 3.图片 4.音频 5.视频 6.语音聊天 7.视频聊天'"`
	Pic         string                `json:"pic" gorm:"type:text;comment:'缩略图'"`
	Url         string                `json:"url" gorm:"type:varchar(350);comment:'文件或者图片地址'"`
}
