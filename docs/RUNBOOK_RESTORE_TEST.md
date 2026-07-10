# Runbook — Restic Restore Test (disaster-recovery drill)

EN-only (project rule: runbooks are not translated).

A backup that has never been restored is not a backup. This runbook documents the
**actually executed** procedure to restore the production restic repository to a
throwaway location and verify the restored DuckDB files open and contain plausible data.
It doubles as the **dress rehearsal for a deploy** (see "Restore = deploy rehearsal"
below and the merge plan step 2).

First executed: **2026-07-07** (LOT V10a, audits 2026-07). Re-run cadence: **quarterly**,
and once **before any big-bang merge** that touches migrations/views.

---

## 1. Discovered production configuration (locations only — no secrets)

The live production backup is the **systemd timer** path, NOT the in-app scheduler
described in `docs/BACKUP_RESTORE.md` (that in-app `pkg/duckdbbackup` scheduler is
disabled — `backup_enabled` defaults to `false`). The two mechanisms are distinct; this
runbook targets the one that actually runs in prod.

| Item | Value |
|------|-------|
| Driver | systemd `levelup-restic-backup.timer` → `.service` → `/opt/levelup/scripts/restic-backup.sh` |
| Schedule | daily `04:00 UTC` (`OnCalendar=*-*-* 04:00:00`, `Persistent=true`) |
| Repo | `/opt/levelup/restic-repo` (local disk on the VPS — same disk as the data) |
| Password file | `/opt/levelup/.restic-password` (root, mode 600) — location only; never copy into any versioned file |
| Scope | `data/titles` (all titles incl. `halo_5`), `data/auth`, `db_profiles.json`, `app_settings.json`, `.env.local` |
| Excluded | media (`data/media`), cache, logs |
| Retention | `--keep-daily 7 --keep-weekly 4 --prune` |
| Consistency | the service stops `levelup` briefly so DuckDB files are at rest during the snapshot |
| Restic (VPS) | 0.14.0 |

Reference docs already in the repo: `scripts/RESTIC_BACKUP.md` (install + in-place
rollback on the VPS) and `docs/BACKUP_RESTORE.md` (the separate in-app scheduler path).

### VPS access constraints (do not violate)

The VPS is small (2 vCPU / 2 GB, ~400 MB free). During a restore *test*, treat the VPS
as **read-only**: no writes to the VPS disk, no service restart, and NEVER
`restic forget/prune/unlock/check --read-data` (RAM/IO heavy or destructive). The
restore below streams to the **local** machine and touches nothing on the VPS disk.

---

## 2. Restore procedure (method A — local restic over sftp)

Preferred because it writes only to the local machine and reads the VPS repo directly.
Requires a local `restic` binary (0.18.x on Windows works against a 0.14 repo) and SSH
access via the `lvelup` host alias. If no local restic is available, use method B below.

The repo password lives on the VPS at `/opt/levelup/.restic-password`. Pull it into an
environment variable at runtime — never write it to disk, a log, or a versioned file.

```bash
# 1. Point restic at the VPS repo over sftp and inject the password from the VPS.
export RESTIC_PASSWORD="$(ssh lvelup 'cat /opt/levelup/.restic-password')"
export RESTIC_REPOSITORY="sftp:lvelup:/opt/levelup/restic-repo"

# 2. Sanity: list snapshots. Note the age of "latest" (ALARM if it predates the last
#    known reboot with no newer snapshot — see the failure discovered below).
restic snapshots

# 3. Size check before restoring (avoid filling the local disk).
restic stats latest            # restore-size mode; expect ~735 MiB for the full scope

# 4. Restore the latest snapshot into an isolated directory OUTSIDE the git repo,
#    so it is not tracked and survives the session (consumed by the data audit, LOT V9).
restic restore latest --target "/c/Users/Guillaume/Downloads/Scripts/LevelUp-prod-copy"
```

Restore target (documented exact path): **`C:\Users\Guillaume\Downloads\Scripts\LevelUp-prod-copy\`**
— outside the git working tree, no `.gitignore` needed, survives the session.

### Method B — streaming dump (fallback, no local restic)

If no local restic binary is available, stream a directory as a tar over SSH:

```bash
# restic 0.14 supports `dump <snapshot> <dir>` -> tar on stdout for a whole directory.
ssh lvelup "RESTIC_REPOSITORY=/opt/levelup/restic-repo \
  RESTIC_PASSWORD_FILE=/opt/levelup/.restic-password \
  restic dump latest /opt/levelup/data/titles" > LevelUp-prod-copy/titles.tar
