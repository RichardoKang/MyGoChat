package main

import (
	gwServer "MyGoChat/internal/gateway/server"
	"MyGoChat/internal/gateway/socket"
	myKafka "MyGoChat/pkg/kafka"
	"MyGoChat/pkg/log"
	"context"
	"net/http"
	"os"
)

func main() {

	// 【重要】网关必须知道自己的 ID
	gatewayID := os.Getenv("GATEWAY_ID")
	if gatewayID == "" {
		gatewayID = "gateway-default" // 本地测试用
		log.Logger.Warn("GATEWAY_ID not set, using default")
	}

	kafkaProducer := myKafka.InitProducer()
	defer kafkaProducer.CloseProducer()

	hub := socket.NewHub(kafkaProducer)
	go hub.Run()
	defer hub.Stop()

	// 【新增】启动 Gateway 自己的消费者
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deliveryTopic := "im_delivery_" + gatewayID
	consumer := myKafka.InitConsumer(deliveryTopic, deliveryTopic)

	// 【关键】: 将 hub.DispatchMessage 作为处理器
	go myKafka.StartConsumer(ctx, consumer, hub.DispatchMessage)

	newRouter := gwServer.NewGatewayRouter(hub) // userService 仍用于 JWT

	s := &http.Server{
		Addr:    ":8081", // Gateway 运行在 8081
		Handler: newRouter,
	}
	err := s.ListenAndServe()
	if err != nil {
		log.Logger.Error("Gateway server error", log.Any("serverError", err))
	}
}
