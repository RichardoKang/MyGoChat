package socket

import (
	pb "MyGoChat/pkg/api/v1"
	"MyGoChat/pkg/config"
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

// DispatchMessage 是 Kafka Delivery Topic 的消息处理器
// 消息流程： Logic Service -> Kafka (Delivery Topic) -> Gateway Consumer -> DispatchMessage -> Client.send -> writePump -> WebSocket
func (h *Hub) DispatchMessage(_ context.Context, kafkaMsg kafka.Message) error {
	// Step 1: 反序列化 Protobuf 消息
	var msg pb.Message
	if err := proto.Unmarshal(kafkaMsg.Value, &msg); err != nil {
		log.Logger.Sugar().Errorf("DispatchMessage: 反序列化失败: %v", err)
		return err
	}

	// Step 2: 获取消息的目标接收者
	recipientUUID := msg.RecipientUUID
	if recipientUUID == "" {
		log.Logger.Sugar().Warnf("DispatchMessage: 消息缺少接收者UUID")
		return nil
	}

	// Step 3: 将 Protobuf 消息转换为 JSON 格式，方便前端解析
	jsonMsg, err := convertProtoToJSON(&msg)
	if err != nil {
		log.Logger.Sugar().Errorf("DispatchMessage: 转换JSON失败: %v", err)
		return err
	}

	// Step 4: 在本 Gateway 的客户端连接池中查找接收者
	// 使用读锁保护并发访问 clients map
	h.mu.RLock()
	recipientClient, ok := h.clients[recipientUUID]
	h.mu.RUnlock()

	if ok {
		// Step 5a: 用户在本 Gateway 在线，通过 channel 推送到 writePump
		// writePump 会将消息通过 WebSocket 发送给客户端
		select {
		case recipientClient.send <- jsonMsg:
			log.Logger.Sugar().Debugf("Dispatched message to user %s", recipientUUID)
		default:
			log.Logger.Sugar().Infof("Client channel full, closing connection: %s", recipientUUID)
			h.mu.Lock()
			close(recipientClient.send)
			delete(h.clients, recipientClient.userUUID)
			h.mu.Unlock()
		}
	} else {
		// Step 5b: 用户不在本 Gateway 上
		log.Logger.Sugar().Debugf("User not connected to this gateway: %s", recipientUUID)
	}

	return nil
}

// convertProtoToJSON 将 Protobuf Message 转换为 JSON 格式
// 用于发送给 WebSocket 客户端，方便前端解析
func convertProtoToJSON(msg *pb.Message) ([]byte, error) {
	// 构建一个前端友好的 JSON 结构
	jsonData := map[string]interface{}{
		"id":             msg.Id,
		"conversationID": msg.ConversationID,
		"senderUUID":     msg.SenderUUID,
		"recipientUUID":  msg.RecipientUUID,
		"sendAt":         msg.SendAt,
		"contentType":    msg.ContentType,
		"messageType":    msg.MessageType,
		"senderName":     msg.SenderName,
		"avatar":         msg.Avatar,
	}

	// 解包 Body 字段
	if msg.Body != nil {
		switch msg.ContentType {
		case 1: // Text
			var textBody pb.TextBody
			if err := msg.Body.UnmarshalTo(&textBody); err == nil {
				jsonData["body"] = map[string]interface{}{
					"content": textBody.Content,
				}
			}
		case 2, 3, 4: // Image, File, Voice
			var fileBody pb.FileAttachment
			if err := msg.Body.UnmarshalTo(&fileBody); err == nil {
				jsonData["body"] = map[string]interface{}{
					"url":      fileBody.Url,
					"fileName": fileBody.FileName,
					"size":     fileBody.Size,
					"mimeType": fileBody.MimeType,
				}
			}
		}
	}

	return json.Marshal(jsonData)
}

// Run 负责客户端连接的注册和注销
func (h *Hub) Run() {
	log.Logger.Info("WebSocket Hub started")

	for {
		select {
		case client := <-h.register:
			// 用户连接：注册到本地连接池
			h.mu.Lock()
			h.clients[client.userUUID] = client
			h.mu.Unlock()

			// Key: user_gateway:{userUUID}, Value: gatewayID
			if h.redis != nil {
				err := h.redis.Set(h.ctx, "user_gateway:"+client.userUUID, h.gatewayID, 0).Err()
				if err != nil {
					log.Logger.Sugar().Errorf("Failed to register user online status: %v", err)
				}
			}

			log.Logger.Sugar().Infof("Client connected: %s on gateway %s", client.userUUID, h.gatewayID)

		case client := <-h.unregister:
			// 用户断开：从本地连接池移除
			h.mu.Lock()
			if _, ok := h.clients[client.userUUID]; ok {
				delete(h.clients, client.userUUID)
				close(client.send)
			}
			h.mu.Unlock()

			// 从 Redis 中移除用户路由信息
			// 这样 Logic 服务就知道用户已离线，会将消息存入离线队列
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
		"useruuid":  userUUID,
		"timestamp": time.Now().Unix(),
	}

	// 序列化为 JSON
	requestData, err := json.Marshal(syncRequest)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to marshal sync request: %v", err)
		return
	}

	// 发送到 Logic 服务的同步主题
	syncTopic := config.GetConfig().Kafka.Topics.Sync_request
	err = h.Producer.SendMessage(syncTopic, requestData)
	if err != nil {
		log.Logger.Sugar().Errorf("Failed to send sync request: %v", err)
	} else {
		log.Logger.Sugar().Debugf("Sent offline message sync request for user: %s", userUUID)
	}
}
