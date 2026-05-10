package database

import (
	"github.com/VladKovDev/chat-bot/internal/infrastructure/repository/postgres"
	"github.com/spf13/viper"
)

func LoadConfig(v *viper.Viper) (postgres.Config, error) {
	bindEnv(v)

	cfg := SetDefaultConfig()

	if err := v.UnmarshalKey("database", &cfg); err != nil {
		return postgres.Config{}, err
	}
	if v.IsSet("database.host") {
		cfg.Host = v.GetString("database.host")
	}
	if v.IsSet("database.port") {
		cfg.Port = v.GetInt("database.port")
	}
	if v.IsSet("database.user") {
		cfg.User = v.GetString("database.user")
	}
	if v.IsSet("database.password") {
		cfg.Password = v.GetString("database.password")
	}
	if v.IsSet("database.name") {
		cfg.Name = v.GetString("database.name")
	}
	if v.IsSet("database.sslmode") {
		cfg.SSLMode = v.GetString("database.sslmode")
	}
	if v.IsSet("database.max_open_conns") {
		cfg.MaxOpenConns = v.GetInt("database.max_open_conns")
	}
	if v.IsSet("database.max_idle_conns") {
		cfg.MaxIdleConns = v.GetInt("database.max_idle_conns")
	}

	if err := Validate(cfg); err != nil {
		return postgres.Config{}, err
	}

	return cfg, nil
}

func bindEnv(v *viper.Viper) {
	_ = v.BindEnv("database.host", "DATABASE_HOST")
	_ = v.BindEnv("database.port", "DATABASE_PORT")
	_ = v.BindEnv("database.user", "DATABASE_USER")
	_ = v.BindEnv("database.password", "DATABASE_PASSWORD")
	_ = v.BindEnv("database.name", "DATABASE_NAME")
	_ = v.BindEnv("database.sslmode", "DATABASE_SSLMODE")
	_ = v.BindEnv("database.max_open_conns", "DATABASE_MAX_OPEN_CONNS")
	_ = v.BindEnv("database.max_idle_conns", "DATABASE_MAX_IDLE_CONNS")
}
