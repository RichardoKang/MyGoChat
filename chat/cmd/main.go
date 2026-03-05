package main

import (
	"MyGoChat/chat/internal/chat"
	"MyGoChat/chat/internal/group"
	"MyGoChat/chat/internal/platform"
	"MyGoChat/chat/internal/relation"
	"MyGoChat/chat/internal/server"
	"MyGoChat/chat/internal/user"
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

	// 自动迁移数据库表结构
	if err := platform.AutoMigrate(dataObj.GetDB(), &user.User{}, &group.Group{}, &relation.Relation{}); err != nil {
		log.Logger.Warn("database auto migrate failed, but continuing...", log.Any("error", err))
	}

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
