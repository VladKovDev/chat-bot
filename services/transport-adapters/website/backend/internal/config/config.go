package config

import "time"

// Config holds the application configuration
type Config struct {
	DecisionEngine DecisionEngine `mapstructure:"decision_engine"`
	Server         Server         `mapstructure:"server"`
	Log            Log            `mapstructure:"log"`
}

// DecisionEngine holds the decision engine client configuration
type DecisionEngine struct {
	URL     string        `mapstructure:"url"`
	Timeout time.Duration `mapstructure:"timeout"`
}

// Server holds the WebSocket server configuration
type Server struct {
	Address         string   `mapstructure:"address"`
	ReadBufferSize  int      `mapstructure:"read_buffer_size"`
	WriteBufferSize int      `mapstructure:"write_buffer_size"`
	AllowedOrigins  []string `mapstructure:"allowed_origins"`
}

// Log holds the logging configuration
type Log struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"` // json or text
}
