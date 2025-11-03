package data

import (
	"MyGoChat/pkg/config"
	"MyGoChat/pkg/log"
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisConfig Redis 连接配置
type RedisConfig struct {
	Addr     string
	Password string
	DB       int

	// 连接池配置
	PoolSize     int
	MinIdleConns int
	MaxRetries   int

	// 超时配置
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolTimeout  time.Duration

	// 空闲连接配置
	IdleTimeout        time.Duration
	IdleCheckFrequency time.Duration
}

// GetDefaultRedisConfig 获取默认 Redis 配置
func GetDefaultRedisConfig() RedisConfig {
	return RedisConfig{
		// 连接池配置 - 根据业务需求调整
		PoolSize:     15, // 连接池大小，一般设置为 CPU 核数 * 2
		MinIdleConns: 5,  // 最小空闲连接数，保持一定的空闲连接提高响应速度
		MaxRetries:   3,  // 最大重试次数

		// 超时配置 - 平衡性能和稳定性
		DialTimeout:  5 * time.Second, // 连接超时
		ReadTimeout:  3 * time.Second, // 读取超时
		WriteTimeout: 3 * time.Second, // 写入超时
		PoolTimeout:  4 * time.Second, // 从池中获取连接的超时时间

		// 空闲连接管理 - 避免连接泄漏
		IdleTimeout:        300 * time.Second, // 5分钟空闲超时
		IdleCheckFrequency: 60 * time.Second,  // 1分钟检查一次空闲连接
	}
}

// NewRedisClient 创建 Redis 客户端 - 推荐的工厂方法
func NewRedisClient(cfg config.YamlConfig) (*redis.Client, error) {
	redisConfig := GetDefaultRedisConfig()

	// 应用用户配置
	redisConfig.Addr = cfg.Redis.Addr
	redisConfig.Password = cfg.Redis.Password
	redisConfig.DB = cfg.Redis.DB

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisConfig.Addr,
		Password: redisConfig.Password,
		DB:       redisConfig.DB,

		// 连接池配置
		PoolSize:     redisConfig.PoolSize,
		MinIdleConns: redisConfig.MinIdleConns,
		MaxRetries:   redisConfig.MaxRetries,

		// 超时配置
		DialTimeout:  redisConfig.DialTimeout,
		ReadTimeout:  redisConfig.ReadTimeout,
		WriteTimeout: redisConfig.WriteTimeout,
		PoolTimeout:  redisConfig.PoolTimeout,

		// 空闲连接检查
		IdleTimeout:        redisConfig.IdleTimeout,
		IdleCheckFrequency: redisConfig.IdleCheckFrequency,

		// 重试配置
		MaxRetryBackoff: 512 * time.Millisecond, // 最大重试间隔
		MinRetryBackoff: 8 * time.Millisecond,   // 最小重试间隔
	})

	// 连接健康检查
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	log.Logger.Sugar().Infof("Redis connected successfully to %s (DB: %d)",
		redisConfig.Addr, redisConfig.DB)

	return rdb, nil
}

// RedisHealthChecker Redis 健康检查器
type RedisHealthChecker struct {
	client *redis.Client
}

func NewRedisHealthChecker(client *redis.Client) *RedisHealthChecker {
	return &RedisHealthChecker{client: client}
}

// HealthCheck 执行健康检查
func (h *RedisHealthChecker) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return h.client.Ping(ctx).Err()
}

// GetStats 获取连接池统计信息
func (h *RedisHealthChecker) GetStats() *redis.PoolStats {
	return h.client.PoolStats()
}

// LogStats 记录连接池统计信息
func (h *RedisHealthChecker) LogStats() {
	stats := h.GetStats()
	log.Logger.Sugar().Infof(
		"Redis Pool Stats - Hits: %d, Misses: %d, Timeouts: %d, TotalConns: %d, IdleConns: %d, StaleConns: %d",
		stats.Hits, stats.Misses, stats.Timeouts,
		stats.TotalConns, stats.IdleConns, stats.StaleConns,
	)
}
