package handler

import (
	"MyGoChat/internal/model"
	"MyGoChat/pkg/common/request"
	"MyGoChat/pkg/common/response"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *Handler) CreateGroup(c *gin.Context) {
	var req request.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.FailMsg(err.Error()))
		return
	}
	req.GroupName = strings.TrimSpace(req.GroupName)
	if req.GroupName == "" {
		c.JSON(http.StatusBadRequest, response.FailMsg("Group name cannot be empty"))
		return
	}

	value, exist := c.Get("useruuid")
	if !exist {
		c.JSON(http.StatusUnauthorized, response.FailMsg("Unauthorized"))
		return
	}
	adminUserUuid := value.(string)

	// 调用服务层创建群组的逻辑
	group := &model.Group{
		Name: req.GroupName,
	}

	if err := h.GroupService.CreateGroup(group, adminUserUuid); err != nil {
		c.JSON(http.StatusInternalServerError, response.FailMsg(err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.SuccessMsg(group))
}

func (h *Handler) GetMyGroups(c *gin.Context) {
	value, exist := c.Get("useruuid")
	if !exist {
		c.JSON(http.StatusUnauthorized, response.FailMsg("Unauthorized"))
		return
	}
	userUuid := value.(string)

	groups, err := h.GroupService.GetMyGroups(userUuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.FailMsg(err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.SuccessMsg(groups))
}

func (h *Handler) JoinGroup(c *gin.Context) {
	var req request.JoinGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.FailMsg(err.Error()))
		return
	}

	value, exist := c.Get("useruuid")
	if !exist {
		c.JSON(http.StatusUnauthorized, response.FailMsg("Unauthorized"))
		return
	}
	userUuid := value.(string)

	if err := h.GroupService.JoinGroup(userUuid, req.GroupNumber, req.Nickname); err != nil {
		c.JSON(http.StatusInternalServerError, response.FailMsg(err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.SuccessMsg("Joined group successfully"))
}

func (h *Handler) GetGroupMembers(c *gin.Context) {
	var req request.GetGroupMembersRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.FailMsg(err.Error()))
		return
	}

	// 使用通过群号获取群成员的方法
	members, err := h.GroupService.GetGroupMembersByGroupNumber(req.GroupNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.FailMsg(err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.SuccessMsg(members))
}
