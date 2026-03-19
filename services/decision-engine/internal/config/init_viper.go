package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

func Init(configPath string, appEnv string) (*viper.Viper, error) {
	v := viper.New()

	if configPath == "" {
		return nil, fmt.Errorf("config path is required")
	}

	fileName := fmt.Sprintf("config.%s", appEnv)

	v.SetConfigName(fileName)
	v.SetConfigType("yaml")
	v.AddConfigPath(configPath)
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}
	// env config
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(configPath)
	v.AddConfigPath(".")
	if err := v.MergeInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("failed to read env: %w", err)
		}
	}

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	return v, nil
}
