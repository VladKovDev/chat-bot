package llm

import (
	"github.com/VladKovDev/chat-bot/internal/infrastructure/llm"
	"github.com/spf13/viper"
)

func LoadConfig(v *viper.Viper) (llm.Config, error) {
	bindEnv(v)

	cfg := SetDefaultConfig()

	if err := v.UnmarshalKey("llm", &cfg); err != nil {
		return llm.Config{}, err
	}

	if err := Validate(cfg); err != nil {
		return llm.Config{}, err
	}

	return cfg, nil
}

func bindEnv(v *viper.Viper) {
	_ = v.BindEnv("llm.base_url")
	_ = v.BindEnv("llm.timeout")
	_ = v.BindEnv("llm.cb_max_requests")
	_ = v.BindEnv("llm.cb_interval")
	_ = v.BindEnv("llm.cb_max_failures")
	_ = v.BindEnv("llm.cb_timeout")
}