// Package config provides secure configuration management for wheresmyprompt.
//
// This package handles loading configuration from environment variables and .env files
// with built-in security measures to prevent path traversal attacks. It uses the
// github.com/caarlos0/env library for environment variable parsing and
// github.com/joho/godotenv for .env file loading.
//
// The configuration loading follows a priority order:
//  1. Environment variables (highest priority)
//  2. .env file in current working directory
//  3. Default values (if any)
//
// Security features:
//   - Path traversal protection for .env file loading
//   - Secure file path resolution using filepath.Abs and filepath.Rel
//   - Validation against directory traversal attempts
//
// Example usage:
//
//	import "github.com/toozej/wheresmyprompt/pkg/config"
//
//	func main() {
//		conf := config.GetEnvVars()
//		fmt.Printf("SNNote: %s\n", conf.SNNote)
//	}
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config represents the application configuration structure.
//
// This struct defines all configurable parameters for the wheresmyprompt
// application. Fields are tagged with struct tags that correspond to
// environment variable names for automatic parsing.
//
// Configuration supports both Simplenote integration and local file access,
// with flexible authentication options including 1Password integration.
//
// Example:
//
//	type Config struct {
//		SNNote       string `env:"SN_NOTE" envDefault:"LLM Prompts"`
//		SNCredential string `env:"SN_CREDENTIAL"`
//		SNUsername   string `env:"SN_USERNAME"`
//		SNPassword   string `env:"SN_PASSWORD"`
//		FilePath     string `env:"FILEPATH"`
//	}
type Config struct {
	// SNNote specifies the name of the Simplenote note containing prompts.
	// It is loaded from the SN_NOTE environment variable.
	// Defaults to "LLM Prompts" if not set.
	SNNote string `env:"SN_NOTE" envDefault:"LLM Prompts"`

	// SNCredential specifies the 1Password item name for Simplenote credentials.
	// It is loaded from the SN_CREDENTIAL environment variable.
	SNCredential string `env:"SN_CREDENTIAL"`

	// SNUsername specifies the Simplenote username or 1Password field name.
	// It is loaded from the SN_USERNAME environment variable.
	SNUsername string `env:"SN_USERNAME"`

	// SNPassword specifies the Simplenote password or 1Password field name.
	// It is loaded from the SN_PASSWORD environment variable.
	SNPassword string `env:"SN_PASSWORD"`

	// FilePath specifies the local file path for prompts (overrides Simplenote).
	// It is loaded from the FILEPATH environment variable.
	FilePath string `env:"FILEPATH"`
}

// GetEnvVars loads and returns the application configuration from environment
// variables and .env files with comprehensive security validation.
//
// This function performs the following operations:
//  1. Securely determines the current working directory
//  2. Constructs and validates the .env file path to prevent traversal attacks
//  3. Loads .env file if it exists in the current directory
//  4. Parses environment variables into the Config struct
//  5. Returns the populated configuration
//
// Security measures implemented:
//   - Path traversal detection and prevention using filepath.Rel
//   - Absolute path resolution for secure path operations
//   - Validation against ".." sequences in relative paths
//   - Safe file existence checking before loading
//
// The function will terminate the program with os.Exit(1) if any critical
// errors occur during configuration loading, such as:
//   - Current directory access failures
//   - Path traversal attempts detected
//   - .env file parsing errors
//   - Environment variable parsing failures
//
// Returns:
//   - Config: A populated configuration struct with values from environment
//     variables and/or .env file
//
// Example:
//
//	// Load configuration
//	conf := config.GetEnvVars()
//
//	// Use configuration
//	if conf.SNNote != "" {
//		fmt.Printf("Using note: %s\n", conf.SNNote)
//	}
func GetEnvVars() Config {
	// Get current working directory for secure file operations
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current working directory: %s\n", err)
		os.Exit(1)
	}

	// Construct secure path for .env file within current directory
	envPath := filepath.Join(cwd, ".env")

	// Ensure the path is within our expected directory (prevent traversal)
	cleanEnvPath, err := filepath.Abs(envPath)
	if err != nil {
		fmt.Printf("Error resolving .env file path: %s\n", err)
		os.Exit(1)
	}
	cleanCwd, err := filepath.Abs(cwd)
	if err != nil {
		fmt.Printf("Error resolving current directory: %s\n", err)
		os.Exit(1)
	}
	relPath, err := filepath.Rel(cleanCwd, cleanEnvPath)
	if err != nil || strings.Contains(relPath, "..") {
		fmt.Printf("Error: .env file path traversal detected\n")
		os.Exit(1)
	}

	// Load .env file if it exists
	if _, err := os.Stat(envPath); err == nil {
		if err := godotenv.Load(envPath); err != nil {
			fmt.Printf("Error loading .env file: %s\n", err)
			os.Exit(1)
		}
	}

	// Parse environment variables into config struct
	var conf Config
	if err := env.Parse(&conf); err != nil {
		fmt.Printf("Error parsing environment variables: %s\n", err)
		os.Exit(1)
	}

	return conf
}
