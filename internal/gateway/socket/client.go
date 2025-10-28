package socket

import (
	pb "MyGoChat/api/v1"
	"MyGoChat/pkg/log"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

// Client 是一个中间人，代表一个连接到服务器的用户。
type Client struct {
	hub    *Hub
	conn   *websocket.Conn // 与客户端的 WebSocket 连接
	send   chan []byte
	userID uint
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
		msg := &pb.Message{}
		if err := proto.Unmarshal(messageBytes, msg); err != nil {
			log.Logger.Sugar().Errorf("Error unmarshalling message: %v", err)
			continue
		}
		// SenderID in proto is a string, so convert uint to string
		msg.SenderID = uint32(c.userID)
		c.hub.broadcast <- msg
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
