package main

import (
	"MyGoChat/internal/chat"
	"MyGoChat/internal/group"
	"MyGoChat/internal/platform"
	"MyGoChat/internal/relation"
	"MyGoChat/internal/server"
	"MyGoChat/internal/user"
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
	dataObj, cleanup, err := platform.NewData(cfg)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	userRepo := user.NewUserRepo(dataObj)
	groupRepo := group.NewGroupRepo(dataObj)
	chatRepo := chat.NewChatRepo(dataObj)
	relationRepo := relation.NewRelationRepo(dataObj)

	kafkaProducer := mq.InitProducer()
	defer kafkaProducer.CloseProducer()

	// 创建会话创建器
	convCreator := chat.NewConversationCreator(chatRepo)

	// Init Services
	chatService := chat.NewService(chatRepo, relationRepo, groupRepo, userRepo, dataObj.GetRedisClient(), kafkaProducer)
	userService := user.NewService(userRepo, dataObj.GetRedisClient())
	groupService := group.NewService(groupRepo, userRepo, relationRepo)
	relationService := relation.NewService(relationRepo, userRepo, groupRepo, convCreator)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 消息处理消费者
	consumer := mq.InitConsumer(cfg.Kafka.Topics.Ingest, "logic_service_group")
	go mq.StartConsumer(ctx, consumer, chatService.ProcessMessage)

	// 同步请求消费者
	syncConsumer := mq.InitConsumer(cfg.Kafka.Topics.Sync_request, "logic_sync_group")
	go mq.StartConsumer(ctx, syncConsumer, chatService.ProcessSyncRequest)

	// Init Router
	uHandler := user.NewHandler(userService)
	gHandler := group.NewHandler(groupService)
	cHandler := chat.NewHandler(chatService)
	rHandler := relation.NewHandler(relationService)

	newRouter := server.NewRouter(uHandler, gHandler, cHandler, rHandler)

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
