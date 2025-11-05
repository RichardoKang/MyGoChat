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
			info.Use(middleware.JWTAuthMiddleware())
			{
				info.PUT("/update", h.Update)
			}
		}
		// hostname/api/group
		group := api.Group("/group")
		{
			group.Use(middleware.JWTAuthMiddleware())
			group.POST("/create", h.CreateGroup)
			group.GET("/mine", h.GetMyGroups)
			group.POST("/join", h.JoinGroup)
			group.GET("/:groupnumber/members", h.GetGroupMembers)
		}

		// hostname/api/message - 消息相关API
		message := api.Group("/message")
		{
			message.Use(middleware.JWTAuthMiddleware())
			message.POST("/send", h.SendMessage)                         // 发送消息（HTTP）
			message.GET("/history/:conversationId", h.GetMessageHistory) // 获取消息历史
			message.POST("/sync-offline", h.SyncOfflineMessages)         // 同步离线消息
			message.POST("/mark-read", h.MarkMessagesAsRead)             // 标记消息已读
		}

		// hostname/api/conversation - 会话相关API
		conversation := api.Group("/conversation")
		{
			conversation.Use(middleware.JWTAuthMiddleware())
			conversation.GET("/list", h.GetConversations)                           // 获取会话列表
			conversation.POST("/private", h.CreatePrivateConversation)              // 创建私聊会话
			conversation.POST("/group", h.CreateGroupConversation)                  // 创建/获取群聊会话
			conversation.GET("/group/:groupNumber", h.GetConversationByGroupNumber) // 通过群号获取会话ID
			conversation.POST("/users", h.GetConversationByUsers)                   // 通过用户获取私聊会话ID
		}

	}

	return r
}
