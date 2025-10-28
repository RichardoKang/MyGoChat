package kafka

import (
	"MyGoChat/pkg/config"
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

var writer *kafka.Writer

// InitProducer 初始化Kafka生产者, 生产者用来发送消息到Kafka主题
func InitProducer() *kafka.Writer {
	cfg := config.GetConfig()
	writer = &kafka.Writer{
		Addr:         kafka.TCP(cfg.Kafka.Brokers...),
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne, // 确保消息至少被一个副本确认
		Async:        true,
	}
	return writer
}

// SendMessage 发送消息到指定的Kafka主题
func SendMessage(topic string, message []byte) error {
	// 设置一个10秒的超时时间，以防止发送消息时阻塞过久
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Value: message,
	})
}

// CloseProducer 关闭Kafka生产者，释放资源
func CloseProducer() {
	if writer != nil {
		writer.Close()
	}
}
