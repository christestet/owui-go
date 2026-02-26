# AGENTS.md

## Project Description
`owui` is a Go-based CLI tool for managing multiple Open WebUI instances via the Open WebUI REST API. Focus: Single binary, shell-native autocomplete, interactive TUI pickers, self-update with checksum validation, cross-platform (Linux/macOS, amd64/arm64).
we will use a config.json to store the configuration of the cli tool.

## Project Details
This is a cli tool written in go.
This cli tool is used to manage multiple instances of open webui. 
Its usage experience should be similar to kubectl or docker.
For the management of open webui, we will use the open webui api. You can have a look into the openapi spec at ./openapi-reference/*-openapi.json. 
Each version of open webui has its own openapi spec, e.g. ./openapi-reference/0.8.5-openapi.json for version 0.8.5.

## Skills
We are using the golang-pro skill to help us build it. - you can view the skill at .agents/skills/golang-pro/SKILL.md

## Packages 
To build the cli tool, we will use the following tools:
 - go version go1.26.0 darwin/arm64 as current version
 - `github.com/spf13/cobra` for cli framework
 - `github.com/spf13/viper` for configuration management
 - `github.com/charmbracelet/bubbletea` for ui
 - `github.com/charmbracelet/lipgloss` for styling
 - `github.com/charmbracelet/bubbles` for ui components
 - `github.com/charmbracelet/huh` for forms and wizards
 - `github.com/creativeprojects/go-selfupdate` for self update the binary

## Config File

### Path Resolution (stdlib only)

```go
func ConfigPath() (string, error) {
    base, err := os.UserConfigDir() // ~/.config on Linux, ~/Library/Application Support on macOS
    if err != nil {
        return "", err
    }
    dir := filepath.Join(base, "owui")
    os.MkdirAll(dir, 0700)
    return filepath.Join(dir, "config.json"), nil
}
```

**Resolved paths:**

- Linux: `~/.config/owui/config.json`
- macOS: `~/Library/Application Support/owui/config.json`

## Sample Config File

```json
{
  "cli": {
    "version": "1.2.0",
    "checksum": "sha256:a3f9c2...",
    "last_update_check": "2026-02-25T22:00:00Z"
  },
  "active_instance": "home-lab", // the active instance to use, if not set, use the first instance
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
    "output_format": "pretty", //or json
    "timeout_seconds": 30
  }
}
```

## CLI Commands

```bash
# update the cli tool
owui update

# show help and version
owui help

#### instances ####
# list all instances
owui instances list

# switch to a different instance
owui instances use <instance-name>

# add a new instance
owui instances add

# remove an instance
owui instances remove <instance-name>

#### users ####
# list all users
owui users list --filter <user-role> <admin|user|pending> -i | --instance <instance-name> (or leave empty to use the active instance)

# add a new user
owui users add <user-name> <email> -> please refer to the openapi spec for the exact requiredfield names

# remove an user
owui users remove <user-name> (with tab auto completion)

...

#### groups ####
# list all groups
owui groups list

# add a new group
owui groups add <group-name> --description <group-description>
# remove an group
owui groups remove <group-name> (with tab auto completion)

# add a user or users to a group
owui groups add-users <group-name> <user-name> <user-name> ... (auto complete)

# remove a user or users from a group
owui groups remove-users <group-name> <user-name> <user-name> ... (auto complete)

#### -> have a look at the openapi spec for the exact field names and types
```

All `owui` commands should support the following flags except `owui update` and `owui help`:
- `-i | --instance <instance-name>` (or leave empty to use the active instance)
- `-o | --output <output-format>` (or leave empty to use console output, which is the default, or json)

Filters should be implemented as follows:
- `-f | --filter <filter>` (or leave empty to use no filter)

## CICD and Release

- We are using GitHub Actions to build and release the cli tool.
- we will provide arm and amd64 binaries for linux and darwin.
- we will provide a checksum file for each release.
- we will use the `go-selfupdate` library to update the cli tool.

## Git and GitHub
- We are using GitHub Actions to build and release the cli tool.
- We are developing this tool on dev branch, and will merge to main branch when we want to release a new version.
- We are using conventional commits to version our commits.

## Testing
Have a look at: .agents/skills/golang-pro/references/testing.md
Always make shure to test your code and always check if new features break existing functionality and if the tests still covering all the functionality.

## Project Structure
Have a look at: .agents/skills/golang-pro/references/project-structure.md
