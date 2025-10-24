package router

import (
	v1 "MyGoChat/api/v1"
	"MyGoChat/internal/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	// hostname/api
	api := r.Group("/api")
	{
		api.GET("/", func(c *gin.Context) {
			c.String(http.StatusOK, "Hello, It`s My Go!!!!!")
		})

		// hostname/api/user
		user := api.Group("/user")
		{
			user.POST("/register", v1.Register)
			user.POST("/login", v1.Login)

			info := user.Group("/info")
			info.Use(middleware.JWTAuthMiddleware())
			{
				info.PUT("/update", v1.Update)
			}
		}
		// hostname/api/group
		group := api.Group("/group")
		{
			group.Use(middleware.JWTAuthMiddleware())
			group.POST("/create", v1.CreateGroup)
			group.GET("/mine", v1.GetMyGroups)
			group.POST("/join", v1.JoinGroup)
		}

	}

	return r
}
