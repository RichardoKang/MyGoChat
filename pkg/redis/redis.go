package redis

import (
	"MyGoChat/pkg/config"
	"context"

	"github.com/go-redis/redis/v8"
)

var (
	Rdb *redis.Client
	Ctx = context.Background()
)

// InitRedisClient 初始化 Redis 客户端
func InitRedisClient(cfg *config.YamlConfig) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	if _, err := client.Ping(Ctx).Result(); err != nil {
		panic(err)
	}

	return client
}

// 兼容旧的全局初始化方式
func init() {
	cfg := config.GetConfig()
	Rdb = redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	if _, err := Rdb.Ping(Ctx).Result(); err != nil {
		panic(err)
	}
}
