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

// parseJWTToken 解析JWT token获取用户UUID
//func parseJWTToken(tokenString string) (string, error) {
//	// JWT token格式: header.payload.signature
//	parts := strings.Split(tokenString, ".")
//	if len(parts) != 3 {
//		return "", errors.New("invalid JWT format")
//	}
//
//	// 解码payload部分
//	payload := parts[1]
//
//	// 添加padding如果需要
//	switch len(payload) % 4 {
//	case 2:
//		payload += "=="
//	case 3:
//		payload += "="
//	}
//
//	decoded, err := base64.URLEncoding.DecodeString(payload)
//	if err != nil {
//		return "", errors.New("failed to decode payload: " + err.Error())
//	}
//
//	// 解析JSON
//	var claims struct {
//		UserUUID string `json:"useruuid"`
//		Username string `json:"username"`
//		Exp      int64  `json:"exp"`
//	}
//
//	if err := json.Unmarshal(decoded, &claims); err != nil {
//		return "", errors.New("failed to parse claims: " + err.Error())
//	}
//
//	if claims.UserUUID == "" {
//		return "", errors.New("useruuid not found in token")
//	}
//
//	// 简单的过期检查（可选）
//	// 这里不做严格的过期检查，只要能解析出UUID即可
//
//	return claims.UserUUID, nil
//}
