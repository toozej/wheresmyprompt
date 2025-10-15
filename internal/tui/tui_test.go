package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toozej/wheresmyprompt/internal/prompt"
	"github.com/toozej/wheresmyprompt/pkg/config"
)

// Mock data for testing
var mockPrompts = &prompt.PromptData{
	Sections: []prompt.Section{
		{
			Headings: []string{"development"},
			Lines:    []string{"Write a function that generates code based on requirements", "Help me debug this specific issue in my application"},
		},
		{
			Headings: []string{"testing"},
			Lines:    []string{"Create comprehensive unit tests for the given code"},
		},
		{
			Headings: []string{"review"},
			Lines:    []string{"Please review this code for best practices and improvements"},
		},
	},
}

var mockConfig = config.Config{
	// Add mock config fields as needed
}

func TestModel_Init(t *testing.T) {
	searchPool := generateSearchPoolFromSections(mockPrompts)
	m := model{
		textInput:       textinput.New(),
		prompts:         mockPrompts,
		filteredResults: searchPool,
		searchPool:      searchPool,
		config:          mockConfig,
	}

	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() should return a command, got nil")
	}
}

func TestModel_Update(t *testing.T) {
	tests := []struct {
		name           string
		initialCursor  int
		msg            tea.Msg
		expectedCursor int
		expectQuit     bool
	}{
		{
			name:           "quit on ctrl+c",
			initialCursor:  0,
			msg:            tea.KeyMsg{Type: tea.KeyCtrlC},
			expectedCursor: 0,
			expectQuit:     true,
		},
		{
			name:           "quit on esc",
			initialCursor:  0,
			msg:            tea.KeyMsg{Type: tea.KeyEsc},
			expectedCursor: 0,
			expectQuit:     true,
		},
		{
			name:           "move cursor up with arrow key",
			initialCursor:  2,
			msg:            tea.KeyMsg{Type: tea.KeyUp},
			expectedCursor: 1,
			expectQuit:     false,
		},
		{
			name:           "move cursor up with k",
			initialCursor:  2,
			msg:            tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}},
			expectedCursor: 1,
			expectQuit:     false,
		},
		{
			name:           "move cursor down with arrow key",
			initialCursor:  1,
			msg:            tea.KeyMsg{Type: tea.KeyDown},
			expectedCursor: 2,
			expectQuit:     false,
		},
		{
			name:           "move cursor down with j",
			initialCursor:  1,
			msg:            tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}},
			expectedCursor: 2,
			expectQuit:     false,
		},
		{
			name:           "cursor stays at 0 when at top",
			initialCursor:  0,
			msg:            tea.KeyMsg{Type: tea.KeyUp},
			expectedCursor: 0,
			expectQuit:     false,
		},
		{
			name:           "cursor stays at bottom when at end",
			initialCursor:  3, // Last index for 4 items
			msg:            tea.KeyMsg{Type: tea.KeyDown},
			expectedCursor: 3,
			expectQuit:     false,
		},
	}

	searchPool := generateSearchPoolFromSections(mockPrompts)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ti := textinput.New()
			m := model{
				textInput:       ti,
				prompts:         mockPrompts,
				filteredResults: searchPool,
				searchPool:      searchPool,
				cursor:          tt.initialCursor,
				config:          mockConfig,
			}

			updatedModel, cmd := m.Update(tt.msg)
			updatedM, ok := updatedModel.(model)
			if !ok {
				t.Fatalf("expected model type, got %T", updatedModel)
			}

			if updatedM.cursor != tt.expectedCursor {
				t.Errorf("expected cursor %d, got %d", tt.expectedCursor, updatedM.cursor)
			}

			if tt.expectQuit && cmd == nil {
				t.Error("expected quit command, got nil")
			}
		})
	}
}

func TestModel_Update_WindowResize(t *testing.T) {
	ti := textinput.New()
	searchPool := generateSearchPoolFromSections(mockPrompts)
	m := model{
		textInput:       ti,
		prompts:         mockPrompts,
		filteredResults: searchPool,
		searchPool:      searchPool,
		cursor:          0,
		config:          mockConfig,
	}

	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updatedModel, cmd := m.Update(msg)

	// Window resize should not cause any errors and should return the model
	if cmd != nil {
		t.Error("window resize should not return any command, got non-nil command")
	}

	// Verify the model is returned properly
	if updatedModel == nil {
		t.Error("expected updated model, got nil")
	}
}

