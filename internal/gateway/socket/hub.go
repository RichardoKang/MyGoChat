package socket

import (
	pb "MyGoChat/api/v1"
	"MyGoChat/pkg/log"
	"sync"

	"google.golang.org/protobuf/proto"
)

// Hub 维护所有活跃的客户端和广播消息。
type Hub struct {
	clients    map[uint]*Client // 所有活跃的客户端
	broadcast  chan *pb.Message // 来自客户端的广播消息
	register   chan *Client     // 新注册的客户端
	unregister chan *Client     // 注销的客户端
	mu         sync.RWMutex     // 保护 clients 映射的读写锁
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan *pb.Message),
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
			log.Logger.Sugar().Infof("Client connected: %d", client.userID)
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.userID]; ok {
				delete(h.clients, client.userID)
				close(client.send)
				log.Logger.Sugar().Infof("Client disconnected: %d", client.userID)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			log.Logger.Sugar().Infof("HUB: Received message from sender %s for recipient %s", message.GetSenderID(), message.RecipientId)

			// The recipient ID from protobuf is a string, convert it to uint for map lookup
			recipientID := message.RecipientId

			log.Logger.Sugar().Infof("HUB: Parsed recipient ID: %d", recipientID)

			h.mu.RLock()
			recipientClient, ok := h.clients[uint(recipientID)]
			h.mu.RUnlock()

			if ok {
				log.Logger.Sugar().Infof("HUB: Recipient %d found. Forwarding message.", recipientID)
				// Marshal the message using protobuf
				msgBytes, err := proto.Marshal(message)
				if err != nil {
					log.Logger.Sugar().Errorf("HUB: Error marshalling proto message: %v", err)
					continue
				}
				select {
				case recipientClient.send <- msgBytes:
					log.Logger.Sugar().Infof("HUB: Message sent to user %d's send channel.", recipientID)
				default:
					log.Logger.Sugar().Warnf("HUB: User %d's send channel is full or closed. Closing connection.", recipientID)
					close(recipientClient.send)
					delete(h.clients, recipientClient.userID)
				}
			} else {
				log.Logger.Sugar().Warnf("HUB: Recipient client with ID %d not found.", recipientID)
			}
		}
	}
}
