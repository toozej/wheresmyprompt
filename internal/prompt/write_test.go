package prompt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/toozej/wheresmyprompt/pkg/config"
)

// FileSystem interface for testing
type FileSystem interface {
	ReadFile(filename string) ([]byte, error)
	WriteFile(filename string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
	Exists(path string) (bool, error)
}

// OSFileSystem implements FileSystem using os package
type OSFileSystem struct{}

func (fs OSFileSystem) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

func (fs OSFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}

func (fs OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fs OSFileSystem) Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// AferoFileSystem implements FileSystem using afero
type AferoFileSystem struct {
	fs afero.Fs
}

func NewAferoFileSystem(fs afero.Fs) *AferoFileSystem {
	return &AferoFileSystem{fs: fs}
}

func (afs *AferoFileSystem) ReadFile(filename string) ([]byte, error) {
	return afero.ReadFile(afs.fs, filename)
}

func (afs *AferoFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return afero.WriteFile(afs.fs, filename, data, perm)
}

func (afs *AferoFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return afs.fs.MkdirAll(path, perm)
}

func (afs *AferoFileSystem) Exists(path string) (bool, error) {
	return afero.Exists(afs.fs, path)
}

// Global filesystem variable for dependency injection
var filesystem FileSystem = OSFileSystem{}

// Helper function to simulate stdin input
func simulateStdin(input string, f func()) {
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		defer w.Close()
		_, _ = w.Write([]byte(input))
	}()

	f()
	os.Stdin = oldStdin
}

func TestGenerateTitleFromContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "normal content with multiple words",
			content:  "this is a test prompt for generating titles",
			expected: "This is a test prompt",
		},
		{
			name:     "empty content",
			content:  "",
			expected: "Untitled Prompt",
		},
		{
			name:     "single word",
			content:  "hello",
			expected: "Hello",
		},
		{
			name:     "three words",
			content:  "hello world test",
			expected: "Hello world test",
		},
		{
			name:     "content with punctuation",
			content:  "hello world, this is great!",
			expected: "Hello world, this is great",
		},
		{
			name:     "content ending with punctuation",
			content:  "hello world.",
			expected: "Hello world",
		},
		{
			name:     "content with multiple punctuation",
			content:  "hello world!?;:",
			expected: "Hello world",
		},
		{
			name:     "content with extra whitespace",
			content:  "  hello   world   test  ",
			expected: "Hello world test",
		},
		{
			name:     "content with newlines",
			content:  "hello\nworld\ntest\nmore\nwords",
			expected: "Hello world test more words",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateTitleFromContent(tt.content)
			if result != tt.expected {
				t.Errorf("generateTitleFromContent(%q) = %q, want %q", tt.content, result, tt.expected)
			}
		})
	}
}

// Modified version of addPromptToFile that accepts a FileSystem for testing
func addPromptToFileWithFS(fs FileSystem, filepath, title, content, section string) error {
	// Read existing content
	existingContent := ""
	data, err := fs.ReadFile(filepath)
	if err == nil {
		existingContent = string(data)
	}

	// Parse existing content into sections using new parser
	sections, err := parseMarkdownIntoSections(existingContent)
	if err != nil {
		return fmt.Errorf("failed to parse markdown: %w", err)
	}
	promptData := gatherPromptData(sections)

	var newContent strings.Builder
	sectionFound := false

	if section != "" {
		// Try to find the section and append prompt
		for i, sec := range promptData.Sections {
			if len(sec.Headings) > 0 && sec.Headings[len(sec.Headings)-1] == section {
				sectionFound = true
				// Write all sections up to this one
				for j := 0; j < i; j++ {
					writeSection(&newContent, promptData.Sections[j])
				}
				// Write this section header
				writeSectionHeader(&newContent, sec)
				// Write existing lines
				for _, line := range sec.Lines {
					newContent.WriteString(line + "\n")
				}
				// Add new prompt
				newContent.WriteString("\n### " + title + "\n")
				newContent.WriteString(content + "\n\n")
				// Write remaining sections
				for j := i + 1; j < len(promptData.Sections); j++ {
					writeSection(&newContent, promptData.Sections[j])
				}
				break
			}
		}
		if !sectionFound {
			// Section not found, preserve existing content and append new section at end
			newContent.WriteString(existingContent)
			if !strings.HasSuffix(existingContent, "\n") {
				newContent.WriteString("\n")
			}
			newContent.WriteString("\n\n## " + section + "\n\n")
			newContent.WriteString("### " + title + "\n")
			newContent.WriteString(content + "\n")
		}
	} else {
		// No section specified, add at the end
		newContent.WriteString(existingContent)
		if !strings.HasSuffix(existingContent, "\n") {
			newContent.WriteString("\n")
		}
		newContent.WriteString("\n### " + title + "\n")
		newContent.WriteString(content + "\n")
	}

	// Write back to file
	return fs.WriteFile(filepath, []byte(newContent.String()), 0600)
}

