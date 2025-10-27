package kafka

import (
	"MyGoChat/pkg/config"
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

func StartConsumer(ctx context.Context, topic string, groupID string, handler func(message kafka.Message)) {
	cfg := config.GetConfig()
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Kafka.Brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("Closing Kafka consumer...")
				r.Close()
				return
			default:
				m, err := r.ReadMessage(ctx)
				if err != nil {
					log.Printf("Error reading message: %v", err)
					continue
				}
				handler(m)
			}
		}
	}()
}
