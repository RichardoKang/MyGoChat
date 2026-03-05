package server

import (
	"MyGoChat/gateway/internal/socket"
	"MyGoChat/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// NewGatewayRouter 只创建网关所需的路由
func NewGatewayRouter(hub *socket.Hub) *gin.Engine {
	gin.SetMode(gin.DebugMode)
	r := gin.Default()

	r.Use(middleware.CORSMiddleware())

	// WebSocket端点，使用jwt中间件验证
	r.GET("/ws", middleware.JWTAuthMiddleware(), func(c *gin.Context) {
		socket.ServeWs(hub, c)
	})

	// 添加健康检查端点
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "gateway"})
	})

	return r
}
