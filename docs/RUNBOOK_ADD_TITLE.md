# RUNBOOK — Adding a New Title

> EN-only (runbook policy). Created 2026-07-10 as part of the monitoring
> overhaul (A3.6): the in-app "Lab" tab was removed (DC-9) — preparing a new
> title is a development workflow, served by the CLIs below plus this runbook.
> The remaining operational value (per-title diagnostics) lives in the admin
> UI under **Gestion → Titres**.

## Overview

LevelUp is multi-title by design (ADR 0008 path isolation, ADR 0025
title-agnostic refactor). A title is defined by:

| Piece | Location |
|---|---|
| Registry entry (slug, capabilities) | `apps/go-api/internal/domain/title/registry.go` |
| Manifest + mappings (TOML) | `config/titles/{slug}/title.toml` + `mappings/{fields,assets,outcomes,capabilities}.toml` |
| Data adapters (Go) | `apps/go-api/internal/games/{slug}/adapter_data.go` + `adapter_semantic.go` |
| Warehouse databases | `data/titles/{slug}/warehouse/*.duckdb` (via `PathResolver`, never hand-built paths) |
| Migration set (per-title schema) | `internal/migration` (`TitleMigrationSet`, PMT-9) |

Golden rules:

- **Never branch on `slug == "..."`** — declare capabilities and branch on
  `HasCapability` / fine-grained `capabilities.toml` keys (ratchet:
  `no_slug_comparison_test.go`).
- **Degrade gracefully**: a field absent from the TOML mappings means the title
  does not support that product surface — return `ErrCapabilityNotSupported`,
  never panic, never serve another title's data.
- **All filesystem paths go through `PathResolver`.**

## Step-by-step

### 1. Probe the upstream API

Explore what the title's API exposes before writing any code. Existing probes
are the model to copy (throwaway probes are forbidden — extend a tested CLI):

```bash
go run ./apps/go-api/cmd/probe-h5 --help     # Halo 5 (haloapi.com official API)
go run ./apps/go-api/cmd/probe-mcc --help    # MCC (mccapi endpoint)
```

Write a `cmd/probe-{slug}` if the title has a new upstream. Keep it: it becomes
the title's reference client documentation.

### 2. Declare the title

1. Add the slug to `internal/domain/title/registry.go` (status `coming_soon`
   first — the UI renders it read-only).
2. Create `config/titles/{slug}/title.toml` plus the four mapping files:
   `fields.toml`, `assets.toml`, `outcomes.toml`, `capabilities.toml`.
   Tip: the admin UI (**Gestion → Titres**) renders a capabilities.toml draft
   from the declared registry entry ("Copier le brouillon TOML").
3. `make build` — the registry validates manifests at boot.

### 3. Fetch reference metadata

```bash
# Halo 5 example — official API metadata into metadata.duckdb
go run ./apps/go-api/cmd/h5-metadata-fetch --help

# Populate image assets (medals, ranks, playlists)
# populate-assets is a `levelup` CLI subcommand (also shipped in the prod image)
go run ./apps/go-api/cmd/levelup populate-assets --help
go run ./apps/go-api/cmd/populate-playlists-catalog --help
```

For a brand-new title, clone `h5-metadata-fetch` and adapt endpoints; metadata
lives in `data/titles/{slug}/warehouse/metadata.duckdb` (per-title migration
set applies the schema).

### 4. Implement the adapters

`internal/games/{slug}/adapter_data.go` (canonical data) and
`adapter_semantic.go` (labels/assets/outcomes from the TOML). Register both in
the boot `Resolver` (`api/server.go` wiring). See skill `arch-rules`
("Ajouter un nouveau titre") and `internal/games/halo_5/` as the reference
implementation of a second title.

### 5. Sync pipeline

The V2 cycle orchestrator (ADR 0027) is currently mono-title (halo_infinite).
A second title routes through its own live runner (see `sync_v2_wiring.go` and
the H5 `liveRunner` precedent) — do NOT force a new title through the
orchestrator without checking `reference_v2_pipeline_mono_title_h5_routing`.

### 6. Verify

```bash
# Per-title diagnostic (config files, databases, declared vs reality drift)
go run ./apps/go-api/cmd/levelup-titles diagnose --slug {slug}

# Full test suite + integration (persist/sync touched => mandatory)
cd apps/go-api && go test ./... && go test -tags=integration -p 1 ./...
```

In the app: **Admin → Gestion → Titres** shows the registry entry, its
capabilities/feature-matrix, and the same diagnostic (config + databases +
drift) rendered per title.

## See also

- `docs/adr/0025-title-agnostic-refactoring.md` (master plan:
  `.ai/V7/PLAN_TITLE_AGNOSTIC_REFACTORING.md`)
- `docs/adr/0008-*` (path isolation), `docs/adr/0027-*` (sync V2)
- Skills: `arch-rules`, `canonical-types`, `db-schema`
