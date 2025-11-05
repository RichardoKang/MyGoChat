package handler

import (
	pb "MyGoChat/api/v1"
	"MyGoChat/pkg/common/response"
	"MyGoChat/pkg/config"
	"MyGoChat/pkg/log"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// SendMessage 发送消息（私聊/群聊）
func (h *Handler) SendMessage(c *gin.Context) {
	var req struct {
		ConversationID string      `json:"conversation_id"` // 可选，如果为空则自动创建
		RecipientUUID  string      `json:"recipient_uuid"`  // 私聊时必填
		ContentType    int32       `json:"content_type" binding:"required"`
		Body           interface{} `json:"body" binding:"required"`
		MessageType    int32       `json:"message_type" binding:"required"` // 1=私聊, 2=群聊
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.FailMsg("请求参数错误: "+err.Error()))
		return
	}

	// 从JWT中获取用户ID
	senderUUID, exists := c.Get("useruuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.FailMsg("未授权"))
		return
	}

	// 验证并获取/创建会话ID
	conversationID := req.ConversationID
	if conversationID == "" {
		// 自动创建私聊会话
		if req.MessageType == 1 {
			if req.RecipientUUID == "" {
				c.JSON(http.StatusBadRequest, response.FailMsg("私聊必须指定接收者UUID"))
				return
			}
			ctx := context.Background()
			conv, err := h.ConvService.GetOrCreatePrivateConversation(
				ctx,
				senderUUID.(string),
				req.RecipientUUID,
			)
			if err != nil {
				log.Logger.Sugar().Errorf("Failed to create conversation: %v", err)
				c.JSON(http.StatusInternalServerError, response.FailMsg("创建会话失败"))
				return
			}
			conversationID = conv.ID.Hex()
			log.Logger.Sugar().Infof("Created/Retrieved conversation: %s", conversationID)
		} else {
			c.JSON(http.StatusBadRequest, response.FailMsg("群聊必须指定会话ID"))
			return
		}
	}

	// 构造 Protobuf 消息
	msg := &pb.Message{
		ConversationID: conversationID,
		SenderUUID:     senderUUID.(string),
		RecipientUUID:  req.RecipientUUID,
		ContentType:    req.ContentType,
		MessageType:    req.MessageType,
		SendAt:         time.Now().Unix(),
	}

	// 转换 Body 为 google.protobuf.Any
	anyBody, err := convertToProtobufAny(req.ContentType, req.Body)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to convert body: %v", err)
		c.JSON(http.StatusBadRequest, response.FailMsg("消息体格式错误: "+err.Error()))
		return
	}
	msg.Body = anyBody

	// 序列化并发送到 Kafka
	msgData, err := proto.Marshal(msg)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to marshal message: %v", err)
		c.JSON(http.StatusInternalServerError, response.FailMsg("消息序列化失败"))
		return
	}

	// 发送到 Kafka
	cfg := config.GetConfig()
	kafkaProducer := h.MessageService.GetProducer()
	err = kafkaProducer.SendMessage(cfg.Kafka.Topics.Ingest, msgData)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to send to Kafka: %v", err)
		c.JSON(http.StatusInternalServerError, response.FailMsg("消息发送失败"))
		return
	}

	log.Logger.Sugar().Infof("Message sent successfully from %s to conversation %s", senderUUID, conversationID)

	c.JSON(http.StatusOK, response.SuccessMsg(gin.H{
		"conversation_id": conversationID,
		"message_id":      msg.Id,
		"send_at":         msg.SendAt,
	}))
}

