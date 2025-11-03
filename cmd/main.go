package main

import (
	"MyGoChat/internal/gateway/socket"
	"MyGoChat/internal/logic/data"
	"MyGoChat/internal/logic/server"
	"MyGoChat/internal/logic/service"
	"MyGoChat/pkg/config"
	myKafka "MyGoChat/pkg/kafka"
	"MyGoChat/pkg/log"
	"net/http"
	"time"
)

func main() {
	log.InitLogger(config.GetConfig().Log.Path, config.GetConfig().Log.Level)
	log.Logger.Info("start server", log.String("start", "start web sever..."))

	// Init Data
	dataObj, cleanup, err := data.NewData(config.GetConfig())
	if err != nil {
		panic(err)
	}
	defer cleanup()

	convRepo := data.NewConversationRepo(dataObj)

	// Init Services
	userService := service.NewUserService(dataObj)
	groupService := service.NewGroupService(dataObj)
	messageService := service.NewMessageService(dataObj)
	convService := service.NewConversationService(convRepo)

	kafkaProducer := myKafka.InitProducer()
	defer kafkaProducer.CloseProducer()

	hub := socket.NewHub(kafkaProducer, groupService, dataObj)
	go hub.Run()
	defer hub.Stop()

	newRouter := server.NewRouter(hub, userService, groupService, messageService, convService)

	s := &http.Server{
		Addr:           ":8080",
		Handler:        newRouter,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	err = s.ListenAndServe()
	if nil != err {
		log.Logger.Error("server error", log.Any("serverError", err))
	}
}
