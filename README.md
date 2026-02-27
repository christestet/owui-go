# owui

[![Go Version](https://img.shields.io/badge/go-1.26-00ADD8.svg?style=flat&logo=go)](https://golang.org)
[![CI](https://github.com/christestet/owui-go/actions/workflows/ci.yml/badge.svg)](https://github.com/christestet/owui-go/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A fast and flexible CLI written in Go for managing multiple Open WebUI instances.

Single binary, cross-platform (Linux/macOS, amd64/arm64).

## Features

- Manage multiple Open WebUI instances (add, remove, switch, list)
- Health checking of instances
- Interactive TUI wizards for adding instances
- JSON and pretty-printed table output formats
- Shell completions (bash, zsh, fish, powershell)
- Configuration per-user with auto-detection of config directory
- Self-update: `owui update` replaces the binary in-place; background check notifies on new releases

## Installation

### Quick install (Self-contained)

The recommended way to install `owui` is to a user-local directory. This does not require `sudo`.

```sh
curl -fsSL https://raw.githubusercontent.com/christestet/owui-go/main/scripts/install.sh | sh
```

The script installs to `~/.local/bin` by default. If this directory is not in your `PATH`, the script will provide instructions on how to add it.

On macOS (`zsh`), the installer also tries to set up shell completion automatically. If your current shell session was already running during install, reload your shell config once:

```sh
source ~/.zshrc
```

### Uninstall

To completely remove `owui` and its configuration:

```sh
curl -fsSL https://raw.githubusercontent.com/christestet/owui-go/main/scripts/uninstall.sh | sh
```

Alternatively, you can manually remove the files:

```sh
rm ~/.local/bin/owui

# Remove configuration
rm -rf ~/.config/owui                       # Linux
rm -rf ~/Library/Application\ Support/owui  # macOS
```

If you installed to a custom directory:

```sh
rm ~/.local/bin/owui
```

### Go install

```sh
go install github.com/christestet/owui-go/cmd/owui@latest
```

### Build from source

```sh
git clone https://github.com/christestet/owui-go.git
cd owui-go
make build
# Binary is at ./bin/owui
```

## Usage

```sh
# Add a new instance (interactive wizard)
owui instances add

# List all configured instances
owui instances list

# Switch active instance
owui instances use home-lab

# Check instance health
owui instances health

# Remove an instance
owui instances remove work

# Install shell completions (bash/zsh/fish)
owui completion install

# Update owui to the latest release
owui update

# Show version
owui version
```

### Global flags

| Flag                | Short | Description                                       |
| ------------------- | ----- | ------------------------------------------------- |
| `--instance <name>` | `-i`  | Use a specific instance instead of the active one |
| `--output <format>` | `-o`  | Output format: `pretty` (default) or `json`       |
| `--filter <filter>` | `-f`  | Filter results                                    |

## Configuration

Configuration is stored at:

- **Linux:** `~/.config/owui/config.json`
- **macOS:** `~/Library/Application Support/owui/config.json`

The config file is created automatically when you add your first instance with `owui instances add`.

Example config:

```json
{
  "cli": {
    "version": "1.0.0",
    "checksum": "sha256:a3f9c2...",
    "last_update_check": "2026-02-25T22:00:00Z"
  },
  "active_instance": "home-lab",
  "instances": {
    "home-lab": {
      "url": "http://192.168.1.10:3000",
      "api_key": "sk-xxxx",
      "added_at": "2026-01-10T10:00:00Z"
    },
    "work": {
      "url": "https://owui.company.com",
      "api_key": "sk-yyyy",
      "added_at": "2026-02-01T08:30:00Z"
    }
  },
  "settings": {
    "output_format": "pretty",
    "timeout_seconds": 30
  }
}
```

## Development

```sh
make deps      # Download and tidy dependencies
make build     # Build binary to bin/owui
make test      # Run tests with race detection
make run       # Run directly with go run
make fmt       # Format code
make lint      # Run go vet
make clean     # Remove build artifacts
```

## License

MIT License - see [LICENSE](LICENSE) for details.