// convertToProtobufAny 将 JSON body 转换为 google.protobuf.Any 类型
func convertToProtobufAny(contentType int32, body interface{}) (*anypb.Any, error) {
	switch contentType {
	case 1: // 文本消息
		if bodyMap, ok := body.(map[string]interface{}); ok {
			if content, exists := bodyMap["content"]; exists {
				textBody := &pb.TextBody{
					Content: content.(string),
				}
				return anypb.New(textBody)
			}
		}
		return nil, fmt.Errorf("invalid text body format, expected {\"content\": \"...\"}")

	case 2, 3, 4: // 图片、文件、语音消息
		if bodyMap, ok := body.(map[string]interface{}); ok {
			fileBody := &pb.FileAttachment{
				Url:      getStringValue(bodyMap, "url"),
				FileName: getStringValue(bodyMap, "fileName"),
				Size:     getInt64Value(bodyMap, "size"),
				MimeType: getStringValue(bodyMap, "mimeType"),
			}
			return anypb.New(fileBody)
		}
		return nil, fmt.Errorf("invalid file body format")

	default:
		return nil, fmt.Errorf("unsupported content type: %d", contentType)
	}
}

// 辅助函数：安全地从 map 中获取字符串值
func getStringValue(m map[string]interface{}, key string) string {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// 辅助函数：安全地从 map 中获取 int64 值
func getInt64Value(m map[string]interface{}, key string) int64 {
	if val, exists := m[key]; exists {
		switch v := val.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case float64:
			return int64(v)
		}
	}
	return 0
}

// GetMessageHistory 获取消息历史记录
func (h *Handler) GetMessageHistory(c *gin.Context) {
	conversationID := c.Param("conversationId")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, response.FailMsg("会话ID不能为空"))
		return
	}

	// 解析分页参数
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// 获取消息历史
	messages, err := h.MessageService.GetMessageHistory(conversationID, limit, offset)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to get message history: %v", err)
		c.JSON(http.StatusInternalServerError, response.FailMsg("获取消息历史失败"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessMsg(gin.H{
		"messages": messages,
		"limit":    limit,
		"offset":   offset,
	}))
}

