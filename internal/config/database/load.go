package database

import (
	"github.com/VladKovDev/chat-bot/internal/infrastructure/repository"
	"github.com/spf13/viper"
)

func LoadConfig(v *viper.Viper) (repository.Config, error) {
	bindEnv(v)

	cfg := SetDefaultConfig()

	if err := v.UnmarshalKey("database", &cfg); err != nil {
		return repository.Config{}, err
	}

	if err := Validate(cfg); err != nil {
		return repository.Config{}, err
	}

	return cfg, nil
}

func bindEnv(v *viper.Viper) {
	_ = v.BindEnv("database.host")
	_ = v.BindEnv("database.port")
	_ = v.BindEnv("database.user")
	_ = v.BindEnv("database.password")
	_ = v.BindEnv("database.name")
	_ = v.BindEnv("database.sslmode")
	_ = v.BindEnv("database.max_open_conns")
	_ = v.BindEnv("database.max_idle_conns")
}
