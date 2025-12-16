package kafka

import (
	"MyGoChat/pkg/config"
	"MyGoChat/pkg/log"
	"context"

	"github.com/segmentio/kafka-go"
)

type Consumer interface {
	ReadMessage(context.Context) (kafka.Message, error)
	Close() error
}

func InitConsumer(topic, groupID string) Consumer {
	cfg := config.GetConfig()
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Kafka.Brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

}

type MessageHandler func(ctx context.Context, msg kafka.Message) error

// StartConsumer 阻塞式消费者
// 建议加上 error 返回值以便 Handler 能够反馈处理结果
func StartConsumer(ctx context.Context, r Consumer, handler MessageHandler) {
	defer func() {
		if err := r.Close(); err != nil {
			log.Logger.Sugar().Errorf("Error closing consumer: %v", err)
		}
	}()

	log.Logger.Sugar().Info("Kafka consumer started")

	for {
		// 1. 检查上下文是否已取消 (避免在关闭时还在拉取新消息)
		select {
		case <-ctx.Done():
			return
		default:
		}

		// 2. 读取消息
		m, err := r.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // 上下文取消导致的错误，直接退出
			}
			log.Logger.Sugar().Errorf("Error reading message: %v", err)
			continue
		}

		// 3. 【关键】将 ctx 传递给 Handler
		// 这样 Handler 内部的 DB 操作就能感知到 main 函数的关闭信号了
		if err := handler(ctx, m); err != nil {
			log.Logger.Sugar().Errorf("Handler failed: %v", err)
			// TODO: 这里可以根据 error 类型决定是否需要重试或提交偏移量
		}
	}
}
