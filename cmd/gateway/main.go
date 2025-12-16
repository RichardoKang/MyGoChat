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
	// 初始化 Redis 客户端
	redisClient := myRedis.Rdb

	// kafka producer负责向 Logic Service 发送消息
	kafkaProducer := mq.InitProducer()
	defer kafkaProducer.CloseProducer()

	hub := socket.NewHub(kafkaProducer, redisClient, gatewayID)
	go hub.Run()
	defer hub.Stop()

	// kafka producer负责监控 Kafka topic为 Delivery+gatewayID 的消息，把消息分发到对应的client
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deliveryTopic := cfg.Kafka.Topics.Delivery + gatewayID
	consumer := mq.InitConsumer(deliveryTopic, deliveryTopic)
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
