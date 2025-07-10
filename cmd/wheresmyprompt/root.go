// Package cmd provides command-line interface functionality for the wheresmyprompt application.
//
// This package implements the root command and manages the command-line interface
// using the cobra library. It handles configuration, logging setup, and command
// execution for the wheresmyprompt prompt management application.
//
// The package integrates with several components:
//   - Configuration management through pkg/config
//   - Prompt processing through internal/prompt
//   - TUI interface through internal/tui
//   - Manual pages through pkg/man
//   - Version information through pkg/version
//   - Language detection through pkg/languaged
//
// Key features:
//   - Fuzzy search and management of LLM prompts
//   - Support for both Simplenote and local file sources
//   - Interactive TUI mode and CLI mode operation
//   - Clipboard integration for prompt copying
//   - Section-based prompt organization
//   - Debug logging support
//
// Example usage:
//
//	import "github.com/toozej/wheresmyprompt/cmd/wheresmyprompt"
//
//	func main() {
//		cmd.Execute()
//	}
package cmd

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/toozej/wheresmyprompt/internal/prompt"
	"github.com/toozej/wheresmyprompt/internal/tui"
	"github.com/toozej/wheresmyprompt/pkg/config"
	"github.com/toozej/wheresmyprompt/pkg/languaged"
	"github.com/toozej/wheresmyprompt/pkg/man"
	"github.com/toozej/wheresmyprompt/pkg/version"
)

// conf holds the application configuration loaded from environment variables.
// It is populated during package initialization and can be modified by command-line flags.
var (
	conf config.Config
	// debug controls the logging level for the application.
	// When true, debug-level logging is enabled through logrus.
	debug bool
	// Command-line flags
	all         bool
	oneShot     bool
	oneShotClip bool
	section     string
	write       string
	load        string
)

var rootCmd = &cobra.Command{
	Use:              "wheresmyprompt",
	Short:            "Fuzzy search and manage LLM prompts from Markdown/Simplenote",
	Long:             `A tool to fuzzy search, manage, and copy LLM prompts from a Markdown or Simplenote note`,
	Args:             cobra.ArbitraryArgs,
	PersistentPreRun: rootCmdPreRun,
	Run:              rootCmdRun,
}

func rootCmdRun(cmd *cobra.Command, args []string) {
	// Check for required binaries
	if err := prompt.CheckRequiredBinaries(conf); err != nil {
		log.Fatal(err)
	}

	// Handle loading prompts from a local file, preferring command line flag over environment variable
	if (conf.FilePath == "" && load != "") || (conf.FilePath != "" && load != "") {
		conf.FilePath = load
	}

	// Handle write mode (adding new prompt)
	if write != "" {
		if err := prompt.WritePrompt(conf, write, args); err != nil {
			log.Fatal(err)
		}
		return
	}

	// Load prompts
	prompts, err := prompt.LoadPrompts(conf)
	if err != nil {
		log.Fatal(err)
	}

	// Determine section to use: command-line flag or detected language
	sectionToUse := section
	// However do not auto-detect the section if --all is specified
	// because that would be confusing (user might expect all sections to be searched).
	if sectionToUse == "" && !all {
		if cwd, err := os.Getwd(); err == nil {
			lang, err := languaged.DetectPrimaryLanguage(cwd)
			if err == nil && lang != "" {
				sectionToUse = lang
			}
		}
	}
	fmt.Println("Using section:", sectionToUse)

	// Handle --all mode
	if all {
		if len(args) == 0 {
			log.Fatal("--all mode requires a search term")
		}
		results := prompt.FindAllMatches(prompts, args[0], sectionToUse)
		if len(results) == 0 {
			fmt.Println("No matches found")
			os.Exit(1)
		}
		for _, p := range results {
			fmt.Printf("\n%s\n\n", p)
		}
		return
	}

	// Handle one-shot mode
	if oneShot {
		query := ""
		if len(args) > 0 {
			query = args[0]
		}
		result := prompt.FindBestMatch(prompts, query, sectionToUse)
		if result == "" {
			fmt.Println("No match found")
			os.Exit(1)
		}
		fmt.Printf("\n%s\n\n", result)
		return
	}

	// Handle one-shot-clip mode
	if oneShotClip {
		query := ""
		if len(args) > 0 {
			query = args[0]
		}
		result := prompt.FindBestMatch(prompts, query, sectionToUse)
		if result == "" {
			fmt.Println("No match found")
			os.Exit(1)
		}
		if err := prompt.CopyToClipboard(result); err != nil {
			log.Fatal("Failed to copy to clipboard: ", err)
		}
		return
	}

	// Handle section listing
	if section := sectionToUse; section != "" && len(args) == 0 {
		results := prompt.GetSectionPrompts(prompts, section)
		for _, p := range results {
			fmt.Printf("\n%s\n\n", p)
		}
		return
	}

	// Handle CLI mode (any flags specified)
	if cmd.Flags().NFlag() > 0 || len(args) > 0 {
		// CLI mode - search and output to stdout
		searchTerm := ""
		if len(args) > 0 {
			searchTerm = args[0]
		}
		results := prompt.SearchPrompts(prompts, searchTerm, sectionToUse)
		for _, p := range results {
			fmt.Printf("\n%s\n\n", p)
		}
		return
	}

	// Default: TUI mode
	if err := tui.RunTUI(prompts, conf); err != nil {
		log.Fatal(err)
	}
}

func rootCmdPreRun(cmd *cobra.Command, args []string) {
	if debug {
		log.SetLevel(log.DebugLevel)
	}
}

// Execute runs the root command and handles any execution errors.
// This is the main entry point for the CLI application.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func init() {
	// Get configuration from environment variables
	conf = config.GetEnvVars()

	// Create rootCmd-level flags
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug-level logging")
	rootCmd.Flags().BoolVarP(&all, "all", "a", false, "Show all fuzzy matches for the search term")
	rootCmd.Flags().BoolVarP(&oneShot, "one-shot", "o", false, "Select best match and print to stdout")
	rootCmd.Flags().BoolVarP(&oneShotClip, "one-shot-clip", "c", false, "Select best match and copy to clipboard")
	rootCmd.Flags().StringVarP(&section, "section", "s", "", "Search within specific section")
	rootCmd.Flags().StringVarP(&write, "write", "w", "", "Add new prompt to note")
	rootCmd.Flags().StringVarP(&load, "load", "l", "", "Load a local file of prompts instead of from Simplenote")

	// Add sub-commands
	rootCmd.AddCommand(
		man.NewManCmd(),
		version.Command(),
	)
}
