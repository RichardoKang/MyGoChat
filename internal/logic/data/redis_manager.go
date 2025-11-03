package data

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"MyGoChat/pkg/log"

	"github.com/go-redis/redis/v8"
)

// RedisManager Redis 管理器 - 封装常用操作和监控
type RedisManager struct {
	client        *redis.Client
	healthChecker *RedisHealthChecker
}

// NewRedisManager 创建 Redis 管理器
func NewRedisManager(client *redis.Client) *RedisManager {
	return &RedisManager{
		client:        client,
		healthChecker: NewRedisHealthChecker(client),
	}
}

// GetClient 获取 Redis 客户端
func (rm *RedisManager) GetClient() *redis.Client {
	return rm.client
}

// StartPeriodicHealthCheck 启动定期健康检查
func (rm *RedisManager) StartPeriodicHealthCheck(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			if err := rm.healthChecker.HealthCheck(); err != nil {
				log.Logger.Sugar().Errorf("Redis health check failed: %v", err)
			} else {
				log.Logger.Sugar().Debug("Redis health check passed")
			}

			// 每次健康检查时记录连接池状态
			rm.healthChecker.LogStats()
		}
	}()
}

// SafeOperation Redis 安全操作封装 - 带重试和错误处理
func (rm *RedisManager) SafeOperation(operation func(*redis.Client) error, maxRetries int) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = operation(rm.client)
		if err == nil {
			return nil
		}

		log.Logger.Sugar().Warnf("Redis operation failed (attempt %d/%d): %v", i+1, maxRetries, err)

		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond) // 递增延迟
		}
	}

	return fmt.Errorf("redis operation failed after %d attempts: %w", maxRetries, err)
}

// SetWithExpiration 设置键值对并带过期时间
func (rm *RedisManager) SetWithExpiration(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return rm.SafeOperation(func(client *redis.Client) error {
		var data []byte
		var err error

		switch v := value.(type) {
		case string:
			data = []byte(v)
		case []byte:
			data = v
		default:
			data, err = json.Marshal(value)
			if err != nil {
				return fmt.Errorf("failed to marshal value: %w", err)
			}
		}

		return client.Set(ctx, key, data, expiration).Err()
	}, 3)
}

// GetWithUnmarshal 获取并反序列化数据
func (rm *RedisManager) GetWithUnmarshal(ctx context.Context, key string, dest interface{}) error {
	return rm.SafeOperation(func(client *redis.Client) error {
		val, err := client.Get(ctx, key).Result()
		if err != nil {
			return err
		}

		return json.Unmarshal([]byte(val), dest)
	}, 3)
}

// ListPush 向列表推送数据
func (rm *RedisManager) ListPush(ctx context.Context, key string, values ...interface{}) error {
	return rm.SafeOperation(func(client *redis.Client) error {
		return client.LPush(ctx, key, values...).Err()
	}, 3)
}

// ListPopRange 批量弹出列表数据
func (rm *RedisManager) ListPopRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	var result []string
	err := rm.SafeOperation(func(client *redis.Client) error {
		var err error
		result, err = client.LRange(ctx, key, start, stop).Result()
		return err
	}, 3)

	return result, err
}

// DeleteKeys 批量删除键
func (rm *RedisManager) DeleteKeys(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	return rm.SafeOperation(func(client *redis.Client) error {
		return client.Del(ctx, keys...).Err()
	}, 3)
}

// GetConnectionInfo 获取连接信息
func (rm *RedisManager) GetConnectionInfo() map[string]interface{} {
	stats := rm.healthChecker.GetStats()

	return map[string]interface{}{
		"pool_stats": map[string]interface{}{
			"hits":        stats.Hits,
			"misses":      stats.Misses,
			"timeouts":    stats.Timeouts,
			"total_conns": stats.TotalConns,
			"idle_conns":  stats.IdleConns,
			"stale_conns": stats.StaleConns,
		},
		"network": rm.client.Options().Network,
		"db":      rm.client.Options().DB,
		"addr":    rm.client.Options().Addr,
	}
}

// Close 关闭 Redis 连接
func (rm *RedisManager) Close() error {
	log.Logger.Sugar().Info("Closing Redis connection...")
	return rm.client.Close()
}
