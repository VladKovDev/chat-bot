package logger

import (
	"github.com/VladKovDev/chat-bot/pkg/logger"
	"github.com/spf13/viper"
)

func LoadConfig(v *viper.Viper) (logger.Config, error) {
	bindEnv(v)

	cfg := SetDefaultConfig()

	if err := v.UnmarshalKey("logger", &cfg); err != nil {
		return logger.Config{}, err
	}

	if err := Validate(cfg); err != nil {
		return logger.Config{}, err
	}

	return cfg, nil
}

func bindEnv(v *viper.Viper) {
	_ = v.BindEnv("logger.level")
	_ = v.BindEnv("logger.format")
	_ = v.BindEnv("logger.output")
	_ = v.BindEnv("logger.enable_colors")
	_ = v.BindEnv("logger.file_path")
	_ = v.BindEnv("logger.max_size")
	_ = v.BindEnv("logger.max_backups")
	_ = v.BindEnv("logger.max_age")
	_ = v.BindEnv("logger.compress")
	_ = v.BindEnv("logger.conn_max_lifetime")
	_ = v.BindEnv("logger.conn_max_idle_time")
	_ = v.BindEnv("logger.health_check_period")
}
