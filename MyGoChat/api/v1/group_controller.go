package v1

import (
	"MyGoChat/internal/logic/service"
	"MyGoChat/internal/model"
	"MyGoChat/pkg/common/request"
	"MyGoChat/pkg/common/response"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CreateGroup(c *gin.Context) {
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

	value, exist := c.Get("user_uuid")
	if !exist {
		c.JSON(http.StatusUnauthorized, response.FailMsg("Unauthorized"))
		return
	}
	adminUserUuid := value.(string)

	// 调用服务层创建群组的逻辑
	group := &model.Group{
		Name: req.GroupName,
	}

	if err := service.GroupService.CreateGroup(group, adminUserUuid); err != nil {
		c.JSON(http.StatusInternalServerError, response.FailMsg(err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.SuccessMsg(group))
}
