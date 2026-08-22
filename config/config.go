package config

import (
	"fmt"
	"strings"

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

	// 配置键用点分隔嵌套（如 jwt.expire_hours），而环境变量规范形如
	// TS_JWT_EXPIRE_HOURS。必须把点替换为下划线，AutomaticEnv 才能在
	// os.LookupEnv 时命中真实变量；否则 viper 会去查找带点的非法变量名，
	// 覆盖永远无法生效，导致“配置变了但会话寿命未变”的不一致。
	viper.SetEnvPrefix("TS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
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

// normalizeExpireHours 把过期小时数钳制到合法下界，保证签发的令牌至少有
// 1 小时寿命。配置缺失/为 0/负值都收敛为 1，而非让上层减一后变成 0。
func normalizeExpireHours(hours int) int {
	if hours < 1 {
		return 1
	}
	return hours
}
