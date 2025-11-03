package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Message struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	ConversationID primitive.ObjectID `bson:"conversationID"` // 索引: 属于哪个会话
	SenderUUID     string             `bson:"senderUUID"`     // 索引: 谁发的 (用户的 UUID string)
	SendAt         int64              `bson:"sendAt"`         // 索引: 消息发送时间
	ContentType    int16              `bson:"contentType"`    // 1=text, 2=image, 3=file, 4=voice
	Body           any                `bson:"body"`           // ContentType=1: "Hello" (string)， ContentType=2/3/...: FileAttachment 结构体
	Metadata       *MessageMetadata   `bson:"metadata,omitempty"`
	DeletedAt      *time.Time         `bson:"deletedAt,omitempty"`
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

//// IMessageRepo 定义了消息数据的接口
//type IMessageRepo interface {
//	CreateMessage(ctx context.Context, msg *Message) error
//	GetMessagesByConversationID(ctx context.Context, convID string, limit int64) ([]*Message, error)
//}
