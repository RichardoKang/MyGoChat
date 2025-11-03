package server

import (
	"MyGoChat/internal/gateway/socket"
	"MyGoChat/internal/logic/middleware"
	"MyGoChat/internal/logic/service" // <-- 注意：这是暂时的依赖

	"github.com/gin-gonic/gin"
)

// NewGatewayRouter 只创建网关所需的路由
func NewGatewayRouter(hub *socket.Hub, userService *service.UserService) *gin.Engine {
	gin.SetMode(gin.DebugMode)
	r := gin.Default()

	r.Use(middleware.CORSMiddleware())

	r.GET("/ws", middleware.JWTAuthMiddleware(userService), func(c *gin.Context) {
		socket.ServeWs(hub, c)
	})

	return r
}
