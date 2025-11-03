package server

import (
	"MyGoChat/internal/gateway/socket"
	"MyGoChat/internal/logic/middleware"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
)

// NewGatewayRouter 只创建网关所需的路由
func NewGatewayRouter(hub *socket.Hub, userService interface{}) *gin.Engine {
	gin.SetMode(gin.DebugMode)
	r := gin.Default()

	r.Use(middleware.CORSMiddleware())

	// 简化版本：不使用复杂的JWT中间件，直接从token参数解析
	r.GET("/ws", func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			c.JSON(401, gin.H{"error": "Missing token parameter"})
			return
		}

		// 解析JWT获取用户信息
		userUUID, err := parseJWTToken(token)
		if err != nil {
			c.JSON(401, gin.H{"error": "Invalid token: " + err.Error()})
			return
		}

		c.Set("userUUID", userUUID)
		socket.ServeWs(hub, c)
	})

	// 添加健康检查端点
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "gateway"})
	})

	return r
}

// parseJWTToken 解析JWT token获取用户UUID
func parseJWTToken(tokenString string) (string, error) {
	// JWT token格式: header.payload.signature
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return "", errors.New("invalid JWT format")
	}

	// 解码payload部分
	payload := parts[1]

	// 添加padding如果需要
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return "", errors.New("failed to decode payload: " + err.Error())
	}

	// 解析JSON
	var claims struct {
		UserUUID string `json:"useruuid"`
		Username string `json:"username"`
		Exp      int64  `json:"exp"`
	}

	if err := json.Unmarshal(decoded, &claims); err != nil {
		return "", errors.New("failed to parse claims: " + err.Error())
	}

	if claims.UserUUID == "" {
		return "", errors.New("useruuid not found in token")
	}

	// 简单的过期检查（可选）
	// 这里不做严格的过期检查，只要能解析出UUID即可

	return claims.UserUUID, nil
}
