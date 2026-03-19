package logger

import (
	"fmt"

	"github.com/VladKovDev/chat-bot/pkg/logger"
)

func Validate(cfg logger.Config) error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[cfg.Level] {
		return fmt.Errorf("logger level must be (debug, info, warn, error), got: %v", cfg.Level)
	}

	validFormats := map[string]bool{
		"console": true,
		"json":    true,
	}
	if !validFormats[cfg.Format] {
		return fmt.Errorf("logger format must be (string, json), got: %v", cfg.Format)
	}

	validOutputs := map[string]bool{
		"stdout": true,
		"stderr": true,
		"file":   true,
	}
	if !validOutputs[cfg.Output] {
		return fmt.Errorf("logger output must be (stdout, stderr), got: %v", cfg.Output)
	}

	if cfg.Output == "file" && cfg.FilePath == "" {
		return fmt.Errorf("logger file_path required when output is 'file'")
	}

	if cfg.MaxBackups < 0 {
		return fmt.Errorf("logger max backups must be non-negative, got: %v", cfg.MaxBackups)
	}

	if cfg.MaxAge < 0 {
		return fmt.Errorf("logger max age must be non-negative, got: %v", cfg.MaxAge)
	}

	return nil
}
