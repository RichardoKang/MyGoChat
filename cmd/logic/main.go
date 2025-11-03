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
	log.InitLogger(config.GetConfig().Log.Path, config.GetConfig().Log.Level)
	log.Logger.Info("start server", log.String("start", "start web server..."))

	// Init Data
	dataObj, cleanup, err := data.NewData(config.GetConfig())
	if err != nil {
		panic(err)
	}
	defer cleanup()

	convRepo := data.NewConversationRepo(dataObj)
	msgRepo := data.NewMessageRepo(dataObj)

	// Init Services
	userService := service.NewUserService(dataObj)
	groupService := service.NewGroupService(dataObj)
	messageService := service.NewMessageService(msgRepo, convRepo, dataObj.GetRedisClient(), mq.InitProducer())
	convService := service.NewConversationService(convRepo)

	kafkaProducer := mq.InitProducer()
	defer kafkaProducer.CloseProducer()

	cfg := config.GetConfig()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	consumer := mq.InitConsumer(cfg.Kafka.Topics.Ingest, "logic_service_group")

	go mq.StartConsumer(ctx, consumer, messageService.ProcessMessage)

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
