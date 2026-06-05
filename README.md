# owui

[![Go Version](https://img.shields.io/badge/go-1.26-00ADD8.svg?style=flat&logo=go)](https://golang.org)
[![CI](https://github.com/christestet/owui-go/actions/workflows/ci.yml/badge.svg)](https://github.com/christestet/owui-go/actions/workflows/ci.yml)
[![Docs](https://github.com/christestet/owui-go/actions/workflows/docs.yml/badge.svg)](https://christestet.github.io/owui-go/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

This tool helps you seamlessly switch between and manage different instances of [Open WebUI](https://github.com/open-webui/open-webui) — an awesome open-source project that provides a user-friendly interface for interacting with various LLMs.

Single binary, cross-platform (Linux/macOS, amd64/arm64).

## Features

- Manage multiple [Open WebUI](https://github.com/open-webui/open-webui) instances (add, remove, switch, list)
- Health checking of instances
- Manage users (list, create, remove, update roles, group membership)
- Manage groups (list, add, remove, update, members)
- Manage models (list, show details, enable/disable, set visibility, group access)
- Interactive TUI wizards for adding instances
- JSON and pretty-printed table output formats
- Shell completions (bash, zsh, fish, powershell)
- Configuration per-user with auto-detection of config directory
- Self-update: `owui update` replaces the binary in-place; background check notifies on new releases

## About Open WebUI

This tool is designed to work with [Open WebUI](https://github.com/open-webui/open-webui), an awesome open-source project that provides a user-friendly web interface for interacting with various Large Language Models (LLMs). Open WebUI allows you to run multiple instances with different configurations or LLM backends.

If you haven't already, check out the [Open WebUI project](https://github.com/open-webui/open-webui) to learn more about this powerful platform.

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

# List all models
owui models list

# Show model details
owui models show claude-sonnet

# Enable/disable models
owui models set-status enable llama-3.1
owui models set-status disable gpt-4o

# Change model visibility
owui models set-visibility public claude-sonnet
owui models set-visibility private gpt-4o

# Add a model to groups
owui models add-to-group --model gpt-4o --groups developers backend-team

# Remove a model from groups
owui models remove-from-group claude-sonnet --groups designers

# Install shell completions (bash/zsh/fish)
owui completion install

# Update owui to the latest release
owui update

# Show version
owui version
```

<!-- BEGIN:CLI_COMMANDS -->

### CLI Command Reference (auto-generated)

Run `make docs-readme` to refresh this section.

| Command | Description |
| --- | --- |
| `owui` | owui is a CLI to manage Open WebUI instances |
| `owui completion` | Generate completion script |
| `owui completion install` | Install completion script for the current shell |
| `owui functions` | Manage functions in an Open WebUI instance |
| `owui functions list` | List all functions |
| `owui groups` | Manage groups in an Open WebUI instance |
| `owui groups add` | Create a new group |
| `owui groups add-users` | Add user(s) to a group |
| `owui groups list` | List all groups |
| `owui groups members` | Show group details and members |
| `owui groups remove` | Remove one or more groups |
| `owui groups remove-users` | Remove user(s) from a group |
| `owui groups show-models` | List models a group can read or write |
| `owui groups show-permissions` | Show group permissions |
| `owui groups show-tools` | List tools a group can read or write |
| `owui groups update` | Update a group's name, description, or permissions |
| `owui instances` | Manage Open WebUI instances |
| `owui instances add` | Add a new instance |
| `owui instances health` | Check the health of all instances or a specified instance |
| `owui instances list` | List all instances |
| `owui instances remove` | Remove an instance |
| `owui instances use` | Switch the active instance |
| `owui models` | Manage models in an Open WebUI instance |
| `owui models add-to-group` | Add model(s) to group(s) |
| `owui models list` | List all models |
| `owui models remove-from-group` | Remove a model from group(s) |
| `owui models set-status` | Enable or disable models |
| `owui models set-visibility` | Set model visibility to public or private |
| `owui models show` | Show model details |
| `owui pipelines` | Manage pipelines and valves in an Open WebUI instance |
| `owui pipelines add` | Add a pipeline registration |
| `owui pipelines list` | List pipeline registrations and pipes |
| `owui pipelines remove` | Remove a pipeline registration |
| `owui pipelines show` | Show details for a pipe |
| `owui pipelines upload` | Upload a pipeline file |
| `owui pipelines valves` | Manage pipe valves |
| `owui pipelines valves show` | Show current valves for a pipe |
| `owui pipelines valves spec` | Show valves spec for a pipe |
| `owui pipelines valves update` | Update valves for a pipe |
| `owui tools` | Manage tools in an Open WebUI instance |
| `owui tools list` | List all tools |
| `owui update` | Update owui to the latest version |
| `owui users` | Manage users in an Open WebUI instance |
| `owui users add-to-group` | Add user(s) to a group |
| `owui users create` | Create a new user |
| `owui users list` | List all users |
| `owui users remove` | Remove a user |
| `owui users remove-from-group` | Remove user(s) from a group |
| `owui users update-role` | Update a user's role |
| `owui version` | Print the version number of owui |

<!-- END:CLI_COMMANDS -->

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

### Documentation

The docs site is built with [Astro](https://astro.build) +
[Starlight](https://starlight.astro.build) under `docs/` and published to
[GitHub Pages](https://christestet.github.io/owui-go/). The CLI reference is
generated from the Cobra command tree.

```sh
make docs-deps   # Install docs site dependencies (npm ci)
make docs-dev    # Generate the CLI reference, then run the site with live reload
make docs-site   # Generate the CLI reference, then build the static site
make docs-readme # Refresh the auto-generated CLI table in this README
```

### Releases

Releases use [release-please](https://github.com/googleapis/release-please) with
[Conventional Commits](https://www.conventionalcommits.org/). Merging the
release PR it opens cuts a `v*` tag, which triggers
[GoReleaser](https://goreleaser.com) to build and publish the binaries.

## License

MIT License - see [LICENSE](LICENSE) for details.
