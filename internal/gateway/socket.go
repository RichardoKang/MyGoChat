package gateway

import (
	"MyGoChat/internal/logic/service"
	"MyGoChat/internal/model"
	"MyGoChat/pkg/log"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/gorilla/websocket"
)

// Client 是一个中间人，代表一个连接到服务器的用户。
type Client struct {
	hub    *Hub
	conn   *websocket.Conn // 与客户端的 WebSocket 连接
	send   chan []byte
	userID uint
}

// Hub 维护所有活跃的客户端和广播消息。
type Hub struct {
	clients    map[uint]*Client    // 所有活跃的客户端
	broadcast  chan *model.Message // 来自客户端的广播消息
	register   chan *Client        // 新注册的客户端
	unregister chan *Client        // 注销的客户端
	mu         sync.RWMutex        // 保护 clients 映射的读写锁
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan *model.Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[uint]*Client),
	}
}

func (h *Hub) Run() {
	log.Logger.Info("WebSocket Hub started")
	for {
		select {
		case client := <-h.register: // 新客户端注册
			h.mu.Lock()
			h.clients[client.userID] = client // 将客户端添加到活跃客户端列表
			h.mu.Unlock()
			log.Logger.Sugar().Info("Client connected: %d", client.userID)
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.userID]; ok {
				delete(h.clients, client.userID)
				close(client.send)
				log.Logger.Sugar().Info("Client disconnected: %d", client.userID)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			// Get sender's information
			sender, err := service.UserService.GetUserByID(message.SenderId)
			if err != nil {
				log.Logger.Sugar().Errorf("Error getting sender info: %v", err)
				continue
			}
			message.Sender = *sender

			h.mu.RLock()
			recipientClient, ok := h.clients[message.RecipientId]
			h.mu.RUnlock()
			if ok {
				// 将消息发送给指定的接收者,marshal用来将结构体转换为JSON格式
				msgBytes, err := json.Marshal(message)
				if err != nil {
					log.Logger.Sugar().Errorf("Error marshalling message: %v", err) // 记录错误但继续处理
					continue
				}
				select {
				case recipientClient.send <- msgBytes: // 发送消息到接收者的发送通道
				default:
					close(recipientClient.send)               // 如果发送通道阻塞，关闭通道并移除客户端
					delete(h.clients, recipientClient.userID) // 注意：这里需要加锁以保护 clients 映射
				}
			} else {
				log.Logger.Sugar().Warnf("Recipient client not found: %d", message.RecipientId)
			}
		}
	}
}

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
	userID, exists := c.Get("userID")
	if !exists {
		log.Logger.Error("User ID not found in context")
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		log.Logger.Error("User ID in context is not of type uint")
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Logger.Sugar().Errorf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), userID: uid}
	hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Logger.Sugar().Errorf("WebSocket read error: %v", err)
			}
			break
		}
		var msg model.Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Logger.Sugar().Errorf("Error unmarshalling message: %v", err)
			continue
		}
		msg.SenderId = c.userID
		c.hub.broadcast <- &msg
	}
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		}
	}
}
