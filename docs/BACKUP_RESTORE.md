# Backup & Restore — LevelUp

French version: [FR/BACKUP_RESTORE.md](FR/BACKUP_RESTORE.md)

LevelUp stores all data in DuckDB files under `data/titles/<slug>/`. There are **two complementary backup paths**:

| Path | What it protects | Format | Driven by |
|------|------------------|--------|-----------|
| **Automatic restic snapshots** (recommended) | Every DuckDB of every title (warehouse + per-player) | Parquet+Zstd staged, then a `restic` snapshot | `pkg/duckdbbackup` scheduler (runs inside the server), `cmd/backup-once`, restored with `cmd/restore` |
| **Manual per-player export** | A single player `stats.duckdb` | Parquet+Zstd files on disk | `levelup backup` / `levelup restore` |

A dedicated one-shot, `levelup restore-csr`, recovers historical per-match CSR from a legacy DuckDB backup.

---

## Data layout (`data/titles/<slug>/`)

All paths are resolved by `internal/domain/title` `PathResolver`. For the default title `halo_infinite`:

```
data/titles/halo_infinite/
├── warehouse/
│   ├── shared_matches_v2.duckdb   # shared matches/participants/medals/...
│   ├── metadata.duckdb            # reference data (ranks, citations, ...)
│   ├── shared_pve.duckdb          # Firefight stats
│   └── shared_social.duckdb       # likes/favorites/social state
├── players/
│   └── <Gamertag>/
│       ├── stats.duckdb           # player enrichment DB
│       ├── archive/               # cold Parquet archives (`levelup archive`)
│       └── captures/              # local media captures
└── backups/
    └── <Gamertag>/                # manual per-player Parquet exports
```

The automatic restic scheduler discovers every title under `data/titles/` and protects each warehouse DB plus every `players/<Gamertag>/stats.duckdb`. Backup keys are namespaced by slug (`<slug>:shared_matches_v2`, `<slug>:metadata`, `<slug>:shared_pve`, `<slug>:shared_social`, `<slug>:player:<Gamertag>`).

---

## 1. Automatic restic snapshots

### How it works

The server runs a backup scheduler (`ops.NewLevelUpBackupScheduler`, wired in `cmd/server/main.go`). Each cycle:

1. Discovers all DuckDB targets under `data/titles/`.
2. Skips unchanged DBs (fingerprint manifest `.manifest.json` in the staging dir).
3. Runs `PRAGMA integrity_check` on changed DBs (a degraded DB is still backed up, with a warning).
4. Exports each changed DB's `BASE TABLE`s to Parquet+Zstd into a staging tree.
5. Creates a single `restic` snapshot of the staging dir, then applies the retention policy (`restic forget --prune`).

The first cycle runs immediately at startup, then every `interval` (default `6h`). If the `restic` binary is not on `PATH`, the scheduler logs a warning and disables itself.

### Configuration

Behaviour comes from `app_settings.json`; machine paths come from environment variables.

| Setting (`app_settings.json`) | Default | Meaning |
|-------------------------------|---------|---------|
| `backup_enabled` | `false` | Enable the periodic scheduler |
| `backup_interval` | `6h` | Go duration between cycles |
| `backup_keep_daily` | `7` | `restic forget --keep-daily` |
| `backup_keep_weekly` | `4` | `restic forget --keep-weekly` |
| `backup_keep_monthly` | `12` | `restic forget --keep-monthly` |

| Environment variable | Default | Meaning |
|----------------------|---------|---------|
| `RESTIC_REPOSITORY` | _(unset)_ | restic repo location (required to back up) |
| `RESTIC_PASSWORD` | _(unset)_ | repo password |
| `RESTIC_PASSWORD_FILE` | _(unset)_ | file containing the repo password |
| `RESTIC_BIN` | `restic` | path to the `restic` binary |
| `LEVELUP_BACKUP_DIR` | `data/backups` | local staging directory |

When neither `RESTIC_PASSWORD` nor `RESTIC_PASSWORD_FILE` is set, restic is invoked with `--insecure-no-password` (unencrypted repo — suited to a local single-user setup).

