package lemmatizer

import (
	"github.com/VladKovDev/chat-bot/internal/infrastructure/lemmatizer"
	"github.com/spf13/viper"
)

func LoadConfig(v *viper.Viper) (lemmatizer.Config, error) {
	bindEnv(v)

	cfg := SetDefaultConfig()

	if err := v.UnmarshalKey("lemmatizer", &cfg); err != nil {
		return lemmatizer.Config{}, err
	}

	if err := Validate(cfg); err != nil {
		return lemmatizer.Config{}, err
	}

	return cfg, nil
}

func bindEnv(v *viper.Viper) {
	_ = v.BindEnv("lemmatizer.base_url")
	_ = v.BindEnv("lemmatizer.timeout")
	_ = v.BindEnv("lemmatizer.cb_max_requests")
	_ = v.BindEnv("lemmatizer.cb_interval")
	_ = v.BindEnv("lemmatizer.cb_max_failures")
	_ = v.BindEnv("lemmatizer.cb_timeout")
}