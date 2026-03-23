# curseforge-gateway

Homelab AI Platform — CurseForge modpack/mod validation gateway.

A minimal authenticated HTTP API that validates CurseForge project and file IDs and returns metadata. Kids browse CurseForge in their browser and bring IDs to the chatbot — this gateway validates those IDs and keeps the API key server-side.

Part of the [homelab-ai](https://github.com/lobo235/homelab-ai) platform.

## Quick Start

```bash
cp .env.example .env
# Fill in CF_API_KEY and GATEWAY_API_KEY
go run ./cmd/server
```

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/health` | No | Health check + CurseForge reachability |
| GET | `/modpacks/{projectID}` | Bearer | Validate modpack; return name, summary, game versions |
| GET | `/modpacks/{projectID}/files` | Bearer | List available server-pack files |
| GET | `/modpacks/{projectID}/files/{fileID}` | Bearer | Get a single modpack file by ID |
| GET | `/mods/{projectID}` | Bearer | Validate mod; return name, summary |
| GET | `/mods/{projectID}/files` | Bearer | List available files for a mod |
| GET | `/mods/{projectID}/files/{fileID}` | Bearer | Get a single mod file by ID |

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CF_API_KEY` | Yes | — | CurseForge API key |
| `GATEWAY_API_KEY` | Yes | — | Bearer token for callers |
| `PORT` | No | `8080` | Listen port |
| `LOG_LEVEL` | No | `info` | Log level: debug, info, warn, error |

## Build & Test

```bash
make build    # Build binary
make test     # Run tests
make lint     # Run linter
make cover    # Coverage report
make run      # Run locally
```

## Docker

```bash
docker build -t curseforge-gateway .
docker run --env-file .env -p 8080:8080 curseforge-gateway
```

## License

Private — part of the homelab-ai platform.
