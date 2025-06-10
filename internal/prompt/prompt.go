// Package prompt provides functionality for loading, searching, and managing LLM prompts.
// It supports both local Markdown files and Simplenote integration, with fuzzy searching
// capabilities and a terminal user interface for interactive prompt selection.
package prompt

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/spf13/viper"
	"github.com/toozej/wheresmyprompt/pkg/config"
)

// Prompt represents a single LLM prompt with its metadata.
// It contains the prompt's title, content, and the section it belongs to.
type Prompt struct {
	Title   string // The prompt's title or name
	Content string // The actual prompt content
	Section string // The section this prompt belongs to (optional)
}

// PromptData contains the structured data for all prompts.
// It provides both a flat list of all prompts and organized sections
// for efficient searching and categorization.
type PromptData struct {
	Prompts  []Prompt            // All prompts in a flat list
	Sections map[string][]Prompt // Prompts organized by section
}

// CheckRequiredBinaries verifies that all required external binaries are available on the system.
// It checks for sncli (when using Simplenote) and op (1Password CLI) based on the configuration.
// Returns an error if any required binary is missing.
func CheckRequiredBinaries(conf config.Config) error {
	// Always check for sncli if not using filepath
	if conf.FilePath == "" {
		if _, err := exec.LookPath("sncli"); err != nil {
			return fmt.Errorf("sncli binary not found: %w", err)
		}
	}

	// Check for op binary for 1Password integration
	if _, err := exec.LookPath("op"); err != nil {
		return fmt.Errorf("1password CLI (op) binary not found: %w", err)
	}

	return nil
}

// LoadPrompts loads prompts from either a local Markdown file or Simplenote.
// The source is determined by the FilePath field in the configuration.
// If FilePath is empty, it loads from Simplenote; otherwise, it loads from the specified file.
// Returns structured prompt data or an error if loading fails.
func LoadPrompts(conf config.Config) (*PromptData, error) {
	var content string
	var err error

	if conf.FilePath != "" {
		content, err = loadFromFile(conf.FilePath)
	} else {
		content, err = loadFromSimplenote(conf)
	}

	if err != nil {
		return nil, err
	}

	return parseMarkdown(content), nil
}

// loadFromFile reads prompts from a local markdown file.
// Returns the file content as a string or an error if reading fails.
func loadFromFile(filepath string) (string, error) {
	data, err := os.ReadFile(filepath) // #nosec G304
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filepath, err)
	}
	return string(data), nil
}

// loadFromSimplenote fetches the note from Simplenote using the sncli command.
// It ensures authentication is set up before attempting to fetch the note.
// Returns the note content as a string or an error if fetching fails.
func loadFromSimplenote(conf config.Config) (string, error) {
	// First, ensure we're logged in to sncli
	if err := ensureSimplenoteAuth(conf); err != nil {
		return "", err
	}

	// Use sncli to get the note
	cmd := exec.Command("sncli", "dump", conf.SNNote) // #nosec G204
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to fetch note '%s' from Simplenote: %w", conf.SNNote, err)
	}

	return string(output), nil
}

// ensureSimplenoteAuth ensures we're authenticated with Simplenote.
// It supports both direct credentials and 1Password integration for credential management.
// Returns an error if authentication setup fails.
func ensureSimplenoteAuth(conf config.Config) error {
	// Check if already authenticated
	cmd := exec.Command("sncli", "list", conf.SNNote) // #nosec G204
	if err := cmd.Run(); err == nil {
		return nil // Already authenticated
	}

	var username, password string

	// Authenticate using Simplenote credentials directly
	if conf.SNUsername != "" && conf.SNPassword != "" && conf.SNCredential == "" {
		username = conf.SNUsername
		password = conf.SNPassword
	} else {
		// Authenticate using 1Password via op CLI
		if conf.SNCredential == "" {
			return fmt.Errorf("SN_CREDENTIAL op item must be set in config for 1Password integration")
		}
		if conf.SNUsername == "" {
			return fmt.Errorf("SN_USERNAME op item must be set in config for 1Password integration")
		}
		if conf.SNPassword == "" {
			return fmt.Errorf("SN_PASSWORD op item must be set in config for 1Password integration")
		}

		// Fetch username from 1Password
		opUserCmd := exec.Command("op", "item", "get", conf.SNCredential, "--field", conf.SNUsername) // #nosec G204
		userOut, err := opUserCmd.Output()
		if err != nil {
			return fmt.Errorf("failed to fetch SN_USERNAME from 1Password: %w", err)
		}
		username = strings.TrimSpace(string(userOut))

		// Fetch password from 1Password
		opPassCmd := exec.Command("op", "item", "get", conf.SNCredential, "--field", conf.SNPassword, "--reveal") // #nosec G204
		passOut, err := opPassCmd.Output()
		if err != nil {
			return fmt.Errorf("failed to fetch SN_PASSWORD from 1Password: %w", err)
		}
		password = strings.TrimSpace(string(passOut))
	}

	// Set SN_USERNAME and SN_PASSWORD as environment variables for sncli
	// since sncli uses these for authentication rather than a login command
	if err := os.Setenv("SN_USERNAME", username); err != nil {
		return fmt.Errorf("failed to set SN_USERNAME env var: %w", err)
	}
	if err := os.Setenv("SN_PASSWORD", password); err != nil {
		return fmt.Errorf("failed to set SN_PASSWORD env var: %w", err)
	}

	return nil
}

