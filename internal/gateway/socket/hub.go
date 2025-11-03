package socket

import (
	pb "MyGoChat/api/v1"
	myKafka "MyGoChat/pkg/kafka"
	"MyGoChat/pkg/log"
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

type Hub struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	Producer   *myKafka.Producer
	redis      *redis.Client
	gatewayID  string // 网关唯一标识
	ctx        context.Context
}

func NewHub(producer *myKafka.Producer, redisClient *redis.Client, gatewayID string) *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[string]*Client),
		Producer:   producer,
		redis:      redisClient,
		gatewayID:  gatewayID,
		ctx:        context.Background(),
	}
}

// 【新增】DispatchMessage 是 Kafka 处理器
func (h *Hub) DispatchMessage(kafkaMsg kafka.Message) {
	var msg pb.Message
	if err := proto.Unmarshal(kafkaMsg.Value, &msg); err != nil {
		log.Logger.Sugar().Errorf("DispatchMessage: 反序列化失败: %v", err)
		return
	}

	// 获取消息的接收者UUID
	recipientUUID := msg.RecipientUUID
	if recipientUUID == "" {
		log.Logger.Sugar().Warnf("DispatchMessage: 消息缺少接收者UUID")
		return
	}

	// 查找接收者客户端连接
	h.mu.RLock()
	recipientClient, ok := h.clients[recipientUUID]
	h.mu.RUnlock()

	if ok {
		// 用户在线，直接推送
		select {
		case recipientClient.send <- kafkaMsg.Value:
			log.Logger.Sugar().Debugf("Message delivered to online user: %s", recipientUUID)
		default:
			// 客户端 channel 满了，关闭连接
			log.Logger.Sugar().Warnf("Client channel full, closing connection: %s", recipientUUID)
			h.mu.Lock()
			close(recipientClient.send)
			delete(h.clients, recipientClient.userUUID)
			h.mu.Unlock()
		}
	} else {
		// 用户在此网关不在线 (可能在别的网关，或已离线)
		log.Logger.Sugar().Debugf("User not connected to this gateway: %s", recipientUUID)
	}
}

// 【修改】Run() 只负责注册/注销
func (h *Hub) Run() {
	log.Logger.Info("WebSocket Hub started")

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.userUUID] = client
			h.mu.Unlock()

			// 在 Redis 中注册用户在线状态
			if h.redis != nil {
				err := h.redis.Set(h.ctx, "user_gateway:"+client.userUUID, h.gatewayID, 0).Err()
				if err != nil {
					log.Logger.Sugar().Errorf("Failed to register user online status: %v", err)
				}
			}

			log.Logger.Sugar().Infof("Client connected: %s", client.userUUID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.userUUID]; ok {
				delete(h.clients, client.userUUID)
				close(client.send)
			}
			h.mu.Unlock()

			// 从 Redis 中移除用户在线状态
			if h.redis != nil {
				err := h.redis.Del(h.ctx, "user_gateway:"+client.userUUID).Err()
				if err != nil {
					log.Logger.Sugar().Errorf("Failed to unregister user online status: %v", err)
				}
			}

			log.Logger.Sugar().Infof("Client disconnected: %s", client.userUUID)
		}
	}
}

// Stop 优雅关闭 Hub
func (h *Hub) Stop() {
	log.Logger.Info("Stopping WebSocket Hub...")

	h.mu.Lock()
	defer h.mu.Unlock()

	// 关闭所有客户端连接
	for userUUID, client := range h.clients {
		close(client.send)

		// 从 Redis 中移除用户在线状态
		if h.redis != nil {
			err := h.redis.Del(h.ctx, "user_gateway:"+userUUID).Err()
			if err != nil {
				log.Logger.Sugar().Errorf("Failed to cleanup user online status: %v", err)
			}
		}
	}

	// 清空客户端映射
	h.clients = make(map[string]*Client)

	log.Logger.Info("WebSocket Hub stopped")
}

// requestOfflineMessageSync 请求同步用户的离线消息
func (h *Hub) requestOfflineMessageSync(userUUID string) {
	// 构造同步请求消息
	syncRequest := map[string]interface{}{
		"action":    "sync_offline",
		"user_uuid": userUUID,
		"timestamp": time.Now().Unix(),
	}

	// 序列化为 JSON
	requestData, err := json.Marshal(syncRequest)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to marshal sync request: %v", err)
		return
	}

	// 发送到 Logic 服务的同步主题
	syncTopic := "im_sync_request"
	err = h.Producer.SendMessage(syncTopic, requestData)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to send sync request: %v", err)
	} else {
		log.Logger.Sugar().Debugf("Sent offline message sync request for user: %s", userUUID)
	}
}
