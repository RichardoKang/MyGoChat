package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type (
	YamlConfig struct {
		AppName     string            `yaml:"AppName"`
		SeverID     string            `yaml:"ServerID"`
		PostgresSQL PostgresSQLConfig `yaml:"PostgresSQL"`
		Log         LogConfig         `yaml:"Log"`
		JwtSecret   JwtSecretConfig   `yaml:"JwtSecret"`
		Kafka       KafkaConfig       `yaml:"Kafka"`
		Mongo       MongoConfig       `yaml:"MongoDB"`
		Redis       RedisConfig       `yaml:"Redis"`
	}

	RedisConfig struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	}

	PostgresSQLConfig struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		DBName   string `yaml:"dbname"`
		SSLMode  string `yaml:"sslmode"`
		TimeZone string `yaml:"timezone"`
	}
	LogConfig struct {
		Path  string `yaml:"path"`
		Level string `yaml:"level"`
	}
	JwtSecretConfig struct {
		SecretKey string `yaml:"SecretKey"`
	}

	// KafkaTopicsConfig 用于存放所有 Kafka 主题的名称
	KafkaTopicsConfig struct {
		Ingest       string `yaml:"ingest"`
		Sync_request string `yaml:"sync_request"`
		Delivery     string `yaml:"delivery"`
	}

	KafkaConfig struct {
		Brokers []string          `yaml:"brokers"`
		Topics  KafkaTopicsConfig `yaml:"topics"`
	}

	MongoConfig struct {
		URI      string `yaml:"uri"`
		Database string `yaml:"database"`
	}
)

var config YamlConfig

func init() {
	root, err := getProjectRoot()
	if err != nil {
		panic(fmt.Errorf("Failed to find project root: %w", err))
	}

	configPath := filepath.Join(root, "configs", "config.dev.yaml")

	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Fatal error config file: %w", err))
	}

	if err := viper.Unmarshal(&config); err != nil {
		panic(fmt.Errorf("Unable to decode into struct: %w", err))
	}
}

func GetConfig() YamlConfig {
	return config
}

// getProjectRoot 向上查找 go.mod 文件来确定项目根目录
func getProjectRoot() (string, error) {
	// 从当前工作目录开始
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// 循环向上查找
	for {
		// 检查 go.mod 是否在此目录
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			// 找到了 go.mod，此目录就是项目根目录
			return dir, nil
		}

		// 向上移动一个目录
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			// 到达了文件系统根目录，仍未找到
			break
		}
		dir = parentDir
	}

	return "", fmt.Errorf("go.mod not found")
}
