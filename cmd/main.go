package main

import (
	"MyGoChat/internal/gateway"
	"MyGoChat/internal/logic/server"
	"MyGoChat/pkg/config"
	_ "MyGoChat/pkg/db" // 导入db包以触发init函数
	"MyGoChat/pkg/log"
	"net/http"
	"time"
)

func main() {
	log.InitLogger(config.GetConfig().Log.Path, config.GetConfig().Log.Level)
	log.Logger.Info("config", log.Any("config", config.GetConfig()))
	log.Logger.Info("start server", log.String("start", "start web sever..."))

	// Create and run the WebSocket hub
	hub := gateway.NewHub()
	go hub.Run()

	// Pass the hub to the router
	newRouter := server.NewRouter(hub)

	s := &http.Server{
		Addr:           ":8080",
		Handler:        newRouter,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	err := s.ListenAndServe()
	if nil != err {
		log.Logger.Error("server error", log.Any("serverError", err))
	}
}
