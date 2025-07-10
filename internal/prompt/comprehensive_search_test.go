package prompt

import (
	"strings"
	"testing"
)

func TestComprehensiveSearch(t *testing.T) {
	content := `# Test Prompts

## documentation

Document each function and package with comments and an overview / purpose for each using the standard methodology for the language (godoc, Python docstring, etc.)

Write extensive usage documentation in Markdown including realistic examples.

Generate a README.md file a repository containing the following code. It should contain an introductory description, usage instructions, installation instructions, and a summary of its major functionality.

Generate a DEVELOPMENT.md file for this repository. It should contain instructions on how to develop the code, tools and technologies used in the code, etc.

Generate a diagram using PlantUML and outputting a SVG graphic file which describes the overview of the application and how to use it. Embed this SVG graphic in the README.md as well under the introductory description section. The PlantUML code used to generate the SVG graphic should also be outputted such that it can live alongside the SVG graphic file in the repository under the directory docs/.

## coding

Create a function that handles authentication

Write unit tests for all functions

Implement error handling throughout the application
`

	sections, err := parseMarkdownIntoSections(content)
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}
	data := gatherPromptData(sections)

	tests := []struct {
		name        string
		query       string
		section     string
		expectCount int
		expectFirst string // partial match for first result
	}{
		{
			name:        "Multi-word exact match",
			query:       "standard methodology",
			section:     "documentation",
			expectCount: 1,
			expectFirst: "Document each function and package",
		},
		{
			name:        "Single word match",
			query:       "authentication",
			section:     "coding",
			expectCount: 1,
			expectFirst: "Create a function that handles authentication",
		},
		{
			name:        "Partial word fuzzy match",
			query:       "func handles",
			section:     "coding",
			expectCount: 1,
			expectFirst: "Create a function that handles authentication",
		},
		{
			name:        "Word not in any prompt",
			query:       "nonexistent",
			section:     "documentation",
			expectCount: 0,
			expectFirst: "",
		},
		{
			name:        "Multi-word where one word doesn't match",
			query:       "standard nonexistent",
			section:     "documentation",
			expectCount: 0,
			expectFirst: "",
		},
		{
			name:        "Case insensitive match",
			query:       "AUTHENTICATION",
			section:     "coding",
			expectCount: 1,
			expectFirst: "Create a function that handles authentication",
		},
		{
			name:        "Fuzzy match with close word",
			query:       "function auth", // close words that exist in the text
			section:     "coding",
			expectCount: 1,
			expectFirst: "Create a function that handles authentication",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := SearchPrompts(data, tt.query, tt.section)

			if len(results) != tt.expectCount {
				t.Errorf("Expected %d results, got %d", tt.expectCount, len(results))
				for i, result := range results {
					t.Logf("Result %d: %s", i+1, result)
				}
			}

			if tt.expectCount > 0 && len(results) > 0 {
				if !strings.Contains(results[0], tt.expectFirst) {
					t.Errorf("Expected first result to contain '%s', got: %s", tt.expectFirst, results[0])
				}
			}
		})
	}
}
