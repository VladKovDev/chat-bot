package main

import (
	"fmt"
	"os"

	"github.com/VladKovDev/web-adapter/internal/app"
	"github.com/VladKovDev/web-adapter/internal/config"
	"github.com/VladKovDev/web-adapter/pkg/logger"
)

func main() {
	// Get config path from environment
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./configs"
	}

	// Get app environment from environment
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "local"
	}

	// Initialize viper configuration
	v, err := config.Init(configPath, appEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize config: %v\n", err)
		os.Exit(1)
	}

	// Create application
	application, err := app.New(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create application: %v\n", err)
		os.Exit(1)
	}

	// Run application
	if err := application.Run(); err != nil {
		application.Logger.Error("application error", logger.Err(err))
		if err := application.Shutdown(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to shutdown application: %v\n", err)
		}
		os.Exit(1)
	}

	// Shutdown application
	if err := application.Shutdown(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to shutdown application: %v\n", err)
		os.Exit(1)
	}
}
