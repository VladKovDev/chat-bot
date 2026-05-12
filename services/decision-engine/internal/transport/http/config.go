package http

import "time"

// Config holds the HTTP server configuration
type Config struct {
	// Server configuration
	Address         string        `mapstructure:"address"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	ReadHeadTimeout time.Duration `mapstructure:"read_head_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	MaxHeaderBytes  int           `mapstructure:"max_header_bytes"`

	// Middleware configuration
	Timeout         time.Duration `mapstructure:"timeout"`
	BodyLimit       int64         `mapstructure:"body_limit"`
	EnableLogs      bool          `mapstructure:"enable_logs"`
	EnableRecovery  bool          `mapstructure:"enable_recovery"`
	AdminResetToken string        `mapstructure:"admin_reset_token"`
}
