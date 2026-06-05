---
title: Configuration
description: Where owui stores its configuration and the config file format.
---

Configuration is stored at:

- **Linux:** `~/.config/owui/config.json`
- **macOS:** `~/Library/Application Support/owui/config.json`

The config file is created automatically when you add your first instance with
`owui instances add`.

## Example config

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
