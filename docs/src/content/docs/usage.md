---
title: Usage
description: Common owui commands and global flags.
---

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

For the full, auto-generated list of commands and flags, see the
[CLI Reference](reference/cli/owui/).

## Global flags

| Flag                | Short | Description                                        |
| ------------------- | ----- | -------------------------------------------------- |
| `--instance <name>` | `-i`  | Use a specific instance instead of the active one  |
| `--output <format>` | `-o`  | Output format: `pretty` (default) or `json`        |
| `--filter <filter>` | `-f`  | Filter results                                     |
