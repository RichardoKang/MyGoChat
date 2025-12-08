package chat

import (
	"MyGoChat/pkg/common/request"
	"MyGoChat/pkg/common/response"
	"MyGoChat/pkg/log"
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// SendMessage 发送消息（私聊/群聊）
func (h *Handler) SendMessage(c *gin.Context) {
	var req request.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.FailMsg("请求参数错误: "+err.Error()))
		return
	}

	senderUUID, exists := c.Get("useruuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.FailMsg("未授权"))
		return
	}

	resp, err := h.service.SendMessage(c.Request.Context(), senderUUID.(string), &req)

	if err != nil {
		log.Logger.Sugar().Errorf("SendMessage failed: %v", err)
		c.JSON(http.StatusInternalServerError, response.FailMsg(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessMsg(resp))
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
	messages, err := h.service.GetMessageHistory(conversationID, limit, offset)
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

	err := h.service.SyncOfflineMessages(userUUID.(string))
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

	err := h.service.MarkMessagesAsRead(req.MessageIDs)
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
	conversations, err := h.service.GetConversationsByUserID(ctx, userUUID.(string), limit)
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
	conversation, err := h.service.GetPrivateConversation(ctx, userUUID.(string), req.TargetUserUUID)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to create private conversation: %v", err)
		c.JSON(http.StatusInternalServerError, response.FailMsg("创建私聊会话失败"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessMsg(conversation))
}
