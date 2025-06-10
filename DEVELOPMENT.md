# Development

## Project Structure
```
wheresmyprompt/
├── cmd/wheresmyprompt/
│   └── root.go              # CLI command setup
├── internal/prompt/
│   ├── prompt.go            # Core prompt logic
│   └── tui.go              # Bubbletea TUI
├── pkg/
│   ├── config/
│   │   └── config.go       # Environment configuration
│   ├── man/                # Manual pages (from starter)
│   └── version/            # Version info (from starter)
├── main.go
├── go.mod
└── README.md
```

## Key Dependencies

- **Cobra**: CLI framework
- **Viper**: Configuration management
- **Bubbletea**: TUI framework
- **Lipgloss**: TUI styling
- **Fuzzy**: Fuzzy string matching
- **Logrus**: Structured logging

## Building

```bash
# Install dependencies
make local-deps

# Build
make local-build
```

## Testing Integration

```bash
# Test Simplenote connection
wheresmyprompt -d  # Debug mode to see connection details

# Test local file mode
echo "# Test\n## Section\n### Prompt\nTest content" > test.md
FILEPATH=test.md wheresmyprompt
```

## Troubleshooting

### Common Issues

1. **sncli not found**: Install with `pip install sncli`
2. **op not found**: Install 1Password CLI
3. **Authentication failed**: Check SN_USERNAME and SN_PASSWORD
4. **No clipboard utility**: Install xclip/xsel on Linux
5. **Note not found**: Verify SN_NOTE matches your Simplenote note title

### Debug Mode

Enable debug logging to troubleshoot issues:

```bash
wheresmyprompt -d
```

### Manual Authentication

If 1Password integration fails, set credentials directly:

```bash
export SN_USERNAME="your@email.com"
export SN_PASSWORD="your_password"
wheresmyprompt
```

## Roadmap

- [ ] Write functionality (`-w` flag)
- [ ] Section management (create/delete sections)
- [ ] Prompt editing within TUI
- [ ] Export/import functionality
- [ ] Multiple note support
- [ ] Tagging system
- [ ] History/favorites
- [ ] API integration for other note services


## changes required to update golang version
- `make update-golang-version`
