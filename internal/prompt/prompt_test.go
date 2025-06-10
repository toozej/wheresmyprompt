package prompt

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/toozej/wheresmyprompt/pkg/config"
)

// Test helpers and mock data
const testMarkdownContent = `# Test Prompts

## Code Review
### Code Review Checklist
Please review this code for:
- Security vulnerabilities
- Performance issues
- Best practices

### Bug Analysis
Analyze this bug report and provide:
1. Root cause analysis
2. Proposed fix
3. Prevention strategies

## Writing
### Email Template
Write a professional email template for:
- Clear subject line
- Polite greeting
- Concise body
- Professional closing

### Documentation
Create documentation that includes:
- Overview
- Installation steps
- Usage examples
`

func TestCheckRequiredBinaries(t *testing.T) {
	tests := []struct {
		name        string
		config      config.Config
		expectError bool
		skipReason  string
	}{
		{
			name: "filepath set - should not check sncli",
			config: config.Config{
				FilePath: "/some/path",
			},
			expectError: false,
		},
		{
			name: "filepath empty - should check sncli",
			config: config.Config{
				FilePath: "",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			err := CheckRequiredBinaries(tt.config)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create in-memory filesystem
	fs := afero.NewMemMapFs()

	tests := []struct {
		name        string
		filepath    string
		fileContent string
		setupFile   bool
		expectError bool
	}{
		{
			name:        "successful file read",
			filepath:    "/test/prompts.md",
			fileContent: "# Test Content\n\nHello World",
			setupFile:   true,
			expectError: false,
		},
		{
			name:        "file not found",
			filepath:    "/nonexistent/file.md",
			setupFile:   false,
			expectError: true,
		},
		{
			name:        "empty file",
			filepath:    "/test/empty.md",
			fileContent: "",
			setupFile:   true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup file if needed
			if tt.setupFile {
				err := afero.WriteFile(fs, tt.filepath, []byte(tt.fileContent), 0644)
				if err != nil {
					t.Fatalf("Failed to setup test file: %v", err)
				}
			}

			// Mock os.ReadFile by temporarily replacing the filesystem
			// Since we can't easily mock os.ReadFile, we'll test the logic differently
			// For this test, we'll create actual temp files
			if tt.setupFile {
				tmpFile, err := os.CreateTemp("", "test_*.md")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				defer os.Remove(tmpFile.Name())
				defer tmpFile.Close()

				_, err = tmpFile.WriteString(tt.fileContent)
				if err != nil {
					t.Fatalf("Failed to write to temp file: %v", err)
				}

				content, err := loadFromFile(tmpFile.Name())
				if tt.expectError && err == nil {
					t.Error("Expected error but got none")
				}
				if !tt.expectError && err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if !tt.expectError && content != tt.fileContent {
					t.Errorf("Expected content %q, got %q", tt.fileContent, content)
				}
			} else {
				_, err := loadFromFile(tt.filepath)
				if !tt.expectError {
					t.Error("Expected no error but got one")
				}
				if tt.expectError && err == nil {
					t.Error("Expected error but got none")
				}
			}
		})
	}
}

func TestParseMarkdown(t *testing.T) {
	tests := []struct {
		name             string
		content          string
		expectedPrompts  int
		expectedSections int
		expectSpecific   map[string]string // section -> prompt title
	}{
		{
			name:             "valid markdown with sections and prompts",
			content:          testMarkdownContent,
			expectedPrompts:  4,
			expectedSections: 2,
			expectSpecific: map[string]string{
				"Code Review": "Code Review Checklist",
				"Writing":     "Email Template",
			},
		},
		{
			name:             "empty content",
			content:          "",
			expectedPrompts:  0,
			expectedSections: 0,
		},
		{
			name: "content without sections",
			content: `### Standalone Prompt
This is a prompt without a section.`,
			expectedPrompts:  1,
			expectedSections: 0,
		},
		{
			name: "section without prompts",
			content: `## Empty Section
No prompts here.`,
			expectedPrompts:  0,
			expectedSections: 0,
		},
		{
			name: "multiple prompts in one section",
			content: `## Test Section
### First Prompt
Content 1

### Second Prompt  
Content 2`,
			expectedPrompts:  2,
			expectedSections: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Disable debug output for tests
			viper.Set("debug", false)

			data := parseMarkdown(tt.content)

			if len(data.Prompts) != tt.expectedPrompts {
				t.Errorf("Expected %d prompts, got %d", tt.expectedPrompts, len(data.Prompts))
			}

			if len(data.Sections) != tt.expectedSections {
				t.Errorf("Expected %d sections, got %d", tt.expectedSections, len(data.Sections))
			}

			// Check specific prompt/section combinations
			for section, expectedTitle := range tt.expectSpecific {
				sectionPrompts, exists := data.Sections[section]
				if !exists {
					t.Errorf("Expected section %q to exist", section)
					continue
				}

				found := false
				for _, prompt := range sectionPrompts {
					if prompt.Title == expectedTitle {
						found = true
						if prompt.Section != section {
							t.Errorf("Expected prompt %q to belong to section %q, got %q",
								expectedTitle, section, prompt.Section)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected to find prompt %q in section %q", expectedTitle, section)
				}
			}
		})
	}
}

func TestSearchPrompts(t *testing.T) {
	data := parseMarkdown(testMarkdownContent)

	tests := []struct {
		name             string
		query            string
		section          string
		expectedCount    int
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:          "empty query returns all prompts",
			query:         "",
			section:       "",
			expectedCount: 4,
		},
		{
			name:          "search by title",
			query:         "checklist",
			section:       "",
			expectedCount: 1,
			shouldContain: []string{"Security vulnerabilities"},
		},
		{
			name:          "search by content",
			query:         "email",
			section:       "",
			expectedCount: 1,
			shouldContain: []string{"professional email"},
		},
		{
			name:          "search within specific section",
			query:         "",
			section:       "Code Review",
			expectedCount: 2,
		},
		{
			name:          "search within specific section with query",
			query:         "bug",
			section:       "Code Review",
			expectedCount: 1,
			shouldContain: []string{"Root cause analysis"},
		},
		{
			name:          "search in non-existent section",
			query:         "test",
			section:       "NonExistent",
			expectedCount: 0,
		},
		{
			name:          "no matches found",
			query:         "xyznomatch",
			section:       "",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := SearchPrompts(data, tt.query, tt.section)

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
			}

			// Check that results contain expected strings
			for _, expected := range tt.shouldContain {
				found := false
				for _, result := range results {
					if strings.Contains(result, expected) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected results to contain %q", expected)
				}
			}

			// Check that results don't contain unexpected strings
			for _, unexpected := range tt.shouldNotContain {
				for _, result := range results {
					if strings.Contains(result, unexpected) {
						t.Errorf("Expected results not to contain %q", unexpected)
					}
				}
			}
		})
	}
}

