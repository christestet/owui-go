#!/bin/sh
set -e

BINARY="owui"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

log() {
    echo "$@" >&2
}

# 1. Remove binary
if [ -f "$INSTALL_DIR/$BINARY" ]; then
    log "Removing binary from $INSTALL_DIR/$BINARY..."
    rm "$INSTALL_DIR/$BINARY"
else
    # Try finding it in path if it wasn't in the default install dir
    EXISTING_PATH=$(which $BINARY 2>/dev/null)
    if [ -n "$EXISTING_PATH" ]; then
        log "Removing binary from $EXISTING_PATH..."
        if [ -w "$EXISTING_PATH" ]; then
            rm "$EXISTING_PATH"
        else
            log "Permission denied for $EXISTING_PATH, trying with sudo..."
            sudo rm "$EXISTING_PATH"
        fi
    else
        log "Binary $BINARY not found."
    fi
fi

# 2. Remove configuration
case "$(uname -s)" in
    Darwin)
        CONF_DIR="$HOME/Library/Application Support/owui"
        ;;
    *)
        CONF_DIR="$HOME/.config/owui"
        ;;
esac

if [ -d "$CONF_DIR" ]; then
    log "Removing configuration directory $CONF_DIR..."
    rm -rf "$CONF_DIR"
fi

# 3. Remove shell completions (best effort)
log "Removing shell completions..."
rm -f "$HOME/.zfunc/_$BINARY" 2>/dev/null || true
rm -f "$HOME/.zsh/completions/_$BINARY" 2>/dev/null || true
rm -f "$HOME/.bash_completion.d/$BINARY" 2>/dev/null || true
rm -f "$HOME/.local/share/bash-completion/completions/$BINARY" 2>/dev/null || true
rm -f "$HOME/.config/fish/completions/$BINARY.fish" 2>/dev/null || true

# 4. Remove shell config lines added by 'owui completion install' (best effort)
MARKER="# owui shell completions"

remove_owui_block() {
    rcfile="$1"
    if [ -f "$rcfile" ] && grep -q "$MARKER" "$rcfile"; then
        log "Cleaning up $rcfile..."
        # Remove the marker line and the line(s) immediately following it that belong to owui
        sed -i.bak "/$MARKER/,/^$/d" "$rcfile" 2>/dev/null || \
            sed -i '' "/$MARKER/,/^$/d" "$rcfile" 2>/dev/null || true
        rm -f "${rcfile}.bak" 2>/dev/null || true
    fi
}

remove_owui_block "$HOME/.zshrc"
remove_owui_block "$HOME/.bashrc"

log "Successfully uninstalled $BINARY."
