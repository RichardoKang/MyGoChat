package request

type SendMessageRequest struct {
	ConversationID string      `json:"conversation_id"` // 可选，如果为空则自动创建
	TargetName     string      `json:"target_name" binding:"required"`
	ContentType    int32       `json:"content_type" binding:"required"`
	Body           interface{} `json:"body" binding:"required"`
	MessageType    int32       `json:"message_type" binding:"required"` // 1=私聊, 2=群聊
}
