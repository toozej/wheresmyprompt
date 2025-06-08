package config

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

func TestGetEnvVars(t *testing.T) {
	fs := afero.NewMemMapFs()

	tests := []struct {
		name           string
		mockEnv        map[string]string
		mockEnvFile    string
		expectError    bool
		expectUsername string
	}{
		{
			name: "Valid environment variable",
			mockEnv: map[string]string{
				"USERNAME": "testuser",
			},
			expectError:    false,
			expectUsername: "",
		},
		{
			name:           "Valid .env file",
			mockEnvFile:    "username=testenvfileuser\n",
			expectError:    false,
			expectUsername: "testenvfileuser",
		},
		{
			name:           "No environment variables or .env file",
			expectError:    false,
			expectUsername: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset Viper settings before each test
			viper.Reset()

			// Mock environment variables
			for key, value := range tt.mockEnv {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Mock .env file if applicable
			if tt.mockEnvFile != "" {
				if err := afero.WriteFile(fs, ".env", []byte(tt.mockEnvFile), 0644); err != nil {
					t.Fatalf("Failed to write mock .env file: %v", err)
				}
				viper.SetFs(fs) // Ensure Viper uses the mocked filesystem
				viper.SetConfigFile(".env")
				if err := viper.ReadInConfig(); err != nil {
					t.Fatalf("failed to read mock .env file: %v", err)
				}
			}

			// Call function
			conf := GetEnvVars()

			// Verify output
			if conf.Username != tt.expectUsername {
				t.Errorf("expected username %q, got %q", tt.expectUsername, conf.Username)
			}
		})
	}
}
