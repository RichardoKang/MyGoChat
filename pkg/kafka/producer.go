package kafka

import (
	"MyGoChat/pkg/config"
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

// InitProducer 初始化Kafka生产者, 生产者用来发送消息到Kafka主题
func InitProducer() *Producer {
	cfg := config.GetConfig()
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Kafka.Brokers...),
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne, // 确保消息至少被一个副本确认
		Async:        true,
	}
	return &Producer{writer: writer}
}

// SendMessage 发送消息到指定的Kafka主题（无 Key）
func (p *Producer) SendMessage(topic string, message []byte) error {
	return p.SendMessageWithKey(topic, nil, message)
}

// SendMessageWithKey 发送消息到指定的Kafka主题（带 Key）
// 使用 Key 可以保证相同 Key 的消息发送到同一个分区，确保消息顺序性
func (p *Producer) SendMessageWithKey(topic string, key, message []byte) error {
	// 设置一个10秒的超时时间，以防止发送消息时阻塞过久
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   key,
		Value: message,
	})
}

// CloseProducer 关闭Kafka生产者，释放资源
func (p *Producer) CloseProducer() {
	if p.writer != nil {
		p.writer.Close()
	}
}
