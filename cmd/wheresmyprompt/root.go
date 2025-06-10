// Package cmd provides the command-line interface for wheresmyprompt.
// It handles argument parsing, configuration, and orchestrates the main application logic
// for searching, managing, and copying LLM prompts.
package cmd

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/automaxprocs/maxprocs"

	"github.com/toozej/wheresmyprompt/internal/prompt"
	"github.com/toozej/wheresmyprompt/internal/tui"
	"github.com/toozej/wheresmyprompt/pkg/config"
	"github.com/toozej/wheresmyprompt/pkg/man"
	"github.com/toozej/wheresmyprompt/pkg/version"
)

var conf config.Config

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
	if (conf.FilePath == "" && viper.GetString("load") != "") || (conf.FilePath != "" && viper.GetString("load") != "") {
		conf.FilePath = viper.GetString("load")
	}

	// Handle write mode (adding new prompt)
	if writePrompt := viper.GetString("write"); writePrompt != "" {
		if err := prompt.WritePrompt(conf, writePrompt, args); err != nil {
			log.Fatal(err)
		}
		return
	}

	// Load prompts
	prompts, err := prompt.LoadPrompts(conf)
	if err != nil {
		log.Fatal(err)
	}

	// Handle --all mode
	if viper.GetBool("all") {
		if len(args) == 0 {
			log.Fatal("--all mode requires a search term")
		}
		results := prompt.FindAllMatches(prompts, args[0], viper.GetString("section"))
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
	if viper.GetBool("one-shot") {
		if len(args) == 0 {
			log.Fatal("one-shot mode requires a search term")
		}
		result := prompt.FindBestMatch(prompts, args[0], viper.GetString("section"))
		if result == "" {
			fmt.Println("No match found")
			os.Exit(1)
		}
		fmt.Printf("\n%s\n\n", result)
		return
	}

	// Handle one-shot-clip mode
	if viper.GetBool("one-shot-clip") {
		if len(args) == 0 {
			log.Fatal("one-shot-clip mode requires a search term")
		}
		result := prompt.FindBestMatch(prompts, args[0], viper.GetString("section"))
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
	if section := viper.GetString("section"); section != "" && len(args) == 0 {
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
		results := prompt.SearchPrompts(prompts, searchTerm, viper.GetString("section"))
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
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return
	}
	if viper.GetBool("debug") {
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
	_, err := maxprocs.Set()
	if err != nil {
		log.Error("Error setting maxprocs: ", err)
	}

	// Get configuration from environment variables
	conf = config.GetEnvVars()

	// Create rootCmd-level flags
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug-level logging")
	rootCmd.Flags().BoolP("all", "a", false, "Show all fuzzy matches for the search term")
	rootCmd.Flags().BoolP("one-shot", "o", false, "Select best match and print to stdout")
	rootCmd.Flags().BoolP("one-shot-clip", "c", false, "Select best match and copy to clipboard")
	rootCmd.Flags().StringP("section", "s", "", "Search within specific section")
	rootCmd.Flags().StringP("write", "w", "", "Add new prompt to note")
	rootCmd.Flags().StringP("load", "l", "", "Load a local file of prompts instead of from Simplenote")

	// Add sub-commands
	rootCmd.AddCommand(
		man.NewManCmd(),
		version.Command(),
	)
}
