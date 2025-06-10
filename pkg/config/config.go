// Package config provides configuration management for wheresmyprompt.
// It handles loading configuration from environment variables and .env files,
// supporting both Simplenote and local file sources for prompts.
package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config holds the application configuration settings.
// It supports both Simplenote integration and local file access,
// with flexible authentication options including 1Password integration.
type Config struct {
	SNNote       string `mapstructure:"sn_note"`       // The name of the Simplenote note containing prompts
	SNCredential string `mapstructure:"sn_credential"` // 1Password item name for Simplenote credentials
	SNUsername   string `mapstructure:"sn_username"`   // Simplenote username or 1Password field name
	SNPassword   string `mapstructure:"sn_password"`   // Simplenote password or 1Password field name
	FilePath     string `mapstructure:"filepath"`      // Local file path for prompts (overrides Simplenote)
}

// GetEnvVars loads configuration from environment variables and .env files.
// It first attempts to load from a .env file if present, then reads from environment variables.
// Default values are set for common configuration options.
// Returns a populated Config struct or exits on configuration errors.

func GetEnvVars() Config {
	if _, err := os.Stat(".env"); err == nil {
		// Initialize Viper from .env file
		viper.SetConfigFile(".env")

		// Read the .env file
		if err := viper.ReadInConfig(); err != nil {
			fmt.Printf("Error reading .env file: %s\n", err)
			os.Exit(1)
		}
	}

	// Enable reading environment variables
	viper.AutomaticEnv()

	// Set default values
	viper.SetDefault("sn_note", "LLM Prompts")

	// Setup conf struct with items from environment variables
	var conf Config
	if err := viper.Unmarshal(&conf); err != nil {
		fmt.Printf("Error unmarshalling Viper conf: %s\n", err)
		os.Exit(1)
	}

	return conf
}
