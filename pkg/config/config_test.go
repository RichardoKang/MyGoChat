package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// 初始化配置（模拟 init 函数）
	viper.SetConfigName("config.dev")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("../../configs")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	assert.NoError(t, err, "读取配置文件失败")

	err = viper.Unmarshal(&config)
	assert.NoError(t, err, "反序列化配置失败")

	// 验证关键字段（根据你的 YamlConfig 结构调整）
	assert.NotEmpty(t, config.AppName, "AppName 为空")
	assert.NotEmpty(t, config.PostgresSQL.Host, "PostgresSQL.Host 为空")
	assert.Greater(t, config.PostgresSQL.Port, 0, "PostgresSQL.Port 无效")
}
