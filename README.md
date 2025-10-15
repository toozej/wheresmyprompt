# wheresmyprompt

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/toozej/wheresmyprompt)
[![Go Report Card](https://goreportcard.com/badge/github.com/toozej/wheresmyprompt)](https://goreportcard.com/report/github.com/toozej/wheresmyprompt)
![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/toozej/wheresmyprompt/cicd.yaml)
![Docker Pulls](https://img.shields.io/docker/pulls/toozej/wheresmyprompt)
![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/toozej/wheresmyprompt/total)

![Screenshot](img/avatar.webp)

- pronounced with a Shrek accent
- search prompts stored in a Markdown file (optionally grabbed from Simplenote Note), and use selected prompt as a LLM system prompt
- designed to be used in a LLM tool pipeline
	- [files2prompt](github.com/toozej/files2prompt) or [files-to-prompt](github.com/simonw/files-to-prompt) to gather workspace
	- [wheresmyprompt](github.com/toozej/wheresmyprompt) to set a system prompt
	- [llm](github.com/simonw/llm) to operate with local or remote large language models
   - [waffles](https://github.com/toozej/waffles) to orchestrate the above tools in an easy-to-use pipeline
- cute mascot named ["Gogre"](img/avatar.webp)

## 🚀 Features

- **TUI Mode**: Interactive fuzzy search with clipboard integration
- **CLI Mode**: Command-line search with stdout output
- **Simplenote Integration**: Fetch prompts from your "LLM Prompts" note
- **Local File Support**: Work with local markdown files
- **Section Support**: Organize and search within prompt sections
- **Section Auto-Detection**: If the `--section` flag is not provided, wheresmyprompt will automatically detect the primary programming language of your current directory and use it as the section
- **Cross-platform Clipboard**: Automatic clipboard integration

## 🛠️ Prerequisites

### Required Binaries

1. **1Password CLI (`op`)**: For secure credential management
   ```bash
   # macOS
   brew install 1password-cli
   
   # Linux/Windows: Download from https://developer.1password.com/docs/cli/get-started/
   ```

2. **Simplenote CLI (`sncli`)**: For Simplenote integration
   ```bash
   pip install sncli
   ```

3. (Or optionally, run `make local-deps` to install above dependencies)

### Environment Variables

Set these via 1Password CLI or directly:

```bash
export SN_NOTE="LLM Prompts"           # Note title in Simplenote
export SN_USERNAME="your@email.com"   # Simplenote username
export SN_PASSWORD="your_password"    # Simplenote password
export FILEPATH="/path/to/local.md"   # Optional: use local file instead
```

## 📦 Installation

```bash
git clone https://github.com/toozej/wheresmyprompt
cd wheresmyprompt
make install
```

## 🖥️ Usage

### TUI Mode (Default)

Interactive fuzzy search interface:

```bash
wheresmyprompt
```

- Type to search prompts
- Use ↑/↓ or k/j to navigate
- Press Enter to copy selected prompt to clipboard
- Press Ctrl+C or Esc to quit

### CLI Mode

#### Search and display all prompts:
```bash
wheresmyprompt -s golang
# or simply
wheresmyprompt
# (auto-detects section based on repo language if -s/--section is not specified)
```

#### One-shot mode (best match to stdout):
```bash
wheresmyprompt -o "code review"
```

#### Search within specific section:
```bash
wheresmyprompt -s golang "error handling"
# or just
wheresmyprompt "error handling"
# (auto-detects section based on repo language)
```

#### Add new prompt (planned feature):
```bash
wheresmyprompt -w "Write unit tests for this Go function"
```

## 📝 Note Format

Your Simplenote "LLM Prompts" note should be structured like this:

```markdown
# LLM Prompts

## Golang

### Code Review Prompt
Review this Go code for best practices, potential bugs, and performance issues. Pay attention to error handling, memory usage, and concurrency patterns.

### Unit Test Generator
Generate comprehensive unit tests for the following Go function. Include edge cases, error conditions, and table-driven tests where appropriate.

## Python

### Data Analysis Helper
Analyze this dataset and provide insights. Create visualizations using matplotlib or seaborn, and suggest statistical tests if applicable.

### Code Optimization
Optimize this Python code for better performance. Consider algorithmic improvements, memory usage, and Pythonic patterns.

## Writing

### Technical Documentation
Write clear, comprehensive documentation for this technical concept. Include examples, use cases, and common pitfalls.

### Email Templates
Draft a professional email for [specific situation]. Keep it concise, clear, and actionable.
```

## ⚙️ Configuration Options

### Environment Variables

- `SN_NOTE`: Simplenote note title (default: "LLM Prompts")
- `SN_CREDENTIAL`: Your Simplenote credential 1password item
- `SN_USERNAME`: Your Simplenote username, or 1password username field name
- `SN_PASSWORD`: Your Simplenote password, or 1password password field name
- `FILEPATH`: Path to local markdown file (skips Simplenote if set)

### 1Password Integration

Store credentials securely in 1Password and populate environment variables:

```bash
# Set up 1Password CLI authentication
eval $(op signin)

# Use 1Password references in your shell profile
vim .env
# set SN_CREDENTIAL, SN_USERNAME, and SN_PASSWORD environment variables
```

## 🏷️ Command Line Flags

- `-d, --debug`: Enable debug logging
- `-o, --one-shot`: Select best match and print to stdout
- `-s, --section`: Search within specific section (optional; auto-detected based off current working directory's primary programming language if not set)
- `-w, --write`: Add new prompt to note (planned)

## 💡 Examples

### TUI Search
```bash
# Launch interactive search
wheresmyprompt

# Search and navigate with keyboard
# - Type "golang error" to filter
# - Use arrows to select
# - Press Enter to copy to clipboard
```

### CLI Search
```bash
# Show all Golang prompts
wheresmyprompt -s golang

# Find best match for "unit test"
wheresmyprompt -o "unit test"

# Search for "review" in Python section
wheresmyprompt -s python "review"
```

### File-based Usage
```bash
# Use local markdown file instead of Simplenote
export FILEPATH="./my-prompts.md"
wheresmyprompt
```

## 🖥️ Supported Platforms

- **macOS**: Uses `pbcopy` for clipboard
- **Linux**: Uses `xclip` or `xsel` for clipboard
- **Windows**: Uses `clip` for clipboard