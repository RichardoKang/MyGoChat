package chat

import (
	"MyGoChat/pkg/common/request"
	"MyGoChat/pkg/common/response"
	"MyGoChat/pkg/log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 处理聊天相关的 HTTP 请求
type Handler struct {
	service *Service
}

// NewHandler 创建聊天 Handler 实例
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// Send 发送消息（私聊/群聊）
func (h *Handler) Send(c *gin.Context) {
	var req request.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.Warn("SendMessage: invalid request body",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, response.FailMsg("请求参数错误: "+err.Error()))
		return
	}

	// 校验消息类型
	if req.MessageType != 1 && req.MessageType != 2 {
		c.JSON(http.StatusBadRequest, response.FailMsg("无效的消息类型，必须为 1(私聊) 或 2(群聊)"))
		return
	}

	// 校验内容类型
	if req.ContentType < 1 || req.ContentType > 4 {
		c.JSON(http.StatusBadRequest, response.FailMsg("无效的内容类型，必须为 1-4"))
		return
	}

	// 从 JWT 中间件中获取 SenderUUID
	senderUUID, exists := c.Get("useruuid")
	if !exists {
		log.Logger.Warn("SendMessage: useruuid not found in context")
		c.JSON(http.StatusUnauthorized, response.FailMsg("未授权：无法获取用户身份"))
		return
	}

	// 类型断言
	senderUUIDStr, ok := senderUUID.(string)
	if !ok || senderUUIDStr == "" {
		log.Logger.Error("SendMessage: useruuid type assertion failed")
		c.JSON(http.StatusUnauthorized, response.FailMsg("未授权：用户身份无效"))
		return
	}

	// 调用 Service 发送消息
	msg, err := h.service.SendMessage(c.Request.Context(), senderUUIDStr, &req)
	if err != nil {
		log.Logger.Error("SendMessage: failed to send message",
			zap.String("senderUUID", senderUUIDStr),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, response.FailMsg(err.Error()))
		return
	}

	log.Logger.Info("SendMessage: message sent successfully",
		zap.String("messageID", msg.Id),
		zap.String("conversationID", msg.ConversationID),
		zap.String("senderUUID", senderUUIDStr),
	)

	c.JSON(http.StatusOK, response.SuccessMsg(gin.H{
		"message_id":      msg.Id,
		"conversation_id": msg.ConversationID,
		"send_at":         msg.SendAt,
	}))
}

// SendMessage 发送消息（私聊/群聊）- 保留旧方法名以保持兼容性
func (h *Handler) SendMessage(c *gin.Context) {
	h.Send(c)
}

// GetHistory 获取历史消息
func (h *Handler) GetHistory(c *gin.Context) {
	conversationID := c.Param("conversationId")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, response.FailMsg("会话ID不能为空"))
		return
	}

	// 解析分页参数
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100 // 限制最大返回数量
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// 获取消息历史
	messages, err := h.service.GetMessageHistory(conversationID, limit, offset)
	if err != nil {
		log.Logger.Error("GetHistory: failed to get message history",
			zap.String("conversationID", conversationID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, response.FailMsg("获取消息历史失败"))
		return
	}

	// 如果没有消息，返回空数组而非 nil
	if messages == nil {
		messages = []*Message{}
	}

	c.JSON(http.StatusOK, response.SuccessMsg(gin.H{
		"messages":        messages,
		"limit":           limit,
		"offset":          offset,
		"conversation_id": conversationID,
	}))
}

// GetMessageHistory 获取消息历史记录 - 保留旧方法名以保持兼容性
// Deprecated: Use GetHistory instead
func (h *Handler) GetMessageHistory(c *gin.Context) {
	h.GetHistory(c)
}

