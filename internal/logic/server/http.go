package server

import (
	v1 "MyGoChat/api/v1"
	"MyGoChat/internal/gateway"
	"MyGoChat/internal/logic/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewRouter(hub *gateway.Hub) *gin.Engine {
	gin.SetMode(gin.DebugMode)

	r := gin.Default()

	// CORS Middleware
	r.Use(middleware.CORSMiddleware())

	r.GET("/ws", middleware.JWTAuthMiddleware(), func(c *gin.Context) {
		gateway.ServeWs(hub, c)
	})

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
			group.GET("/:groupnumber/members", v1.GetGroupMembers)
		}

	}

	return r
}
