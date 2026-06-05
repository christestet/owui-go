---
title: Install
description: Install owui via the install script, go install, or build from source.
---

## Quick install (self-contained)

The recommended way to install `owui` is into a user-local directory. This does
not require `sudo`.

```sh
curl -fsSL https://raw.githubusercontent.com/christestet/owui-go/main/scripts/install.sh | sh
```

The script installs to `~/.local/bin` by default. If this directory is not in
your `PATH`, the script will provide instructions on how to add it.

On macOS (`zsh`), the installer also tries to set up shell completion
automatically. If your current shell session was already running during
install, reload your shell config once:

```sh
source ~/.zshrc
```

## Go install

```sh
go install github.com/christestet/owui-go/cmd/owui@latest
```

## Build from source

```sh
git clone https://github.com/christestet/owui-go.git
cd owui-go
make build
# Binary is at ./bin/owui
```

## Uninstall

To completely remove `owui` and its configuration:

```sh
curl -fsSL https://raw.githubusercontent.com/christestet/owui-go/main/scripts/uninstall.sh | sh
```

Alternatively, remove the files manually:

```sh
rm ~/.local/bin/owui

# Remove configuration
rm -rf ~/.config/owui                       # Linux
rm -rf ~/Library/Application\ Support/owui  # macOS
```
