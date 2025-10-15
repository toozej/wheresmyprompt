// Package prompt provides functionality for loading, searching, and managing LLM prompts.
// It supports both local Markdown files and Simplenote integration, with fuzzy searching
// capabilities and a terminal user interface for interactive prompt selection.
package prompt

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/toozej/wheresmyprompt/pkg/config"
)

// Prompt represents a single LLM prompt with its metadata.
// It contains the prompt's content and the section it belongs to.
type Prompt struct {
	Content string // The actual prompt content
	Section string // The section this prompt belongs to
}

// PromptData contains the structured data for all prompts.
// providing a list of sections for efficient searching and categorization.
type PromptData struct {
	Sections []Section // All sections parsed from the markdown
}

// Section represents a heading (any depth) and its associated lines
type Section struct {
	Headings []string // Ordered from top-level heading to deepest sub-heading
	Lines    []string
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

	// Parse the loaded content into []sections
	sections, err := parseMarkdownIntoSections(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse markdown content: %w", err)
	}
	// Gather the loaded sections into structured prompt data
	return gatherPromptData(sections), nil
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

// parseMarkdown parses the markdown file's content into sections grouped by any heading level
func parseMarkdownIntoSections(content string) ([]Section, error) {

	var sections []Section
	var current Section
	var headingStack []string

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		level, headingText := parseHeading(line)
		if level > 0 {
			// Update heading stack
			if len(headingStack) < level {
				// Deeper heading: extend stack
				headingStack = append(headingStack, headingText)
			} else {
				// Replace heading at this level and truncate deeper levels
				headingStack = append(headingStack[:level-1], headingText)
			}

			// Save previous section
			if len(current.Lines) > 0 {
				sections = append(sections, current)
			}
			// Start new section
			current = Section{
				Headings: append([]string(nil), headingStack...), // copy
			}
		} else {
			current.Lines = append(current.Lines, line)
		}
	}
	// Save last section
	if len(current.Lines) > 0 {
		sections = append(sections, current)
	}

	return sections, scanner.Err()
}

// parseHeading returns heading level and text, or (0, "") if not a heading
func parseHeading(line string) (int, string) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "#") {
		return 0, ""
	}
	level := 0
	for i := 0; i < len(line) && line[i] == '#'; i++ {
		level++
	}
	// Require at least one space after hashes
	if len(line) > level && line[level] == ' ' {
		return level, strings.TrimSpace(line[level:])
	}
	return 0, ""
}

// gatherPromptData gathers the markdown content from []sections into structured prompt data.
// Returns a PromptData structure containing all parsed prompts organized by sections.
func gatherPromptData(sections []Section) *PromptData {
	return &PromptData{
		Sections: sections,
	}
}

// Helper: match full section path (nested headings)
func searchPoolBySectionPath(data *PromptData, sectionPath []string) []Prompt {
	var searchPool []Prompt
	for _, sec := range data.Sections {
		// Always skip the first heading (Markdown file title)
		if len(sec.Headings) < 2 {
			continue
		}
		// Compare sectionPath to sec.Headings[1:]
		if len(sec.Headings)-1 == len(sectionPath) {
			match := true
			for i := range sectionPath {
				if sec.Headings[i+1] != sectionPath[i] {
					match = false
					break
				}
			}
			if match {
				for _, line := range sec.Lines {
					if strings.TrimSpace(line) != "" {
						searchPool = append(searchPool, Prompt{
							Content: line,
							Section: sec.Headings[len(sec.Headings)-1],
						})
					}
				}
			}
		}
	}
	return searchPool
}

// Helper: match single section name (lowest-level heading)
func searchPoolBySingleSection(data *PromptData, section string) []Prompt {
	var searchPool []Prompt
	for _, sec := range data.Sections {
		if len(sec.Headings) > 0 && sec.Headings[len(sec.Headings)-1] == section {
			for _, line := range sec.Lines {
				if strings.TrimSpace(line) != "" {
					searchPool = append(searchPool, Prompt{
						Content: line,
						Section: section,
					})
				}
			}
		}
	}
	return searchPool
}

