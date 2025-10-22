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

func Register(c *gin.Context) {
	var req request.UserRegisterRequest
	var res response.UserResponse
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.FailMsg(err.Error()))
		return
	}

	user := model.User{
		Username: req.Username,
		Password: req.Password,
	}

	res = response.UserResponse{
		Username: user.Username,
		Email:    user.Email,
		Avatar:   user.Avatar,
	}

	token, err := service.UserService.Register(&user)
	if err != nil {
		c.JSON(http.StatusOK, response.FailMsg(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessMsg(gin.H{"user": res, "token": token}))
}

func Login(c *gin.Context) {
	var req request.UserLoginRequest
	var res response.UserResponse
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.FailMsg(err.Error()))
		return
	}

	user := model.User{
		Username: req.Username,
		Password: req.Password,
	}

	res = response.UserResponse{
		Username: user.Username,
		Email:    user.Email,
		Avatar:   user.Avatar,
	}

	token, err := service.UserService.Login(&user)
	// 处理返回...
	if err != nil {
		c.JSON(http.StatusOK, response.FailMsg(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessMsg(gin.H{"user": res, "token": token}))
}

func Update(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}
	// 传入token，解析在service层
	authHeader := c.GetHeader("Authorization")
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	err := service.UserService.Update(&user, tokenString)
	if err != nil {
		c.JSON(http.StatusOK, response.FailMsg(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessMsg(gin.H{"user": user}))

}
