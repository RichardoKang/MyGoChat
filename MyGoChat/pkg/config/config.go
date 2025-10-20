package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type (
	YamlConfig struct {
		AppName     string
		PostgresSQL PostgresSQLConfig
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
)

var config YamlConfig

func init() {
	viper.SetConfigName("config.dev")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("../../configs")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}

	// Unmarshal 用来将读取到的配置信息反序列化到结构体中
	if err := viper.Unmarshal(&config); err != nil {
		panic(fmt.Errorf("Unable to decode into struct: %w \n", err))
	}
}

func GetConfig() YamlConfig {
	return config
}
