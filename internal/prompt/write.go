// Package write provides functionality for adding new prompts to notes.
// It supports both local Markdown files and Simplenote integration,
// with automatic title generation and section organization.
package prompt

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/toozej/wheresmyprompt/pkg/config"
)

// Allow test overrides
var loadFromSimplenoteFunc = loadFromSimplenote
var ensureSimplenoteAuthFunc = ensureSimplenoteAuth

// WritePrompt adds a new prompt to the configured note source.
// It can handle prompts provided via command line arguments, flags, or interactive input.
// The prompt is automatically organized into sections and formatted according to the
// established Markdown structure. For Simplenote integration, it updates the remote note.
// Returns an error if the write operation fails.
func WritePrompt(conf config.Config, promptContent string, args []string) error {
	// Determine the prompt title and content
	var title, content string

	switch {
	case promptContent != "":
		// Content provided via -w flag
		title = generateTitleFromContent(promptContent)
		content = promptContent
	case len(args) > 0:
		// Content provided as arguments
		content = strings.Join(args, " ")
		title = generateTitleFromContent(content)
	default:
		// Read from stdin
		fmt.Print("Enter prompt title: ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		title = scanner.Text()

		fmt.Print("Enter prompt content (press Ctrl+D when done):\n")
		var contentLines []string
		for scanner.Scan() {
			contentLines = append(contentLines, scanner.Text())
		}
		content = strings.Join(contentLines, "\n")
	}

	if title == "" || content == "" {
		return fmt.Errorf("both title and content are required")
	}

	// Get section from command line or prompt user
	section := ""
	if len(args) > 1 {
		section = args[1] // Second argument could be section
	}

	if section == "" {
		fmt.Print("Enter section (optional, press Enter to skip): ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		section = strings.TrimSpace(scanner.Text())
	}

	return addPromptToNote(conf, title, content, section)
}

// generateTitleFromContent creates a title from the first few words of content
func generateTitleFromContent(content string) string {
	words := strings.Fields(content)
	if len(words) == 0 {
		return "Untitled Prompt"
	}

	// Take first 5 words or less
	maxWords := 5
	if len(words) < maxWords {
		maxWords = len(words)
	}

	title := strings.Join(words[:maxWords], " ")

	// Capitalize first letter
	if len(title) > 0 {
		title = strings.ToUpper(string(title[0])) + title[1:]
	}

	// Remove trailing punctuation
	title = strings.TrimRight(title, ".,!?;:")

	return title
}

// addPromptToNote adds the new prompt to the Simplenote note
func addPromptToNote(conf config.Config, title, content, section string) error {
	if conf.FilePath != "" {
		return addPromptToFile(conf.FilePath, title, content, section)
	}
	return addPromptToSimplenote(conf, title, content, section)
}

// addPromptToFile adds the prompt to a local markdown file
func addPromptToFile(filepath, title, content, section string) error {
	// Read existing content
	existingContent := ""
	data, err := os.ReadFile(filepath) // #nosec G304
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
	return os.WriteFile(filepath, []byte(newContent.String()), 0600)
}

// writeSectionHeader writes the markdown header for a section
func writeSectionHeader(b *strings.Builder, sec Section) {
	for i, heading := range sec.Headings {
		b.WriteString(strings.Repeat("#", i+1) + " " + heading + "\n")
	}
}

// writeSection writes the full section (header + lines)
func writeSection(b *strings.Builder, sec Section) {
	writeSectionHeader(b, sec)
	for _, line := range sec.Lines {
		b.WriteString(line + "\n")
	}
}

// addPromptToSimplenote adds the prompt to the Simplenote note
func addPromptToSimplenote(conf config.Config, title, content, section string) error {
	// First, ensure authentication
	if err := ensureSimplenoteAuthFunc(conf); err != nil {
		return err
	}

	// Get current note content
	currentContent, err := loadFromSimplenoteFunc(conf)
	if err != nil {
		return fmt.Errorf("failed to load current note: %w", err)
	}

	// Create updated content
	var newContent strings.Builder
	newContent.WriteString(currentContent)

	if section != "" {
		// Try to add to existing section
		if !addToExistingSection(&newContent, currentContent, title, content, section) {
			// Section doesn't exist, create it
			if !strings.HasSuffix(currentContent, "\n") {
				newContent.WriteString("\n")
			}
			newContent.WriteString("\n## " + section + "\n\n")
			newContent.WriteString("### " + title + "\n")
			newContent.WriteString(content + "\n")
		}
	} else {
		// Add at the end without section
		if !strings.HasSuffix(currentContent, "\n") {
			newContent.WriteString("\n")
		}
		newContent.WriteString("\n### " + title + "\n")
		newContent.WriteString(content + "\n")
	}

	// Prepare JSON note for import
	note := map[string]interface{}{
		"tags":             []string{},
		"deleted":          false,
		"shareURL":         "",
		"publishURL":       "",
		"content":          newContent.String(),
		"systemTags":       []string{},
		"modificationDate": float64(time.Now().Unix()),
		"creationDate":     float64(time.Now().Unix()),
		"key":              conf.SNNote,
		"version":          1,
		"syncdate":         float64(time.Now().Unix()),
		"localkey":         conf.SNNote,
		"savedate":         float64(time.Now().Unix()),
	}
	notes := []interface{}{note}
	jsonBytes, err := json.Marshal(notes)
	if err != nil {
		return fmt.Errorf("failed to marshal note JSON: %w", err)
	}

	// Import the note using sncli import -
	cmd := exec.Command("sncli", "import", "-") // #nosec G204
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}
	go func() {
		defer stdin.Close()
		// nosemgrep: go.lang.security.audit.dangerous-command-write.dangerous-command-write
		_, _ = stdin.Write(jsonBytes)
	}()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to import note to Simplenote: %w", err)
	}

	fmt.Printf("Successfully added prompt '%s' to note '%s'\n", title, conf.SNNote)
	if section != "" {
		fmt.Printf("Section: %s\n", section)
	}

	return nil
}

// addToExistingSection tries to add the prompt to an existing section
func addToExistingSection(newContent *strings.Builder, currentContent, title, content, section string) bool {
	lines := strings.Split(currentContent, "\n")
	sectionHeader := "## " + section

	// Reset the builder and rebuild with the new prompt
	newContent.Reset()

	for i, line := range lines {
		if strings.TrimSpace(line) == sectionHeader {
			// Found the section, add all lines up to here
			for j := 0; j <= i; j++ {
				newContent.WriteString(lines[j] + "\n")
			}

			// Find the end of this section
			k := i + 1
			for k < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[k]), "## ") {
				newContent.WriteString(lines[k] + "\n")
				k++
			}

			// Add the new prompt
			newContent.WriteString("\n### " + title + "\n")
			newContent.WriteString(content + "\n")

			// Add remaining sections
			for j := k; j < len(lines); j++ {
				newContent.WriteString(lines[j] + "\n")
			}

			return true
		}
	}

	return false
}
