package transport

import (
	"github.com/VladKovDev/chat-bot/internal/transport/http"
	"github.com/spf13/viper"
)

func LoadConfig(v *viper.Viper) (http.Config, error) {
	bindEnv(v)

	cfg := SetDefaultConfig()

	if err := v.UnmarshalKey("http", &cfg); err != nil {
		return http.Config{}, err
	}

	if err := Validate(cfg); err != nil {
		return http.Config{}, err
	}

	return cfg, nil
}

func bindEnv(v *viper.Viper) {
	_ = v.BindEnv("http.address")
	_ = v.BindEnv("http.read_timeout")
	_ = v.BindEnv("http.read_head_timeout")
	_ = v.BindEnv("http.write_timeout")
	_ = v.BindEnv("http.idle_timeout")
	_ = v.BindEnv("http.max_header_bytes")
	_ = v.BindEnv("http.timeout")
	_ = v.BindEnv("http.body_limit")
	_ = v.BindEnv("http.enable_logs")
	_ = v.BindEnv("http.enable_recovery")
}