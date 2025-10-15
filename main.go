// Package main provides the entry point for the wheresmyprompt CLI application.
// wheresmyprompt is a tool to fuzzy search, manage, and copy LLM prompts from Markdown or Simplenote notes.
package main

import cmd "github.com/toozej/wheresmyprompt/cmd/wheresmyprompt"

func main() {
	cmd.Execute()
}
