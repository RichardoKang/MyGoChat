package server

import (
	"MyGoChat/internal/chat"
	"MyGoChat/internal/group"
	"MyGoChat/internal/middleware"
	"MyGoChat/internal/relation"
	"MyGoChat/internal/user"

	"net/http"

	"github.com/gin-gonic/gin"
)

func NewRouter(
	userHandler *user.Handler,
	groupHandler *group.Handler,
	chatHandler *chat.Handler,
	relaHandler *relation.Handler,

) *gin.Engine {
	gin.SetMode(gin.DebugMode)

	r := gin.Default()

	// CORS Middleware
	r.Use(middleware.CORSMiddleware())

	// hostname/api
	api := r.Group("/api")
	{
		api.GET("/", func(c *gin.Context) {
			c.String(http.StatusOK, "Hello, It`s My Go!!!!!")
		})

		// hostname/api/user
		user := api.Group("/user")
		{
			user.POST("/register", userHandler.Register) // 用户注册
			user.POST("/login", userHandler.Login)       // 用户登录

			info := user.Group("/info")
			info.Use(middleware.JWTAuthMiddleware())
			{
				info.PUT("/update", userHandler.Update) // 更新用户信息
			}
		}
		// hostname/api/group
		group := api.Group("/group")
		{
			group.Use(middleware.JWTAuthMiddleware())
			group.POST("/create", groupHandler.CreateGroup)
		}

		// hostname/api/message - 消息相关API
		message := api.Group("/message")
		{
			message.Use(middleware.JWTAuthMiddleware())
			message.POST("/send", chatHandler.SendMessage)                         // 发送消息（HTTP）
			message.GET("/history/:conversationId", chatHandler.GetMessageHistory) // 获取历史消息
			message.POST("/sync-offline", chatHandler.SyncOfflineMessages)         // 同步离线消息 // 标记消息为已读
		}

		relations := api.Group("/relations")
		{
			relations.Use(middleware.JWTAuthMiddleware())
			relations.POST("/add-group", relaHandler.JoinGroupRelation)     // 加入群组
			relations.POST("/add-friend", relaHandler.CreateFriendRelation) // 添加好友
			relations.GET("list", relaHandler.ListUserRelations)            // 获取用户关系列表
		}

	}

	return r
}
