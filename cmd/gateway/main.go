// cmd/gateway/main.go
package main

import (
	gwServer "MyGoChat/internal/gateway/server" // 导入我们新创建的 gateway 路由
	"MyGoChat/internal/gateway/socket"
	"MyGoChat/internal/logic/data"    // <-- 临时依赖
	"MyGoChat/internal/logic/service" // <-- 临时依赖
	"MyGoChat/pkg/config"
	myKafka "MyGoChat/pkg/kafka"
	"MyGoChat/pkg/log"
	"net/http"
	"time"
)

func main() {
	log.InitLogger(config.GetConfig().Log.Path, config.GetConfig().Log.Level)
	log.Logger.Info("start GATEWAY server", log.String("start", "start gateway server..."))

	// --- 临时的依赖 ---
	// hub.go 目前仍然直接依赖 logic 的 service 和 data
	// 这是我们下一步要修复的，但为了启动，我们先初始化它们
	dataObj, cleanup, err := data.NewData(config.GetConfig())
	if err != nil {
		panic(err)
	}
	defer cleanup()
	groupService := service.NewGroupService(dataObj)
	userService := service.NewUserService(dataObj)

	// Init Kafka Producer
	kafkaProducer := myKafka.InitProducer()
	defer kafkaProducer.CloseProducer()

	// Init Hub (注意，我们传入了临时的依赖)
	hub := socket.NewHub(kafkaProducer, groupService, dataObj)
	go hub.Run()
	defer hub.Stop()

	// Init Router (使用我们新创建的 gateway 专用 Router)
	newRouter := gwServer.NewGatewayRouter(hub, userService)

	// Start HTTP Server
	s := &http.Server{
		Addr:           ":8081", // Gateway 服务运行在 8081 (避免端口冲突)
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
