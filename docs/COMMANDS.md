# Common commands — LevelUp

French version: [FR/COMMANDS.md](FR/COMMANDS.md)

> Cheat-sheet for the current stack: Go backend (`apps/go-api`) + React/Vite frontend (`apps/web`).
> Operational tooling is the `levelup` CLI (`apps/go-api/cmd/levelup`). Make targets live in the
> root `Makefile`. DuckDB access requires CGO (see [Tests](#tests)).

---

## Run the app

```bash
make dev          # Go API (air, :8000) + Vite frontend (:5173) — Ctrl+C stops both
make go-api-dev   # Go API only (air hot-reload)
make web          # Frontend only (Vite, :5173)
make stop         # Stop dev servers (kills by port, API + 5173)
make restart      # stop + dev
```

Open http://localhost:5173 once `make dev` is running.

---

## Build

```bash
make go-api-build   # CGO_ENABLED=1 go build -> apps/go-api/bin/server
make install-web    # npm install in apps/web
make generate-types # TypeScript types from apps/go-api/api/openapi.yaml
make check-types    # tsc -b (typecheck only)
```

---

## `levelup` CLI

Built from `apps/go-api/cmd/levelup`. Run via `go run` (CGO required) or build a binary.
Use `LEVELUP_REPO_ROOT` to point at the data repo (auto-detected if omitted).

```bash
cd apps/go-api
CGO_ENABLED=1 go run ./cmd/levelup <command> [flags]
# Per-command help:
CGO_ENABLED=1 go run ./cmd/levelup <command> --help
```

### Sync (Halo API)

```bash
# Delta sync — new matches only
go run ./cmd/levelup sync-delta --gamertag YourGamertag
go run ./cmd/levelup sync-delta --all --max-matches 25
# flags: --match-type all|matchmaking|custom|local  --rps N  --token-pool-size N

# Full sync — walk last N API matches, insert what's missing (fills gaps)
go run ./cmd/levelup sync-full --gamertag YourGamertag --max-matches 500

# Backfill Xbox achievements (admin one-shot)
go run ./cmd/levelup sync-achievements --all [--dry-run]
```

### Backfill (mostly local, Go-only; CSR/weapons need Halo tokens)

```bash
go run ./cmd/levelup backfill --gamertag X --citations        [--force]
go run ./cmd/levelup backfill --all          --lusr           [--force]
go run ./cmd/levelup backfill --gamertag X --perf             [--force]
go run ./cmd/levelup backfill --gamertag X --engagement-scores
go run ./cmd/levelup backfill --gamertag X --csr             [--force]   # Halo tokens
go run ./cmd/levelup backfill --all          --shared-csr     [--dry-run] # Halo tokens
go run ./cmd/levelup backfill --all          --weapons        [--force]   # film CDN
go run ./cmd/levelup backfill --gamertag X --citations-recompute-all
```

### Backup / restore

```bash
go run ./cmd/levelup backup  --gamertag X [--output-dir D] [--compression-level 9]
go run ./cmd/levelup restore --gamertag X --backup-dir D [--replace] [--dry-run] [--tables T1,T2]
go run ./cmd/levelup restore-csr --gamertag X --backup PATH [--dry-run] [--mode preserve|overwrite]
```

### Metadata / seed / migration

```bash
go run ./cmd/levelup seed career-ranks | citation-mappings | medals | rank-translations
go run ./cmd/levelup seed-demo            # generate anonymized demo data (data/demo/)
go run ./cmd/levelup migrate              # migrate data into the multi-title namespace
go run ./cmd/levelup add-title --name "Halo MCC" [--slug s] [--capabilities matchmaking,media] [--xbox-id X] [--steam-id S]
```

### Media

```bash
go run ./cmd/levelup index-media --gamertag X [--force-rescan] [--buffer-min N]
```

### Diagnostics & ops

```bash
go run ./cmd/levelup healthcheck [--verbose]
go run ./cmd/levelup diagnose --db PATH [--verbose]
go run ./cmd/levelup check-env
go run ./cmd/levelup gate-check [--gamertag X] [--json]
go run ./cmd/levelup compare-db --go-db PATH --python-db PATH [--json]
```

### Prestige — coach grammar tuning analyzer

Read-only analyzer (never opens DBs RW). Produces **recommendations** to adjust the
coach synthesis grammar (`config/coach_advisor/synthesis_grammar.toml`) from Prestige
telemetry (completion rate per grammar metric). Application stays **manual**: a human
reads the report and edits the TOML — no auto-PR, no runtime override.

```bash
# All players of a title (default halo_infinite), text report:
go run ./cmd/prestige-tuning-analyze
# Single player, JSON output:
go run ./cmd/prestige-tuning-analyze --player JGtm --format json
# Custom thresholds (rule: completion < min-completion over >= min-sample accepted coach challenges):
go run ./cmd/prestige-tuning-analyze --min-completion 0.30 --min-sample 50 --source coach
# flags: --format text|json  --player SLUG|GAMERTAG  --title SLUG
#        --min-completion 0..1  --min-sample N  --source coach|user|pilot_mode  --grammar PATH
```

Below `--min-sample`: "insufficient data" (no recommendation on noise). A telemetry
metric absent from the grammar is flagged as an orphan (naming drift / legacy challenge).

### Maintenance (server stopped for ART/alias rebuilds)

```bash
go run ./cmd/levelup rebuild-pme-art --all | --gamertag X   # rebuild player_match_enrichment ART index
go run ./cmd/levelup consolidate-aliases                    # merge xbox_aliases into shared.xuid_aliases
go run ./cmd/levelup recompute-friends [--dry-run]          # recompute is_with_friends across player DBs
go run ./cmd/levelup replay-events --gamertag X             # re-parse highlight events
go run ./cmd/levelup reset-bitmasks                         # reset skill/participants/PVE backfill bits
go run ./cmd/levelup engagement-coefs [--with-scores]      # recompute engagement coefficients
```

### Notifications

```bash
go run ./cmd/levelup notify-version --version v1.2.3
go run ./cmd/levelup notify-sync --gamertag X --op sync_delta --duration 120s [--matches N]
```

Full list: `go run ./cmd/levelup help`.

---

## Tests

### Go (see [testing.md](testing.md))

```bash
# Fast, no DuckDB (CGO off)
make go-api-test
# or directly:
cd apps/go-api && CGO_ENABLED=0 go test ./internal/domain/... ./internal/analysis/... ./contracttest/... -count=1

# Full suite with DuckDB (CGO on — needs a C toolchain / MinGW on Windows)
cd apps/go-api && CGO_ENABLED=1 LEVELUP_DEMO_MODE=true go test ./... -timeout 5m -count=1

make go-api-coverage   # coverage report
make go-api-lint       # go vet
```

### Frontend (`apps/web`)

```bash
make test-web        # vitest run
make test-e2e        # Playwright (needs `make dev` running)
make test-e2e-ui     # Playwright UI mode
# or via npm in apps/web:
npm run test:run
npm run test:coverage
npm run lint
```

---

## Environment variables

| Variable | Purpose |
|----------|---------|
| `LEVELUP_REPO_ROOT` | Data repo root (auto-detected if absent) |
| `LEVELUP_API_PORT` | Go API port (default `8000`) |
| `LEVELUP_DEMO_MODE` | Demo mode (used by the test targets) |
| `LEVELUP_NOTIFY_VERSIONS` | Set to `1` to enable version notifications in prod |
| `DISCORD_WEBHOOK_URL` | Discord webhook (overrides `app_settings.json`) |
| `CGO_ENABLED` | Must be `1` for any DuckDB-touching build/test |

---

## Data paths

```
data/
  warehouse/metadata.duckdb         # referentials (maps, playlists, medals)
  warehouse/shared_matches_v2.duckdb # shared matches/medals/events/aliases
  warehouse/shared_pve.duckdb       # Firefight stats
  players/{gamertag}/stats.duckdb   # per-player enrichment
  players/{gamertag}/archive/       # Parquet archives
db_profiles.json                    # player profiles (multi-title)
app_settings.json                   # app settings
.env.local                          # Azure tokens / secrets
```

See [ARCHITECTURE_V6.md](ARCHITECTURE_V6.md) for the full data model.
