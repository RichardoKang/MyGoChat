package main

import (
	gwServer "MyGoChat/internal/gateway/server"
	"MyGoChat/internal/gateway/socket"
	"MyGoChat/pkg/config"
	mq "MyGoChat/pkg/kafka"
	"MyGoChat/pkg/log"
	myRedis "MyGoChat/pkg/redis"
	"context"
	"net/http"
	"os"
)

func main() {
	// 初始化配置和日志
	cfg := config.GetConfig()
	log.InitLogger(cfg.Log.Path, cfg.Log.Level)

	gatewayID := os.Getenv("GATEWAY_ID")
	if gatewayID == "" {
		gatewayID = "gateway-default" // 本地测试用
		log.Logger.Warn("GATEWAY_ID not set, using default")
	}

	// 使用 pkg/redis 包中的全局 Redis 实例
	redisClient := myRedis.Rdb

	// kafka producer负责向 Logic Service 发送消息
	kafkaProducer := mq.InitProducer()
	defer kafkaProducer.CloseProducer()

	hub := socket.NewHub(kafkaProducer, redisClient, gatewayID)
	go hub.Run()
	defer hub.Stop()

	// 【新增】启动 Gateway 自己的消费者, 处理来自 Logic Service 的消息
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deliveryTopic := cfg.Kafka.Topics.Delivery + gatewayID
	consumer := mq.InitConsumer(deliveryTopic, deliveryTopic)

	// 【关键】: 将 hub.DispatchMessage 作为处理器
	go mq.StartConsumer(ctx, consumer, hub.DispatchMessage)

	newRouter := gwServer.NewGatewayRouter(hub)

	s := &http.Server{
		Addr:    ":8081", // Gateway 运行在 8081
		Handler: newRouter,
	}
	err := s.ListenAndServe()
	if err != nil {
		log.Logger.Error("Gateway server error", log.Any("serverError", err))
	}
}