# then extract locally: tar -xf titles.tar -C LevelUp-prod-copy/
```

Method A was used on 2026-07-07 (local restic 0.18.1 present); B is documented as the
zero-dependency fallback.

---

## 3. Post-restore verification

No local `duckdb` CLI is required; use the repo's read-only Go query tool
`apps/go-api/cmd/tmpdbq/main.go` (opens with `?access_mode=read_only`). CGO env on
Windows:

```powershell
$env:CGO_ENABLED='1'; $env:PATH = "C:\msys64\ucrt64\bin;$env:PATH"
cd apps/go-api
go run cmd/tmpdbq/main.go <db_path> "<SQL>"
```

Run these checks on the restored tree (`LevelUp-prod-copy/data/titles/...`). Do NOT run
concurrent `go run`/`go build` (Windows build-cache corruption — run sequentially).

Checks performed and expected shape:

- **shared_matches_v2** (per title): `COUNT(*) FROM match_registry`, `match_participants`,
  and `MIN/MAX` of `COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC')`.
- **metadata**: table list non-empty; spot-count `maps_catalog`, `medal_definitions`,
  `playlists_catalog`, `asset_translations`.
- **shared_social**: table list contains the `_latest` views (media, squad, prestige).
- **shared_pve** (Infinite only): `COUNT(*) FROM pve_match_stats`.
- **1-2 player DBs** per title: `COUNT(*) FROM mv_player_matches`,
  `player_match_enrichment`, `match_skill_rank_latest`.
- **auth tokens**: count of `data/auth/watcher_tokens/*.json`.

### Reference counts (snapshot `9e96ed20`, 2026-06-27) — baseline for the data audit (V9a)

| DB | Metric | Value |
|----|--------|-------|
| Infinite shared_matches_v2 | match_registry / participants | 1780 / 26577 |
| Infinite shared_matches_v2 | start range | 2021-11-19 → 2026-06-25 |
| Infinite metadata | maps / medals / playlists / asset_tr | 123 / 167 / 35 / 9702 |
| Infinite shared_pve | pve_match_stats | 20 |
| Infinite player Madina97294 | matches / enrich / csr_latest | 1182 / 4120 / 1172 |
| Infinite player JGtm | matches / enrich / csr_latest | 958 / 4674 / 948 |
| Infinite player Chocoboflor | matches / enrich / csr_latest | 490 / 2428 / 499 |
| Infinite player XxDaemonGamerxX | matches / enrich / csr_latest | 22 / 157 / 32 |
| H5 shared_matches_v2 | match_registry / participants | 3032 / 24208 |
| H5 shared_matches_v2 | start range | 2015-10-29 → 2023-04-05 |
| H5 metadata | table count | 13 |
| H5 shared_social | table count | 33 |
| H5 player Madina97294 | mv_player_matches | 1424 |
| auth watcher_tokens | *.json files | 9 |

Note on magnitude: `shared_matches_v2.match_registry` holds *shared* match rows (~1.8k
Infinite); the "tens of thousands" of rows live at participant level (26.5k Infinite) and
in per-player enrichment. Both are plausible for 6 tracked players.

---

## 4. Restore = deploy rehearsal

The restored copy is not throwaway: it is the substrate for the **deploy dress rehearsal**
(merge plan step 2). Point a local server at the restored `data/` and boot the branch
binary to confirm boot-time migrations/views apply cleanly on **real production data**
before pushing to `main` (push main = auto-deploy). Any migration surprise must surface
here, not in production. This same copy also feeds the read-only data audit (LOT V9a).

---

## 5. Recommended cadence and follow-ups

- Run this restore test **quarterly** and **before every big-bang merge**.
- Keep the `.restic-password` copied **off-VPS** (the repo is single-disk on the VPS; if
  the disk is lost, both data and repo are lost — `scripts/RESTIC_BACKUP.md` "Limites").
- If `restic snapshots` shows a stale "latest", investigate the timer immediately
  (see the failure mode discovered on 2026-07-07: the ExecStart script lacked the
  executable bit, so every automated 04:00 run failed with `status=203/EXEC` while manual
  runs succeeded — every snapshot to date was manual).