func TestFindBestMatch(t *testing.T) {
	data := parseMarkdown(testMarkdownContent)

	tests := []struct {
		name          string
		query         string
		section       string
		expectEmpty   bool
		shouldContain string
	}{
		{
			name:          "find best match",
			query:         "checklist",
			section:       "",
			expectEmpty:   false,
			shouldContain: "Security vulnerabilities",
		},
		{
			name:        "no match found",
			query:       "nomatchforthis",
			section:     "",
			expectEmpty: true,
		},
		{
			name:          "best match in section",
			query:         "email",
			section:       "Writing",
			expectEmpty:   false,
			shouldContain: "professional email",
		},
		{
			name:        "no match in wrong section",
			query:       "email",
			section:     "Code Review",
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindBestMatch(data, tt.query, tt.section)

			if tt.expectEmpty && result != "" {
				t.Errorf("Expected empty result, got %q", result)
			}

			if !tt.expectEmpty && result == "" {
				t.Error("Expected non-empty result, got empty string")
			}

			if tt.shouldContain != "" && !strings.Contains(result, tt.shouldContain) {
				t.Errorf("Expected result to contain %q, got %q", tt.shouldContain, result)
			}
		})
	}
}

func TestGetSectionPrompts(t *testing.T) {
	data := parseMarkdown(testMarkdownContent)

	tests := []struct {
		name          string
		section       string
		expectedCount int
		shouldContain []string
	}{
		{
			name:          "existing section",
			section:       "Code Review",
			expectedCount: 2,
			shouldContain: []string{"Security vulnerabilities", "Root cause analysis"},
		},
		{
			name:          "another existing section",
			section:       "Writing",
			expectedCount: 2,
			shouldContain: []string{"professional email", "documentation"},
		},
		{
			name:          "non-existent section",
			section:       "NonExistent",
			expectedCount: 0,
		},
		{
			name:          "empty section name",
			section:       "",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := GetSectionPrompts(data, tt.section)

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
			}

			for _, expected := range tt.shouldContain {
				found := false
				for _, result := range results {
					if strings.Contains(result, expected) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected results to contain %q", expected)
				}
			}
		})
	}
}

