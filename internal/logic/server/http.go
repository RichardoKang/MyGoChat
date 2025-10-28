package server

import (
	"MyGoChat/internal/gateway/socket"
	"MyGoChat/internal/logic/handler"
	"MyGoChat/internal/logic/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewRouter(hub *socket.Hub) *gin.Engine {
	gin.SetMode(gin.DebugMode)

	r := gin.Default()

	// CORS Middleware
	r.Use(middleware.CORSMiddleware())

	r.GET("/ws", middleware.JWTAuthMiddleware(), func(c *gin.Context) {
		socket.ServeWs(hub, c)
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
			user.POST("/register", handler.Register)
			user.POST("/login", handler.Login)

			info := user.Group("/info")
			info.Use(middleware.JWTAuthMiddleware())
			{
				info.PUT("/update", handler.Update)
			}
		}
		// hostname/api/group
		group := api.Group("/group")
		{
			group.Use(middleware.JWTAuthMiddleware())
			group.POST("/create", handler.CreateGroup)
			group.GET("/mine", handler.GetMyGroups)
			group.POST("/join", handler.JoinGroup)
			group.GET("/:groupnumber/members", handler.GetGroupMembers)
		}

	}

	return r
}
