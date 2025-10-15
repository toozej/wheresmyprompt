package prompt

import (
	"strings"
	"testing"
)

func TestFuzzySearchIssue(t *testing.T) {
	// Create test markdown content similar to the user's issue
	content := `# Test Prompts

## documentation

Document each function and package with comments and an overview / purpose for each using the standard methodology for the language (godoc, Python docstring, etc.)

Write extensive usage documentation in Markdown including realistic examples.

Generate a README.md file a repository containing the following code. It should contain an introductory description, usage instructions, installation instructions, and a summary of its major functionality.

Generate a DEVELOPMENT.md file for this repository. It should contain instructions on how to develop the code, tools and technologies used in the code, etc.

Generate a diagram using PlantUML and outputting a SVG graphic file which describes the overview of the application and how to use it. Embed this SVG graphic in the README.md as well under the introductory description section. The PlantUML code used to generate the SVG graphic should also be outputted such that it can live alongside the SVG graphic file in the repository under the directory docs/.
`

	// Parse content
	sections, err := parseMarkdownIntoSections(content)
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	data := gatherPromptData(sections)

	// Search for "standard methodology" in "documentation" section
	results := SearchPrompts(data, "standard methodology", "documentation")

	t.Logf("Number of results: %d", len(results))
	for i, result := range results {
		t.Logf("Result %d: %s", i+1, strings.TrimSpace(result))
		t.Logf("---")
	}

	// The issue: we expect only 1 result (the first prompt that contains "standard methodology")
	// But we're getting 2 results, including the PlantUML prompt which doesn't contain these words

	// Check that only the first prompt is returned
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// Check that the returned result contains both "standard" and "methodology"
	if len(results) > 0 {
		result := strings.ToLower(results[0])
		if !strings.Contains(result, "standard") || !strings.Contains(result, "methodology") {
			t.Errorf("First result should contain both 'standard' and 'methodology': %s", results[0])
		}
	}
}
