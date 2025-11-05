package socket

import (
	pb "MyGoChat/api/v1"
	"MyGoChat/pkg/config"
	"MyGoChat/pkg/log"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	// 心跳相关常量
	writeWait      = 10 * time.Second    // 写操作超时时间
	pongWait       = 60 * time.Second    // 等待 pong 消息的超时时间
	pingPeriod     = (pongWait * 9) / 10 // ping 消消息发送周期，必须小于 pongWait
	maxMessageSize = 512                 // 最大消息大小
)

// Client 是一个中间人，代表一个连接到服务器的用户。
type Client struct {
	hub      *Hub
	conn     *websocket.Conn // 与客户端的 WebSocket 连接
	send     chan []byte
	userUUID string
}

// readPump 从 WebSocket 连接中读取消息并将其发送到Hub的kafka producer.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	// 设置连接参数
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		log.Logger.Sugar().Debugf("Received pong from client: %s", c.userUUID)
		return nil
	})

	ingestTopic := config.GetConfig().Kafka.Topics.Ingest

	for {
		// 读取消息
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Logger.Sugar().Errorf("Unexpected websocket close error: %v", err)
			} else {
				log.Logger.Sugar().Debugf("Client disconnected: %s, error: %v", c.userUUID, err)
			}
			break
		}

		// 尝试解析消息 - 支持 JSON 和 protobuf 两种格式
		msg, err := c.parseMessage(messageBytes)
		if err != nil {
			log.Logger.Sugar().Errorf("Error parsing message: %v", err)
			log.Logger.Sugar().Debugf("Raw message bytes: %s", string(messageBytes))
			continue
		}

		// 设置发送者UUID
		msg.SenderUUID = c.userUUID

		// 序列化为protobuf发送给Kafka
		serializedMsg, err := proto.Marshal(msg)
		if err != nil {
			log.Logger.Sugar().Errorf("Error marshalling message: %v", err)
			continue
		}

		producer := c.hub.Producer
		err = producer.SendMessage(ingestTopic, serializedMsg)
		if err != nil {
			log.Logger.Sugar().Errorf("Error sending message to Kafka: %v", err)
			return
		}
	}
}

// writePump 将消息从集线器发送到 WebSocket 连接。
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub 关闭了 channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				log.Logger.Sugar().Errorf("Error getting next writer: %v", err)
				return
			}

			w.Write(message)

			// 添加排队的聊天消息到当前的 WebSocket 消息
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				log.Logger.Sugar().Errorf("Error closing writer: %v", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Logger.Sugar().Errorf("Error sending ping message: %v", err)
				return
			}
		}
	}
}

// parseMessage 解析消息，支持 JSON 和 protobuf 两种格式
func (c *Client) parseMessage(messageBytes []byte) (*pb.Message, error) {
	// 先尝试解析为 protobuf 格式
	var msg pb.Message
	if err := proto.Unmarshal(messageBytes, &msg); err == nil {
		log.Logger.Sugar().Debugf("Parsed as protobuf message")
		return &msg, nil
	}

	// 如果 protobuf 解析失败，尝试解析为 JSON 格式
	return c.parseJSONMessage(messageBytes)
}

// parseJSONMessage 解析 JSON 格式的消息并转换为 protobuf 格式
func (c *Client) parseJSONMessage(messageBytes []byte) (*pb.Message, error) {
	// 定义临时结构体来解析 JSON
	var jsonMsg struct {
		ID             string      `json:"id"`
		ConversationID string      `json:"conversationID"`
		SenderUUID     string      `json:"senderUUID"`
		SendAt         int64       `json:"sendAt"`
		ContentType    int32       `json:"contentType"`
		Body           interface{} `json:"body"`
		MessageType    int32       `json:"messageType"`
		RecipientUUID  string      `json:"recipientUUID"`
		SenderName     string      `json:"senderName"`
		Avatar         string      `json:"avatar"`
	}

	if err := json.Unmarshal(messageBytes, &jsonMsg); err != nil {
		log.Logger.Sugar().Errorf("Failed to parse as JSON: %v", err)
		return nil, err
	}

	log.Logger.Sugar().Debugf("Parsed as JSON message: %+v", jsonMsg)

	// 转换为 protobuf 消息
	msg := &pb.Message{
		Id:             jsonMsg.ID,
		ConversationID: jsonMsg.ConversationID,
		SenderUUID:     jsonMsg.SenderUUID,
		SendAt:         jsonMsg.SendAt,
		ContentType:    jsonMsg.ContentType,
		MessageType:    jsonMsg.MessageType,
		RecipientUUID:  jsonMsg.RecipientUUID,
		SenderName:     jsonMsg.SenderName,
		Avatar:         jsonMsg.Avatar,
	}

	// 处理 body 字段 - 转换为 google.protobuf.Any
	if jsonMsg.Body != nil {
		anyBody, err := c.convertBodyToAny(jsonMsg.ContentType, jsonMsg.Body)
		if err != nil {
			log.Logger.Sugar().Errorf("Failed to convert body to Any: %v", err)
			return nil, err
		}
		msg.Body = anyBody
	}

	return msg, nil
}

// convertBodyToAny 将 JSON body 转换为 google.protobuf.Any 类型
func (c *Client) convertBodyToAny(contentType int32, body interface{}) (*anypb.Any, error) {
	switch contentType {
	case 1: // 文本消息
		// 解析 body 中的 content 字段
		if bodyMap, ok := body.(map[string]interface{}); ok {
			if content, exists := bodyMap["content"]; exists {
				textBody := &pb.TextBody{
					Content: content.(string),
				}
				return anypb.New(textBody)
			}
		}
		return nil, fmt.Errorf("invalid text body format")

	case 2, 3, 4: // 图片、文件、语音消息
		if bodyMap, ok := body.(map[string]interface{}); ok {
			fileBody := &pb.FileAttachment{
				Url:      getStringValue(bodyMap, "url"),
				FileName: getStringValue(bodyMap, "fileName"),
				Size:     getInt64Value(bodyMap, "size"),
				MimeType: getStringValue(bodyMap, "mimeType"),
			}
			return anypb.New(fileBody)
		}
		return nil, fmt.Errorf("invalid file body format")

	default:
		return nil, fmt.Errorf("unsupported content type: %d", contentType)
	}
}

// 辅助函数：安全地从 map 中获取字符串值
func getStringValue(m map[string]interface{}, key string) string {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// 辅助函数：安全地从 map 中获取 int64 值
func getInt64Value(m map[string]interface{}, key string) int64 {
	if val, exists := m[key]; exists {
		switch v := val.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case float64:
			return int64(v)
		}
	}
	return 0
}
