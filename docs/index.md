# owui CLI

A fast and flexible CLI written in Go for managing multiple Open WebUI instances.

## Quick Start

```bash
# Install
go install github.com/christestet/owui-go/cmd/owui@latest

# Add an instance
owui instances add

# List users
owui users list

# List and manage models
owui models list
owui models show claude-sonnet

# Check instance health
owui instances health
```

## Commands

| Command | Description |
|---------|-------------|
| `owui instances` | Manage Open WebUI instances |
| `owui users` | Manage users |
| `owui groups` | Manage groups |
| `owui models` | Manage models (list, show, enable/disable, visibility, group access) |
| `owui completion` | Generate shell completions |
| `owui update` | Update owui to the latest version |
| `owui version` | Print version information |

## CLI Reference

See the full [CLI Reference](cli/owui.md) for detailed command documentation.
