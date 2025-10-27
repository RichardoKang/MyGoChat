package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type (
	YamlConfig struct {
		AppName     string
		PostgresSQL PostgresSQLConfig
		Log         LogConfig
		JwtSecret   JwtSecretConfig
	}

	PostgresSQLConfig struct {
		Host     string
		Port     int
		User     string
		Password string
		DBName   string
		SSLMode  string
		TimeZone string
	}
	LogConfig struct {
		Path  string
		Level string
	}
	JwtSecretConfig struct {
		SecretKey string
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
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}

	if err := viper.Unmarshal(&config); err != nil {
		panic(fmt.Errorf("Unable to decode into struct: %w \n", err))
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