// Helper: match single section name (higher-level heading)
func searchPoolByParentSection(data *PromptData, section string) []Prompt {
	var searchPool []Prompt
	for _, sec := range data.Sections {
		if len(sec.Headings) > 1 {
			for i, heading := range sec.Headings[:len(sec.Headings)-1] {
				if heading == section {
					for _, line := range sec.Lines {
						if strings.TrimSpace(line) != "" {
							searchPool = append(searchPool, Prompt{
								Content: line,
								Section: sec.Headings[len(sec.Headings)-1],
							})
						}
					}
					break
				}
				if i == len(sec.Headings)-2 {
					break
				}
			}
		}
	}
	return searchPool
}

// Helper: all prompts (no section specified)
func searchPoolAllPrompts(data *PromptData) []Prompt {
	var searchPool []Prompt
	for _, sec := range data.Sections {
		if len(sec.Headings) > 0 {
			sectionTitle := sec.Headings[len(sec.Headings)-1]
			for _, line := range sec.Lines {
				if strings.TrimSpace(line) != "" {
					searchPool = append(searchPool, Prompt{
						Content: line,
						Section: sectionTitle,
					})
				}
			}
		}
	}
	return searchPool
}

// generateSearchPool creates a slice of Prompt structs for each line in the relevant sections.
// Returns a slice of Prompt structs containing the content and section for each line.
func generateSearchPool(data *PromptData, section string) []Prompt {
	if section == "" {
		// No section specified: return all prompts
		return searchPoolAllPrompts(data)
	}
	sectionPath := strings.Split(section, ",")
	for i := range sectionPath {
		sectionPath[i] = strings.TrimSpace(sectionPath[i])
	}
	if len(sectionPath) > 1 {
		// Comma-separated: treat as nested headings
		return searchPoolBySectionPath(data, sectionPath)
	}
	// Single section name: try lowest-level heading match first
	pool := searchPoolBySingleSection(data, sectionPath[0])
	if len(pool) > 0 {
		return pool
	}
	// If not found, try parent section match
	return searchPoolByParentSection(data, sectionPath[0])
}

// SearchPrompts performs fuzzy search on prompts using the provided query.
// If a section is specified, it searches only within that section.
// If the query is empty, it returns all prompts (or all prompts in the specified section).
// Returns a slice of prompt content strings matching the search criteria.
func SearchPrompts(data *PromptData, query, section string) []string {
	searchPool := generateSearchPool(data, section)
	if len(searchPool) == 0 {
		return []string{}
	}

	if query == "" {
		results := make([]string, len(searchPool))
		for i, p := range searchPool {
			results[i] = p.Content
		}
		return results
	}

	// Split query into individual words for better matching
	queryWords := strings.Fields(strings.ToLower(query))
	if len(queryWords) == 0 {
		return []string{}
	}

	type MatchResult struct {
		Content string
		Score   int // Lower is better (total distance across all words)
		Index   int
	}

	var matches []MatchResult

	// For each prompt in the search pool
	for i, prompt := range searchPool {
		totalDistance := 0
		matchedWords := 0
		content := strings.ToLower(prompt.Content)

		// Check if all query words have reasonable matches in this prompt
		for _, word := range queryWords {
			// First try exact word match
			if strings.Contains(content, word) {
				matchedWords++
				// Give exact matches a very low distance (high priority)
				totalDistance += 1
				continue
			}

			// If no exact match, try fuzzy match on individual word
			wordMatches := fuzzy.RankFindNormalizedFold(word, []string{content})
			if len(wordMatches) > 0 && wordMatches[0].Distance < 100 { // reasonable fuzzy match threshold
				matchedWords++
				totalDistance += wordMatches[0].Distance
			}
		}

		// Only include this prompt if ALL query words were found
		if matchedWords == len(queryWords) {
			matches = append(matches, MatchResult{
				Content: prompt.Content,
				Score:   totalDistance,
				Index:   i,
			})
		}
	}

	// Sort matches by score (lower is better)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score < matches[j].Score
	})

	// Extract just the content
	results := make([]string, len(matches))
	for i, match := range matches {
		results[i] = match.Content
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
	for _, sec := range data.Sections {
		if len(sec.Headings) > 0 && sec.Headings[len(sec.Headings)-1] == section {
			return []string{strings.Join(sec.Lines, "\n")}
		}
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
