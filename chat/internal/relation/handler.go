package relation

import (
	"MyGoChat/pkg/common/request"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) JoinGroupRelation(c *gin.Context) {

	var req request.JoinGroupRelationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.JoinGroupRelation(c.Request.Context(), req.Username, req.GroupNumber); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Joined group successfully"})

}

func (h *Handler) CreateFriendRelation(c *gin.Context) {
	var req request.CreateFriendRelationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.CreateFriendRelation(c.Request.Context(), req.Username, req.FriendUsername); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
}

func (h *Handler) ListUserRelations(c *gin.Context) {
	var req request.ListUserRelationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	relations, err := h.service.ListUserRelation(c.Request.Context(), req.Username)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"relations": relations})
}
