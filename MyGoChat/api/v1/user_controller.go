package v1

import (
	"MyGoChat/internal/logic/service"
	"MyGoChat/internal/model"
	"MyGoChat/pkg/common/request"
	"MyGoChat/pkg/common/response"
	"net/http"

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
	var req request.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.FailMsg(err.Error()))
		return
	}

	user := model.User{
		Nickname: req.Nickname,
		Email:    req.Email,
		Avatar:   req.Avatar,
	}

	// 传入token，解析在service层
	value, exist := c.Get("useruuid")
	if !exist {
		c.JSON(http.StatusUnauthorized, response.FailMsg("Unauthorized"))
		return
	}
	userUuid := value.(string)

	err := service.UserService.Update(&user, userUuid)
	if err != nil {
		c.JSON(http.StatusOK, response.FailMsg(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessMsg(gin.H{"user": user}))

}
