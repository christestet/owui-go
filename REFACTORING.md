# Refactoring TODO

Follow-up work from the senior engineer + architect code review. The two
ship-blockers and all **Wave 1** quick fixes are already merged (commits
`c1d8eeb`, `53ba87b`, `1e880a5`). This file tracks the remaining backlog.

Last reviewed: 2026-07-20 against the Open WebUI 0.10.2 API specification.

Priority key: **P1** = highest leverage, do first · **P2** = valuable
consolidation · **P3** = small / opportunistic.

---

## Completed since the original review

- **Stable user/group resolution and interactive IDs:** Commit `3d65c46`
  introduced shared ID-first resolvers, ambiguity errors, human-readable
  selectors with stable ID values, and unambiguous completions. User identifiers
  now support exact ID, unique email, and unique username/name; groups support
  exact ID or unique name. Existing group members are filtered from both
  add-users entry points before batch requests.
- **gofmt drift:** `internal/` is currently gofmt-clean; the previously noted
  drift in `internal/api/models.go` and `internal/cli/groups/show_models.go` no
  longer exists.
- **List pagination contracts:** Verified against
  `openapi-reference/0.10.2-openapi.json`: groups, tools, and functions return
  complete arrays and do not expose pagination parameters. No pagination helper
  migration is currently required for those endpoints.

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

Open WebUI 0.10.2 reinforces that these are resource-specific concerns: users
and models expose different server-side query/order parameters, groups expose a
`share` parameter, and tools/functions expose no equivalent list filter.

**Approach:** Make `filter` a *local* flag per command with command-specific
shell completion and documented allowed values, or a small filter-spec helper in
`shared`. Update generated docs afterward.

**Verify:** `go test ./...`; `owui models list -f <TAB>` offers valid values;
`make docs`.

### 3.2 Finish entity resolver consolidation
**Status:** Partially completed by `3d65c46`. User and group resolution now lives
in `internal/cli/shared/filters.go`; interactive selectors carry stable IDs and
duplicate names fail safely in non-interactive mode.

**Remaining:** `findModelByID` / `findModelByNameOrID`
(`internal/cli/models/models.go`) are still package-local, and name resolution
still returns the first matching model. The same resolve→confirm→apply skeleton
also repeats across mutation commands.

**Approach:** Add an ambiguity-safe shared model resolver (or a small generic
resolver over name/ID accessors), migrate model commands to it, and then decide
whether factoring the common mutation skeleton improves clarity enough to
justify the abstraction.

**Verify:** `go test ./...`.

### 3.3 Type the pipelines layer & share the HTTP helper
**Why:** `internal/api/pipelines.go` returns `any` / `map[string]any` everywhere
(`…Raw`, `decodeUntypedJSON`) while the rest of `internal/api` is strongly typed,
and the multipart upload path (pipelines.go ~119-166) reimplements auth/status/
size-limit handling instead of using `sendRequest` (`internal/api/client.go`).

**OpenAPI constraint:** The 0.10.2 specification still declares the successful
responses for pipeline registration, list, add, upload, and valves endpoints as
empty schemas (`{}`). Full response types therefore cannot be generated or
validated from the specification alone.

**Approach:** Extract a shared `doRequest(req)` (headers + status mapping +
`io.LimitReader`) used by both `sendRequest` and the multipart uploader. Keep raw
transport decoding at the API boundary, normalize empirically stable
registration/list fields into the existing `Registration` / `Pipe` inventory
types, and protect those assumptions with fixture-based contract tests. Keep
`any` for genuinely dynamic valve payloads and undocumented response fields.

**Verify:** `go test ./...`; manual `owui pipelines list/add/upload` against an
instance if available.

### 3.4 Add a targeted OpenAPI contract-drift check
**Why:** API paths and Go request/response types are maintained manually. The
0.10.2 specification adds and removes paths and schemas while several runtime
contracts used by `owui` retain the same JSON shape under different schema
names. A version bump can therefore silently invalidate an endpoint, method,
parameter, or top-level response shape.

**Approach:** Add a small validation tool or test that checks the endpoints used
by `internal/api` against the supported OpenAPI file: path + method, security,
required parameters, and top-level response shape for typed endpoints. Maintain
an explicit allowlist for intentionally undocumented/dynamic pipeline schemas;
do not generate the full client from the current specification.

**Verify:** Run the check against the current supported specification and a
fixture containing a deliberate breaking change.

---

## P3 — Small / opportunistic

- **`omitempty` on a struct value:** `ModelMeta.Capabilities`
  (`internal/api/models.go:22`) — `omitempty` has no effect on a non-pointer
  struct. Open WebUI 0.10.2 explicitly defines `capabilities` as optional and
  nullable, and `ModelMeta` permits additional properties. Use
  `*ModelCapabilities` to preserve absent/null semantics and decide whether
  unknown metadata must be retained via `json.RawMessage` or a custom unmarshal.
- **Healthcheck auth header:** `Healthcheck` (`internal/api/client.go`) sends the
  `Authorization` header through `sendRequest`, while `/health` has no security
  requirement in both the 0.9.5 and 0.10.2 specifications. Harmless; drop for
  clarity or let the planned shared request helper opt out of auth explicitly.
- **Background update-check race (full fix):** Wave 1 only *shrank* the
  lost-update window (`internal/cli/root/root.go`). A complete fix needs saves
  to be serialized or a file lock, or persisting `LastUpdateCheck` via a narrow
  read-modify-write that doesn't round-trip the whole config.

---

## Suggested sequencing
1. **2.1** API client interface (unblocks cheap tests for everything after).
2. **2.2** render package (largest dedup win).
3. Finish **3.2**, then **3.1**, **3.4**, and **3.3** (consolidation and API
   contract hardening, now easy to test).
4. Sweep **P3** items as small standalone commits.

## Global verification for any PR here
- `go build ./... && go vet ./... && gofmt -l internal/` (expect empty)
- `go test ./...` and `go test -race ./internal/api/ ./internal/config/ ./internal/cli/root/`
- `make docs && make docs-readme-check` when help text / output changes
