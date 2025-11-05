package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Conversation struct {
	ID   primitive.ObjectID `bson:"_id,omitempty"`
	Type int                `bson:"type"` // 1=私聊, 2=群聊

	// 'participants' 字段是实现群聊/私聊的关键
	Participants []string `bson:"participants"` // 存 user_id 列表

	// (用于群聊)
	GroupNumber  string `bson:"groupNumber,omitempty"`
	GroupOwnerID string `bson:"groupOwnerID,omitempty"`

	// 会话信息
	LastMessage          interface{} `bson:"lastMessage,omitempty"` // 最后一条消息内容
	LastMessageTimestamp int64       `bson:"lastMessageTimestamp"`  // 最后消息时间戳
	CreatedAt            time.Time   `bson:"createdAt"`             // 会话创建时间
	UpdatedAt            time.Time   `bson:"updatedAt"`             // 会话更新时间
}
