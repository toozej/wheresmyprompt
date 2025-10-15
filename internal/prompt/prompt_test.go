package prompt

import (
	"os"
	"runtime"
	"strings"
	"testing"

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

func newPromptDataFromContent(content string) *PromptData {
	sections, err := parseMarkdownIntoSections(content)
	if err != nil {
		panic(err)
	}
	return gatherPromptData(sections)
}

func TestParseMarkdownIntoSections(t *testing.T) {
	sections, err := parseMarkdownIntoSections(testMarkdownContent)
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}
	if len(sections) == 0 {
		t.Error("Expected non-zero sections")
	}
	// Check that we have at least the expected headings
	found := false
	for _, sec := range sections {
		for _, h := range sec.Headings {
			if h == "Code Review" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("Expected to find 'Code Review' heading")
	}
}

func TestSearchPrompts(t *testing.T) {
	data := newPromptDataFromContent(testMarkdownContent)

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
			expectedCount: 17, // all non-empty lines in all sections (including bullet points and numbered items)
		},
		{
			name:          "search by content",
			query:         "email",
			section:       "",
			expectedCount: 2, // fuzzy search finds multiple matches
		},
		{
			name:          "search within specific section",
			query:         "",
			section:       "Code Review",
			expectedCount: 8, // all lines in Code Review section
		},
		{
			name:          "search within specific section with query",
			query:         "bug",
			section:       "Code Review",
			expectedCount: 1,
			shouldContain: []string{"Analyze this bug report and provide:"},
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
	data := newPromptDataFromContent(testMarkdownContent)

	tests := []struct {
		name          string
		query         string
		section       string
		expectEmpty   bool
		shouldContain string
	}{
		{
			name:          "find best match",
			query:         "code",
			section:       "",
			expectEmpty:   false,
			shouldContain: "Please review this code for:",
		},
		{
			name:        "no match found",
			query:       "nomatchforthis",
			section:     "",
			expectEmpty: true,
		},
		{
			name:          "best match in section",
			query:         "documentation",
			section:       "Writing",
			expectEmpty:   false,
			shouldContain: "Create documentation that includes:",
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
	data := newPromptDataFromContent(testMarkdownContent)

	tests := []struct {
		name          string
		section       string
		expectedCount int
		shouldContain []string
	}{
		{
			name:          "existing section",
			section:       "Code Review Checklist",
			expectedCount: 1,
			shouldContain: []string{"Please review this code for:"},
		},
		{
			name:          "another existing section",
			section:       "Email Template",
			expectedCount: 1,
			shouldContain: []string{"Write a professional email template for:"},
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
		Content: "This is test content",
		Section: "Test Section",
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
		Sections: make([]Section, 0),
	}

	if data.Sections == nil {
		t.Error("Expected Sections slice to be initialized")
	}
	if len(data.Sections) != 0 {
		t.Error("Expected empty Sections slice")
	}
}

// Benchmark tests
func BenchmarkParseMarkdown(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := parseMarkdownIntoSections(testMarkdownContent)
		if err != nil {
			b.Fatalf("Failed to parse markdown: %v", err)
		}
	}
}

func BenchmarkSearchPrompts(b *testing.B) {
	sections, err := parseMarkdownIntoSections(testMarkdownContent)
	if err != nil {
		b.Fatalf("Failed to parse markdown: %v", err)
	}
	data := gatherPromptData(sections)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		SearchPrompts(data, "checklist", "")
	}
}

// Test parsing markdown content
func TestParseMarkdownWithDebug(t *testing.T) {
	// Just ensure it doesn't panic
	sections, err := parseMarkdownIntoSections(testMarkdownContent)
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}
	data := gatherPromptData(sections)
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
			expected: 3, // Empty line creates an additional section
		},
		{
			name:     "content with markdown formatting",
			content:  "## Section\n### Prompt\n**Bold** and _italic_ text\n- List item\n- Another item",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sections, err := parseMarkdownIntoSections(tt.content)
			if err != nil {
				t.Fatalf("Failed to parse markdown: %v", err)
			}
			data := gatherPromptData(sections)
			if len(data.Sections) != tt.expected {
				t.Errorf("Expected %d sections, got %d", tt.expected, len(data.Sections))
			}
		})
	}
}
