# FAQ — LevelUp

French version: [FR/FAQ.md](FR/FAQ.md)

> LevelUp is a Halo stats dashboard. Stack: a Go backend (`apps/go-api`) and a React/Vite frontend (`apps/web`). There is no Python anymore. For the full install walkthrough see [INSTALL.md](INSTALL.md); for sync details see [SYNC_GUIDE.md](SYNC_GUIDE.md).

## Install & prerequisites

### What do I need installed?

- **Go 1.26+** (the DuckDB driver requires CGO — a C toolchain such as MinGW on Windows is needed for the full test suite)
- **Node.js + npm** (frontend)
- **GNU Make** and **Git**
- **Air** for Go hot reload: `go install github.com/air-verse/air@latest`

No Python is required. Full steps: [INSTALL.md](INSTALL.md).

### How do I start the app?

From the repo root:

```bash
make dev
```

This starts the Go API on port 8000 and the Vite frontend on http://localhost:5173. On first launch the in-browser setup wizard guides you through gamertag + Xbox sign-in. To stop everything: `make stop` (or `Ctrl+C` in the `make dev` terminal).

### The frontend won't start ("Module not found")

Install the frontend dependencies:

```bash
cd apps/web && npm install
```

## Configuration & tokens

### My Xbox token expired

Do **not** re-capture tokens to fix a transient 401 — a green sync means the tokens are good and the auth pool refreshes them automatically.

If the refresh token is genuinely expired, reconnect in the app: **Settings → Xbox connection → Reconnect** (re-runs the Device Code flow). For headless players:

```bash
go run ./apps/go-api/cmd/token-capture/ <Gamertag>
```

Tokens are the single source of truth in `data/auth/watcher_tokens/{xuid}.json` (see [ADR 0023](adr/0023-auth-tokens-single-source.md)). The player must be declared in `db_profiles.json` (with `xuid`) first.

### How do I add a new player?

1. Declare the player in `db_profiles.json` (with their `xuid`).
2. Onboard via the in-app wizard (Xbox SSO) or, headless, `go run ./apps/go-api/cmd/token-capture/ <Gamertag>`.
3. Sync happens automatically once the token is in the store (see below).

## Sync

### Do I need to run a sync command?

No. Sync runs **inside the Go server**: a presence watcher queues a delta sync when a player finishes a match, and an auto-sync scheduler periodically delta-syncs every player. Day-to-day operation needs no manual action. Details: [SYNC_GUIDE.md](SYNC_GUIDE.md).

Manual CLI commands exist for bootstrap and gap-filling: `levelup sync-delta` / `sync-full` / `backfill` (from `apps/go-api/cmd/levelup`). Do not run shared-DB CLI tools while the server holds the DuckDB handle — stop the server first.

### What's the difference between delta and full sync?

- **Delta** — fetches only matches newer than the last watermark. Fast; the default for the watcher and the scheduler.
- **Full** — walks the last N API matches and inserts any that are missing (gap-filling). Use after a long outage, an import, or a watermark issue.

## Data

### Where is my data stored?

Under `data/`, in a title-agnostic layout `data/titles/{slug}/` (default slug `halo_infinite`):

```
data/
├── auth/watcher_tokens/{xuid}.json              # OAuth/MSAL token store (ADR 0023)
└── titles/halo_infinite/
    ├── warehouse/
    │   ├── metadata.duckdb                       # referentials
    │   ├── shared_matches_v2.duckdb              # shared match data
    │   └── shared_pve.duckdb                     # Firefight stats
    └── players/{gamertag}/stats.duckdb           # per-player enrichments
```

Full schema and rationale: [ARCHITECTURE_V6.md](ARCHITECTURE_V6.md).

## Development

### How do I run the tests?

```bash
# Fast subset, no DuckDB (no CGO)
cd apps/go-api
CGO_ENABLED=0 go test ./internal/domain/... ./internal/analysis/... ./contracttest/... -count=1

# Full suite with DuckDB (requires CGO)
CGO_ENABLED=1 LEVELUP_DEMO_MODE=true go test ./... -timeout 5m -count=1

# Frontend
cd apps/web && npm run typecheck && npm test
```

The convenience target `make go-api-test` runs the fast subset. Full matrix (Windows MinGW, coverage ratchet): [testing.md](testing.md).

### How do I report a bug?

Open a GitHub issue with your OS, the full error message, and steps to reproduce. Logs are written per category under `logs/*.log`.

## Misc

### Does LevelUp collect any data?

No. All data stays on your machine; no telemetry is sent.

### Is the project affiliated with 343 Industries or Microsoft?

No. LevelUp is an unofficial, community project.
