package data

import (
	"MyGoChat/pkg/config"
	"MyGoChat/pkg/log"
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Data struct {
	mdb *mongo.Database
	rdb *redis.Client
	db  *gorm.DB
}

func NewData(cfg config.YamlConfig) (*Data, func(), error) {

	data := &Data{
		mdb: initMongoDB(cfg),
		rdb: initRedisDB(cfg),
		db:  initPostgresDB(cfg),
	}

	log.Logger.Sugar().Infof("init data success")
	cleanup := func() {
		sqlDB, _ := data.db.DB()
		sqlDB.Close()
		data.mdb.Client().Disconnect(context.Background())

		// 这里可以添加关闭MongoDB连接的逻辑
	}
	return data, cleanup, nil
}

func (d *Data) GetDB() *gorm.DB {
	return d.db
}

func (d *Data) GetMongoDB() *mongo.Database {
	return d.mdb
}

func (d *Data) GetRedisClient() *redis.Client {
	return d.rdb
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
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
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