func TestCopyToClipboard(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		expectError bool
		skipReason  string
	}{
		{
			name:        "copy simple text",
			text:        "Hello, World!",
			expectError: false,
		},
		{
			name:        "copy empty text",
			text:        "",
			expectError: false,
		},
		{
			name:        "copy multiline text",
			text:        "Line 1\nLine 2\nLine 3",
			expectError: false,
		},
		{
			name:        "copy text with special characters",
			text:        "Special chars: !@#$%^&*()",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			err := CopyToClipboard(tt.text)

			// The actual clipboard operation might fail in CI/CD environments
			// where clipboard utilities aren't available, so we'll check for
			// the specific error types we expect
			if runtime.GOOS == "linux" {
				// On Linux, if neither xclip nor xsel is available, we expect a specific error
				if err != nil && strings.Contains(err.Error(), "no clipboard utility found") {
					t.Skip("Clipboard utilities not available in test environment")
				}
			}

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				// Only fail if it's not a missing utility error
				if !strings.Contains(err.Error(), "not found") &&
					!strings.Contains(err.Error(), "no clipboard utility") {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestLoadPrompts(t *testing.T) {
	tests := []struct {
		name        string
		config      config.Config
		setupFile   bool
		fileContent string
		expectError bool
	}{
		{
			name: "load from file - success",
			config: config.Config{
				FilePath: "", // Will be set in test
			},
			setupFile:   true,
			fileContent: testMarkdownContent,
			expectError: false,
		},
		{
			name: "load from file - file not found",
			config: config.Config{
				FilePath: "/nonexistent/file.md",
			},
			setupFile:   false,
			expectError: true,
		},
		{
			name: "load from simplenote - no filepath",
			config: config.Config{
				FilePath: "",
				SNNote:   "test-note",
			},
			setupFile:   false,
			expectError: true, // Will fail because sncli is not available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tempFile *os.File
			var err error

			if tt.setupFile {
				tempFile, err = os.CreateTemp("", "test_*.md")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				defer os.Remove(tempFile.Name())
				defer tempFile.Close()

				_, err = tempFile.WriteString(tt.fileContent)
				if err != nil {
					t.Fatalf("Failed to write to temp file: %v", err)
				}

				tt.config.FilePath = tempFile.Name()
			}

			data, err := LoadPrompts(tt.config)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectError && data == nil {
				t.Error("Expected data but got nil")
			}
		})
	}
}

// Test the Prompt struct
func TestPromptStruct(t *testing.T) {
	prompt := Prompt{
		Title:   "Test Prompt",
		Content: "This is test content",
		Section: "Test Section",
	}

	if prompt.Title != "Test Prompt" {
		t.Errorf("Expected title %q, got %q", "Test Prompt", prompt.Title)
	}
	if prompt.Content != "This is test content" {
		t.Errorf("Expected content %q, got %q", "This is test content", prompt.Content)
	}
	if prompt.Section != "Test Section" {
		t.Errorf("Expected section %q, got %q", "Test Section", prompt.Section)
	}
}

// Test the PromptData struct
func TestPromptDataStruct(t *testing.T) {
	data := &PromptData{
		Prompts:  make([]Prompt, 0),
		Sections: make(map[string][]Prompt),
	}

	if data.Prompts == nil {
		t.Error("Expected Prompts slice to be initialized")
	}
	if data.Sections == nil {
		t.Error("Expected Sections map to be initialized")
	}
	if len(data.Prompts) != 0 {
		t.Error("Expected empty Prompts slice")
	}
	if len(data.Sections) != 0 {
		t.Error("Expected empty Sections map")
	}
}

// Benchmark tests
func BenchmarkParseMarkdown(b *testing.B) {
	for i := 0; i < b.N; i++ {
		parseMarkdown(testMarkdownContent)
	}
}

func BenchmarkSearchPrompts(b *testing.B) {
	data := parseMarkdown(testMarkdownContent)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		SearchPrompts(data, "checklist", "")
	}
}

// Test with debug mode enabled
func TestParseMarkdownWithDebug(t *testing.T) {
	// Enable debug mode
	viper.Set("debug", true)
	defer viper.Set("debug", false)

	// Capture output would be complex, so we just ensure it doesn't panic
	data := parseMarkdown(testMarkdownContent)
	if data == nil {
		t.Error("Expected data but got nil")
	}
}

// Test edge cases for markdown parsing
func TestParseMarkdownEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name:     "headers with extra spaces",
			content:  "##  Section  \n###   Prompt   \nContent",
			expected: 1,
		},
		{
			name:     "empty lines between sections",
			content:  "## Section\n\n### Prompt\nContent\n\n### Another\nMore content",
			expected: 2,
		},
		{
			name:     "content with markdown formatting",
			content:  "## Section\n### Prompt\n**Bold** and _italic_ text\n- List item\n- Another item",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := parseMarkdown(tt.content)
			if len(data.Prompts) != tt.expected {
				t.Errorf("Expected %d prompts, got %d", tt.expected, len(data.Prompts))
			}
		})
	}
}
