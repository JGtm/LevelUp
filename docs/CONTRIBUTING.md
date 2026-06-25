# Contributing to LevelUp

French version: [FR/CONTRIBUTING.md](FR/CONTRIBUTING.md)

Thank you for your interest in contributing to LevelUp! This document explains how to participate in the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Repository Layout](#repository-layout)
- [Environment Setup](#environment-setup)
- [How to Contribute](#how-to-contribute)
- [Branch Strategy](#branch-strategy)
- [Go Backend Standards](#go-backend-standards)
- [Frontend Standards](#frontend-standards)
- [Agent Workflow (thought_log + skills)](#agent-workflow-thought_log--skills)
- [Pull Request Process](#pull-request-process)
- [Reporting a Bug](#reporting-a-bug)
- [Proposing a Feature](#proposing-a-feature)
- [Open Source Credits](#open-source-credits)

---

## Code of Conduct

This project follows a respectful and inclusive code of conduct. Be kind to other contributors.

---

## Repository Layout

LevelUp is a Go backend + React frontend monorepo. The two applications live under `apps/`:

| Path | Stack | Role |
|------|-------|------|
| `apps/go-api/` | Go (CGO + DuckDB) | HTTP API, sync engine, analysis, CLI tooling |
| `apps/web/` | React 19 / Vite / TypeScript | Dashboard frontend |
| `docs/` | Markdown | English documentation (`docs/FR/` = French mirror) |
| `data/` | DuckDB / Parquet / JSON | Warehouses, per-player DBs, auth token store, config |
| `.ai/` | Markdown | Agent working memory (project map, thought log, plans) |
| `.claude/skills/` | Markdown | Agent skills (architecture rules, conventions) |

Key areas inside `apps/go-api/`:

| Path | Role |
|------|------|
| `cmd/server/` | HTTP server entrypoint |
| `cmd/levelup/` | Operations CLI (backup, restore, sync, seed, migrate, ...) |
| `cmd/*` | One-shot diagnostic / backfill / migration tools |
| `internal/api/` | HTTP handlers, middleware, router |
| `internal/analysis/` | Pure stateless algorithms (temporal, breakdown, narrative) |
| `internal/service/` | Orchestration (repo access + analysis) |
| `internal/sync/` | Sync engine (delta / full, persist pipeline) |
| `internal/platform/duckdb/` | DuckDB access, leases, shared_social writes |
| `internal/games/canonical/` | Cross-title canonical types |
| `internal/migration/` | Schema migration steps |

Architecture references: `docs/ARCHITECTURE_V6.md`, `docs/FOUNDATIONS_GUIDE.md`, and the ADRs in `docs/adr/`.

---

## Environment Setup

### Prerequisites

- **Go** matching the version in `apps/go-api/go.mod`. DuckDB requires **CGO**, so a C toolchain is mandatory. On Windows use MSYS2/MinGW `gcc` with `CGO_ENABLED=1`.
- **Node.js** (LTS) + npm, for `apps/web/`.
- **air** for Go hot-reload (`go install github.com/air-verse/air@latest`) — used by `make dev`.
- Git.

### One-command dev

From the repo root:

```bash
make dev
```

This starts the Go API (via `air`, default port 8000) and the Vite frontend (port 5173). Open `http://localhost:5173`. `Ctrl+C` stops both. `make stop` force-kills the dev servers by port; `make restart` does `stop` then `dev`.

Install frontend dependencies the first time:

```bash
make install-web      # = cd apps/web && npm install
```

### Auth tokens

Auth tokens have a single source of truth: `data/auth/watcher_tokens/{xuid}.json`, managed by `MultiUserTokenStore` (see `docs/adr/0023-auth-tokens-single-source.md`). The player must first be declared in `db_profiles.json` (with `xuid`). Onboarding options:

```bash
# Advanced onboarding (device-code capture, writes directly to the store)
go run ./cmd/token-capture/ <Gamertag>

# Import a refresh token from stdin
go run ./cmd/token-import/ <Gamertag>
```

Never use `.env.local` or `sync_meta` as a credential source (legacy fallbacks only).

---

## How to Contribute

1. Fork and clone the repository.
2. Create a work branch (see [Branch Strategy](#branch-strategy)) — **never commit on `main`**.
3. Implement your change following the standards below.
4. Run the relevant lint + tests (Go and/or frontend).
5. Add a `.ai/thought_log.md` entry (see [Agent Workflow](#agent-workflow-thought_log--skills)).
6. Open a Pull Request using Conventional Commits.

---

## Branch Strategy

Rule (from `CLAUDE.md`): **1 task = 1 branch, N commits**. Sequential phases of the same task are commits on a single branch, not separate branches.

```bash
# Correct — phases of one task = commits on one branch
git checkout -b refactor/cleanup-all
git commit -m "refactor(phase1): dead code cleanup"
git commit -m "refactor(phase2): DRY violations"
```

Applied rules:

- **Never work on `main`** — no exception. If you are on `main`, create a work branch first.
- Check the current branch before committing: `git branch --show-current`.
- Create a new branch for every new feature/fix from the current branch (`git checkout -b <type>/<name>`).
- Do not switch branches if unrelated work is already in progress on the current branch.
- Pushing `main` triggers an automatic production deploy — merge to `main` deliberately.

Commit message format (Conventional Commits):

```
<type>(<scope>): <description>
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`. Examples:

```
feat(api): add CSR-by-playlist endpoint
fix(sync): correct Firefight mode parsing
docs: update contributing guide
```

---

## Go Backend Standards

Tooling equivalents (formatting / linting / typing) are enforced by `gofmt`, `go vet`, and `golangci-lint` (config: `apps/go-api/.golangci.yml`).

### Format and vet

```bash
cd apps/go-api
gofmt -l .            # list unformatted files (must be empty)
go vet ./...
```

The golangci-lint config enables `revive`, `gocyclo`, `funlen`, `lll`, `goconst`, `unconvert`, `unparam`, `bodyclose`, `noctx`, `prealloc`, plus the standard set and `staticcheck`. Thresholds: cyclomatic complexity 15, function length 100 lines / 80 statements, line length 220, argument limit 7. `gofmt` + `goimports` are the formatters.

```bash
cd apps/go-api && golangci-lint run
# Makefile shortcut (vet on domain + analysis):
make go-api-lint
```

### Tests

DuckDB requires CGO. Two test tiers (full detail in `docs/testing.md`):

```bash
# Fast tier — no DuckDB (CGO off): domain + analysis + contract
cd apps/go-api
CGO_ENABLED=0 go test ./internal/domain/... ./internal/analysis/... ./contracttest/... -timeout 60s -count=1
# Makefile shortcut:
make go-api-test

# Full tier — with DuckDB (CGO on)
cd apps/go-api
CGO_ENABLED=1 LEVELUP_DEMO_MODE=true go test ./... -timeout 5m -count=1
```

On Windows, ensure MinGW `gcc` is on PATH (`CC=gcc`, `CGO_ENABLED=1`). Note: `go test -race` is incompatible with the DuckDB driver unless you pass `-gcflags=all=-d=checkptr=0`.

Coverage is a non-regressing ratchet (baseline in `apps/go-api/coverage_baseline.txt`):

```bash
make go-api-coverage              # quick func summary
make go-api-test-coverage-ratchet # enforce ratchet vs baseline
```

See `docs/testing.md` for the per-layer test patterns (handlers mock service, in-memory DuckDB, validation gates).

### Architecture rules

- `internal/analysis/` = pure stateless algorithms (no DB access). `internal/service/` = orchestration (repo + analysis).
- New writes to a shared DB on a per-match path go through the persist pipeline (`BatchBuilder.Build()` then `BatchQueue.Submit()`) — no concurrent UPSERT/UPDATE on critical tables (ART-safe, see ADR 0019, 0026).
- State tables are append-only (read via `<table>_latest` views). Guard test: `internal/sync/no_art_patterns_test.go`.
- All DuckDB access via context managers / leases — no bare `db.Close()` leaks.
- Read the `arch-rules`, `db-schema`, `canonical-types`, and `go-features` agent skills before changing backend structure.

---

## Frontend Standards

From `apps/web/` (scripts defined in `apps/web/package.json`):

```bash
npm run typecheck     # tsc -b — no type errors
npm run lint          # eslint .
npm run lint:fields   # guard against hardcoded API field names
npm run test:run      # vitest run (no watch)
npm run test:e2e      # playwright (requires `make dev` running)
```

Makefile shortcuts: `make check-types`, `make test-web`, `make test-e2e`.

Vitest note: run tests outside any sandbox that blocks worker processes; typecheck and eslint are safe in sandbox.

### Color tokens (mandatory)

No raw hex (`#RRGGBB`) and no Tailwind color classes (`text-red-*`, `bg-green-*`, ...) in `apps/web/src/features/` or `apps/web/src/components/`, except documented exceptions. Semantic colors must go through `tokenCssVar(token)` (JSX), `resolveToken(token)` (Plotly/SVG), or `getSeriesColors(n, tokens[])` (series). Raw palettes are centralized in `apps/web/src/lib/accessibility/palettes/`. See the `color-tokens` skill.

### i18n

User-facing strings must be provided in both **FR and EN** via the i18n manifests (TOML manifests + custom linter, see ADR 0003). Do not hardcode display strings in components.

### Charts and pages

Use the canonical ECharts wrappers (`apps/web/src/components/charts/README.md`) and the foundations (canonical types + adapters + i18n + chart wrappers) described in `docs/FOUNDATIONS_GUIDE.md`. See the `frontend-patterns` and `foundations-usage` skills.

---

## Agent Workflow (thought_log + skills)

This repo is maintained partly by AI agents. Two conventions apply to every contributor:

1. **thought_log (mandatory)** — before every commit (or at minimum before handing back), add an entry to `.ai/thought_log.md` with: date `[YYYY-MM-DD]`, task title, status (In progress / Done), the main technical decision, observed results, and the conclusion / next step. A missing entry means the task is not finished.
2. **Agent skills** — consult the relevant skill in `.claude/skills/{arch-rules, canonical-types, color-tokens, foundations-usage, delivery-checklist, plan-review, halo-modes, db-schema, frontend-patterns, go-features}/SKILL.md` before committing structural changes.

Before starting, also review the `.ai/` working memory: `project_map.md`, `thought_log.md`, `data_lineage.md`.

---

## Pull Request Process

### Checklist

Before submitting a PR, verify:

- [ ] On a work branch, not `main`
- [ ] Go: `gofmt -l .` clean, `go vet ./...` clean, `golangci-lint run` passes
- [ ] Go tests pass (fast tier always; full CGO tier for DB/sync changes)
- [ ] Coverage ratchet does not regress
- [ ] Frontend: `npm run typecheck`, `npm run lint`, `npm run test:run` pass
- [ ] No raw hex / Tailwind color classes in `features/` or `components/`
- [ ] FR + EN i18n strings provided for new UI text
- [ ] `docs/FR/` mirror updated if a `docs/` file changed
- [ ] `.ai/thought_log.md` entry added
- [ ] Commit messages follow Conventional Commits

### Review

A maintainer will get back to you for clarification questions, improvement suggestions, then validation and merge.

---

## Reporting a Bug

### Before Reporting

1. Check that the bug has not already been reported.
2. Reproduce on the latest `main`.

### Create an Issue

Include:

- **Description**: observed vs. expected behaviour.
- **Reproduction**: steps to reproduce.
- **Environment**: OS, Go version (`go version`), Node version, browser.
- **Logs**: full error messages. Go logs are written per-category under `logs/*.log` (e.g. `logs/handlers.log`, `logs/general.log`), not only stdout — grep all of them.

```markdown
## Bug

### Description
The dashboard does not load matches for player X.

### Reproduction
1. Open the dashboard
2. Select player X
3. Observe the error

### Environment
- OS: Windows 11
- Go: go1.x
- Node: 20.x

### Logs
(error message from logs/handlers.log)
```

---

## Proposing a Feature

### Before Proposing

1. Check that the feature has not already been proposed or is in progress (`.ai/` plans).
2. Think through the implementation against the architecture (ADRs, foundations).

### Create an Issue

Include:

- **Description**: what does the feature do?
- **Motivation**: why is it useful?
- **Implementation** (optional): which layer (analysis/service/handler, or frontend feature) and which canonical types it touches.

---

## Open Source Credits

This project relies on several community components. Credits are centralised in [ACKNOWLEDGMENTS.md](ACKNOWLEDGMENTS.md). Before adding a major external dependency, document it there to keep attribution clear.

---

## Questions?

If you have questions, open an issue with the `question` tag.

---

**Thank you for contributing to LevelUp!**