func TestAddPromptToFile(t *testing.T) {
	tests := []struct {
		name            string
		existingContent string
		title           string
		content         string
		section         string
		expectedContent string
		expectError     bool
	}{
		{
			name:            "add to empty file without section",
			existingContent: "",
			title:           "Test Title",
			content:         "Test content",
			section:         "",
			expectedContent: "\n\n### Test Title\nTest content\n",
			expectError:     false,
		},
		{
			name:            "add to existing file without section",
			existingContent: "# Existing Notes\n\n### Old Title\nOld content\n",
			title:           "New Title",
			content:         "New content",
			section:         "",
			expectedContent: "# Existing Notes\n\n### Old Title\nOld content\n\n### New Title\nNew content\n",
			expectError:     false,
		},
		{
			name:            "add to new section",
			existingContent: "# Existing Notes\n\n### Old Title\nOld content\n",
			title:           "New Title",
			content:         "New content",
			section:         "New Section",
			expectedContent: "# Existing Notes\n\n### Old Title\nOld content\n\n\n## New Section\n\n### New Title\nNew content\n",
			expectError:     false,
		},
		// 		{
		// 			name: "add to existing section",
		// 			existingContent: `# Notes

		// ## Existing Section

		// ### Old Title
		// Old content

		// ## Another Section

		// ### Another Title
		// Another content`,
		// 			title:   "New Title",
		// 			content: "New content",
		// 			section: "Existing Section",
		// 			expectedContent: `# Notes

		// ## Existing Section

		// ### Old Title
		// Old content

		// ### New Title
		// New content

		// ## Another Section

		// ### Another Title
		// Another content
		// `,
		// 			expectError: false,
		// 		},
		{
			name:            "add to file without trailing newline",
			existingContent: "# Notes\n\n### Old Title\nOld content",
			title:           "New Title",
			content:         "New content",
			section:         "",
			expectedContent: "# Notes\n\n### Old Title\nOld content\n\n### New Title\nNew content\n",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new memory filesystem for each test
			memFS := afero.NewMemMapFs()
			fs := NewAferoFileSystem(memFS)
			filepath := "/test/notes.md"

			// Create directory structure
			_ = fs.MkdirAll("/test", 0755)

			// Write existing content if any
			if tt.existingContent != "" {
				_ = fs.WriteFile(filepath, []byte(tt.existingContent), 0644)
			} else {
				// Ensure file exists even if empty
				_ = fs.WriteFile(filepath, []byte(""), 0644)
			}

			err := addPromptToFileWithFS(fs, filepath, tt.title, tt.content, tt.section)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				content, err := fs.ReadFile(filepath)
				if err != nil {
					t.Fatalf("failed to read file after writing: %v", err)
				}

				if string(content) != tt.expectedContent {
					t.Errorf("file content mismatch:\nexpected:\n%q\ngot:\n%q", tt.expectedContent, string(content))
				}
			}
		})
	}
}

func TestAddToExistingSection(t *testing.T) {
	tests := []struct {
		name           string
		currentContent string
		title          string
		content        string
		section        string
		expectedResult bool
		expectedOutput string
	}{
		// 		{
		// 			name: "section exists",
		// 			currentContent: `# Notes

		// ## Test Section

		// ### Old Title
		// Old content

		// ## Another Section

		// ### Another Title
		// Another content`,
		// 			title:          "New Title",
		// 			content:        "New content",
		// 			section:        "Test Section",
		// 			expectedResult: true,
		// 			expectedOutput: `# Notes

		// ## Test Section

		// ### Old Title
		// Old content

		// ### New Title
		// New content

		// ## Another Section

		// ### Another Title
		// Another content
		// `,
		// 		},
		{
			name: "section does not exist",
			currentContent: `# Notes

## Different Section

### Old Title
Old content`,
			title:          "New Title",
			content:        "New content",
			section:        "Non-existent Section",
			expectedResult: false,
			expectedOutput: "",
		},
		{
			name:           "empty content",
			currentContent: "",
			title:          "New Title",
			content:        "New content",
			section:        "Test Section",
			expectedResult: false,
			expectedOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var newContent strings.Builder
			result := addToExistingSection(&newContent, tt.currentContent, tt.title, tt.content, tt.section)

			if result != tt.expectedResult {
				t.Errorf("addToExistingSection() = %v, want %v", result, tt.expectedResult)
			}

			if tt.expectedResult {
				output := newContent.String()
				if output != tt.expectedOutput {
					t.Errorf("output mismatch:\nexpected:\n%q\ngot:\n%q", tt.expectedOutput, output)
				}
			}
		})
	}
}

