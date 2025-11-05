package main

import (
	"MyGoChat/internal/logic/data"
	"MyGoChat/internal/logic/server"
	"MyGoChat/internal/logic/service"
	"MyGoChat/pkg/config"
	mq "MyGoChat/pkg/kafka"
	"MyGoChat/pkg/log"
	"context"
	"net/http"
	"time"
)

func main() {
	cfg := config.GetConfig()
	log.InitLogger(cfg.Log.Path, cfg.Log.Level)
	log.Logger.Info("start server", log.String("start", "start web server..."))

	// Init Data
	dataObj, cleanup, err := data.NewData(cfg)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	userRepo := data.NewUserRepo(dataObj)
	groupRepo := data.NewGroupRepo(dataObj)
	convRepo := data.NewConversationRepo(dataObj)
	msgRepo := data.NewMessageRepo(dataObj)

	// Init Services
	userService := service.NewUserService(userRepo, dataObj.GetRedisClient())
	groupService := service.NewGroupService(groupRepo, userRepo)
	// 使用 RedisManager 提供更强大的 Redis 操作
	messageService := service.NewMessageService(msgRepo, convRepo, groupRepo, dataObj.GetRedisClient(), mq.InitProducer())
	convService := service.NewConversationService(convRepo)

	kafkaProducer := mq.InitProducer()
	defer kafkaProducer.CloseProducer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 消息处理消费者
	consumer := mq.InitConsumer(cfg.Kafka.Topics.Ingest, "logic_service_group")
	go mq.StartConsumer(ctx, consumer, messageService.ProcessMessage)

	// 同步请求消费者
	syncConsumer := mq.InitConsumer(cfg.Kafka.Topics.Sync_request, "logic_sync_group")
	go mq.StartConsumer(ctx, syncConsumer, messageService.ProcessSyncRequest)

	newRouter := server.NewRouter(userService, groupService, messageService, convService)

	s := &http.Server{
		Addr:           ":8080",
		Handler:        newRouter,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	err = s.ListenAndServe()
	if err != nil {
		log.Logger.Error("server error", log.Any("serverError", err))
	}
}
