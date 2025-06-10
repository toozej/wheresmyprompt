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
		name         string
		mockEnv      map[string]string
		mockEnvFile  string
		expectError  bool
		expectConfig Config
	}{
		// TODO fix valid environment variables test
		// {
		// 	name: "Valid environment variables",
		// 	mockEnv: map[string]string{
		// 		"SN_NOTE":     "Test Note",
		// 		"SN_USERNAME": "testuser",
		// 		"SN_PASSWORD": "testpass",
		// 		"FILEPATH":    "/tmp/testfile.txt",
		// 	},
		// 	expectError: false,
		// 	expectConfig: Config{
		// 		SNNote:     "Test Note",
		// 		SNUsername: "testuser",
		// 		SNPassword: "testpass",
		// 		FilePath:   "/tmp/testfile.txt",
		// 	},
		// },
		{
			name:        "Valid .env file",
			mockEnvFile: "sn_note=LLM Prompts\nsn_username=username\nsn_password=password\nfilepath=/tmp/envfile.txt\n",
			expectError: false,
			expectConfig: Config{
				SNNote:     "LLM Prompts",
				SNUsername: "username",
				SNPassword: "password",
				FilePath:   "/tmp/envfile.txt",
			},
		},
		{
			name:        "No environment variables or .env file (defaults)",
			expectError: false,
			expectConfig: Config{
				SNNote:     "LLM Prompts",
				SNUsername: "",
				SNPassword: "",
				FilePath:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()

			// Mock environment variables
			for key, value := range tt.mockEnv {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Mock .env file if applicable
			if tt.mockEnvFile != "" {
				if err := afero.WriteFile(fs, ".env", []byte(tt.mockEnvFile), 0600); err != nil {
					t.Fatalf("Failed to write mock .env file: %v", err)
				}
				viper.SetFs(fs)
				viper.SetConfigFile(".env")
				if err := viper.ReadInConfig(); err != nil {
					t.Fatalf("failed to read mock .env file: %v", err)
				}
			}

			conf := GetEnvVars()

			if conf.SNNote != tt.expectConfig.SNNote {
				t.Errorf("expected SNNote %q, got %q", tt.expectConfig.SNNote, conf.SNNote)
			}
			if conf.SNUsername != tt.expectConfig.SNUsername {
				t.Errorf("expected SNUsername %q, got %q", tt.expectConfig.SNUsername, conf.SNUsername)
			}
			if conf.SNPassword != tt.expectConfig.SNPassword {
				t.Errorf("expected SNPassword %q, got %q", tt.expectConfig.SNPassword, conf.SNPassword)
			}
			if conf.FilePath != tt.expectConfig.FilePath {
				t.Errorf("expected FilePath %q, got %q", tt.expectConfig.FilePath, conf.FilePath)
			}
		})
	}
}
