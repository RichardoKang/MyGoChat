package socket

import (
	pb "MyGoChat/api/v1"
	"MyGoChat/pkg/config"
	myKafka "MyGoChat/pkg/kafka"
	"MyGoChat/pkg/log"
	"context"
	"strconv"
	"sync"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

type Hub struct {
	clients    map[uint]*Client
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	Producer   *myKafka.Producer
	// ... 移除所有 consumer 和 logic 依赖 ...
}

func NewHub(producer *myKafka.Producer) *Hub {
	// ... (移除 consumer, ctx, cancel, groupService, redis)
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[uint]*Client),
		Producer:   producer,
	}
}

// 【新增】DispatchMessage 是 Kafka 处理器
func (h *Hub) DispatchMessage(kafkaMsg kafka.Message) {
	var msg pb.Message
	if err := proto.Unmarshal(kafkaMsg.Value, &msg); err != nil {
		log.Logger.Sugar().Errorf("DispatchMessage: 反序列化失败: %v", err)
		return
	}

	var recipientID uint

	if msg.MessageType == 1 { // 私聊
		// RecipientID 字段在你的 proto 里是 int32
		// 而你的 client.userID 是 uint
		// 这里假设它们都是 uint
		recipientID = uint(msg.RecipientID)
	} else if msg.MessageType == 2 {
		// TODO: Logic 服务应该把消息推给群里的每一个人
		// Kafka 消息应该包含一个 'to_user_id' 字段，而不是群ID
		// (这是下一步要优化的，现在我们先假设 RecipientID 是 UserID)
		recipientID = uint(msg.RecipientID) // <-- 这是一个待修复的逻辑
	}

	h.mu.RLock()
	recipientClient, ok := h.clients[recipientID]
	h.mu.RUnlock()

	if ok {
		select {
		case recipientClient.send <- kafkaMsg.Value:
			// 成功
		default:
			// 客户端 channel 满了，关闭
			h.mu.Lock()
			close(recipientClient.send)
			delete(h.clients, recipientClient.userID)
			h.mu.Unlock()
		}
	} else {
		// 用户在此网关不在线 (正常，可能在别的网关，或已离线)
	}
}

// 【修改】Run() 只负责注册/注销
func (h *Hub) Run() {
	log.Logger.Info("WebSocket Hub started")
	// ... (移除 consumer 启动代码)
	// ... (移除 severID 和 redis)

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.userID] = client
			// 【重要】: 注册 Redis 的逻辑应该移到这里
			// h.redis.HSet(h.ctx, "online_users", ...)
			h.mu.Unlock()
			log.Logger.Sugar().Infof("Client connected: %d", client.userID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.userID]; ok {
				delete(h.clients, client.userID)
				close(client.send)
				// 【重要】: 注销 Redis 的逻辑也应该移到这里
				// h.redis.HDel(h.ctx, "online_users", ...)
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) Stop() {
	// ... (移除 consumer.Close())
}
