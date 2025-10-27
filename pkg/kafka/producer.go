package kafka

import (
	"MyGoChat/pkg/config"
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

var writer *kafka.Writer

func InitProducer() {
	cfg := config.GetConfig()
	writer = &kafka.Writer{
		Addr:         kafka.TCP(cfg.Kafka.Brokers...),
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne, // 确保消息至少被一个副本确认
		Async:        true,
	}
}

func SendMessage(topic string, message []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Value: message,
	})
}

func CloseProducer() {
	if writer != nil {
		writer.Close()
	}
}