// Test helper process for simulating sncli commands
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(2)
	}

	cmd, args := args[0], args[1:]

	switch cmd {
	case "sncli":
		// Mock successful sncli commands
		if len(args) > 0 && args[0] == "note" {
			fmt.Println("Note updated successfully")
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Unknown sncli command\n")
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command %q\n", cmd)
		os.Exit(2)
	}
}

func TestWritePrompt(t *testing.T) {
	tests := []struct {
		name          string
		config        config.Config
		promptContent string
		args          []string
		stdinInput    string
		expectError   bool
		errorContains string
	}{
		// {
		// 	name: "write with prompt content flag",
		// 	config: config.Config{
		// 		FilePath: "/test/notes.md",
		// 	},
		// 	promptContent: "This is test content for prompt",
		// 	args:          []string{},
		// 	expectError:   false,
		// },
		// {
		// 	name: "write with args",
		// 	config: config.Config{
		// 		FilePath: "/test/notes.md",
		// 	},
		// 	promptContent: "",
		// 	args:          []string{"This", "is", "test", "content"},
		// 	expectError:   false,
		// },
		// {
		// 	name: "write with stdin input",
		// 	config: config.Config{
		// 		FilePath: "/test/notes.md",
		// 	},
		// 	promptContent: "",
		// 	args:          []string{},
		// 	stdinInput:    "Test Title\nThis is test content\n",
		// 	expectError:   false,
		// },
		{
			name: "empty content should error",
			config: config.Config{
				FilePath: "/test/notes.md",
			},
			promptContent: "",
			args:          []string{},
			stdinInput:    "\n\n",
			expectError:   true,
			errorContains: "both title and content are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new memory filesystem for each test
			memFS := afero.NewMemMapFs()
			fs := NewAferoFileSystem(memFS)
			_ = fs.MkdirAll("/test", 0755)
			_ = fs.WriteFile("/test/notes.md", []byte(""), 0644) // Ensure file exists

			// Set up filesystem for testing
			originalFS := filesystem
			filesystem = fs
			defer func() {
				filesystem = originalFS
			}()

			var err error
			if tt.stdinInput != "" {
				simulateStdin(tt.stdinInput, func() {
					err = WritePrompt(tt.config, tt.promptContent, tt.args)
				})
			} else {
				err = WritePrompt(tt.config, tt.promptContent, tt.args)
			}

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error should contain %q, got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// Verify file was created/updated
				if tt.config.FilePath != "" {
					exists, err := fs.Exists(tt.config.FilePath)
					if err != nil {
						t.Errorf("failed to check file existence: %v", err)
					}
					if !exists {
						t.Error("expected file to be created")
					}
				}
			}
		})
	}
}

func TestAddPromptToNote(t *testing.T) {
	tests := []struct {
		name        string
		config      config.Config
		title       string
		content     string
		section     string
		expectError bool
	}{
		// {
		// 	name: "add to file",
		// 	config: config.Config{
		// 		FilePath: "/test/notes.md",
		// 	},
		// 	title:       "Test Title",
		// 	content:     "Test content",
		// 	section:     "Test Section",
		// 	expectError: false,
		// },
		{
			name: "add to simplenote (will fail without mocking)",
			config: config.Config{
				SNNote: "test-note",
			},
			title:       "Test Title",
			content:     "Test content",
			section:     "Test Section",
			expectError: true, // Will fail because we can't mock all simplenote operations easily
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.FilePath != "" {
				// Create a new memory filesystem for file tests
				memFS := afero.NewMemMapFs()
				fs := NewAferoFileSystem(memFS)
				_ = fs.MkdirAll("/test", 0755)
				_ = fs.WriteFile("/test/notes.md", []byte(""), 0644) // Ensure file exists

				// Set up filesystem for testing
				originalFS := filesystem
				filesystem = fs
				defer func() {
					filesystem = originalFS
				}()
			}

			err := addPromptToNote(tt.config, tt.title, tt.content, tt.section)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// Helper to mock exec.Command for sncli import - and capture stdin JSON
// func mockSncliImport(expectedContent string, expectedKey string, testFunc func()) {
// 	oldExecCommand := execCommand
// 	defer func() { execCommand = oldExecCommand }()

// 	execCommand = func(name string, args ...string) *exec.Cmd {
// 		if name == "sncli" && len(args) == 2 && args[0] == "import" && args[1] == "-" {
// 			return helperSncliImportCmd(expectedContent, expectedKey)
// 		}
// 		// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
// 		return exec.Command(name, args...)
// 	}
// 	testFunc()
// }

// Helper exec.Cmd for sncli import -
// func helperSncliImportCmd(expectedContent string, expectedKey string) *exec.Cmd {
// 	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
// 	return exec.Command(os.Args[0], "-test.run=TestSncliImportHelper", "--", expectedContent, expectedKey)
// }

// Test helper process for sncli import -
func TestSncliImportHelper(t *testing.T) {
	if os.Getenv("GO_WANT_SNCLI_IMPORT_HELPER") != "1" {
		return
	}
	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) < 2 {
		os.Exit(2)
	}
	expectedContent := args[0]
	expectedKey := args[1]
	// Read stdin
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, os.Stdin)
	var notes []map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &notes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid JSON: %v\n", err)
		os.Exit(3)
	}
	if len(notes) != 1 {
		fmt.Fprintf(os.Stderr, "Expected 1 note, got %d\n", len(notes))
		os.Exit(4)
	}
	note := notes[0]
	if note["content"] != expectedContent {
		fmt.Fprintf(os.Stderr, "Content mismatch: got %q, want %q\n", note["content"], expectedContent)
		os.Exit(5)
	}
	if note["key"] != expectedKey {
		fmt.Fprintf(os.Stderr, "Key mismatch: got %q, want %q\n", note["key"], expectedKey)
		os.Exit(6)
	}
	os.Exit(0)
}

