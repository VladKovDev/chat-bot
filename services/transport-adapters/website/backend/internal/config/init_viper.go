package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Init initializes viper configuration
func Init(configPath string, appEnv string) (*viper.Viper, error) {
	v := viper.New()

	if configPath == "" {
		configPath = "./configs"
	}

	if appEnv == "" {
		appEnv = "local"
	}

	fileName := fmt.Sprintf("config.%s", appEnv)

	v.SetConfigName(fileName)
	v.SetConfigType("yaml")
	v.AddConfigPath(configPath)
	v.AutomaticEnv()

	// Try to read config file (optional)
	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		// Config file not found, use defaults and env vars
	}

	// Try to read .env file (optional)
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(configPath)
	v.AddConfigPath(".")
	if err := v.MergeInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("failed to read env: %w", err)
		}
		// .env file not found, use defaults
	}

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	return v, nil
}