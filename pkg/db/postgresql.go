package db

//
//import (
//	"MyGoChat/pkg/config"
//	"fmt"
//
//	"gorm.io/driver/postgres"
//	"gorm.io/gorm"
//	"gorm.io/gorm/logger"
//)
//
//var _db *gorm.DB // 下划线表示包内私有变量，外部无法访问
//
//func getDSN() string {
//	username := config.GetConfig().PostgresSQL.User
//	password := config.GetConfig().PostgresSQL.Password
//	host := config.GetConfig().PostgresSQL.Host
//	port := config.GetConfig().PostgresSQL.Port
//	dbname := config.GetConfig().PostgresSQL.DBName
//	timezone := config.GetConfig().PostgresSQL.TimeZone
//	sslmode := config.GetConfig().PostgresSQL.SSLMode
//
//	// 添加验证
//	if host == "" || username == "" || password == "" || dbname == "" || port <= 0 {
//		panic("数据库配置不完整或无效")
//	}
//
//	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s", host, username, password, dbname, port, sslmode, timezone)
//
//	return dsn
//}
//
//func init() {
//
//	var err error
//	_db, err = gorm.Open(postgres.New(postgres.Config{
//		DSN:                  getDSN(),
//		PreferSimpleProtocol: true,
//	}), &gorm.Config{
//		Logger: logger.Default.LogMode(logger.Info),
//	})
//
//	if err != nil {
//		panic("连接数据库失败, error=" + err.Error())
//	}
//
//	sqlDB, _ := _db.DB()
//
//	//设置数据库连接池参数
//	sqlDB.SetMaxOpenConns(100)
//	sqlDB.SetMaxIdleConns(20)
//}
//
//func GetDB() *gorm.DB {
//	return _db
//}
