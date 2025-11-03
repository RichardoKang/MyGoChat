package data

import (
	"MyGoChat/pkg/config"
	"MyGoChat/pkg/log"
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Data struct {
	Mdb          *mongo.Database
	Rdb          *redis.Client
	RedisManager *RedisManager // 新增：Redis 管理器
	Db           *gorm.DB
}

func NewData(cfg config.YamlConfig) (*Data, func(), error) {
	rdb := initRedisDB(cfg)

	data := &Data{
		Mdb:          initMongoDB(cfg),
		Rdb:          rdb,
		RedisManager: NewRedisManager(rdb),
		Db:           initPostgresDB(cfg),
	}

	// 启动 Redis 健康检查 (每5分钟检查一次)
	data.RedisManager.StartPeriodicHealthCheck(5 * time.Minute)

	log.Logger.Sugar().Infof("init data success")
	cleanup := func() {
		log.Logger.Sugar().Info("Cleaning up data connections...")

		// 关闭 PostgreSQL 连接
		if sqlDB, err := data.Db.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				log.Logger.Sugar().Errorf("Failed to close PostgreSQL connection: %v", err)
			} else {
				log.Logger.Sugar().Info("PostgreSQL connection closed")
			}
		}

		// 关闭 MongoDB 连接
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := data.Mdb.Client().Disconnect(ctx); err != nil {
			log.Logger.Sugar().Errorf("Failed to close MongoDB connection: %v", err)
		} else {
			log.Logger.Sugar().Info("MongoDB connection closed")
		}

		// 关闭 Redis 连接
		if data.RedisManager != nil {
			if err := data.RedisManager.Close(); err != nil {
				log.Logger.Sugar().Errorf("Failed to close Redis connection: %v", err)
			} else {
				log.Logger.Sugar().Info("Redis connection closed")
			}
		}

		log.Logger.Sugar().Info("All data connections cleaned up")
	}
	return data, cleanup, nil
}

func (d *Data) GetDB() *gorm.DB {
	return d.Db
}

func (d *Data) GetMongoDB() *mongo.Database {
	return d.Mdb
}

func (d *Data) GetRedisClient() *redis.Client {
	return d.Rdb
}

// GetRedisManager 获取 Redis 管理器 - 推荐使用这个方法
func (d *Data) GetRedisManager() *RedisManager {
	return d.RedisManager
}

func initPostgresDB(cfg config.YamlConfig) *gorm.DB {
	var db *gorm.DB //
	var err error
	db, err = gorm.Open(postgres.New(postgres.Config{
		DSN:                  getDSN(cfg),
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		panic("连接数据库失败, error=" + err.Error())
	}

	sqlDB, _ := db.DB()

	//设置数据库连接池参数
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(20)
	return db
}

func initMongoDB(cfg config.YamlConfig) *mongo.Database {
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.Mongo.URI))
	if err != nil {
		panic("连接MongoDB失败, error=" + err.Error())
	}

	mdb := mongoClient.Database(cfg.Mongo.Database)

	return mdb
}

func initRedisDB(cfg config.YamlConfig) *redis.Client {
	// 使用优化后的 Redis 客户端创建方法
	rdb, err := NewRedisClient(cfg)
	if err != nil {
		panic("连接Redis失败, error=" + err.Error())
	}
	return rdb
}

func getDSN(cfg config.YamlConfig) string {
	username := cfg.PostgresSQL.User
	password := cfg.PostgresSQL.Password
	host := cfg.PostgresSQL.Host
	port := cfg.PostgresSQL.Port
	dbname := cfg.PostgresSQL.DBName
	timezone := cfg.PostgresSQL.TimeZone
	sslmode := cfg.PostgresSQL.SSLMode

	// 添加验证
	if host == "" || username == "" || password == "" || dbname == "" || port <= 0 {
		panic("数据库配置不完整或无效")
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s", host, username, password, dbname, port, sslmode, timezone)

	return dsn
}