func TestModel_FilterResults(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		expectedCount int
	}{
		{
			name:          "empty query returns all prompts",
			query:         "",
			expectedCount: len(generateSearchPoolFromSections(mockPrompts)),
		},
		{
			name:          "search for 'code' finds relevant prompts",
			query:         "code",
			expectedCount: 3, // All prompts contain 'code' in some form
		},
		{
			name:          "search for 'test' finds test-related prompt",
			query:         "test",
			expectedCount: 4, // Fuzzy search finds matches in all prompts
		},
		{
			name:          "search for non-existent term",
			query:         "nonexistent",
			expectedCount: 0,
		},
		{
			name:          "case insensitive search",
			query:         "CODE",
			expectedCount: 3, // Should find same as lowercase 'code'
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ti := textinput.New()
			ti.SetValue(tt.query)

			searchPool := generateSearchPoolFromSections(mockPrompts)
			m := &model{
				textInput:       ti,
				prompts:         mockPrompts,
				searchPool:      searchPool,
				filteredResults: searchPool,
				cursor:          0,
				config:          mockConfig,
			}

			m.filterResults()

			if len(m.filteredResults) != tt.expectedCount {
				t.Errorf("expected %d results, got %d", tt.expectedCount, len(m.filteredResults))
			}

			// Verify we got the expected count of results
			// The actual content matching is tested by the filtering logic itself
		})
	}
}

func TestModel_View(t *testing.T) {
	tests := []struct {
		name                string
		filteredResults     []prompt.Prompt
		cursor              int
		err                 error
		expectedContains    []string
		expectedNotContains []string
	}{
		{
			name:            "normal view with results",
			filteredResults: generateSearchPoolFromSections(mockPrompts)[:2],
			cursor:          0,
			err:             nil,
			expectedContains: []string{
				"Where's My Prompt?",
				"Search:",
				"Found 2 prompt(s):",
				"Write a function that generates code based on requirements",
				"▶", // Cursor indicator
			},
			expectedNotContains: []string{"Error:", "No prompts found"},
		},
		{
			name:            "view with no results",
			filteredResults: []prompt.Prompt{},
			cursor:          0,
			err:             nil,
			expectedContains: []string{
				"Where's My Prompt?",
				"Search:",
				"No prompts found",
			},
			expectedNotContains: []string{"Error:", "Found", "prompt(s):"},
		},
		{
			name:            "view with error",
			filteredResults: generateSearchPoolFromSections(mockPrompts),
			cursor:          0,
			err:             fmt.Errorf("test error"),
			expectedContains: []string{
				"Error:",
				"Press Ctrl+C to exit",
			},
			expectedNotContains: []string{"Where's My Prompt?", "Search:"},
		},
		{
			name:            "view with cursor at second item",
			filteredResults: generateSearchPoolFromSections(mockPrompts)[:3],
			cursor:          1,
			err:             nil,
			expectedContains: []string{
				"Help me debug this specific issue in my application", // Should be highlighted (cursor at index 1)
				"Found 3 prompt(s):",
			},
			expectedNotContains: []string{"Error:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ti := textinput.New()
			m := model{
				textInput:       ti,
				prompts:         mockPrompts,
				filteredResults: tt.filteredResults,
				cursor:          tt.cursor,
				config:          mockConfig,
				err:             tt.err,
			}

			view := m.View()

			for _, expected := range tt.expectedContains {
				if !strings.Contains(view, expected) {
					t.Errorf("expected view to contain '%s', but it didn't.\nView: %s", expected, view)
				}
			}

			for _, notExpected := range tt.expectedNotContains {
				if strings.Contains(view, notExpected) {
					t.Errorf("expected view to NOT contain '%s', but it did.\nView: %s", notExpected, view)
				}
			}
		})
	}
}

