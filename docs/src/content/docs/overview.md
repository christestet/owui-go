---
title: Overview
description: What owui is and what it can do.
---

`owui` is a fast and flexible CLI written in Go for managing multiple
[Open WebUI](https://github.com/open-webui/open-webui) instances — an
open-source project that provides a user-friendly interface for interacting
with various LLMs.

It ships as a single binary and runs cross-platform (Linux/macOS,
amd64/arm64).

## Features

- Manage multiple Open WebUI instances (add, remove, switch, list)
- Health checking of instances
- Manage users (list, create, remove, update roles, group membership)
- Manage groups (list, add, remove, update, members, permissions)
- Manage models (list, show details, enable/disable, set visibility, group access)
- Manage functions, tools and pipelines
- Interactive TUI wizards for adding instances
- JSON and pretty-printed table output formats
- Shell completions (bash, zsh, fish, powershell)
- Per-user configuration with auto-detection of the config directory
- Self-update: `owui update` replaces the binary in-place; a background check
  notifies you of new releases

## About Open WebUI

`owui` is designed to work with
[Open WebUI](https://github.com/open-webui/open-webui), which provides a
user-friendly web interface for interacting with various Large Language Models
(LLMs) and lets you run multiple instances with different configurations or LLM
backends.
