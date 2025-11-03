package server

import (
	"MyGoChat/internal/gateway/socket"
	"MyGoChat/internal/logic/middleware"

	"github.com/gin-gonic/gin"
)

// NewGatewayRouter 只创建网关所需的路由
func NewGatewayRouter(hub *socket.Hub) *gin.Engine {
	gin.SetMode(gin.DebugMode)
	r := gin.Default()

	r.Use(middleware.CORSMiddleware())

	r.GET("/ws", middleware.JWTAuthMiddleware(), func(c *gin.Context) {
		socket.ServeWs(hub, c)
	})

	return r
}
