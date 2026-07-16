# Installation Guide — LevelUp

French version: [FR/INSTALL.md](FR/INSTALL.md)

> Complete guide to install and configure LevelUp on your machine.

---

## Windows — Recommended local install

This Go migration repo no longer ships one-click launchers.
The standard entry point is now `make dev`.

### Step 1 — Download LevelUp

Go to the GitHub project page → green **Code** button → **Download ZIP**.
Extract the folder wherever you like (e.g. Desktop or `C:\LevelUp\`).

> If you know Git, you can also clone:
> ```bash
> git clone https://github.com/JGtm/LevelUp_with_SPNKr.git
> ```

### Step 2 — Install local tooling

1. Install Go 1.26+ and Node.js on your machine
2. Install Air for Go hot reload:
   ```bash
   go install github.com/air-verse/air@latest
   ```
3. Install frontend dependencies:
   ```bash
   cd apps/web && npm install && cd ../..
   ```

### Step 3 — Start the app

```bash
make dev
```

This starts the Go API on port 8000 and the Vite frontend on http://localhost:5173.

### Step 4 — Setup Wizard in the browser

On the first launch, LevelUp detects it is not yet configured and shows a **guided wizard**.
Choose your path:

#### Xbox Express (recommended — 2 steps)

**v6 — Zero Azure configuration required.** LevelUp bundles its own client ID.

**Step 1 — Enter your gamertag**

Type your Xbox gamertag in the wizard. LevelUp creates your local profile automatically.

**Step 2 — Authenticate via Device Code**

The wizard displays a short code and the URL `https://xbox.com/activate`.

1. Open [https://xbox.com/activate](https://xbox.com/activate) in your browser
2. Enter the code shown in the wizard
3. Sign in with your Microsoft/Xbox account

That's it — LevelUp retrieves your XUID, completes your profile and persists the OAuth
refresh token to the single token store (`data/auth/watcher_tokens/{xuid}.json`,
see [ADR 0023](adr/0023-auth-tokens-single-source.md)), then proceeds to the smoke test.

#### Advanced onboarding (headless / CLI)

Use this path when the interactive wizard is not accessible (server, headless, reverse proxy).
The player must already be declared in `db_profiles.json` with their `xuid` before running these
commands (the token store is addressed by xuid). Tokens go straight to the single store — there is
**no `.env.local` manipulation** (see [ADR 0023](adr/0023-auth-tokens-single-source.md)).

```bash
# Device Code Flow in the browser, refresh token written to the store
cd apps/go-api && go run ./cmd/token-capture/ <Gamertag>

# Or import a refresh token obtained elsewhere (read from stdin)
cd apps/go-api && go run ./cmd/token-import/ <Gamertag>
```

After capture/import, restart the server: the auth Pool finds the token in the store and works
immediately.

#### Note for forks / developers

The bundled client ID is tied to this project's Azure App Registration.
If you fork LevelUp, create your own (free) Azure App Registration and set:

```env
# .env.local
LEVELUP_OAUTH_CLIENT_ID=your_own_client_id
```

See [CONFIGURATION.md](CONFIGURATION.md) for the full Azure registration walkthrough.
This env var takes precedence over the bundled ID. Note that `.env.local` is config-only
(client ID): it is **not** a credential store — refresh tokens live in the token store
(see [ADR 0023](adr/0023-auth-tokens-single-source.md)).

#### Auth provider — SISU (sole provider)

The onboarding Device Code Flow uses the **SISU** provider: the native Xbox device-code
flow, which needs **no Azure app** at all — this is why "Xbox Express" above requires zero
Azure configuration. The former MSAL fallback was removed on 2026-07-15 after SISU was
validated end-to-end; a legacy `"auth_provider": "msal"` in `app_settings.json` is now
ignored (with a boot warning). See [CONFIGURATION.md](CONFIGURATION.md) for the details.

### Step 5 — Smoke test (automatic check on 20 matches)

After Xbox sign-in, the wizard automatically runs a **3-phase smoke test**:

| Phase | What happens |
|-------|--------------|
| Phase 1 — Sync | Synchronize 20 matches from the Halo API |
| Phase 2 — Enrichment | Compute scores, sessions, citations, LUSR/CSR, killer/victim pairs |
| Phase 3 — Verification | Full integrity check of all tables (see below) |

**Checked tables (all required):**

| Table | Database | What is validated |
|-------|----------|-------------------|
| `match_registry` | shared | count > 0 |
| `match_participants` | shared | count > 0 + kills/deaths not NULL |
| `medals_earned` | shared | count > 0 |
| `killer_victim_pairs` | shared | count > 0 |
| `highlight_events` | shared | count > 0 (film clips) |
| `xuid_aliases` | shared | count > 0 |
| `player_match_enrichment` | player | count > 0 + session_id not NULL |
| `performance_score` | shared (via match_participants) | computed score > 0 |
| `match_citations` | player | count > 0 |
| `match_skill_rank` (LUSR/CSR) | player | count > 0 + LUSR/CSR present |
| `sessions` | player | count > 0 |
| `sync_meta` | player | count > 0 |
| shared↔player consistency | cross | counts consistent |

If any check fails, the test offers to **retry**. When everything is green, two options:

- **Full sync** → navigates to the Settings page to fetch your complete history (recommended)
- **Dashboard (20 matches)** → goes directly to the dashboard with the already-synced matches

---

## macOS / Linux

The local workflow is the same as on Windows:

1. Install Go 1.26+, Node.js + npm, and GNU Make
2. Install Air:
   ```bash
   go install github.com/air-verse/air@latest
   ```
3. Install frontend dependencies:
   ```bash
   cd apps/web && npm install && cd ../..
   ```
4. Start the stack:
   ```bash
   make dev
   ```

Then open http://localhost:5173 and complete the in-app wizard.

---

## Developer install

### Requirements
- Go 1.26+
- Node.js + npm
- GNU Make
- Git

```bash
git clone https://github.com/JGtm/LevelUp_with_SPNKr.git
cd LevelUp_with_SPNKr
cd apps/web && npm install && cd ../..
go install github.com/air-verse/air@latest
make dev
```

### Environment healthcheck

```bash
curl http://127.0.0.1:8000/health
make go-api-test
cd apps/web && npm run typecheck
```

### Tests

The DuckDB driver requires CGO. See [testing.md](testing.md) for the full matrix
(CGO=0 fast path, coverage ratchet, Windows MinGW).

```bash
# Go — full suite with DuckDB (CGO)
cd apps/go-api
CGO_ENABLED=1 LEVELUP_DEMO_MODE=true go test ./... -timeout 5m -count=1

# Go — fast subset without DuckDB
CGO_ENABLED=0 go test ./internal/domain/... ./internal/analysis/... ./contracttest/... -count=1

# Frontend
cd apps/web && npm run typecheck && npm test
```

### Update

```bash
git pull origin main
cd apps/web && npm install && cd ../..
go install github.com/air-verse/air@latest
```

See [CONFIGURATION.md](CONFIGURATION.md) for Azure token configuration.

---

## Docker install

### Requirements
- Docker Desktop installed
- Docker Compose v2 available (`docker compose version`)

### Pre-requisites: configuration files

Before the first `docker compose up`, make sure these files exist:

```bash
# If db_profiles.json does not exist yet
echo '{"profiles": {}}' > db_profiles.json

# If app_settings.json does not exist yet
echo '{}' > app_settings.json
```

> **Why?** Docker bind-mount creates a *directory* (not a file) if the source does not exist,
> which would crash the app.

### Start with Docker Compose

```bash
# Build and start
docker compose up --build

# In the background
docker compose up -d

# View logs
docker compose logs -f

# Stop
docker compose down
```

### Docker volumes

| Host path | Container path | Description |
|-----------|----------------|-------------|
| `./data` | `/app/data` | DuckDB data (read/write) |
| `./db_profiles.json` | `/app/db_profiles.json` | Player profiles |
| `./app_settings.json` | `/app/app_settings.json` | Application settings |

---

## Troubleshooting

### Full diagnostic

```bash
curl http://127.0.0.1:8000/health
make go-api-test
cd apps/web && npm run typecheck
```

### "Module not found" error

```bash
cd apps/web && npm install
```

### DuckDB version error

```bash
cd apps/go-api && CGO_ENABLED=1 go test ./... -count=1
```

### Expired OAuth token

In the app → **Settings** → **Xbox connection** → **Reconnect** (re-runs the Device Code flow
and refreshes the token in the store). For headless players, re-run
`cd apps/go-api && go run ./cmd/token-capture/ <Gamertag>`. The refresh token is persisted in
`data/auth/watcher_tokens/{xuid}.json` (single token store, see
[ADR 0023](adr/0023-auth-tokens-single-source.md)).

### Permission Denied (Windows / PowerShell)

```powershell
# Allow PowerShell scripts (once only)
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

---

## Folder structure after install

```
LevelUp/
├── apps/
│   ├── go-api/                      # Go backend (API + sync + CLI under cmd/)
│   └── web/                         # React/Vite frontend
├── data/
│   ├── auth/
│   │   └── watcher_tokens/
│   │       └── {xuid}.json          # OAuth/MSAL token store (ADR 0023)
│   ├── players/
│   │   └── MyGamertag/
│   │       └── stats.duckdb         # Per-player enrichments
│   └── warehouse/
│       ├── metadata.duckdb          # Reference data (maps, medals…)
│       └── shared_matches_v2.duckdb # Shared match data (centralised)
├── db_profiles.json                 # Player profiles (created by the wizard)
├── app_settings.json                # Application settings
└── .env.local                       # Optional config (e.g. LEVELUP_OAUTH_CLIENT_ID for forks)
```

---

## Next steps

1. [Detailed Azure configuration](CONFIGURATION.md)
2. [Sync your matches](SYNC_GUIDE.md)
3. [Explore the dashboard](../README.md)
