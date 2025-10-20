package router

import (
	"MyGoChat/api/v1"
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	routers := gin.Default()

	groups := routers.Group("/api")
	{
		groups.GET("/", func(c *gin.Context) {
			c.String(http.StatusOK, "Hello, It`s My Go!!!!!")
		})
	}

	userRoutes := groups.Group("/user")
	{
		//userRoutes.GET("/user", v1.GetUserList)
		//userRoutes.GET("/user/:uuid", v1.GetUserDetails)
		//userRoutes.GET("/user/name", v1.GetUserOrGroupByName)
		userRoutes.POST("/register", v1.Register)
		//userRoutes.POST("/user/login", v1.Login)
		//userRoutes.PUT("/user", v1.ModifyUserInfo)
	}

	return routers
}