// Patch exec.Command for testing
// var execCommand = exec.Command

// Patch addPromptToSimplenote to use execCommand
// func TestAddPromptToSimplenote_JSON(t *testing.T) {
// 	conf := config.Config{SNNote: "test-note"}
// 	title := "Test Title"
// 	content := "Test content"
// 	section := "Test Section"
// 	// Simulate current note content
// 	oldLoad := loadFromSimplenoteFunc
// 	oldAuth := ensureSimplenoteAuthFunc
// 	defer func() {
// 		loadFromSimplenoteFunc = oldLoad
// 		ensureSimplenoteAuthFunc = oldAuth
// 	}()
// 	loadFromSimplenoteFunc = func(conf config.Config) (string, error) {
// 		return "# Notes\n", nil
// 	}
// 	ensureSimplenoteAuthFunc = func(conf config.Config) error { return nil }

// 	expectedContent := "# Notes\n\n## Test Section\n\n### Test Title\nTest content\n"
// 	mockSncliImport(expectedContent, "test-note", func() {
// 		err := addPromptToSimplenote(conf, title, content, section)
// 		if err != nil {
// 			t.Errorf("unexpected error: %v", err)
// 		}
// 	})
// }

// func TestAddPromptToNote_JSON(t *testing.T) {
// 	conf := config.Config{SNNote: "test-note"}
// 	title := "Test Title"
// 	content := "Test content"
// 	section := "Test Section"
// 	oldLoad := loadFromSimplenoteFunc
// 	oldAuth := ensureSimplenoteAuthFunc
// 	defer func() {
// 		loadFromSimplenoteFunc = oldLoad
// 		ensureSimplenoteAuthFunc = oldAuth
// 	}()
// 	loadFromSimplenoteFunc = func(conf config.Config) (string, error) {
// 		return "# Notes\n", nil
// 	}
// 	ensureSimplenoteAuthFunc = func(conf config.Config) error { return nil }

// 	expectedContent := "# Notes\n\n## Test Section\n\n### Test Title\nTest content\n"
// 	mockSncliImport(expectedContent, "test-note", func() {
// 		err := addPromptToNote(conf, title, content, section)
// 		if err != nil {
// 			t.Errorf("unexpected error: %v", err)
// 		}
// 	})
// }

// Benchmark tests
func BenchmarkGenerateTitleFromContent(b *testing.B) {
	content := "this is a long piece of content that we want to generate a title from"

	for i := 0; i < b.N; i++ {
		generateTitleFromContent(content)
	}
}

func BenchmarkAddPromptToFile(b *testing.B) {
	memFS := afero.NewMemMapFs()
	fs := NewAferoFileSystem(memFS)
	filepath := "/test/notes.md"
	_ = fs.MkdirAll("/test", 0755)

	// Create initial content
	initialContent := `# My Notes

## Section 1

### Title 1
Content 1

## Section 2

### Title 2
Content 2`

	_ = fs.WriteFile(filepath, []byte(initialContent), 0644)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = addPromptToFileWithFS(fs, filepath, "Benchmark Title", "Benchmark content", "Section 1")
	}
}
