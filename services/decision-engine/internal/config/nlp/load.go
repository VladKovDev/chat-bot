package nlp

import (
	infranlp "github.com/VladKovDev/chat-bot/internal/infrastructure/nlp"
	"github.com/spf13/viper"
)

func LoadConfig(v *viper.Viper) (infranlp.EmbedderConfig, error) {
	bindEnv(v)

	cfg := SetDefaultConfig()
	if err := v.UnmarshalKey("nlp", &cfg); err != nil {
		return infranlp.EmbedderConfig{}, err
	}
	if v.IsSet("nlp.base_url") {
		cfg.BaseURL = v.GetString("nlp.base_url")
	}
	if v.IsSet("nlp.timeout") {
		cfg.Timeout = v.GetDuration("nlp.timeout")
	}
	if v.IsSet("nlp.expected_dimension") {
		cfg.ExpectedDimension = v.GetInt("nlp.expected_dimension")
	}
	if err := Validate(cfg); err != nil {
		return infranlp.EmbedderConfig{}, err
	}
	return cfg, nil
}

func bindEnv(v *viper.Viper) {
	_ = v.BindEnv("nlp.base_url", "NLP_BASE_URL")
	_ = v.BindEnv("nlp.timeout", "NLP_TIMEOUT")
	_ = v.BindEnv("nlp.expected_dimension", "NLP_EXPECTED_DIMENSION")
}
