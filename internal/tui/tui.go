// Package tui provides a terminal user interface for interactive prompt selection.
// It uses the Bubble Tea framework to create a responsive, keyboard-driven interface
// with fuzzy search capabilities and live filtering.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/toozej/wheresmyprompt/internal/prompt"
	"github.com/toozej/wheresmyprompt/pkg/config"
)

type model struct {
	textInput       textinput.Model
	prompts         *prompt.PromptData
	searchPool      []prompt.Prompt
	filteredResults []prompt.Prompt
	cursor          int
	config          config.Config
	err             error
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4"))

	promptStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 2).
			MarginTop(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))
)

// RunTUI starts the terminal user interface for interactive prompt selection.
// It creates a searchable, navigable interface where users can fuzzy search through prompts
// and select one to copy to the clipboard. The interface supports keyboard navigation
// with vim-like keybindings and real-time search filtering.
// Returns an error if the TUI fails to start or encounters runtime errors.
func RunTUI(prompts *prompt.PromptData, conf config.Config) error {
	ti := textinput.New()
	ti.Placeholder = "Search prompts..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50

	searchPool := generateSearchPoolFromSections(prompts)

	m := model{
		textInput:       ti,
		prompts:         prompts,
		searchPool:      searchPool,
		filteredResults: searchPool,
		config:          conf,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "enter":
			if len(m.filteredResults) > 0 && m.cursor < len(m.filteredResults) {
				selectedPrompt := m.filteredResults[m.cursor]
				if err := prompt.CopyToClipboard(selectedPrompt.Content); err != nil {
					m.err = err
					return m, nil
				}
				return m, tea.Quit
			}

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.filteredResults)-1 {
				m.cursor++
			}

		default:
			m.textInput, cmd = m.textInput.Update(msg)
			m.filterResults()
			if m.cursor >= len(m.filteredResults) {
				m.cursor = len(m.filteredResults) - 1
			}
			if m.cursor < 0 {
				m.cursor = 0
			}
		}

	case tea.WindowSizeMsg:
		// Handle window resize if needed
	}

	return m, cmd
}

func (m *model) filterResults() {
	query := m.textInput.Value()
	if query == "" {
		m.filteredResults = m.searchPool
		return
	}

	// Prepare data for fuzzy search
	searchData := make([]string, len(m.searchPool))
	for i, p := range m.searchPool {
		searchData[i] = p.Content
	}

	matches := fuzzy.RankFindNormalizedFold(query, searchData)
	m.filteredResults = make([]prompt.Prompt, len(matches))
	for i, match := range matches {
		m.filteredResults[i] = m.searchPool[match.OriginalIndex]
	}
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress Ctrl+C to exit", m.err)
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("Where's My Prompt?"))
	b.WriteString("\n\n")

	// Search input
	b.WriteString("Search: ")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")

	// Results
	if len(m.filteredResults) == 0 {
		b.WriteString("No prompts found.\n")
	} else {
		b.WriteString(fmt.Sprintf("Found %d prompt(s):\n\n", len(m.filteredResults)))

		// Show first few results
		maxDisplay := 5
		if len(m.filteredResults) < maxDisplay {
			maxDisplay = len(m.filteredResults)
		}

		for i := 0; i < maxDisplay; i++ {
			prompt := m.filteredResults[i]
			cursor := " "
			if m.cursor == i {
				cursor = "▶"
			}

			title := prompt.Section
			if m.cursor == i {
				title = selectedStyle.Render(title)
			}

			section := ""
			if prompt.Section != "" {
				section = fmt.Sprintf(" [%s]", prompt.Section)
			}

			b.WriteString(fmt.Sprintf("%s %s%s\n", cursor, title, section))

			// Show preview of content for selected item
			if m.cursor == i {
				preview := prompt.Content
				if len(preview) > 100 {
					preview = preview[:100] + "..."
				}
				b.WriteString(promptStyle.Render(preview))
				b.WriteString("\n")
			}
		}

		if len(m.filteredResults) > maxDisplay {
			b.WriteString(fmt.Sprintf("\n... and %d more\n", len(m.filteredResults)-maxDisplay))
		}
	}

	// Help
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/k up • ↓/j down • enter select & copy • ctrl+c/esc quit"))

	return b.String()
}

// Helper to flatten PromptData.Sections into []Prompt
func generateSearchPoolFromSections(data *prompt.PromptData) []prompt.Prompt {
	var pool []prompt.Prompt
	for _, sec := range data.Sections {
		sectionTitle := ""
		if len(sec.Headings) > 0 {
			sectionTitle = sec.Headings[len(sec.Headings)-1]
		}
		for _, line := range sec.Lines {
			if strings.TrimSpace(line) != "" {
				pool = append(pool, prompt.Prompt{
					Content: line,
					Section: sectionTitle,
				})
			}
		}
	}
	return pool
}
