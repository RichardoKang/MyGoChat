package socket

import (
	pb "MyGoChat/api/v1"
	"MyGoChat/pkg/config"
	"MyGoChat/pkg/log"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
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

	ingestTopic := config.GetConfig().Kafka.Topics.Ingest

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			log.Logger.Sugar().Errorf("Error reading message: %v", err)
			break
		}
		msg := &pb.Message{}
		if err := proto.Unmarshal(messageBytes, msg); err != nil {
			log.Logger.Sugar().Errorf("Error unmarshalling message: %v", err)
			continue
		}

		msg.SenderUUID = c.userUUID

		//c.hub.broadcast <- msg // 发送到Hub的广播通道，现在改为发送到Kafka producer

		serializedMsg, err := proto.Marshal(msg)
		if err != nil {
			log.Logger.Sugar().Errorf("Error marshalling message: %v", err)
			continue
		}

		producer := c.hub.Producer
		err = producer.SendMessage(ingestTopic, serializedMsg)
		if err != nil {
			return
		}
	}
}

// writePump 将消息从集线器发送到 WebSocket 连接。
func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				log.Logger.Sugar().Info("WRITE PUMP: Hub closed channel.")
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				log.Logger.Sugar().Errorf("WRITE PUMP: Error getting next writer: %v", err)
				return
			}
			log.Logger.Sugar().Info("WRITE PUMP: Writing message to websocket.")

			w.Write(message)

			if err := w.Close(); err != nil {
				log.Logger.Sugar().Errorf("WRITE PUMP: Error closing writer: %v", err)
				return
			}
		}
	}
}
