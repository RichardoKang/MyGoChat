package socket

import (
	"MyGoChat/pkg/log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gorilla/websocket"
)

// 配置 WebSocket 升级器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ServeWs 处理 WebSocket 连接请求
func ServeWs(hub *Hub, c *gin.Context) {
	// 从 Gin Context 获取已验证的用户信息
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

	// 将 HTTP 连接升级为 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Logger.Sugar().Errorf("WebSocket upgrade error: %v", err)
		return
	}

	// 创建 Client 实例
	client := &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		userUUID: uuid,
	}

	// 注册到 Hub, Hub.Run() 会处理注册请求，更新 clients map 和 Redis 路由表
	hub.register <- client

	// 触发离线消息同步
	hub.requestOfflineMessageSync(uuid)

	// 启动读写 goroutine
	// writePump: 监听 client.send channel，将消息写入 WebSocket
	// readPump: 从 WebSocket 读取消息，发送到 Kafka ingest topic
	go client.writePump()
	go client.readPump()
}
