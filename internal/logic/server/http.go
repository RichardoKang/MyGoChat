package server

import (
	"MyGoChat/internal/logic/handler"
	"MyGoChat/internal/logic/middleware"
	"MyGoChat/internal/logic/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewRouter(userService *service.UserService, groupService *service.GroupService, messageService *service.MessageService, convService *service.ConversationService) *gin.Engine {
	gin.SetMode(gin.DebugMode)

	r := gin.Default()

	// Create a handler instance with all the services
	h := &handler.Handler{
		UserService:    userService,
		GroupService:   groupService,
		MessageService: messageService,
		ConvService:    convService,
	}

	// CORS Middleware
	r.Use(middleware.CORSMiddleware())

	//r.GET("/ws", middleware.JWTAuthMiddleware(h.UserService), func(c *gin.Context) {
	//	socket.ServeWs(hub, c)
	//})

	// hostname/api
	api := r.Group("/api")
	{
		api.GET("/", func(c *gin.Context) {
			c.String(http.StatusOK, "Hello, It`s My Go!!!!!")
		})

		// hostname/api/user
		user := api.Group("/user")
		{
			user.POST("/register", h.Register)
			user.POST("/login", h.Login)

			info := user.Group("/info")
			info.Use(middleware.JWTAuthMiddleware(h.UserService))
			{
				info.PUT("/update", h.Update)
			}
		}
		// hostname/api/group
		group := api.Group("/group")
		{
			group.Use(middleware.JWTAuthMiddleware(h.UserService))
			group.POST("/create", h.CreateGroup)
			group.GET("/mine", h.GetMyGroups)
			group.POST("/join", h.JoinGroup)
			group.GET("/:groupnumber/members", h.GetGroupMembers)
		}

	}

	return r
}
