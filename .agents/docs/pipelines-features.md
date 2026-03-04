# owui pipelines command

`owui pipelines` is the subcommand that manages Open WebUI pipeline registrations, the contained pipes, and their valves.  
Refer to [Open WebUI v0.8.5 OpenAPI](file:///Users/chris/projects/owui-go/openapi-reference/0.8.5-openapi.json).

This feature intentionally separates two entities:

- **Pipeline registration**: a configured pipeline server entry (from `GET /api/v1/pipelines/list`)
- **Pipe**: a runnable pipe exposed by a registration (from `GET /api/v1/pipelines/?urlIdx=...`)

The CLI must show both layers clearly, because valve operations target a **pipe ID** while some management operations target a **registration ID**.

---

## Domain model

### 1) Pipeline registration

A registration is a configured pipeline endpoint known to Open WebUI.  
It is identified by:

- `registration_id` (server-provided if available)
- `urlIdx` (index used by pipelines API routing)
- registration URL / metadata

### 2) Pipe

A pipe is a concrete unit exposed by a registration.  
It is identified by:

- `pipe_id` (used in `.../pipelines/{pipeline_id}/valves*`)
- parent registration (`registration_id`, `urlIdx`)
- pipe metadata (name, description, raw data)

### 3) Why `urlIdx` matters

Valves and delete operations require `urlIdx` in addition to an ID:

- valves endpoints: `pipeline_id` path + mandatory `urlIdx` query param
- delete endpoint: request body `{id, urlIdx}`

The CLI therefore auto-resolves `urlIdx` from inventory and exposes `--url-idx` as explicit override.

---

## List pipelines and pipes

```bash
owui pipelines list|ls [--filter <query>]
```

Lists both:

1. registered pipeline servers
2. all discovered pipes per registration

Behavior:

- Builds inventory using:
  - `GET /api/v1/pipelines/list` (registrations)
  - `GET /api/v1/pipelines/?urlIdx=<n>` for each registration index
- `--filter` applies client-side substring filtering across registration id/url and pipe id/name
- pretty output renders **two blocks** (`Registered Pipelines` and `Pipes`)
- `-o json` returns one normalized object containing registrations + pipes + unresolved raw payloads

```example terminal output
$ owui pipelines ls
Registered Pipelines
REGISTRATION ID            URL                               URL IDX   PIPES
pipeline-reg-01            http://pipelines-a:9099          0         3
pipeline-reg-02            http://pipelines-b:9099          1         2

Pipes
PIPE ID                    NAME                             REGISTRATION ID      URL IDX
rag_ingest                 RAG Ingest                       pipeline-reg-01      0
sql_assistant              SQL Assistant                    pipeline-reg-01      0
image_tools                Image Tools                      pipeline-reg-01      0
llm_router                 LLM Router                       pipeline-reg-02      1
doc_summary                Document Summary                 pipeline-reg-02      1

Showing 2 registration(s), 5 pipe(s).
```

JSON output:

```bash
owui pipelines list -o json
```

---

## Show pipe details

```bash
owui pipelines show <pipe_id> [--url-idx <n>]
```

Shows detail view of a single pipe with parent registration context.

- positional arg accepts `pipe_id`
- tab completion lists known pipe IDs from inventory
- if a `pipe_id` exists in multiple registrations and `--url-idx` is missing, command fails with clear ambiguity error
- detail output includes `pipe_id`, display name, description, registration ID, URL, `urlIdx`, and raw attributes

### Interactive mode

When called without arguments:

```example terminal output
$ owui pipelines show
? Select pipe (Use arrow keys)
search: rag
    rag_ingest (pipeline-reg-01 / idx=0)
    rag_ingest (pipeline-reg-02 / idx=1)
```

If duplicate `pipe_id` exists, selector shows registration context to disambiguate.

### Non-interactive mode

```bash
owui pipelines show rag_ingest --url-idx 0
```

---

## Manage valves

`valves` is a nested command group:

- `owui pipelines valves show <pipe_id> [--url-idx <n>]`
- `owui pipelines valves spec <pipe_id> [--url-idx <n>]`
- `owui pipelines valves update <pipe_id> [--url-idx <n>] --data '<json>'`

All commands resolve `urlIdx` automatically unless overridden.

### Show current valves

```bash
owui pipelines valves show rag_ingest
```

Uses:

- `GET /api/v1/pipelines/{pipeline_id}/valves?urlIdx=<n>`

### Show valves spec

```bash
owui pipelines valves spec rag_ingest
```

Uses:

- `GET /api/v1/pipelines/{pipeline_id}/valves/spec?urlIdx=<n>`

### Update valves

```bash
owui pipelines valves update rag_ingest --data '{"temperature":0.2,"top_k":20}'
```

Uses:

- `POST /api/v1/pipelines/{pipeline_id}/valves/update?urlIdx=<n>`
- body is a free JSON object (`additionalProperties: true`)

If `--data` is omitted, interactive mode asks for JSON text input and validates it before sending.

```example terminal output
$ owui pipelines valves update rag_ingest
? Enter valves JSON:
{"temperature":0.2,"top_k":20}
Confirm updating valves for pipe 'rag_ingest' (urlIdx=0)? (y/n) y
Successfully updated valves for pipe 'rag_ingest'
```

---

## Add pipeline registration

```bash
owui pipelines add --url <pipeline_url> [--url-idx <n>]
```

Creates a new pipeline registration.

- API endpoint: `POST /api/v1/pipelines/add`
- request body: `AddPipelineForm`
- `urlIdx` strategy:
  - if `--url-idx` provided: use it
  - otherwise: auto-select next free index (`max(existing urlIdx) + 1`, fallback `0`)

### Interactive mode

```example terminal output
$ owui pipelines add
? Pipeline URL: http://pipelines-c:9099
? Use custom urlIdx? (y/n) n
Successfully added pipeline registration for 'http://pipelines-c:9099' (urlIdx=2)
```

---

## Upload pipeline file

```bash
owui pipelines upload --file <path> [--url-idx <n>]
```

Uploads a pipeline file via multipart form data.

- API endpoint: `POST /api/v1/pipelines/upload` (multipart)
- request fields: `file`, `urlIdx`
- `urlIdx` auto/override behavior is the same as `add`

### Interactive mode

```example terminal output
$ owui pipelines upload
? Path to pipeline file: ./pipelines/custom_pipeline.py
? Use custom urlIdx? (y/n) n
Successfully uploaded pipeline file './pipelines/custom_pipeline.py' to urlIdx=2
```

---

## Remove pipeline registration

```bash
owui pipelines remove|rm <registration_id> [--url-idx <n>]
```

Removes one registration.

- API endpoint: `DELETE /api/v1/pipelines/delete`
- request body: `DeletePipelineForm` (`id`, `urlIdx`)
- tab completion suggests registration IDs
- if `--url-idx` missing, CLI resolves it from inventory by registration ID
- confirmation prompt is always shown

### Interactive mode

When no positional argument is provided:

```example terminal output
$ owui pipelines rm
? Select registration to delete (Use arrow keys)
search: reg-0
    pipeline-reg-01 (idx=0, http://pipelines-a:9099)

Confirm deleting registration 'pipeline-reg-01' (urlIdx=0)? (y/n) y
Successfully deleted pipeline registration 'pipeline-reg-01'
```

---

## API Endpoints reference

| Command | API Endpoint | Method | Request Body / Params |
|---------|-------------|--------|-----------------------|
| `pipelines list` inventory | `/api/v1/pipelines/list` | GET | none |
| `pipelines list` pipes per registration | `/api/v1/pipelines/` | GET | query: `urlIdx` (nullable in spec; CLI uses concrete int) |
| `pipelines show` (inventory lookup) | `/api/v1/pipelines/list` + `/api/v1/pipelines/` | GET + GET | resolve by `pipe_id` and `urlIdx` |
| `pipelines valves show` | `/api/v1/pipelines/{pipeline_id}/valves` | GET | path: `pipeline_id`, query: `urlIdx` |
| `pipelines valves spec` | `/api/v1/pipelines/{pipeline_id}/valves/spec` | GET | path: `pipeline_id`, query: `urlIdx` |
| `pipelines valves update` | `/api/v1/pipelines/{pipeline_id}/valves/update` | POST | path: `pipeline_id`, query: `urlIdx`, body: JSON object |
| `pipelines add` | `/api/v1/pipelines/add` | POST | `AddPipelineForm` |
| `pipelines upload` | `/api/v1/pipelines/upload` | POST | multipart: `urlIdx`, `file` |
| `pipelines remove` | `/api/v1/pipelines/delete` | DELETE | `DeletePipelineForm` |

---

## API Schemas reference

### AddPipelineForm

```json
{
  "url": "http://pipelines-a:9099",
  "urlIdx": 0
}
```

### DeletePipelineForm

```json
{
  "id": "pipeline-reg-01",
  "urlIdx": 0
}
```

### Body_upload_pipeline_api_v1_pipelines_upload_post

```json
{
  "urlIdx": 0,
  "file": "<binary>"
}
```

### Important schema note

Most pipelines responses in OpenAPI are typed as `{}`.  
CLI implementation must therefore parse defensively and normalize untyped responses into internal structs for display, completion, and routing.

---

## Global flags

- `--instance`, `-i`: target instance; defaults to active instance
- `--output`, `-o`: `pretty` or `json`
- `--filter`, `-f`: optional client-side filtering for `list`

If no instance is specified, active instance is used. Switch via:

```bash
owui instances use <instance_name>
```

---

## Autocomplete behavior

- **Pipe ID autocomplete:** used by `pipelines show` and `pipelines valves *`
  - source: normalized inventory from `pipelines/list` + `pipelines/?urlIdx=...`
  - when duplicates exist, completion description includes registration and `urlIdx`
- **Registration ID autocomplete:** used by `pipelines remove|rm`
- **Multi-arg exclusion:** for any command accepting multiple IDs in future, already-selected IDs are excluded from suggestions

---

## Implementation notes

### 1) Untyped response normalization

Because responses are untyped in OpenAPI, normalize via `map[string]any` and key fallbacks.

Recommended fallback order:

- registration ID: `id` -> `pipeline_id` -> `name` -> synthetic `registration-<urlIdx>`
- registration URL: `url` -> `base_url` -> `endpoint`
- pipe ID: `id` -> `pipe_id` -> `pipeline_id` -> `name`
- pipe name: `name` -> `title` -> `id`

Always keep a `raw map[string]any` copy in normalized structs for JSON output/debugging.

### 2) Inventory build flow

1. Fetch registrations via `GET /api/v1/pipelines/list`.
2. Derive `urlIdx` for each registration:
   - explicit field if present
   - otherwise stable index by list order.
3. Fetch pipes for each registration via `GET /api/v1/pipelines/?urlIdx=<n>`.
4. Build mapping:
   - `pipe_id -> []candidates{registration_id, urlIdx, pipe}`
   - `registration_id -> registration`.

### 3) Ambiguity handling

When command targets `pipe_id` and multiple candidates exist:

- if `--url-idx` provided: select matching candidate
- else return explicit error:
  - `pipe_id 'rag_ingest' is ambiguous across urlIdx [0,1]; pass --url-idx`

### 4) `urlIdx` default policy

- read commands: resolve from inventory
- `add`/`upload`: next free index by default; allow override
- `remove`: resolve by registration ID; allow override

### 5) JSON mode

For untyped APIs, JSON mode should output normalized objects plus `raw` payload segments.  
Do not silently drop unknown fields.

---

## Test cases and scenarios

### Inventory construction

1. Single registration with multiple pipes.
2. Multiple registrations with overlapping `pipe_id`.
3. Missing expected keys in raw payload (fallback key path exercised).
4. `pipelines/?urlIdx=...` fails for one registration while others succeed.

### Command behavior

1. `pipelines list` pretty output renders two blocks and counts.
2. `pipelines list -o json` returns normalized + raw sections.
3. `pipelines show <pipe_id>` works for unique pipe ID.
4. `pipelines show <pipe_id>` fails with ambiguity without `--url-idx`.
5. `pipelines valves show/spec/update` successful path.
6. `pipelines valves update` rejects invalid JSON in `--data`.
7. `pipelines add` uses explicit and auto-assigned `urlIdx`.
8. `pipelines upload` multipart request includes `file` and `urlIdx`.
9. `pipelines remove` sends `{id,urlIdx}` and respects cancel/confirm.

### Completion

1. `show` and `valves` complete pipe IDs.
2. `remove` completes registration IDs.
3. Completion annotations include context for duplicates.
4. Already-selected values are excluded where applicable.

### UX and failure modes

1. No active instance configured.
2. Zero registrations returned.
3. API 4xx/5xx on each endpoint path.
4. Selected `--url-idx` does not match requested ID.

---

## Non-goals (v1)

- No bulk delete of multiple registrations in one command invocation.
- No automatic schema-driven valve form UI from spec fields (JSON input is sufficient).
- No mutation of pipe metadata beyond valves update.
- No cross-instance merge view in a single command run.

