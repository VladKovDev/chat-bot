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
	if v.IsSet("http.address") {
		cfg.Address = v.GetString("http.address")
	}
	if v.IsSet("http.read_timeout") {
		cfg.ReadTimeout = v.GetDuration("http.read_timeout")
	}
	if v.IsSet("http.read_head_timeout") {
		cfg.ReadHeadTimeout = v.GetDuration("http.read_head_timeout")
	}
	if v.IsSet("http.write_timeout") {
		cfg.WriteTimeout = v.GetDuration("http.write_timeout")
	}
	if v.IsSet("http.idle_timeout") {
		cfg.IdleTimeout = v.GetDuration("http.idle_timeout")
	}
	if v.IsSet("http.max_header_bytes") {
		cfg.MaxHeaderBytes = v.GetInt("http.max_header_bytes")
	}
	if v.IsSet("http.timeout") {
		cfg.Timeout = v.GetDuration("http.timeout")
	}
	if v.IsSet("http.body_limit") {
		cfg.BodyLimit = v.GetInt64("http.body_limit")
	}
	if v.IsSet("http.enable_logs") {
		cfg.EnableLogs = v.GetBool("http.enable_logs")
	}
	if v.IsSet("http.enable_recovery") {
		cfg.EnableRecovery = v.GetBool("http.enable_recovery")
	}
	if v.IsSet("http.admin_reset_token") {
		cfg.AdminResetToken = v.GetString("http.admin_reset_token")
	}

	if err := Validate(cfg); err != nil {
		return http.Config{}, err
	}

	return cfg, nil
}

func bindEnv(v *viper.Viper) {
	_ = v.BindEnv("http.address", "HTTP_ADDRESS")
	_ = v.BindEnv("http.read_timeout", "HTTP_READ_TIMEOUT")
	_ = v.BindEnv("http.read_head_timeout", "HTTP_READ_HEAD_TIMEOUT")
	_ = v.BindEnv("http.write_timeout", "HTTP_WRITE_TIMEOUT")
	_ = v.BindEnv("http.idle_timeout", "HTTP_IDLE_TIMEOUT")
	_ = v.BindEnv("http.max_header_bytes", "HTTP_MAX_HEADER_BYTES")
	_ = v.BindEnv("http.timeout", "HTTP_TIMEOUT")
	_ = v.BindEnv("http.body_limit", "HTTP_BODY_LIMIT")
	_ = v.BindEnv("http.enable_logs", "HTTP_ENABLE_LOGS")
	_ = v.BindEnv("http.enable_recovery", "HTTP_ENABLE_RECOVERY")
	_ = v.BindEnv("http.admin_reset_token", "ADMIN_RESET_TOKEN")
}
