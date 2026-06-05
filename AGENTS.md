# AGENTS.md

## WHAT: Project Description
`owui` is a Go-based CLI tool for managing multiple Open WebUI instances via the Open WebUI REST API. Focus: Single binary, shell-native autocomplete, interactive TUI pickers, self-update with checksum validation, cross-platform (Linux/macOS, amd64/arm64).

## WHY: Project Purpose
We aim to manage multiple instances of Open WebUI with a seamless user experience similar to `kubectl` or `docker`. This is achieved by interacting directly with the Open WebUI API using the provided OpenAPI specifications.

## HOW: Tech Stack and Development
- **Language**: Go 1.26 (darwin/arm64 current)
- **CLI Framework**: `github.com/spf13/cobra` and `github.com/spf13/viper` (configuration management)
- **UI Components**: `github.com/charmbracelet/bubbletea`, `lipgloss`, `bubbles`, `huh`
- **Self Update**: `github.com/creativeprojects/go-selfupdate`

## Progressive Context (Read when relevant)
Do not assume project structure or testing conventions; read these referenced files before working on related tasks:
- **OpenAPI Specs**: `openapi-reference/*-openapi.json` (e.g., `0.8.5-openapi.json`)
- **Coding Standards**: `.agents/skills/golang-pro/SKILL.md`
- **Project Structure**: `.agents/skills/golang-pro/references/project-structure.md`
- **Testing & CI/CD**: Read `.agents/skills/golang-pro/references/testing.md` and check `.github/workflows/`
- **Feature Docs References**: `.agents/docs/`

## Codebase Pointers
Instead of duplicating code instructions, refer directly to the source of truth in the codebase:
- **Configuration Management**: See `internal/config/config.go` for JSON structure, defaults, and OS-specific path resolution (e.g., `~/.config/owui/config.json` vs `~/Library/Application Support/owui/config.json`).
- **CLI Commands**: See `internal/cli/` to understand command structure and flags. Most commands use standard flags like `-i` (instance), `-o` (output), and `-f` (filter).
- **CLI Docs Generation**:
  - `make docs-readme` regenerates the CLI command reference block in `README.md` from Cobra command definitions (`internal/tools/readmegen`).
  - `make docs-cli` regenerates the markdown CLI reference pages into `docs/src/content/docs/reference/cli` (`internal/tools/docgen`); these are gitignored and rebuilt in CI.
  - `make docs-readme-check` is the CI guard and must pass; it fails if `README.md` is out of sync with the Cobra command tree.
- **Documentation Site**: Astro + Starlight under `docs/`, deployed to GitHub Pages via `.github/workflows/docs.yml`. See `.agents/docs/gh-pages.md`. Use `make docs-dev` / `make docs-site` locally.
- **Release Process**: `main`-only branching. Open PRs against `main` with conventional commits. `release-please` (`release-please-config.json`, `.release-please-manifest.json`) aggregates commits into a release PR; merging it cuts a `v*` tag which triggers GoReleaser (`.goreleaser.yaml`) to build and publish binaries (`.github/workflows/release.yml`).

## Hard Rules
- Never use emojis in the code or in the output of the CLI tool.
