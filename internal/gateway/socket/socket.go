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
	// 允许所有来源的连接
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ServeWs 处理来自 gin 上下文的 WebSocket 请求。
func ServeWs(hub *Hub, c *gin.Context) {
	userUUID, exists := c.Get("userUUID")
	if !exists {
		log.Logger.Error("User ID not found in context")
		return
	}

	uuid, ok := userUUID.(string)
	if !ok {
		log.Logger.Error("User ID in context is not of type uint")
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Logger.Sugar().Errorf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), userUUID: uuid}
	hub.register <- client

	// 用户上线，发送离线消息同步请求到 Logic 服务
	hub.requestOfflineMessageSync(uuid)

	go client.writePump()
	go client.readPump()
}