// SyncOfflineMessages 同步离线消息
func (h *Handler) SyncOfflineMessages(c *gin.Context) {
	// 从JWT中获取用户ID
	userUUID, exists := c.Get("useruuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.FailMsg("未授权"))
		return
	}

	err := h.MessageService.SyncOfflineMessages(userUUID.(string))
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to sync offline messages: %v", err)
		c.JSON(http.StatusInternalServerError, response.FailMsg("同步离线消息失败"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessMsg(gin.H{"message": "离线消息同步完成"}))
}

// MarkMessagesAsRead 标记消息为已读
func (h *Handler) MarkMessagesAsRead(c *gin.Context) {
	var req struct {
		MessageIDs []string `json:"message_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.FailMsg("请求参数错误"))
		return
	}

	err := h.MessageService.MarkMessagesAsRead(req.MessageIDs)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to mark messages as read: %v", err)
		c.JSON(http.StatusInternalServerError, response.FailMsg("标记消息已读失败"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessMsg(gin.H{"message": "消息已标记为已读"}))
}

// GetConversations 获取用户的会话列表
func (h *Handler) GetConversations(c *gin.Context) {
	// 从JWT中获取用户ID
	userUUID, exists := c.Get("useruuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.FailMsg("未授权"))
		return
	}

	// 解析分页参数
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	ctx := context.Background()
	conversations, err := h.ConvService.GetConversationsByUserID(ctx, userUUID.(string), limit)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to get conversations: %v", err)
		c.JSON(http.StatusInternalServerError, response.FailMsg("获取会话列表失败"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessMsg(gin.H{
		"conversations": conversations,
		"limit":         limit,
	}))
}

// CreatePrivateConversation 创建私聊会话
func (h *Handler) CreatePrivateConversation(c *gin.Context) {
	var req struct {
		TargetUserUUID string `json:"target_user_uuid" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.FailMsg("请求参数错误"))
		return
	}

	// 从JWT中获取用户ID
	userUUID, exists := c.Get("useruuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.FailMsg("未授权"))
		return
	}

	ctx := context.Background()
	conversation, err := h.ConvService.GetOrCreatePrivateConversation(ctx, userUUID.(string), req.TargetUserUUID)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to create private conversation: %v", err)
		c.JSON(http.StatusInternalServerError, response.FailMsg("创建私聊会话失败"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessMsg(conversation))
}

// CreateGroupConversation 创建群聊会话（当用户加入群聊时自动创建）
func (h *Handler) CreateGroupConversation(c *gin.Context) {
	var req struct {
		GroupNumber string `json:"group_number" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.FailMsg("请求参数错误"))
		return
	}

	// 从JWT中获取用户ID
	userUUID, exists := c.Get("useruuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.FailMsg("未授权"))
		return
	}

	ctx := context.Background()

	// 1. 验证群组是否存在
	group, err := h.GroupService.GetGroupByGroupNumber(req.GroupNumber)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to get group by number: %v", err)
		c.JSON(http.StatusNotFound, response.FailMsg("群组不存在"))
		return
	}

	// 2. 获取群成员列表
	groupMembers, err := h.GroupService.GetGroupMemberUUIDs(req.GroupNumber)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to get group members: %v", err)
		c.JSON(http.StatusInternalServerError, response.FailMsg("获取群成员失败"))
		return
	}

	// 3. 检查当前用户是否是群成员
	isMember := false
	for _, memberUUID := range groupMembers {
		if memberUUID == userUUID.(string) {
			isMember = true
			break
		}
	}
	if !isMember {
		c.JSON(http.StatusForbidden, response.FailMsg("您不是该群组成员"))
		return
	}

	// 4. 创建或获取群聊会话
	conversation, err := h.ConvService.GetOrCreateGroupConversation(ctx, groupMembers, req.GroupNumber)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to create group conversation: %v", err)
		c.JSON(http.StatusInternalServerError, response.FailMsg("创建群聊会话失败"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessMsg(gin.H{
		"conversation":  conversation,
		"group_info":    group,
		"members_count": len(groupMembers),
	}))
}

// GetConversationByGroupNumber 通过群号获取会话ID
func (h *Handler) GetConversationByGroupNumber(c *gin.Context) {
	groupNumber := c.Param("groupNumber")
	if groupNumber == "" {
		c.JSON(http.StatusBadRequest, response.FailMsg("群号不能为空"))
		return
	}

	// 从JWT中获取用户ID
	userUUID, exists := c.Get("useruuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.FailMsg("未授权"))
		return
	}

	ctx := context.Background()

	// 1. 验证用户是否是群成员
	groupMembers, err := h.GroupService.GetGroupMemberUUIDs(groupNumber)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to get group members: %v", err)
		c.JSON(http.StatusNotFound, response.FailMsg("群组不存在"))
		return
	}

	isMember := false
	for _, memberUUID := range groupMembers {
		if memberUUID == userUUID.(string) {
			isMember = true
			break
		}
	}
	if !isMember {
		c.JSON(http.StatusForbidden, response.FailMsg("您不是该群组成员"))
		return
	}

	// 2. 获取群聊会话
	conversation, err := h.ConvService.GetOrCreateGroupConversation(ctx, groupMembers, groupNumber)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to get group conversation: %v", err)
		c.JSON(http.StatusInternalServerError, response.FailMsg("获取群聊会话失败"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessMsg(gin.H{
		"conversation_id": conversation.ID.Hex(),
		"conversation":    conversation,
	}))
}

// GetConversationByUsers 通过用户UUID获取私聊会话ID
func (h *Handler) GetConversationByUsers(c *gin.Context) {
	var req struct {
		TargetUserUUID string `json:"target_user_uuid" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.FailMsg("请求参数错误"))
		return
	}

	// 从JWT中获取用户ID
	userUUID, exists := c.Get("useruuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.FailMsg("未授权"))
		return
	}

	ctx := context.Background()
	conversation, err := h.ConvService.GetOrCreatePrivateConversation(ctx, userUUID.(string), req.TargetUserUUID)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to get private conversation: %v", err)
		c.JSON(http.StatusInternalServerError, response.FailMsg("获取私聊会话失败"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessMsg(gin.H{
		"conversation_id": conversation.ID.Hex(),
		"conversation":    conversation,
	}))
}
