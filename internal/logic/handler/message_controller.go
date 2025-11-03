package handler

import (
	"MyGoChat/pkg/common/response"
	"MyGoChat/pkg/log"
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

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