The retention policy is a single **global** envelope across all titles (one snapshot, one retention policy covers every title's DBs by design).

### Run a snapshot manually

```bash
cd apps/go-api
go run ./cmd/backup-once
```

This runs exactly one cycle synchronously and prints the snapshot id, duration, and the list of exported DBs. It is a no-op (`Skipped`) if nothing changed since the last cycle.

### List and restore snapshots

```bash
cd apps/go-api

# List available snapshots (id, date, host)
go run ./cmd/restore --list

# Restore the most recent snapshot into data/restore/<YYYY-MM-DD>/
go run ./cmd/restore

# Restore the most recent snapshot of a given day
go run ./cmd/restore --date 2026-05-25

# Restore a specific snapshot by short id
go run ./cmd/restore --snapshot 6ba84d2b

# Restore into a custom directory
go run ./cmd/restore --output /tmp/restore/

# Simulate without writing anything
go run ./cmd/restore --dry-run
```

By default `restore` writes a mirror tree under `data/restore/<date>/` (`{slug}/{db}.duckdb`, `{slug}/players/{gamertag}/stats.duckdb`) so you can inspect before swapping anything in.

**In-place production restore** overwrites the live DuckDB files:

```bash
go run ./cmd/restore --live          # asks for confirmation
```

`--live` is incompatible with `--output`. **Stop the Go server first** — the DBs must not be held open while they are replaced.

---

## 2. Manual per-player export (`levelup backup` / `restore`)

Exports a single player's `stats.duckdb` to standalone Parquet+Zstd files (portable, readable by any DuckDB/Parquet tool). Independent of restic.

### Backup

```bash
cd apps/go-api

# Default output: data/titles/<slug>/backups/<Gamertag>/
go run ./cmd/levelup backup --gamertag YourGamertag

# Custom title, output directory and compression level (Zstd 1-22, default 9)
go run ./cmd/levelup backup \
  --gamertag YourGamertag \
  --title halo_infinite \
  --output-dir ./my_backups \
  --compression-level 15
```

Each table is written as `<table>_<timestamp>.parquet`, plus a `backup_metadata_<timestamp>.json` describing the rows/sizes.

### Restore

```bash
cd apps/go-api

# Restore all tables from a backup directory
go run ./cmd/levelup restore \
  --gamertag YourGamertag \
  --backup-dir ./data/titles/halo_infinite/backups/YourGamertag

# Restore selected tables, replacing existing data
go run ./cmd/levelup restore \
  --gamertag YourGamertag \
  --backup-dir ./my_backups \
  --tables player_match_enrichment,match_citations \
  --replace

# Inspect without writing
go run ./cmd/levelup restore --gamertag YourGamertag --backup-dir ./my_backups --dry-run
```

| Flag | Effect |
|------|--------|
| `--title` | Target title slug (default `halo_infinite`) |
| `--tables T1,T2` | Restore only these tables (default: all) |
| `--replace` | `DROP TABLE` before importing (otherwise rows are appended) |
| `--dry-run` | List without modifying |

---

## 3. Recover historical CSR (`levelup restore-csr`)

One-shot, idempotent recovery of per-match CSR values from a legacy DuckDB backup (e.g. a `shared_matches_v2.duckdb` extracted from a snapshot). Intended for the case where CSR rows were overwritten by LUSR before the SQL guard was in place.

```bash
cd apps/go-api

# Inspect the legacy schema and count without writing
go run ./cmd/levelup restore-csr \
  --gamertag YourGamertag \
  --backup /path/to/legacy/shared_matches_v2.duckdb \
  --dry-run

# Apply (overwrite removes the faulty LUSR rows on the affected matches)
go run ./cmd/levelup restore-csr \
  --gamertag YourGamertag \
  --backup /path/to/legacy/shared_matches_v2.duckdb \
  --mode overwrite
```

| Flag | Effect |
|------|--------|
| `--title` | Target title slug (default `halo_infinite`) |
| `--backup` | Path to the legacy `.duckdb` (attached read-only) |
| `--mode preserve\|overwrite` | `overwrite` deletes faulty LUSR rows on matches to restore; `preserve` keeps them |
| `--dry-run` | Inspect schema and counts only |

The command attaches the backup read-only, locates the CSR source table, and re-inserts CSR into `match_skill_rank` with `ON CONFLICT DO NOTHING`.

---

## Practical notes

- **Restore before swapping live data**: prefer a staged restore (`go run ./cmd/restore`) and inspect, then use `--live` only with the server stopped.
- **Migration to a new machine**: copy the restic repository (or the per-player Parquet folder) and restore on the target host with the same env vars.
- **Integrity**: the scheduler records `PRAGMA integrity_check` results in the staging `.manifest.json`; a degraded DB is still snapshotted with a warning so you never lose a recoverable copy.

> Legacy note: the previous Python scripts `scripts/backup_player.py` / `scripts/restore_player.py` have been removed. Use the Go commands above.
