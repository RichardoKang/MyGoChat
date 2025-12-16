package socket

import (
	"MyGoChat/pkg/log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gorilla/websocket"
)

// 配置 WebSocket 升级器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024, // 读取缓冲区大小
	WriteBufferSize: 1024, // 写入缓冲区大小
	// 允许所有来源的连接（生产环境应该配置具体的域名）
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ServeWs 处理 WebSocket 连接请求
// 这是 WebSocket 握手的入口点，由 Gin 路由调用
//
// 完整的连接建立流程：
// 1. HTTP 请求 -> Gin Router (/ws) -> JWTAuthMiddleware (验证 token) -> ServeWs
// 2. ServeWs 从 Gin Context 获取已验证的 userUUID（由 JWT 中间件设置）
// 3. 升级 HTTP 连接为 WebSocket
// 4. 创建 Client 实例并注册到 Hub
// 5. 触发离线消息同步
// 6. 启动读写 goroutine
//
// 安全说明：
// - userUUID 来自 JWT token 解析，是可信的
// - 即使客户端在消息中伪造 SenderUUID，readPump 也会用真实的 userUUID 覆盖
func ServeWs(hub *Hub, c *gin.Context) {
	// Step 1: 从 Gin Context 获取已验证的用户信息
	// 这个值由 JWTAuthMiddleware 在验证 token 后设置
	userUUID, exists := c.Get("useruuid")
	if !exists {
		log.Logger.Error("User ID not found in context - JWT middleware may not have run")
		return
	}

	uuid, ok := userUUID.(string)
	if !ok {
		log.Logger.Error("User ID in context is not of type string")
		return
	}

	// Step 2: 将 HTTP 连接升级为 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Logger.Sugar().Errorf("WebSocket upgrade error: %v", err)
		return
	}

	// Step 3: 创建 Client 实例
	// userUUID 是经过 JWT 验证的真实用户标识
	client := &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256), // 发送缓冲区
		userUUID: uuid,                   // 绑定真实用户ID
	}

	// Step 4: 注册到 Hub
	// Hub.Run() 会处理注册请求，更新 clients map 和 Redis 路由表
	hub.register <- client

	// Step 5: 触发离线消息同步
	// 发送请求到 Kafka sync_request topic，Logic 服务会将离线消息推送给用户
	hub.requestOfflineMessageSync(uuid)

	// Step 6: 启动读写 goroutine
	// writePump: 监听 client.send channel，将消息写入 WebSocket
	// readPump: 从 WebSocket 读取消息，发送到 Kafka ingest topic
	go client.writePump()
	go client.readPump()
}
