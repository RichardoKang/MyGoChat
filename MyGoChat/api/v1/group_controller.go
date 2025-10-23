package v1

import (
	"MyGoChat/pkg/common/request"
	"MyGoChat/pkg/common/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

func CreateGroup(c *gin.Context) {
	var req request.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.FailMsg(err.Error()))
		return
	}

}
