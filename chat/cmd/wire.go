//go:build wireinject
// +build wireinject

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

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
)

// UserSet 聚合 User 模块的所有构造函数
var UserSet = wire.NewSet(
	user.NewUserRepo,
	user.NewService,
	user.NewHandler,
)

// GroupSet 聚合 Group 模块
var GroupSet = wire.NewSet(
	group.NewGroupRepo,
	group.NewService,
	group.NewHandler,
)

// ChatSet 聚合 Chat 模块
var ChatSet = wire.NewSet(
	chat.NewChatRepo,
	chat.NewService,
	chat.NewHandler,
)

// RelationSet 聚合 Relation 模块
var RelationSet = wire.NewSet(
	relation.NewRelationRepo,
	relation.NewService,
	relation.NewHandler,
)

// 声明注入器函数签名
func InitializeApp() (*gin.Engine, func(), error) {
	panic(wire.Build(
		// 1. 基础配置和数据层
		config.GetConfig,
		platform.NewData, // 返回 (*Data, func(), error)
		mq.InitProducer,

		// 2. 从 Data 提取子依赖 (Helper functions)
		// Wire 无法自动识别 data.GetRedisClient 这种方法，需要显式转换或提供 getter
		// 建议在 platform 包里加几个简单的 Provider 函数，或者用 FieldsOf
		provideRedis,

		// 3. 业务模块 Set
		UserSet,
		GroupSet,
		ChatSet,
		RelationSet,

		// 4. HTTP Server
		server.NewRouter,
	))
}

// 辅助函数：从 Data 提取 Redis，供 wire 使用
func provideRedis(data *platform.Data) interface{} {
	return data.GetRedisClient()
}
