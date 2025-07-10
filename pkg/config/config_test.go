package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetEnvVars(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	tests := []struct {
		name         string
		mockEnv      map[string]string
		mockEnvFile  string
		expectError  bool
		expectConfig Config
	}{
		{
			name: "Valid environment variables",
			mockEnv: map[string]string{
				"SN_NOTE":     "Test Note",
				"SN_USERNAME": "testuser",
				"SN_PASSWORD": "testpass",
				"FILEPATH":    "/tmp/testfile.txt",
			},
			expectError: false,
			expectConfig: Config{
				SNNote:     "Test Note",
				SNUsername: "testuser",
				SNPassword: "testpass",
				FilePath:   "/tmp/testfile.txt",
			},
		},
		{
			name:        "Valid .env file",
			mockEnvFile: "SN_NOTE=LLM Prompts\nSN_USERNAME=username\nSN_PASSWORD=password\nFILEPATH=/tmp/envfile.txt\n",
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
			// Change to temp directory for this test
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp dir: %v", err)
			}

			// Clean up environment variables from previous tests
			envVars := []string{"SN_NOTE", "SN_USERNAME", "SN_PASSWORD", "FILEPATH"}
			for _, envVar := range envVars {
				os.Unsetenv(envVar)
			}

			// Mock environment variables
			for key, value := range tt.mockEnv {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Mock .env file if applicable
			if tt.mockEnvFile != "" {
				envPath := filepath.Join(tempDir, ".env")
				if err := os.WriteFile(envPath, []byte(tt.mockEnvFile), 0600); err != nil {
					t.Fatalf("Failed to write mock .env file: %v", err)
				}
				defer os.Remove(envPath)
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