// Sync 同步离线消息
func (h *Handler) Sync(c *gin.Context) {
	// 从 JWT 中间件中获取用户 UUID
	userUUID, exists := c.Get("useruuid")
	if !exists {
		log.Logger.Warn("Sync: useruuid not found in context")
		c.JSON(http.StatusUnauthorized, response.FailMsg("未授权：无法获取用户身份"))
		return
	}

	userUUIDStr, ok := userUUID.(string)
	if !ok || userUUIDStr == "" {
		log.Logger.Error("Sync: useruuid type assertion failed")
		c.JSON(http.StatusUnauthorized, response.FailMsg("未授权：用户身份无效"))
		return
	}

	err := h.service.SyncOfflineMessages(userUUIDStr)
	if err != nil {
		log.Logger.Error("Sync: failed to sync offline messages",
			zap.String("userUUID", userUUIDStr),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, response.FailMsg("同步离线消息失败"))
		return
	}

	log.Logger.Info("Sync: offline messages synced successfully",
		zap.String("userUUID", userUUIDStr),
	)

	c.JSON(http.StatusOK, response.SuccessMsg(gin.H{
		"message": "离线消息同步完成",
	}))
}

// SyncOfflineMessages 同步离线消息 - 保留旧方法名以保持兼容性
// Deprecated: Use Sync instead
func (h *Handler) SyncOfflineMessages(c *gin.Context) {
	h.Sync(c)
}

// GetConversations 获取用户的会话列表
func (h *Handler) GetConversations(c *gin.Context) {
	userUUID, exists := c.Get("useruuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.FailMsg("未授权：无法获取用户身份"))
		return
	}

	userUUIDStr, ok := userUUID.(string)
	if !ok || userUUIDStr == "" {
		c.JSON(http.StatusUnauthorized, response.FailMsg("未授权：用户身份无效"))
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	conversations, err := h.service.GetConversationsByUserID(c.Request.Context(), userUUIDStr, limit)
	if err != nil {
		log.Logger.Error("GetConversations: failed to get conversations",
			zap.String("userUUID", userUUIDStr),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, response.FailMsg("获取会话列表失败"))
		return
	}

	if conversations == nil {
		conversations = []*Conversation{}
	}

	c.JSON(http.StatusOK, response.SuccessMsg(gin.H{
		"conversations": conversations,
		"limit":         limit,
	}))
}

// MarkAsRead 标记消息为已读
func (h *Handler) MarkAsRead(c *gin.Context) {
	var req struct {
		MessageIDs []string `json:"message_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.FailMsg("请求参数错误: "+err.Error()))
		return
	}

	if len(req.MessageIDs) == 0 {
		c.JSON(http.StatusBadRequest, response.FailMsg("消息ID列表不能为空"))
		return
	}

	err := h.service.MarkMessagesAsRead(req.MessageIDs)
	if err != nil {
		log.Logger.Error("MarkAsRead: failed to mark messages as read",
			zap.Strings("messageIDs", req.MessageIDs),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, response.FailMsg("标记已读失败"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessMsg(gin.H{
		"message": "标记已读成功",
		"count":   len(req.MessageIDs),
	}))
}

// CreatePrivateConversation 创建私聊会话
func (h *Handler) CreatePrivateConversation(c *gin.Context) {
	var req struct {
		TargetUserUUID string `json:"target_user_uuid" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.FailMsg("请求参数错误: "+err.Error()))
		return
	}

	// 从JWT中获取用户UUID
	userUUID, exists := c.Get("useruuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.FailMsg("未授权：无法获取用户身份"))
		return
	}

	userUUIDStr, ok := userUUID.(string)
	if !ok || userUUIDStr == "" {
		c.JSON(http.StatusUnauthorized, response.FailMsg("未授权：用户身份无效"))
		return
	}

	conversation, err := h.service.GetPrivateConversation(c.Request.Context(), userUUIDStr, req.TargetUserUUID)
	if err != nil {
		log.Logger.Error("CreatePrivateConversation: failed to get/create private conversation",
			zap.String("userUUID", userUUIDStr),
			zap.String("targetUUID", req.TargetUserUUID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, response.FailMsg("创建私聊会话失败"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessMsg(conversation))
}
