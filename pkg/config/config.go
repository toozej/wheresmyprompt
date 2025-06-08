package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Username string `mapstructure:"username"`
}

func GetEnvVars() Config {
	if _, err := os.Stat(".env"); err == nil {
		// Initialize Viper from .env file
		viper.SetConfigFile(".env") // Specify the name of your .env file

		// Read the .env file
		if err := viper.ReadInConfig(); err != nil {
			fmt.Printf("Error reading .env file: %s\n", err)
			os.Exit(1)
		}
	}

	// Enable reading environment variables
	viper.AutomaticEnv()

	// Setup conf struct with items from environment variables
	var conf Config
	if err := viper.Unmarshal(&conf); err != nil {
		fmt.Printf("Error unmarshalling Viper conf: %s\n", err)
		os.Exit(1)
	}

	return conf
}
