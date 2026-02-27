# owui groups command

owui groups command is the subcommand which manages groups in Open WebUI. Refer and search through [Open WebUI v0.8.5 OpenAPI](file:///Users/christoph/projekte/owui-go/openapi-reference/0.8.5-openapi.json).

Groups in Open WebUI come in two types:
- **local** groups: Manually created groups managed by admins.
- **oauth** groups: Automatically created via OAuth with the description pattern `"Group <group_name> created automatically via OAuth."`.

This distinction is important for filtering, autocomplete, and write operations (only local groups can be modified).

---

## List groups

```bash
owui groups list|ls [--filter oauth|local]
```

This command lists all groups in Open WebUI with their id, name, description, member count and type (local/oauth).

- The `--filter` flag accepts `oauth` or `local` to show only groups of that type.
- Without `--filter`, all groups are shown.
- The type column shows `local` or `oauth` based on the group description pattern.
- API endpoint: `GET /api/v1/groups/`

```example terminal output
$ owui groups list
NAME          DESCRIPTION                                          MEMBERS  TYPE
developers    Development team                                     5        local
designers     Design team                                          3        local
engineering   Group engineering created automatically via OAuth.    12       oauth
marketing     Group marketing created automatically via OAuth.     8        oauth
```

```example terminal output with filter
$ owui groups list --filter local
NAME          DESCRIPTION        MEMBERS  TYPE
developers    Development team   5        local
designers     Design team        3        local
```

JSON output:

```bash
owui groups list -o json
```

---

## Add group (create)

```bash
owui groups add --name <group_name> --description <group_description> [--users <user_1> <user_2> ...] [--permissions <json_string>]
```

This command creates a new group in Open WebUI with the specified name, description, and optionally adds users to it and sets permissions.

- The `--users` flag accepts one or more usernames. Only users with account type `user` (not `admin` or `pending`) can be added. Autocomplete filters accordingly.
- The `--permissions` flag accepts a JSON string defining the group permissions (see Permissions section below).
- API endpoint for creation: `POST /api/v1/groups/create` with `GroupForm` body.
- If `--users` is provided, after creation the command calls `POST /api/v1/groups/id/{id}/users/add` with the resolved user IDs.

### Interactive mode

When `owui groups add` is called without flags, an interactive wizard is shown:

```example terminal output
$ owui groups add
? Group name: backend-team
? Description: Backend development team
? Add users to this group? (y/n) y
? Select users to add (Use arrow keys, space to select)
search: al
    [ ] alice
    [x] bob
    [ ] charlie
    [x] diana

? Set custom permissions? (y/n) n

Successfully created group 'backend-team' with 2 members
```

### Non-interactive mode

```bash
owui groups add --name backend-team --description "Backend development team" --users bob diana
```

```example terminal output
Successfully created group 'backend-team' with 2 members
```

---

## Remove group (delete)

```bash
owui groups remove|rm <group_1> [<group_2> ...]
```

This command deletes one or more groups from Open WebUI by their name.

- Supports multiple group names as positional arguments.
- Autocomplete provides group name suggestions (only local groups, since oauth groups should not be deleted manually).
- A confirmation prompt is shown before deletion.
- API endpoint: `DELETE /api/v1/groups/id/{id}/delete`

### Interactive mode

When `owui groups remove` is called without arguments, an interactive multi-select is shown:

```example terminal output
$ owui groups remove
? Select groups to delete (Use arrow keys, space to select)
search: back
    [x] backend-team
    [ ] designers
    [ ] developers

Confirm deleting 1 group(s): backend-team? (y/n) y
Successfully deleted group 'backend-team'
```

### Non-interactive mode

```bash
owui groups remove backend-team designers
```

```example terminal output
Confirm deleting 2 group(s): backend-team, designers? (y/n) y
Successfully deleted group 'backend-team'
Successfully deleted group 'designers'
```

---

## Update group

```bash
owui groups update <group_name> [--name <new_name>] [--description <new_description>] [--permissions <json_string>]
```

This command updates an existing group's name, description, or permissions.

- The positional argument is the current group name (with autocomplete, only local groups).
- At least one of `--name`, `--description`, or `--permissions` must be provided.
- API endpoint: `POST /api/v1/groups/id/{id}/update` with `GroupUpdateForm` body.
- A confirmation prompt is shown before the update.

### Interactive mode

When `owui groups update` is called without arguments, an interactive flow is shown:

```example terminal output
$ owui groups update
? Select group to update (Use arrow keys)
search: dev
    developers
    designers

? New name (leave empty to keep 'developers'):
? New description (leave empty to keep 'Development team'): Senior development team
? Update permissions? (y/n) n

Confirm updating group 'developers'? (y/n) y
Successfully updated group 'developers'
```

### Non-interactive mode

```bash
owui groups update developers --description "Senior development team"
```

```example terminal output
Confirm updating group 'developers'? (y/n) y
Successfully updated group 'developers'
```

---

## Show group members

```bash
owui groups members|show <group_name>
```

This command shows the details and members of a specific group.

- The positional argument is the group name (with autocomplete for all groups, both local and oauth).
- Fetches group info via `GET /api/v1/groups/id/{id}` and members via `POST /api/v1/groups/id/{id}/users`.
- Displays group metadata (name, description, type, member count, created/updated timestamps) and a table of members.

### Interactive mode

When called without arguments, an interactive select is shown:

```example terminal output
$ owui groups members
? Select group (Use arrow keys)
search: dev
    developers
    designers
    engineering

Group: developers
Description: Development team
Type: local
Members: 5
Created: 2026-01-15 10:30:00
Updated: 2026-02-20 14:00:00

NAME      EMAIL               ROLE
alice     alice@example.com   user
bob       bob@example.com     user
charlie   charlie@example.com user
diana     diana@example.com   user
eve       eve@example.com     user
```

### Non-interactive mode

```bash
owui groups members developers
```

Shows the same output as above.

JSON output:

```bash
owui groups members developers -o json
```

---

## Add users to group

```bash
owui groups add-users <user_1> [<user_2> ...] --group <group_name>
```

This command adds one or more users to an existing group.

- Only users with role `user` can be added (not `admin` or `pending`). Autocomplete filters accordingly.
- The `--group` flag specifies the target group (with autocomplete, only local groups).
- A confirmation prompt is shown before adding.
- API endpoint: `POST /api/v1/groups/id/{id}/users/add` with `UserIdsForm` body.

Note: This is the same functionality as `owui users add-to-group` but accessed from the groups perspective.

### Interactive mode

When called without arguments:

```example terminal output
$ owui groups add-users
? Select group (Use arrow keys)
search: dev
    developers
    designers

? Select users to add (Use arrow keys, space to select)
search: bo
    [ ] alice
    [x] bob
    [ ] charlie
    [x] diana

Confirm adding 2 users to group 'developers'? (y/n) y
Successfully added bob, diana to group 'developers'
```

### Non-interactive mode

```bash
owui groups add-users bob diana --group developers
```

```example terminal output
Confirm adding 2 users to group 'developers'? (y/n) y
Successfully added bob, diana to group 'developers'
```

---

## Remove users from group

```bash
owui groups remove-users <user_1> [<user_2> ...] --group <group_name>
```

This command removes one or more users from an existing group.

- Only users currently in the group are shown in autocomplete.
- The `--group` flag specifies the target group (with autocomplete, only local groups).
- A confirmation prompt is shown before removing.
- API endpoint: `POST /api/v1/groups/id/{id}/users/remove` with `UserIdsForm` body.

Note: This is the same functionality as `owui users remove-from-group` but accessed from the groups perspective.

### Interactive mode

When called without arguments:

```example terminal output
$ owui groups remove-users
? Select group (Use arrow keys)
search: dev
    developers
    designers

? Select users to remove (Use arrow keys, space to select)
search: bo
    [ ] alice
    [x] bob
    [ ] charlie

Confirm removing 1 user from group 'developers'? (y/n) y
Successfully removed bob from group 'developers'
```

### Non-interactive mode

```bash
owui groups remove-users bob --group developers
```

```example terminal output
Confirm removing bob from group 'developers'? (y/n) y
Successfully removed bob from group 'developers'
```

---

## Permissions

Groups in Open WebUI have a `permissions` field that is a free-form JSON object. Permissions can be set during group creation (`owui groups add --permissions`) or updated later (`owui groups update --permissions`).

The permissions flag accepts a JSON string:

```bash
owui groups add --name team --description "My team" --permissions '{"workspace": {"models": true, "knowledge": true, "prompts": false}}'
```

In interactive mode, the user is asked whether they want to set custom permissions. If yes, a text input is shown where they can enter the JSON string.

When displaying group details (`owui groups members|show`), permissions are shown if present:

```example terminal output
Group: developers
Description: Development team
Type: local
Members: 5
Permissions: {"workspace":{"models":true,"knowledge":true,"prompts":false}}
```

In JSON output mode (`-o json`), permissions are included as a nested object.

---

## API Endpoints Reference

| Command | API Endpoint | Method |
|---------|-------------|--------|
| `groups list` | `/api/v1/groups/` | GET |
| `groups add` | `/api/v1/groups/create` | POST |
| `groups add` (with users) | `/api/v1/groups/id/{id}/users/add` | POST |
| `groups remove` | `/api/v1/groups/id/{id}/delete` | DELETE |
| `groups update` | `/api/v1/groups/id/{id}/update` | POST |
| `groups members` | `/api/v1/groups/id/{id}` + `/api/v1/groups/id/{id}/users` | GET + POST |
| `groups add-users` | `/api/v1/groups/id/{id}/users/add` | POST |
| `groups remove-users` | `/api/v1/groups/id/{id}/users/remove` | POST |

---

## Global flags

If no instance is specified with `--instance` or `-i` flag, the command will use the active instance. You can switch the active instance with `owui instances use <instance_name>`.

Output format can be set with `--output` or `-o` flag (`pretty` or `json`). Default is `pretty`.

## Autocomplete behavior

- **Group name autocomplete:** All commands that accept a group name provide shell autocomplete by fetching groups from the active instance API.
  - For write operations (`remove`, `update`, `add-users`, `remove-users`): Only local groups are shown (oauth groups are filtered out).
  - For read operations (`members`/`show`): All groups are shown (both local and oauth).
- **User autocomplete:** Commands that accept usernames (`add`, `add-users`) filter to only show users with role `user` (not `admin` or `pending`).
- **`remove-users` user autocomplete:** Shows only users that are currently members of the selected group.
