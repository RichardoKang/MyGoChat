package socket

import (
	pb "MyGoChat/api/v1"
	"MyGoChat/pkg/config"
	myKafka "MyGoChat/pkg/kafka"
	"MyGoChat/pkg/log"
	"context"
	"sync"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

// Hub 维护所有活跃的客户端和广播消息。
type Hub struct {
	clients         map[uint]*Client // 所有活跃的客户端
	register        chan *Client     // 新注册的客户端
	unregister      chan *Client     // 注销的客户端
	mu              sync.RWMutex     // 保护 clients 映射的读写锁
	Producer        *myKafka.Producer
	PrivateConsumer myKafka.Consumer
	GroupConsumer   myKafka.Consumer
	ctx             context.Context
	cancel          context.CancelFunc
}

func NewHub(producer *myKafka.Producer) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := config.GetConfig()
	// 初始化私聊和群聊的消费者
	privateConsumer := myKafka.InitConsumer(cfg.Kafka.Topics.Private, "private_consumer_group")
	groupConsumer := myKafka.InitConsumer(cfg.Kafka.Topics.Group, "group_consumer_group")

	return &Hub{
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		clients:         make(map[uint]*Client),
		Producer:        producer,
		PrivateConsumer: privateConsumer,
		GroupConsumer:   groupConsumer,
		ctx:             ctx,
		cancel:          cancel,
	}
}

// dispatchPrivateMessage 处理来自Kafka的私聊消息
func (h *Hub) dispatchPrivateMessage(kafkaMsg kafka.Message) {
	var msg pb.Message
	if err := proto.Unmarshal(kafkaMsg.Value, &msg); err != nil {
		log.Logger.Sugar().Errorf("Error unmarshalling private message: %v", err)
		return
	}

	recipientID := msg.RecipientID
	h.mu.RLock()
	recipientClient, ok := h.clients[uint(recipientID)]
	h.mu.RUnlock()

	if ok {
		log.Logger.Sugar().Infof("HUB: Recipient %d found. Forwarding private message.", recipientID)
		select {
		case recipientClient.send <- kafkaMsg.Value:
			log.Logger.Sugar().Infof("HUB: Message sent to user %d's send channel.", recipientID)
		default:
			log.Logger.Sugar().Warnf("HUB: User %d's send channel is full or closed. Closing connection.", recipientID)
			h.mu.Lock()
			close(recipientClient.send)
			delete(h.clients, recipientClient.userID)
			h.mu.Unlock()
		}
	} else {
		log.Logger.Sugar().Warnf("HUB: Recipient client with ID %d not found for private message.", recipientID)
	}
}

// dispatchGroupMessage 处理来自Kafka的群聊消息
func (h *Hub) dispatchGroupMessage(kafkaMsg kafka.Message) {
	var msg pb.Message
	if err := proto.Unmarshal(kafkaMsg.Value, &msg); err != nil {
		log.Logger.Sugar().Errorf("Error unmarshalling group message: %v", err)
		return
	}

	// TODO: 实现群聊消息分发逻辑
	// 1. 从 msg.RecipientID (即 GroupID) 获取所有群成员的 userID 列表。
	// 2. 遍历群成员 userID 列表。
	// 3. 对于每个在线的成员（即存在于 h.clients 中），将消息发送到其 send channel。

	log.Logger.Sugar().Infof("HUB: Received group message for group %d. Dispatch logic is not yet implemented.", msg.RecipientID)

}

func (h *Hub) Run() {
	log.Logger.Info("WebSocket Hub started")

	// 启动私聊和群聊的消费者
	go myKafka.StartConsumer(h.ctx, h.PrivateConsumer, h.dispatchPrivateMessage)
	go myKafka.StartConsumer(h.ctx, h.GroupConsumer, h.dispatchGroupMessage)

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
		}
	}
}

func (h *Hub) Stop() {
	log.Logger.Info("Stopping WebSocket Hub...")
	h.cancel()
	h.PrivateConsumer.Close()
	h.GroupConsumer.Close()
}