// parseMarkdown parses the markdown content into structured prompt data.
// It expects a specific format with sections marked by "## " and prompts marked by "### ".
// Returns a PromptData structure containing all parsed prompts organized by sections.
func parseMarkdown(content string) *PromptData {
	lines := strings.Split(content, "\n")
	data := &PromptData{
		Prompts:  make([]Prompt, 0),
		Sections: make(map[string][]Prompt),
	}

	var currentSection string
	var currentPrompt strings.Builder
	var promptTitle string
	inPrompt := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for section headers (##)
		if strings.HasPrefix(line, "## ") {
			// Save previous prompt if exists
			if inPrompt && promptTitle != "" {
				prompt := Prompt{
					Title:   promptTitle,
					Content: strings.TrimSpace(currentPrompt.String()),
					Section: currentSection,
				}
				data.Prompts = append(data.Prompts, prompt)
				if currentSection != "" {
					data.Sections[currentSection] = append(data.Sections[currentSection], prompt)
				}
			}

			currentSection = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			currentPrompt.Reset()
			promptTitle = ""
			inPrompt = false
			continue
		}

		// Check for prompt headers (###)
		if strings.HasPrefix(line, "### ") {
			// Save previous prompt if exists
			if inPrompt && promptTitle != "" {
				prompt := Prompt{
					Title:   promptTitle,
					Content: strings.TrimSpace(currentPrompt.String()),
					Section: currentSection,
				}
				data.Prompts = append(data.Prompts, prompt)
				if currentSection != "" {
					data.Sections[currentSection] = append(data.Sections[currentSection], prompt)
				}
			}

			promptTitle = strings.TrimSpace(strings.TrimPrefix(line, "### "))
			currentPrompt.Reset()
			inPrompt = true
			continue
		}

		// Add content to current prompt
		if inPrompt && line != "" {
			if currentPrompt.Len() > 0 {
				currentPrompt.WriteString("\n")
			}
			currentPrompt.WriteString(line)
		}
	}

	// Save the last prompt
	if inPrompt && promptTitle != "" {
		prompt := Prompt{
			Title:   promptTitle,
			Content: strings.TrimSpace(currentPrompt.String()),
			Section: currentSection,
		}
		data.Prompts = append(data.Prompts, prompt)
		if currentSection != "" {
			data.Sections[currentSection] = append(data.Sections[currentSection], prompt)
		}
	}

	if viper.GetBool("debug") {
		s, _ := json.MarshalIndent(data, "", "\t")
		fmt.Print(string(s))
	}

	return data
}

// SearchPrompts performs fuzzy search on prompts using the provided query.
// If a section is specified, it searches only within that section.
// If the query is empty, it returns all prompts (or all prompts in the specified section).
// Returns a slice of prompt content strings matching the search criteria.
func SearchPrompts(data *PromptData, query, section string) []string {
	var searchPool []Prompt

	if section != "" {
		if sectionPrompts, exists := data.Sections[section]; exists {
			searchPool = sectionPrompts
		} else {
			return []string{}
		}
	} else {
		searchPool = data.Prompts
	}

	if query == "" {
		results := make([]string, len(searchPool))
		for i, p := range searchPool {
			results[i] = p.Content
		}
		return results
	}

	// First, search only the Title field
	titleData := make([]string, len(searchPool))
	for i, p := range searchPool {
		titleData[i] = p.Title
	}
	titleMatches := fuzzy.RankFindNormalizedFold(query, titleData)
	if len(titleMatches) > 0 {
		results := make([]string, len(titleMatches))
		for i, match := range titleMatches {
			results[i] = searchPool[match.OriginalIndex].Content
		}
		return results
	}

	// If no title matches, search the Content field
	contentData := make([]string, len(searchPool))
	for i, p := range searchPool {
		contentData[i] = p.Content
	}
	contentMatches := fuzzy.RankFindNormalizedFold(query, contentData)
	results := make([]string, len(contentMatches))
	for i, match := range contentMatches {
		results[i] = searchPool[match.OriginalIndex].Content
	}
	return results
}

// FindAllMatches returns all fuzzy search results for the given query and section.
// It is a convenience wrapper for SearchPrompts, returning all matches.
func FindAllMatches(data *PromptData, query, section string) []string {
	return SearchPrompts(data, query, section)
}

// FindBestMatch returns the best fuzzy match for the given query.
// It performs a search and returns the top result, or an empty string if no matches are found.
// This is useful for one-shot operations where you want the single best match.
func FindBestMatch(data *PromptData, query, section string) string {
	results := SearchPrompts(data, query, section)
	if len(results) == 0 {
		return ""
	}
	return results[0]
}

// GetSectionPrompts returns all prompts from a specific section.
// If the section doesn't exist, it returns an empty slice.
// Returns a slice of prompt content strings from the specified section.
func GetSectionPrompts(data *PromptData, section string) []string {
	if sectionPrompts, exists := data.Sections[section]; exists {
		results := make([]string, len(sectionPrompts))
		for i, p := range sectionPrompts {
			results[i] = p.Content
		}
		return results
	}
	return []string{}
}

// CopyToClipboard copies the provided text to the system clipboard.
// It automatically detects the operating system and uses the appropriate clipboard utility:
// - macOS: pbcopy
// - Linux: xclip or xsel
// - Windows: clip
// Returns an error if the clipboard operation fails or if no suitable utility is found.
func CopyToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("no clipboard utility found (xclip or xsel required)")
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
