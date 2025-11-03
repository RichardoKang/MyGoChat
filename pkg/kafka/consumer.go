package kafka

import (
	"MyGoChat/pkg/config"
	"context"
	"sync"

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

// StartConsumer 启动一个Kafka消费者，持续读取消息并通过handler处理。
func StartConsumer(ctx context.Context, r Consumer, handler func(kafka.Message)) (stop func(), done <-chan struct{}) {
	var once sync.Once
	d := make(chan struct{})
	go func() {
		defer close(d)
		for {
			select {
			case <-ctx.Done():
				// 当上下文被取消时，关闭消费者并退出
				once.Do(func() { r.Close() })
				return
			default:
				m, err := r.ReadMessage(ctx)
				if err != nil {
					if ctx.Err() != nil {
						// 上下文取消，优雅退出
						once.Do(func() { r.Close() })
						return
					}
					// 临时错误：记录并继续/重试
					continue
				}
				handler(m)
			}
		}
	}()
	// 返回一个停止函数，用于关闭消费者
	stop = func() { once.Do(func() { r.Close() }) }
	return stop, d
}
