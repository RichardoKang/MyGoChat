package chat

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Conversation struct {
	ID         string `bson:"_id" json:"ID"`
	Type       int    `bson:"type" json:"Type"`              // 1=私聊, 2=群聊
	TargetUUID string `bson:"-" json:"TargetUUID,omitempty"` // 对方的UUID（私聊）或群UUID（群聊），不存入MongoDB
	TargetName string `bson:"-" json:"TargetName,omitempty"` // 对方的用户名（私聊）或群名（群聊），不存入MongoDB

	// 会话信息
	LastMessage          interface{} `bson:"lastMessage,omitempty" json:"LastMessage,omitempty"` // 最后一条消息内容
	LastMessageTimestamp int64       `bson:"lastMessageTimestamp" json:"LastMessageTimestamp"`   // 最后消息时间戳
	CreatedAt            time.Time   `bson:"createdAt" json:"CreatedAt"`                         // 会话创建时间
	UpdatedAt            time.Time   `bson:"updatedAt" json:"UpdatedAt"`                         // 会话更新时间
}

type Message struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"ID"`
	ConversationID string             `bson:"conversationID" json:"ConversationID"` // 会话ID (字符串，支持MD5哈希)
	SenderUUID     string             `bson:"senderUUID" json:"SenderUUID"`         // 发送者 UUID
	SenderName     string             `bson:"senderName" json:"SenderName"`         // 发送者用户名
	SendAt         int64              `bson:"sendAt" json:"SendAt"`                 // 发送时间戳
	ContentType    int16              `bson:"contentType" json:"ContentType"`       // 1=text, 2=image, 3=file, 4=voice
	Body           any                `bson:"body" json:"Body"`                     // 消息内容
	Metadata       *MessageMetadata   `bson:"metadata,omitempty" json:"Metadata,omitempty"`
	DeletedAt      *time.Time         `bson:"deletedAt,omitempty" json:"DeletedAt,omitempty"`
}

// FileAttachment 用于 Body 字段 (当 ContentType 不是 text 时)
type FileAttachment struct {
	URL      string `bson:"url" json:"url"`           // MinIO/S3 的 URL
	FileName string `bson:"fileName" json:"fileName"` // "简历.pdf"
	Size     int64  `bson:"size" json:"size"`         // 文件大小 (bytes)
	MimeType string `bson:"mimeType" json:"mimeType"` // "image/jpeg"
}

// MessageMetadata 用于存储回复、@ 等元数据
type MessageMetadata struct {
	ReplyToMsgID string `bson:"replyToMsgID,omitempty"` // 回复的消息 ID (用 string 存 ObjectID)
}
