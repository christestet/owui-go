# owui models command

owui models command is the subcommand which manages models in Open WebUI. Refer and search through [Open WebUI v0.8.5 OpenAPI](file:///openapi-reference/0.8.5-openapi.json).

Models in Open WebUI have two key properties for filtering and management:

- **Status** (`is_active`): A model is either **enabled** (`is_active: true`) or **disabled** (`is_active: false`). Disabled models are hidden from users.
- **Visibility** (derived from `access_grants`): A model is either **public** (empty `access_grants` array -- accessible to all users) or **private** (non-empty `access_grants` -- restricted to specific groups).

These two dimensions are independent: a model can be enabled+public, enabled+private, disabled+public, or disabled+private.

---

## List models

```bash
owui models list|ls [--filter enabled|disabled|private|public] [--query <search>] [--tag <tag>]
```

This command lists all models in Open WebUI with their name, id, base model, status (enabled/disabled), visibility (public/private), access grant count, and last updated timestamp.

- The `--filter` flag accepts `enabled`, `disabled`, `private`, or `public` to show only models matching that state.
- Without `--filter`, all models are shown.
- The `--query` flag performs a server-side search by model name/id.
- The `--tag` flag filters models by tag.
- The `status` column shows `enabled` or `disabled` based on `is_active`.
- The `visibility` column shows `public` or `private` based on whether `access_grants` is empty or not.
- API endpoint: `GET /api/v1/models/list` (paginated, returns `ModelAccessListResponse`)
- For tab completion and internal lookups we use `GET /api/v1/models` (returns all models, useful for autocomplete).

```example terminal output
$ owui models list
NAME                  ID                        BASE MODEL              STATUS     VISIBILITY   GRANTS   UPDATED
GPT-4o                gpt-4o                    openai/gpt-4o           enabled    public       0        2026-02-20
Claude Sonnet         claude-sonnet             anthropic/claude-3.5    enabled    private      3        2026-02-18
Llama 3.1             llama-3.1                 ollama/llama3.1         disabled   public       0        2026-02-10
Custom Assistant      custom-assistant          gpt-4o                  enabled    private      1        2026-02-25
```

```example terminal output with filter
$ owui models list --filter disabled
NAME                  ID                        BASE MODEL              STATUS     VISIBILITY   GRANTS   UPDATED
Llama 3.1             llama-3.1                 ollama/llama3.1         disabled   public       0        2026-02-10

Showing 1 model(s) matching filter "disabled".
```

```example terminal output with filter
$ owui models list --filter private
NAME                  ID                        BASE MODEL              STATUS     VISIBILITY   GRANTS   UPDATED
Claude Sonnet         claude-sonnet             anthropic/claude-3.5    enabled    private      3        2026-02-18
Custom Assistant      custom-assistant          gpt-4o                  enabled    private      1        2026-02-25

Showing 2 model(s) matching filter "private".
```

JSON output:

```bash
owui models list -o json
```

---

## Show model details

```bash
owui models show <model_name|model_id>
```

This command displays a comprehensive detail view of a single model. Because models carry a lot of information (metadata, status, visibility, capabilities, access grants with resolved group names, owner, timestamps), a plain key-value list would be hard to scan. Instead, the output uses **lipgloss-styled sections** with visual grouping, borders, and color to make the terminal output readable at a glance.

- The positional argument is the model name or id (with tab completion for all model names).
- Tab completion shows `Name (id)` format for all models.
- API endpoints used:
  - `GET /api/v1/models/model?id=<id>` -- fetches `ModelAccessResponse` (includes `access_grants`, `user`, `write_access`)
  - `GET /api/v1/groups/` -- to resolve `principal_id` UUIDs in access grants to human-readable group names

### Interactive mode

When `owui models show` is called without arguments, an interactive select is shown:

```example terminal output
$ owui models show
? Select model (Use arrow keys)
search: claude
    Claude Sonnet (claude-sonnet)
    Custom Assistant (custom-assistant)
    GPT-4o (gpt-4o)
```

### Terminal output (lipgloss-styled)

The detail view is rendered in structured sections using `lipgloss` for borders, padding, and color. The layout uses a **card-style** design with distinct visual blocks. This is the target rendering -- implementation uses `lipgloss.NewStyle()` with `Border()`, `Padding()`, `Foreground()`, and `Width()`.

```example terminal output
$ owui models show claude-sonnet

  Claude Sonnet
  claude-sonnet

  ── General ─────────────────────────────────────
  Base Model        anthropic/claude-3.5-sonnet
  Description       Claude 3.5 Sonnet by Anthropic
  Owner             admin (admin@example.com)
  Created           2026-01-15 10:30:00
  Updated           2026-02-18 14:22:00

  ── Status ──────────────────────────────────────
  Status            enabled
  Visibility        private (3 group grants)

  ── Capabilities ────────────────────────────────
  Vision            yes
  Citations         yes
  Code Interpreter  no

  ── Access Grants ───────────────────────────────
  GROUP             PERMISSION   GRANTED
  developers        read         2026-01-20
  backend-team      read         2026-02-01
  designers         read         2026-02-10

```

#### Layout specification

The output is composed of these visual elements:

1. **Header block**: Model `name` rendered bold/bright, model `id` rendered dim below it. Both left-padded with 2 spaces.

2. **Section dividers**: Each section has a labeled horizontal rule using lipgloss, e.g. `── General ──────...`. The label is rendered with a subtle accent color.

3. **Key-value pairs**: Left column (labels) is right-padded to a fixed width (e.g., 18 chars) and rendered in a muted/dim color. Right column (values) is rendered in the default foreground color. Special values get color treatment:
   - `enabled` = green, `disabled` = red
   - `public` = cyan, `private` = yellow
   - `yes` = green, `no` = dim/gray

4. **Access Grants table**: Rendered as a compact table within the card. Group names are resolved from UUIDs via the groups API. The `GRANTED` column shows the `created_at` timestamp formatted as date. If there are no access grants, show `No access grants -- model is public.` in dim text.

5. **Overall card**: The entire output is wrapped in a lipgloss style with `PaddingLeft(2)` for consistent indentation. No outer border -- the section dividers provide enough visual structure.

#### Adaptive width

The section divider lines adapt to the terminal width (using `lipgloss.Width()` or `term.GetSize()`), capped at a maximum of 60 characters. On narrow terminals the dividers are shorter; the key-value layout remains fixed.

#### Implementation approach

```go
// Pseudo-code for the rendering approach
import (
    "github.com/charmbracelet/lipgloss"
)

var (
    titleStyle   = lipgloss.NewStyle().Bold(true).PaddingLeft(2)
    subtitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).PaddingLeft(2)
    labelStyle   = lipgloss.NewStyle().Width(18).Foreground(lipgloss.Color("245")).PaddingLeft(2)
    valueStyle   = lipgloss.NewStyle()
    enabledStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))  // green
    disabledStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // red
    privateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // yellow
    publicStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))  // cyan
    dividerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).PaddingLeft(2)
    dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func renderSectionDivider(label string, width int) string {
    prefix := "── " + label + " "
    remaining := width - len(prefix)
    return dividerStyle.Render(prefix + strings.Repeat("─", remaining))
}

func renderKeyValue(label, value string) string {
    return labelStyle.Render(label) + valueStyle.Render(value)
}
```

#### JSON output

```bash
owui models show claude-sonnet -o json
```

In JSON mode, the raw `ModelAccessResponse` is printed with `json.MarshalIndent`, plus an added `resolved_groups` field that maps `principal_id` to group name for convenience:

```json
{
  "id": "claude-sonnet",
  "name": "Claude Sonnet",
  "base_model_id": "anthropic/claude-3.5-sonnet",
  "is_active": true,
  "meta": {
    "description": "Claude 3.5 Sonnet by Anthropic",
    "capabilities": { "vision": true, "citations": true }
  },
  "access_grants": [
    {
      "id": "grant-1",
      "principal_type": "group",
      "principal_id": "uuid-123",
      "permission": "read",
      "resolved_group_name": "developers"
    }
  ],
  "user": { "id": "abc123", "name": "admin", "email": "admin@example.com" },
  "created_at": 1705312200,
  "updated_at": 1708265520
}
```

---

## Set status (enable/disable)

```bash
owui models set-status <enable|disable> <model_name> [<model_name_2> ...]
```

This command enables or disables one or more models by toggling their `is_active` state.

- The first positional argument is the action: `enable` or `disable`.
- Subsequent positional arguments are model names (with autocomplete).
- **Smart tab completion:**
  - When action is `enable`: autocomplete only shows models that are currently **disabled** (`is_active: false`).
  - When action is `disable`: autocomplete only shows models that are currently **enabled** (`is_active: true`).
  - Already-selected model names are excluded from subsequent completions.
- A confirmation prompt is shown before toggling.
- API endpoint: `POST /api/v1/models/model/toggle?id=<model_id>` (toggles `is_active`, returns `ModelResponse`).

### Interactive mode

When `owui models set-status` is called without a model name, an interactive flow is shown:

```example terminal output
$ owui models set-status enable
? Select models to enable (Use arrow keys, space to select)
search: llam
    [x] Llama 3.1 (llama-3.1)
    [ ] Old Model (old-model)

Confirm enabling 1 model(s): Llama 3.1? (y/n) y
Successfully enabled model 'Llama 3.1'
```

```example terminal output
$ owui models set-status disable
? Select models to disable (Use arrow keys, space to select)
search: gpt
    [x] GPT-4o (gpt-4o)
    [ ] Claude Sonnet (claude-sonnet)
    [ ] Custom Assistant (custom-assistant)

Confirm disabling 1 model(s): GPT-4o? (y/n) y
Successfully disabled model 'GPT-4o'
```

### Non-interactive mode

```bash
owui models set-status enable llama-3.1 old-model
```

```example terminal output
Confirm enabling 2 model(s): Llama 3.1, Old Model? (y/n) y
Successfully enabled model 'Llama 3.1'
Successfully enabled model 'Old Model'
```

```bash
owui models set-status disable gpt-4o
```

```example terminal output
Confirm disabling 1 model(s): GPT-4o? (y/n) y
Successfully disabled model 'GPT-4o'
```

---

## Set visibility (private/public)

```bash
owui models set-visibility <private|public> <model_name> [<model_name_2> ...]
```

This command changes the visibility of one or more models between public and private.

- The first positional argument is the target visibility: `private` or `public`.
- Subsequent positional arguments are model names (with autocomplete).
- **Smart tab completion:**
  - When action is `public`: autocomplete only shows models that are currently **private** (have non-empty `access_grants`).
  - When action is `private`: autocomplete only shows models that are currently **public** (have empty `access_grants`).
  - Already-selected model names are excluded from subsequent completions.
- **Making a model public:** Clears all `access_grants` -- the model becomes accessible to all users.
- **Making a model private:** Removes all `access_grants`, making the model accessible only to admins/owner. To grant group access afterward, use `owui models add-to-group`.
- A confirmation prompt is shown before changing visibility.
- API endpoint: `POST /api/v1/models/model/access/update` with `ModelAccessGrantsForm` body.

### Interactive mode

When `owui models set-visibility` is called without a model name, an interactive flow is shown:

```example terminal output
$ owui models set-visibility public
? Select models to make public (Use arrow keys, space to select)
search: claude
    [x] Claude Sonnet (claude-sonnet) [3 grants]
    [ ] Custom Assistant (custom-assistant) [1 grant]

Confirm making 1 model(s) public: Claude Sonnet? (y/n) y
Note: This will remove all 3 access grants from 'Claude Sonnet'.
Successfully set model 'Claude Sonnet' to public
```

```example terminal output
$ owui models set-visibility private
? Select models to make private (Use arrow keys, space to select)
search: gpt
    [x] GPT-4o (gpt-4o)
    [ ] Llama 3.1 (llama-3.1)

Confirm making 1 model(s) private: GPT-4o? (y/n) y
Successfully set model 'GPT-4o' to private (admin-only access)
Tip: Use 'owui models add-to-group' to grant access to specific groups.
```

### Non-interactive mode

```bash
owui models set-visibility public claude-sonnet custom-assistant
```

```example terminal output
Confirm making 2 model(s) public: Claude Sonnet, Custom Assistant? (y/n) y
Successfully set model 'Claude Sonnet' to public
Successfully set model 'Custom Assistant' to public
```

---

## Add model to group

```bash
owui models add-to-group [--model <model_name> --groups <group_1> [<group_2> ...]]
owui models add-to-group [--models <model_1> [<model_2> ...] --group <group_name>]
```

This command adds one or more models to one or more groups, granting group members access to the model(s). Both directions are supported:

1. **One model to multiple groups:** `--model <name> --groups <g1> <g2> ...`
2. **Multiple models to one group:** `--models <m1> <m2> ... --group <name>`

- The `--model` / `--models` flag accepts model names (with autocomplete for all models).
- The `--group` / `--groups` flag accepts group names (with autocomplete, only **local** groups -- oauth groups are filtered out, same logic as in `owui groups`).
- When adding a model to a group, an `AccessGrantModel` entry is created with `principal_type: "group"` and `principal_id: <group_id>`, `resource_type: "model"`, `resource_id: <model_id>`, `permission: "read"`.
- If the model is already in the specified group, a warning is shown and that combination is skipped.
- A confirmation prompt is shown before applying changes.
- API endpoint: `POST /api/v1/models/model/access/update` with `ModelAccessGrantsForm` body. The existing `access_grants` must be fetched first (via `GET /api/v1/models/model?id=<id>`) and the new grant(s) merged before sending the update.

### Interactive mode

When `owui models add-to-group` is called without flags, an interactive wizard is shown:

```example terminal output
$ owui models add-to-group
? Select mode (Use arrow keys)
    > Add one model to multiple groups
      Add multiple models to one group

? Select model (Use arrow keys)
search: gpt
    GPT-4o (gpt-4o)
    Custom Assistant (custom-assistant)

? Select groups to grant access (Use arrow keys, space to select)
search: dev
    [x] developers
    [ ] designers
    [x] backend-team

Confirm adding model 'GPT-4o' to 2 group(s): developers, backend-team? (y/n) y
Successfully added model 'GPT-4o' to group 'developers'
Successfully added model 'GPT-4o' to group 'backend-team'
```

```example terminal output (multiple models to one group)
$ owui models add-to-group
? Select mode (Use arrow keys)
      Add one model to multiple groups
    > Add multiple models to one group

? Select models to add (Use arrow keys, space to select)
search:
    [x] GPT-4o (gpt-4o)
    [x] Claude Sonnet (claude-sonnet)
    [ ] Llama 3.1 (llama-3.1)

? Select group (Use arrow keys)
search: dev
    developers
    designers
    backend-team

Confirm adding 2 model(s) to group 'developers': GPT-4o, Claude Sonnet? (y/n) y
Successfully added model 'GPT-4o' to group 'developers'
Successfully added model 'Claude Sonnet' to group 'developers'
```

### Non-interactive mode

```bash
# One model to multiple groups
owui models add-to-group --model gpt-4o --groups developers backend-team
```

```example terminal output
Confirm adding model 'GPT-4o' to 2 group(s): developers, backend-team? (y/n) y
Successfully added model 'GPT-4o' to group 'developers'
Successfully added model 'GPT-4o' to group 'backend-team'
```

```bash
# Multiple models to one group
owui models add-to-group --models gpt-4o claude-sonnet --group developers
```

```example terminal output
Confirm adding 2 model(s) to group 'developers': GPT-4o, Claude Sonnet? (y/n) y
Successfully added model 'GPT-4o' to group 'developers'
Successfully added model 'Claude Sonnet' to group 'developers'
```

---

## Remove model from group

```bash
owui models remove-from-group <model_name> --groups <group_1> [<group_2> ...]
```

This command removes a model from one or more groups, revoking group members' access to the model.

- The positional argument is the model name (with autocomplete for all models that have at least one access grant, i.e., private models).
- The `--groups` flag accepts one or more group names.
- **Smart tab completion for `--groups`:** Only groups where the selected model is actually assigned are shown. This requires fetching the model's current `access_grants` and resolving the `principal_id` values to group names.
- If the model is removed from all its groups, it becomes effectively admin-only. A note is shown to inform the user.
- A confirmation prompt is shown before applying changes.
- API endpoint: `POST /api/v1/models/model/access/update` with `ModelAccessGrantsForm` body. The existing `access_grants` are fetched first (via `GET /api/v1/models/model?id=<id>`), the specified group grants are removed, and the remaining grants are sent as update.

### Interactive mode

When `owui models remove-from-group` is called without arguments, an interactive flow is shown:

```example terminal output
$ owui models remove-from-group
? Select model (Use arrow keys)
search: claude
    Claude Sonnet (claude-sonnet) [3 groups]
    Custom Assistant (custom-assistant) [1 group]

? Select groups to revoke access (Use arrow keys, space to select)
    [x] developers
    [ ] designers
    [x] backend-team

Confirm removing model 'Claude Sonnet' from 2 group(s): developers, backend-team? (y/n) y
Successfully removed model 'Claude Sonnet' from group 'developers'
Successfully removed model 'Claude Sonnet' from group 'backend-team'
Note: Model 'Claude Sonnet' still has access grants for 1 group(s): designers
```

```example terminal output (removing last group)
$ owui models remove-from-group custom-assistant --groups developers
Confirm removing model 'Custom Assistant' from 1 group(s): developers? (y/n) y
Successfully removed model 'Custom Assistant' from group 'developers'
Note: Model 'Custom Assistant' has no remaining group access -- it is now admin-only.
Tip: Use 'owui models set-visibility public' to make it accessible to all users.
```

### Non-interactive mode

```bash
owui models remove-from-group claude-sonnet --groups developers backend-team
```

```example terminal output
Confirm removing model 'Claude Sonnet' from 2 group(s): developers, backend-team? (y/n) y
Successfully removed model 'Claude Sonnet' from group 'developers'
Successfully removed model 'Claude Sonnet' from group 'backend-team'
```

---

## API Endpoints Reference

| Command | API Endpoint | Method | Request Body / Params |
|---------|-------------|--------|-----------------------|
| `models list` | `/api/v1/models/list` | GET | Query params: `query`, `view_option`, `tag`, `order_by`, `direction`, `page` |
| `models list` (for autocomplete) | `/api/v1/models` | GET | Query param: `refresh` (boolean) |
| `models show` | `/api/v1/models/model` | GET | Query param: `id` (model ID) |
| `models show` (resolve groups) | `/api/v1/groups/` | GET | -- |
| `models set-status` | `/api/v1/models/model/toggle` | POST | Query param: `id` (model ID) |
| `models set-visibility` | `/api/v1/models/model/access/update` | POST | `ModelAccessGrantsForm`: `{id, name?, access_grants[]}` |
| `models add-to-group` | `/api/v1/models/model/access/update` | POST | `ModelAccessGrantsForm`: `{id, name?, access_grants[]}` |
| `models remove-from-group` | `/api/v1/models/model/access/update` | POST | `ModelAccessGrantsForm`: `{id, name?, access_grants[]}` |

---

## API Schemas Reference

### ModelAccessListResponse (response from list endpoint)

```json
{
  "items": [ModelAccessResponse, ...],
  "total": 42
}
```

### ModelAccessResponse (single model in list/detail)

```json
{
  "id": "gpt-4o",
  "user_id": "abc123",
  "base_model_id": "openai/gpt-4o",
  "name": "GPT-4o",
  "params": {},
  "meta": {
    "profile_image_url": "/static/favicon.png",
    "description": "OpenAI GPT-4o model",
    "capabilities": {}
  },
  "access_grants": [
    {
      "id": "grant-1",
      "resource_type": "model",
      "resource_id": "gpt-4o",
      "principal_type": "group",
      "principal_id": "group-uuid-123",
      "permission": "read",
      "created_at": 1708000000
    }
  ],
  "is_active": true,
  "updated_at": 1708900000,
  "created_at": 1707000000,
  "user": { "id": "abc123", "name": "admin", "email": "admin@example.com" },
  "write_access": true
}
```

### ModelAccessGrantsForm (request body for access updates)

```json
{
  "id": "gpt-4o",
  "name": "GPT-4o",
  "access_grants": [
    {
      "principal_type": "group",
      "principal_id": "group-uuid-123",
      "permission": "read"
    }
  ]
}
```

### ModelResponse (response from toggle endpoint)

```json
{
  "id": "gpt-4o",
  "user_id": "abc123",
  "base_model_id": "openai/gpt-4o",
  "name": "GPT-4o",
  "params": {},
  "meta": {},
  "access_grants": [],
  "is_active": false,
  "updated_at": 1708900000,
  "created_at": 1707000000
}
```

---

## Global flags

If no instance is specified with `--instance` or `-i` flag, the command will use the active instance. You can switch the active instance with `owui instances use <instance_name>`.

Output format can be set with `--output` or `-o` flag (`pretty` or `json`). Default is `pretty`.

## Autocomplete behavior

- **Model name autocomplete:** All commands that accept a model name provide shell autocomplete by fetching models from the active instance API (`GET /api/v1/models`).
  - `set-status enable`: Only disabled models (`is_active: false`) are shown.
  - `set-status disable`: Only enabled models (`is_active: true`) are shown.
  - `set-visibility public`: Only private models (non-empty `access_grants`) are shown.
  - `set-visibility private`: Only public models (empty `access_grants`) are shown.
  - `show`: All models are shown.
  - `remove-from-group`: Only models with at least one access grant (private models) are shown.
  - `add-to-group`: All models are shown.
  - Multiple model name arguments: Already-selected names are excluded from subsequent completions.
- **Group name autocomplete:** Commands that accept group names (`add-to-group`, `remove-from-group`) filter to only show **local** groups (oauth groups are filtered out, same logic as `owui groups` write operations).
  - `remove-from-group --groups`: Only groups that the selected model is currently assigned to are shown.

## Implementation notes

### Determining visibility

A model's visibility is determined at the CLI level (not an API field):
- **Public:** `len(access_grants) == 0`
- **Private:** `len(access_grants) > 0`

### Merging access grants

When adding or removing groups from a model, the flow is:
1. Fetch current model state via `GET /api/v1/models/model?id=<model_id>` to get existing `access_grants`.
2. Merge the changes (add new grants or remove specified grants).
3. Send the full updated `access_grants` array via `POST /api/v1/models/model/access/update`.

### Model identification

Models are identified by their `id` field (e.g., `gpt-4o`, `claude-sonnet`). The `name` field is a human-readable display name. In autocomplete and interactive selectors, both are shown as `Name (id)` for clarity. Positional arguments and `--model`/`--models` flags accept the model **id**.

### Group resolution

When adding a model to a group, the group name must be resolved to its `id` (UUID) via `GET /api/v1/groups/`. The `principal_id` in access grants is the group UUID, not the group name.
