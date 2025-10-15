// Package main provides diagram generation utilities for the wheresmyprompt project.
//
// This application generates architectural and component diagrams for the wheresmyprompt
// application using the go-diagrams library. It creates visual representations of the
// project structure and component relationships to aid in documentation and understanding.
//
// The generated diagrams are saved as .dot files in the docs/diagrams/go-diagrams/
// directory and can be converted to various image formats using Graphviz.
//
// Usage:
//
//	go run cmd/diagrams/main.go
//
// This will generate:
//   - architecture.dot: High-level architecture showing user interaction flow
//   - components.dot: Component relationships and dependencies
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/blushft/go-diagrams/diagram"
	"github.com/blushft/go-diagrams/nodes/generic"
	"github.com/blushft/go-diagrams/nodes/programming"
)

// main is the entry point for the diagram generation utility.
//
// This function orchestrates the entire diagram generation process:
//  1. Creates the output directory structure
//  2. Changes to the appropriate working directory
//  3. Generates architecture and component diagrams
//  4. Reports successful completion
//
// The function will terminate with log.Fatal if any critical operation fails,
// such as directory creation, navigation, or diagram rendering.
func main() {
	// Ensure output directory exists
	if err := os.MkdirAll("docs/diagrams", 0750); err != nil {
		log.Fatal("Failed to create output directory:", err)
	}

	// Change to docs/diagrams directory
	if err := os.Chdir("docs/diagrams"); err != nil {
		log.Fatal("Failed to change directory:", err)
	}

	// Generate architecture diagram
	generateArchitectureDiagram()

	// Generate component diagram
	generateComponentDiagram()

	fmt.Println("Diagram .dot files generated successfully in ./docs/diagrams/go-diagrams/")
}

// generateArchitectureDiagram creates a high-level architecture diagram showing
// the interaction flow between users and the wheresmyprompt application components.
//
// The diagram illustrates:
//   - User interaction with the CLI application
//   - Configuration management flow
//   - Integration with prompt processing and TUI components
//   - External integrations (Simplenote, clipboard)
//
// The diagram is rendered in top-to-bottom (TB) direction and saved as
// "architecture.dot" in the current working directory. The function will
// terminate the program with log.Fatal if diagram creation or rendering fails.
func generateArchitectureDiagram() {
	d, err := diagram.New(diagram.Filename("architecture"), diagram.Label("WheresMyPrompt Architecture"), diagram.Direction("TB"))
	if err != nil {
		log.Fatal(err)
	}

	// Define components
	user := generic.Blank.Blank(diagram.NodeLabel("User"))
	cli := programming.Language.Go(diagram.NodeLabel("CLI Application"))
	config := generic.Blank.Blank(diagram.NodeLabel("Configuration\n(env/godotenv)"))
	prompt := programming.Language.Go(diagram.NodeLabel("Prompt Processing"))
	tui := programming.Language.Go(diagram.NodeLabel("TUI Interface\n(Bubbletea)"))
	simplenote := generic.Blank.Blank(diagram.NodeLabel("Simplenote\nIntegration"))
	clipboard := generic.Blank.Blank(diagram.NodeLabel("Clipboard\nOperations"))
	logging := generic.Blank.Blank(diagram.NodeLabel("Logging\n(logrus)"))

	// Create connections
	d.Connect(user, cli, diagram.Forward())
	d.Connect(cli, config, diagram.Forward())
	d.Connect(cli, prompt, diagram.Forward())
	d.Connect(cli, tui, diagram.Forward())
	d.Connect(prompt, simplenote, diagram.Forward())
	d.Connect(prompt, clipboard, diagram.Forward())
	d.Connect(cli, logging, diagram.Forward())

	if err := d.Render(); err != nil {
		log.Fatal(err)
	}
}

// generateComponentDiagram creates a detailed component diagram showing the
// relationships and dependencies between different packages in the wheresmyprompt project.
//
// The diagram illustrates:
//   - main.go as the entry point
//   - cmd/wheresmyprompt package handling CLI operations
//   - Integration with configuration, prompt processing, TUI, and utility packages
//   - Data flow between components
//
// The diagram is rendered in left-to-right (LR) direction and saved as
// "components.dot" in the current working directory. The function will
// terminate the program with log.Fatal if diagram creation or rendering fails.
func generateComponentDiagram() {
	d, err := diagram.New(diagram.Filename("components"), diagram.Label("WheresMyPrompt Components"), diagram.Direction("LR"))
	if err != nil {
		log.Fatal(err)
	}

	// Main components
	main := programming.Language.Go(diagram.NodeLabel("main.go"))
	rootCmd := programming.Language.Go(diagram.NodeLabel("cmd/wheresmyprompt\nroot.go"))
	config := programming.Language.Go(diagram.NodeLabel("pkg/config\nconfig.go"))
	prompt := programming.Language.Go(diagram.NodeLabel("internal/prompt\nprompt.go"))
	tui := programming.Language.Go(diagram.NodeLabel("internal/tui\ntui.go"))
	version := programming.Language.Go(diagram.NodeLabel("pkg/version\nversion.go"))
	man := programming.Language.Go(diagram.NodeLabel("pkg/man\nman.go"))
	languaged := programming.Language.Go(diagram.NodeLabel("pkg/languaged\nlanguaged.go"))

	// Create connections showing the flow
	d.Connect(main, rootCmd, diagram.Forward())
	d.Connect(rootCmd, config, diagram.Forward())
	d.Connect(rootCmd, prompt, diagram.Forward())
	d.Connect(rootCmd, tui, diagram.Forward())
	d.Connect(rootCmd, version, diagram.Forward())
	d.Connect(rootCmd, man, diagram.Forward())
	d.Connect(rootCmd, languaged, diagram.Forward())

	if err := d.Render(); err != nil {
		log.Fatal(err)
	}
}