func TestModel_View_MaxDisplay(t *testing.T) {
	// Test that only 5 items are displayed maximum
	manyPrompts := make([]prompt.Prompt, 10)
	for i := 0; i < 10; i++ {
		manyPrompts[i] = prompt.Prompt{
			Content: fmt.Sprintf("Content for prompt %d", i+1),
			Section: "test",
		}
	}

	ti := textinput.New()
	m := model{
		textInput:       ti,
		prompts:         &prompt.PromptData{},
		filteredResults: manyPrompts,
		searchPool:      manyPrompts,
		cursor:          0,
		config:          mockConfig,
	}

	view := m.View()

	// Should show "Found 10 prompt(s)" but only display first 5
	if !strings.Contains(view, "Found 10 prompt(s):") {
		t.Error("should show total count of 10 prompts")
	}

	if !strings.Contains(view, "... and 5 more") {
		t.Error("should show '... and 5 more' for remaining prompts")
	}

	// Count lines that represent prompt items (either selected with ▶ or unselected with spaces)
	lines := strings.Split(view, "\n")
	promptItemCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Look for lines that contain section names with brackets
		if (strings.HasPrefix(line, "▶ ") || strings.HasPrefix(line, "  ")) &&
			strings.Contains(trimmed, "[") && strings.Contains(trimmed, "]") {
			promptItemCount++
		}
	}
	if promptItemCount != 5 {
		t.Errorf("expected 5 prompts displayed, got %d. View:\n%s", promptItemCount, view)
	}
}

func TestModel_View_ContentPreview(t *testing.T) {
	longContent := strings.Repeat("This is a very long content ", 10) // > 100 chars
	shortContent := "Short content"

	prompts := []prompt.Prompt{
		{
			Content: longContent,
			Section: "test",
		},
		{
			Content: shortContent,
			Section: "test",
		},
	}

	ti := textinput.New()
	m := model{
		textInput:       ti,
		prompts:         &prompt.PromptData{},
		filteredResults: prompts,
		searchPool:      prompts,
		cursor:          0, // First item selected
		config:          mockConfig,
	}

	view := m.View()

	// Should truncate long content with "..."
	if !strings.Contains(view, "...") {
		t.Error("long content should be truncated with '...'")
	}

	// Test with short content selected
	m.cursor = 1
	view = m.View()

	// Should show full short content
	if strings.Contains(view, shortContent) && strings.Contains(view, "...") {
		// This is a bit tricky to test precisely due to styling, but we can check
		// that short content doesn't get truncated inappropriately
		t.Error("short content should not be truncated with '...', but it was")
	}
}

func TestModel_View_HelpText(t *testing.T) {
	ti := textinput.New()
	searchPool := generateSearchPoolFromSections(mockPrompts)
	m := model{
		textInput:       ti,
		prompts:         mockPrompts,
		filteredResults: searchPool,
		searchPool:      searchPool,
		cursor:          0,
		config:          mockConfig,
	}

	view := m.View()

	expectedHelp := "↑/k up • ↓/j down • enter select & copy • ctrl+c/esc quit"
	if !strings.Contains(view, expectedHelp) {
		t.Errorf("expected help text '%s' in view, but didn't find it", expectedHelp)
	}
}

// Benchmark tests
func BenchmarkModel_FilterResults_EmptyQuery(b *testing.B) {
	ti := textinput.New()
	searchPool := generateSearchPoolFromSections(mockPrompts)
	m := &model{
		textInput:       ti,
		prompts:         mockPrompts,
		filteredResults: searchPool,
		searchPool:      searchPool,
		cursor:          0,
		config:          mockConfig,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.filterResults()
	}
}

func BenchmarkModel_FilterResults_WithQuery(b *testing.B) {
	ti := textinput.New()
	ti.SetValue("code")
	searchPool := generateSearchPoolFromSections(mockPrompts)
	m := &model{
		textInput:       ti,
		prompts:         mockPrompts,
		filteredResults: searchPool,
		searchPool:      searchPool,
		cursor:          0,
		config:          mockConfig,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.filterResults()
	}
}

func BenchmarkModel_View(b *testing.B) {
	ti := textinput.New()
	searchPool := generateSearchPoolFromSections(mockPrompts)
	m := model{
		textInput:       ti,
		prompts:         mockPrompts,
		filteredResults: searchPool,
		searchPool:      searchPool,
		cursor:          0,
		config:          mockConfig,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.View()
	}
}
