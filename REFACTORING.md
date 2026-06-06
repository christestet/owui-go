# Refactoring TODO

Follow-up work from the senior engineer + architect code review. The two
ship-blockers and all **Wave 1** quick fixes are already merged (commits
`c1d8eeb`, `53ba87b`, `1e880a5`). This file tracks the remaining backlog.

Priority key: **P1** = highest leverage, do first · **P2** = valuable
consolidation · **P3** = small / opportunistic.

---

## Wave 2 — Foundational architecture (P1)

These two are the highest-leverage changes: they unlock lightweight tests and
remove the dominant source of duplication. Do them before Wave 3, ideally one
PR each.

### 2.1 Introduce an API client interface
**Why:** There is no interface in the codebase — `*api.Client` is concrete
everywhere, so every CLI command test must spin up a real `httptest.Server` +
temp config and invoke `cmd.RunE` on package-global command singletons with
defer-resets (fragile, order-sensitive). See `internal/cli/models/models_test.go`
(`newModelsServer`, global-flag mutation) for the pattern this replaces.

**Approach:**
- Define resource interfaces in `internal/api` (or a sub-package), e.g.
  `ModelsAPI`, `UsersAPI`, `GroupsAPI`, `ToolsAPI`, `FunctionsAPI`,
  `PipelinesAPI` — each listing only the methods that resource's commands call.
  `*api.Client` already satisfies them (no behavior change).
- Make `shared.ResolveClient` (`internal/cli/shared/client.go`) return an
  interface (or introduce a small client-factory the commands depend on) so a
  fake can be injected in tests.
- Add a `fakeClient` test helper and migrate a couple of command tests off the
  `httptest` harness to prove the seam works.

**Risk:** Touches every cli package's call site and test setup. Mechanical but
broad. Keep `*api.Client` as the production implementation.

**Verify:** `go build ./... && go test ./...`; confirm at least one command
package has a fast, server-free unit test.

### 2.2 Extract an output-rendering package
**Why:** The `if outputFormat == "json" { MarshalIndent } else { tabwriter }`
block, the empty-state message, and the "Showing N …" footer are reimplemented
in ~13 list/show commands (e.g. `internal/cli/models/list.go`,
`internal/cli/instances/instances.go`, and the groups/users/pipelines/tools/
functions list files). Wording drifts between commands. There is a `prompts`
package for input but no symmetric package for output.

**Approach:**
- Add `internal/cli/render` with:
  - `RenderJSON(w io.Writer, v any) error`
  - `RenderTable[T any](w io.Writer, headers []string, rows []T, cols func(T) []string)`
  - a shared empty-state / footer helper.
- Migrate the list/show commands to use it. Standardize the footer text.

**Verify:** `go test ./...`; then `make docs` + `make docs-readme-check` since
help/example output is doc-generated — confirm no unintended drift.

---

## Wave 3 — Consolidation (P2)

Builds on Wave 2.

### 3.1 Formalize the `--filter` contract
**Why:** `filter` is a single persistent root flag (`internal/cli/root/root.go`,
help text just "filter results") but each command interprets it differently —
models `enabled|disabled|public|private`, groups `local|oauth`, functions
`global|enabled|…`, users free-text — and re-validates inline with a duplicated
`switch … default: invalid filter` block. Valid values are invisible to users,
shell completion, and generated docs.

**Approach:** Make `filter` a *local* flag per command with command-specific
shell completion and documented allowed values, or a small filter-spec helper in
`shared`. Update generated docs afterward.

**Verify:** `go test ./...`; `owui models list -f <TAB>` offers valid values;
`make docs`.

### 3.2 De-duplicate find-by-name/ID helpers
**Why:** `findModelByID` / `findModelByNameOrID` (`internal/cli/models/models.go`)
mirror `shared.FindGroupByName` / `FindUserByName` (`internal/cli/shared/filters.go`).
The same resolve→confirm→apply skeleton repeats across mutation commands.

**Approach:** Consolidate the find-by-name/ID helpers into `shared` (generic over
a name/ID accessor). Optionally factor the common mutation skeleton.

**Verify:** `go test ./...`.

### 3.3 Type the pipelines layer & share the HTTP helper
**Why:** `internal/api/pipelines.go` returns `any` / `map[string]any` everywhere
(`…Raw`, `decodeUntypedJSON`) while the rest of `internal/api` is strongly typed,
and the multipart upload path (pipelines.go ~119-166) reimplements auth/status/
size-limit handling instead of using `sendRequest` (`internal/api/client.go`).

**Approach:** Extract a shared `doRequest(req)` (headers + status mapping +
`io.LimitReader`) used by both `sendRequest` and the multipart uploader. Type the
stable responses (registration/list) where the schema is reliable; keep `any`
only for genuinely dynamic valve payloads.

**Verify:** `go test ./...`; manual `owui pipelines list/add/upload` against an
instance if available.

---

## P3 — Small / opportunistic

- **gofmt drift (pre-existing):** `internal/api/models.go` (the
  `Capabilities ModelCapabilities` alignment) and
  `internal/cli/groups/show_models.go` are gofmt-dirty and untouched by recent
  work. Run `gofmt -w` on both; they would trip a CI gofmt gate.
- **`omitempty` on a struct value:** `ModelMeta.Capabilities`
  (`internal/api/models.go:22`) — `omitempty` has no effect on a non-pointer
  struct. Make it `*ModelCapabilities` if it's ever used as a request body;
  otherwise drop the misleading tag.
- **Verify list pagination contracts:** `ListGroups`, tools, and functions list
  endpoints fetch a single response while users/models auto-paginate
  (`internal/api/users.go`, `groups.go`, `tools.go`, `functions.go`). Confirm
  against the OpenAPI / server whether those endpoints paginate; if so, route
  them through `fetchAllPages`.
- **Healthcheck auth header:** `Healthcheck` (`internal/api/client.go`) sends the
  `Authorization` header to the typically-unauthenticated `/health`. Harmless;
  drop for clarity.
- **Background update-check race (full fix):** Wave 1 only *shrank* the
  lost-update window (`internal/cli/root/root.go`). A complete fix needs saves
  to be serialized or a file lock, or persisting `LastUpdateCheck` via a narrow
  read-modify-write that doesn't round-trip the whole config.

---

## Suggested sequencing
1. **2.1** API client interface (unblocks cheap tests for everything after).
2. **2.2** render package (largest dedup win).
3. **3.2** then **3.1** then **3.3** (consolidation, now easy to test).
4. Sweep **P3** items as small standalone commits.

## Global verification for any PR here
- `go build ./... && go vet ./... && gofmt -l internal/` (expect empty)
- `go test ./...` and `go test -race ./internal/api/ ./internal/config/ ./internal/cli/root/`
- `make docs && make docs-readme-check` when help text / output changes
