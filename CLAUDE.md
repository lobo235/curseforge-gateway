# curseforge-gateway

A minimal Go HTTP API that validates CurseForge modpack/mod project IDs and returns metadata.
Part of the [homelab-ai](https://github.com/lobo235/homelab-ai) platform.

## Module

`github.com/lobo235/curseforge-gateway`

## Quick Start

```bash
cp .env.example .env
# Fill in required values
go run ./cmd/server
```

## Build, Test, Run

> Go is installed at `~/bin/go/bin/go` (also on `$PATH` via `.bashrc`).

```bash
# Build
make build

# Run tests
make test

# Run tests with verbose output
go test -v ./...

# Run linter
make lint

# Coverage report (opens in browser)
make cover

# Run the server (requires .env or env vars)
make run

# Build binary
go build -o curseforge-gateway ./cmd/server
```

## Project Layout

```
curseforge-gateway/
├── Dockerfile
├── Makefile
├── go.mod / go.sum
├── .env.example              # dev template — never commit real values
├── .gitignore
├── .golangci.yml             # strict linter config
├── .githooks/pre-commit      # runs lint + tests; activate with `make hooks`
├── CLAUDE.md                 # this file
├── README.md
├── CHANGELOG.md
├── cmd/
│   └── server/
│       └── main.go           # entry point
└── internal/
    ├── config/
    │   ├── config.go         # ENV var loading & validation
    │   └── config_test.go
    ├── curseforge/
    │   ├── client.go         # CurseForge API wrapper with caching
    │   ├── client_test.go
    │   └── errors.go         # sentinel errors (ErrNotFound, ErrWrongClass)
    └── api/
        ├── server.go         # HTTP mux + Run()
        ├── middleware.go     # Bearer auth + request logging + X-Trace-ID
        ├── handlers.go       # all route handlers
        ├── errors.go         # writeError / writeJSON helpers
        ├── health.go         # GET /health (unauthenticated)
        └── server_test.go    # handler tests via httptest
```

## Configuration

All config via ENV vars. Loaded from `.env` in development (via `godotenv`; missing file silently ignored). In production, secrets are injected by Nomad Vault Workload Identity — the app never talks to Vault directly.

| Var | Required | Default | Purpose |
|-----|----------|---------|---------|
| `CF_API_KEY` | yes | — | CurseForge API key (x-api-key header) |
| `GATEWAY_API_KEY` | yes | — | Bearer token for callers of this API |
| `PORT` | no | `8080` | Listen port |
| `LOG_LEVEL` | no | `info` | Verbosity: `debug`, `info`, `warn`, `error` |

## Architecture

```
cmd/server/main.go               — entry point, wires deps, handles SIGINT/SIGTERM
internal/config/config.go        — ENV-based config with validation
internal/curseforge/client.go    — CurseForge API wrapper with in-memory caching
internal/curseforge/errors.go    — sentinel errors
internal/api/server.go           — HTTP server, route registration
internal/api/middleware.go       — bearerAuth + requestLogger + X-Trace-ID propagation
internal/api/handlers.go         — route handlers
internal/api/errors.go           — writeError / writeJSON helpers
internal/api/health.go           — GET /health handler (unauthenticated)
```

## API Routes

All routes except `/health` require `Authorization: Bearer <GATEWAY_API_KEY>`.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/health` | No | Ping CurseForge API; return `{"status":"ok","version":"..."}` |
| GET | `/modpacks/{projectID}` | Bearer | Validate modpack (classId=4471); return name, summary, game versions |
| GET | `/modpacks/{projectID}/files` | Bearer | List available files for a modpack |
| GET | `/modpacks/{projectID}/files/{fileID}` | Bearer | Get a single modpack file by ID |
| GET | `/mods/{projectID}` | Bearer | Validate mod (classId=6); return name, summary |
| GET | `/mods/{projectID}/files` | Bearer | List available files for a mod |
| GET | `/mods/{projectID}/files/{fileID}` | Bearer | Get a single mod file by ID |
| GET | `/search?query=...&type=modpack\|mod` | Bearer | Search CurseForge by name (default type: modpack) |

### Caching

- Project metadata: 30-minute in-memory TTL
- File lists: 5-minute in-memory TTL

## Testing Approach

Tests use `httptest.NewServer` with mock implementations of the `curseforgeClient` interface — no live CurseForge API dependencies required.

```
internal/config/config_test.go   — unit tests for config loading
internal/curseforge/client_test.go — unit tests for CurseForge client (httptest mock server)
internal/api/server_test.go      — handler tests via httptest
```

Key patterns:
- Each test registers a mock implementation, calls the handler, asserts return value
- Table-driven tests for input validation cases
- Test both success paths and error paths (upstream 404, 5xx, wrong classId)

## Coding Conventions

- No external router, ORM, or framework — minimal dependency footprint
- Error responses always use `writeError(w, status, code, message)` with machine-readable `code`
- Route handlers return `http.HandlerFunc`
- All upstream errors wrapped with `fmt.Errorf("context: %w", err)`
- `X-Trace-ID` header propagated from request context to all upstream calls and log lines
- Structured JSON logging via `log/slog`; version logged on startup; every request access-logged

## Security Rules

> **Claude must enforce all rules below on every commit and push without exception.**

1. **Never commit secrets:** No `.env`, tokens, API keys, passwords, or credentials of any kind.
2. **Never commit infrastructure identifiers:** No real hostnames, IP addresses, datacenter names, node pool names, Consul service names, Vault paths with real values, Traefik routing rules with real domains, or any value that reveals homelab architecture. Use generic placeholders (`dc1`, `default`, `example.com`, `your-node-pool`, `your-service`).
3. **Unknown files:** If `git status` shows a file Claude didn't create, ask the operator before staging it.
4. **Pre-commit checks (must all pass before committing):**
   - `go test ./...` — all tests must pass
   - `golangci-lint run` — no lint errors
5. **Docs accuracy:** Review all changed `.md` files before committing — documentation must reflect the current state of the code in the same commit.
6. **Version bump:** Before any `git commit`, review the changes and determine the appropriate SemVer bump (MAJOR/MINOR/PATCH). Present the rationale and proposed new version to the operator and wait for confirmation before tagging or referencing the new version.
7. **Push confirmation:** Before any `git push`, show the operator a summary of what will be pushed (commits, branch, remote) and wait for explicit confirmation.
8. **Commit messages:** Must not contain real hostnames, IPs, or infrastructure identifiers.

## Versioning & Releases

SemVer (`MAJOR.MINOR.PATCH`). Git tags are the source of truth.

```bash
git tag v1.2.3 && git push origin v1.2.3
```

This triggers the Docker workflow which publishes:
- `ghcr.io/lobo235/curseforge-gateway:v1.2.3`
- `ghcr.io/lobo235/curseforge-gateway:v1.2`
- `ghcr.io/lobo235/curseforge-gateway:latest`
- `ghcr.io/lobo235/curseforge-gateway:<short-sha>`

Version is embedded at build time: `-ldflags "-X main.version=v1.2.3"` — defaults to `"dev"` for local builds. Exposed in `GET /health` response and logged on startup.

## Docker

```bash
# Build (version defaults to "dev")
docker build -t curseforge-gateway .

# Build with explicit version
docker build --build-arg VERSION=v1.2.3 -t curseforge-gateway .

# Run
docker run --env-file .env -p 8080:8080 curseforge-gateway
```

Multi-stage build: `golang:1.24-alpine` → `alpine:3.21`. Statically compiled (`CGO_ENABLED=0`).

## Known Limitations

- Cache is in-memory only; restarting the service clears the cache.
