package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Conversation struct {
	ID   primitive.ObjectID `bson:"_id,omitempty"`
	Type int16              `bson:"type"` // 1=私聊, 2=群聊

	// 'participants' 字段是实现群聊/私聊的关键
	Participants []uint `bson:"participants"` // 存 user_id 列表

	// (用于群聊)
	GroupNumber  string `bson:"groupNumber,omitempty"`
	GroupOwnerID string `bson:"groupOwnerID,omitempty"`

	// (用于会话列表排序，由 logic 服务在收到新消息时更新)
	LastMessageTimestamp int64 `bson:"lastMessageTimestamp"`
}
