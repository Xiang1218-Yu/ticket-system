package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config 持有应用全部配置。
// 单一职责：仅描述配置项的结构，不负责加载与校验逻辑（由 Load 完成）。
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
	Upload   UploadConfig   `mapstructure:"upload"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Driver string `mapstructure:"driver"`
	DSN    string `mapstructure:"dsn"`
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

type LogConfig struct {
	Level    string `mapstructure:"level"`
	Dir      string `mapstructure:"dir"`
	Filename string `mapstructure:"filename"`
}

type UploadConfig struct {
	Dir       string `mapstructure:"dir"`
	MaxSizeMB int    `mapstructure:"max_size_mb"`
}

// Load 从 config.yaml（及环境变量覆盖）读取并返回配置。
// 单一职责：仅负责“把配置加载到内存”，不含任何业务或初始化副作用。
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("../")
	viper.AddConfigPath("../config")

	viper.SetEnvPrefix("TS")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}
	cfg.JWT.ExpireHours = normalizeExpireHours(cfg.JWT.ExpireHours)
	return &cfg, nil
}

func normalizeExpireHours(hours int) int {
	if hours < 1 {
		return 1
	}
	return hours
}
