# Runbook — Deploy Checklist (executable)

EN-only (project rule: runbooks are not translated).

Consolidates deploy hazards that were previously scattered across ADRs and agent memories
into a single tick-box list. **Pushing to `main` = automatic production deploy** (GitHub
Actions `deploy.yml` → `scripts/deploy.sh` on the VPS). Warn the user before pushing.

Each item cites its source so it can be re-verified against the code. Structure:
**pre-deploy → deploy → post-deploy → rollback**.

---

## Pre-deploy

- [ ] **CI on the branch is green** (`.github/workflows`, gate V1e) AND the final local
      gate passed: `go build && go vet && go test ./...`, then
      `-tags=integration -p 1 -timeout 900s ./...` exit 0 (serial `-p 1` is mandatory —
      integration DuckDB gate gives false green otherwise). Front: purge cache, typecheck,
      lint, vitest.
- [ ] **Deploy dress rehearsal on the restored prod copy** (see
      `docs/RUNBOOK_RESTORE_TEST.md`, merge plan step 2): point a local server at the
      restored `data/`, boot the branch binary, confirm boot-time migrations/views apply
      (PME view without dead columns, `_latest` views, `deprecatedPlayerAggregates`,
      prestige/halo5 extraction) and smoke the key pages on real data. No migration
      surprise may remain.
- [ ] **GO/NO-GO — irreversible migrations inventory.** From the deploy dress rehearsal,
      list every NON-reversible migration in the diff (append-only rebuilds, view drops,
      column drops). If any makes the DB incompatible with the previous binary, rollback
      requires a restic restore too — this is the GO/NO-GO criterion (merge plan step 7).
- [ ] **`.env.local` on the VPS has `LEVELUP_ENV=production`** (prod guard armed;
      `scripts/deploy.sh` step 1b warns but does not block if missing → the server boots in
      DEV posture with localhost-only CORS/CSRF).
- [ ] **No host-side backfill running** before triggering a demo regen: a background
      `cmd/backfill_*` holds the DuckDB mono-writer lock; stopping prod for the seed would
      prevent it reopening → crash-loop. `deploy.yml` job `deploy-demo` guards this with
      `pgrep -f '[b]ackfill'` and SKIPs the regen (prod preserved) — confirm the guard is
      intact if that job changed (source: memory `deploy_hazards`, `.github/workflows/deploy.yml`).
- [ ] **Announce the deploy window** (auto-deploy, calm hour, user available).

## Deploy

- [ ] Merge on `main` with a **merge commit, no squash** (per-lot history = traceability):
      `git checkout main && git pull && git merge refactor/audits-2026-07`, then `git push`.
- [ ] `scripts/deploy.sh` runs on the VPS (via `deploy.yml`): `git reset --hard origin/main`,
      `docker compose down`, `up -d --build`, image prune, BuildKit cache bound to 5 GB
      (source: `scripts/deploy.sh` — the 5 GB cap prevents the 2026-06-27 disk-fill incident).
- [ ] **Demo regen is NON-destructive**: it does NOT `rm` `data/demo/warehouse|players`
      before seeding (incident 2026-06-05 left the demo empty on seed failure). The ONLY
      `rm -rf` is for **phantom-directory JSON stubs** (`data/demo/db_profiles.json`,
      `app_settings.json`) that Docker creates as directories at bind-mount time — remove
      those before the real files are written by `seed-demo` (source: `scripts/deploy.sh`
      step 2a, `deploy.yml` job `deploy-demo`).

## Post-deploy

- [ ] **`GET /health` returns 200** on `127.0.0.1:8000` — the healthcheck opens metadata +
      shared read-only and returns match count + DuckDB version, so 200 confirms both the
      binary is up and the DBs open (source: `scripts/deploy.sh` step 4; deploy fails if it
      does not respond within 90 s).
- [ ] **Boot logs show migrations OK, no FATAL.** Logs are per-category files under
      `/opt/levelup/data/logs/*.log` — grep ALL of them, not just one:
      `grep -riE 'FATAL|panic' /opt/levelup/data/logs/*.log`. Check `migration.log`,
      `duckdb.log`, `provider.log`, `server.crash.log` in particular (source: memory
      `auth_logs_per_category_file`; log set verified on the VPS).
- [ ] **`/debug/vars` reachable admin-only.** Mounted behind `RequireAuth` +
      `RequireAdmin`, exposes the `levelup` expvar namespace (source:
      `internal/api/server_apiv1.go` `r.Mount("/debug/vars", http.DefaultServeMux)` inside
      the admin group). Confirm an anonymous request is rejected and an admin request
      returns JSON.
- [ ] **`legacy_source_used_*` telemetry visible in `/debug/vars`** (key `levelup`).
      Counters: `legacy_source_used_duckdb_msal`, `_duckdb_oauth`, `_env_oauth`,
      `_watcher_legacy` (source: `internal/observability/legacy_source.go`). This is the
      machine signal that arms D2 (ADR 0023 Phase 5): while > 0, installs still depend on
      the legacy auth fallback.
- [ ] **NOTE THE D1A PRODUCTION DATE.** D1A (legacy-source telemetry) goes live with this
      merge. Record the exact deploy date in the parent plan §6 — it arms D2 at **≥ 7 days**
      after. TODO fill-in below:

      > **D1A live in production on: `__________` (YYYY-MM-DD).**
      > D2 (`refactor/adr0023-phase5`) may start on/after `live_date + 7d`, gated on
      > `legacy_source_used_*` observed over that window (parent plan step 8).

- [ ] **shared_social durability after writes.** Any social write path must `CHECKPOINT`
      shared_social (ADR 0022) — without it the WAL can be lost (incident #7659). If a
      social write ran during/after deploy, confirm the `CHECKPOINT` fired (source: ADR 0022,
      memory `shared_social_durable_writes_checkpoint`).
- [ ] **First full auto-sync monitored.** Watch `sync.log` for the first complete cycle
      after boot (source: merge plan step 6).
- [ ] Smoke the key pages on prod: Home, Career, Squad, Explorer, Sessions (FR + EN, one
      Infinite + one H5 player).
- [ ] **Xbox device-code endpoint reachable (SSO login).** The `xboxDeviceCodeURL` constant
      is only exercised by an opt-in network guard (double-gated: `integration` build tag +
      `LEVELUP_DEVICE_ENDPOINT_LIVE_CHECK` env — it never runs in `go test ./...` nor in the
      anti-ART `-tags=integration` suite, so no CI flake). It is NOT run by any pipeline —
      run it by hand when validating SSO or after touching the auth/device-code path
      (incident 2026-07-13: the URL regressed to 404 while all mocked tests stayed green):

      ```bash
      LEVELUP_DEVICE_ENDPOINT_LIVE_CHECK=1 \
        go test -tags=integration -run TestXboxDeviceCodeEndpointReachable \
        ./apps/go-api/internal/platform/auth/
      ```

## Rollback

- [ ] **Rollback = `git revert -m 1 <merge-commit>` + push** (redeploys the previous
      application state; deploy is auto). Source: merge plan step 7.
- [ ] **Before relying on revert, re-check the irreversible-migrations inventory** from
      pre-deploy. If a migration made the DB incompatible with the previous binary, the
      revert is NOT enough — you must ALSO restore the DBs from restic
      (`scripts/RESTIC_BACKUP.md` in-place restore: stop `levelup`, `restic restore latest
      --target /opt/levelup`, start `levelup`). This is the GO/NO-GO criterion decided in
      pre-deploy.
- [ ] Confirm `/health` 200 and clean logs after rollback, same as post-deploy.
