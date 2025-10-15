// Package languaged provides programming language detection functionality for the wheresmyprompt application.
//
// This package analyzes repository contents to automatically detect the primary programming
// language being used. It supports detection through multiple methods including file
// extensions, shebang lines, and .gitattributes linguist-language overrides.
//
// The detection process:
//  1. Scans all files in the repository directory tree
//  2. Identifies languages using file extensions and shebang analysis
//  3. Respects .gitattributes linguist-language overrides
//  4. Counts lines of code per language
//  5. Returns the language with the most lines of code
//
// Supported languages include:
//   - Go, Python, JavaScript, TypeScript, Java, C/C++, C#
//   - Ruby, PHP, Rust, Swift, Kotlin, Objective-C, Scala
//   - Shell scripts, Lua, Haskell, HTML, CSS, and more
//
// Example usage:
//
//	import "github.com/toozej/wheresmyprompt/pkg/languaged"
//
//	// Detect primary language in current directory
//	lang, err := languaged.DetectPrimaryLanguage(".")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Primary language: %s\n", lang)
package languaged

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// extensionToLanguage maps file extensions to programming languages.
var extensionToLanguage = map[string]string{
	".go":    "Golang",
	".py":    "Python",
	".js":    "JavaScript",
	".ts":    "TypeScript",
	".java":  "Java",
	".c":     "C",
	".cpp":   "C++",
	".cs":    "C#",
	".rb":    "Ruby",
	".php":   "PHP",
	".rs":    "Rust",
	".swift": "Swift",
	".kt":    "Kotlin",
	".m":     "Objective-C",
	".scala": "Scala",
	".sh":    "Shell",
	".lua":   "Lua",
	".hs":    "Haskell",
	".html":  "HTML",
	".css":   "CSS",
}

// shebangToLanguage maps common shebang interpreters to languages.
var shebangToLanguage = map[string]string{
	"python":  "Python",
	"python3": "Python",
	"python2": "Python",
	"bash":    "Shell",
	"sh":      "Shell",
	"ruby":    "Ruby",
	"node":    "JavaScript",
	"perl":    "Perl",
	"php":     "PHP",
	"lua":     "Lua",
}

// DetectPrimaryLanguage analyzes a repository directory and returns its primary programming language.
//
// This function performs comprehensive language detection by:
//  1. Walking the entire directory tree starting from repoPath
//  2. Identifying file languages using extensions and shebang analysis
//  3. Respecting .gitattributes linguist-language overrides
//  4. Counting lines of code for each detected language
//  5. Returning the language with the highest line count
//
// The function skips common non-source directories (.git, vendor, node_modules)
// and hidden directories to focus on actual source code. Files that cannot be
// identified or read are silently skipped.
//
// Parameters:
//   - repoPath: Path to the repository root directory to analyze
//
// Returns:
//   - string: Name of the primary language (e.g., "Go", "Python", "JavaScript")
//   - error: Error if directory cannot be accessed or walked
//
// Special cases:
//   - Returns "Unknown" if no recognizable source files are found
//   - Empty repositories return "Unknown" without error
//   - Unreadable files are skipped without causing errors
//
// Example:
//
//	// Detect language in current directory
//	lang, err := DetectPrimaryLanguage(".")
//	if err != nil {
//		return fmt.Errorf("language detection failed: %w", err)
//	}
//
//	// Use detected language for section filtering
//	if lang != "Unknown" {
//		fmt.Printf("Detected %s project\n", lang)
//	}
func DetectPrimaryLanguage(repoPath string) (string, error) {
	languageLineCounts := make(map[string]int)

	// Load linguist-language overrides from .gitattributes
	overrides, _ := parseGitattributes(filepath.Join(repoPath, ".gitattributes"))

	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable
		}
		relPath, _ := filepath.Rel(repoPath, path)

		// Skip directories like .git, vendor, node_modules
		if info.IsDir() {
			base := info.Name()
			if strings.HasPrefix(base, ".") || base == "vendor" || base == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		var lang string

		// Check if this file is overridden in .gitattributes
		if overrideLang, ok := overrides[relPath]; ok {
			lang = overrideLang
		} else {
			ext := strings.ToLower(filepath.Ext(info.Name()))
			if knownLang, ok := extensionToLanguage[ext]; ok {
				lang = knownLang
			} else {
				// Try detect by shebang
				shebangLang, err := detectLanguageByShebang(path)
				if err == nil && shebangLang != "" {
					lang = shebangLang
				} else {
					return nil // skip unknown
				}
			}
		}

		// Count lines
		lineCount, err := countLines(path)
		if err != nil {
			return nil // skip unreadable
		}
		languageLineCounts[lang] += lineCount
		return nil
	})
	if err != nil {
		return "", err
	}

	// Find language with most lines
	var primaryLang string
	maxLines := 0
	for lang, count := range languageLineCounts {
		if count > maxLines {
			primaryLang = lang
			maxLines = count
		}
	}

	if primaryLang == "" {
		return "Unknown", nil
	}
	return primaryLang, nil
}

// parseGitattributes parses .gitattributes for linguist-language overrides.
func parseGitattributes(path string) (map[string]string, error) {
	overrides := make(map[string]string)

	file, err := os.Open(path) // #nosec G304
	if err != nil {
		return overrides, nil // no .gitattributes is fine
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	linguistRe := regexp.MustCompile(`linguist-language=([^\s]+)`)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			pattern := parts[0]
			for _, attr := range parts[1:] {
				if matches := linguistRe.FindStringSubmatch(attr); len(matches) == 2 {
					// For simplicity, store exact file names
					// Real gitattributes can use globs, but we keep it simple here
					cleanPattern := strings.TrimPrefix(pattern, "/")
					overrides[cleanPattern] = matches[1]
				}
			}
		}
	}
	return overrides, nil
}

// detectLanguageByShebang reads first line and returns detected language.
func detectLanguageByShebang(path string) (string, error) {
	f, err := os.Open(path) // #nosec G304
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#!") {
			for key, lang := range shebangToLanguage {
				if strings.Contains(line, key) {
					return lang, nil
				}
			}
		}
	}
	return "", nil
}

// countLines counts the number of lines in a file.
func countLines(path string) (int, error) {
	f, err := os.Open(path) // #nosec G304
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
	}
	return count, nil
}
