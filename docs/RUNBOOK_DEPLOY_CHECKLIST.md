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
      lint, vitest. Since 2026-07-26 the pipeline enforces this too (the deploy waits for the
      CI verdict, see the Deploy section) — checking it here still matters: it is what avoids
      burning a ~30-minute deploy window to learn that the build was red.
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
- [ ] **The deploy WAITS for the CI verdict** (D29, 2026-07-26 — `deploy.yml` job `attente-ci`).
      Before that date CI and Deploy ran in parallel: production received the code before any
      verdict, and a red CI had no consequence (40 % of `main` runs red over 14 days, no visible
      effect). Now the first deploy job polls the Actions API every 30 s for the CI run of the
      **same `head_sha`** and releases the deploy only on `success`. Plan for:
      - **Latency**: production is updated ~25-35 min after the push (CI wall time, bounded by
        the `go-coverage` job) plus the usual deploy time. It is no longer immediate.
      - **CI red = nothing deployed.** The job fails with an explicit message; fix and re-push,
        or use the emergency valve below. Do not re-run the deploy on a red CI — by design it
        will keep refusing.
      - **No CI run for that sha = nothing deployed either**: `attente-ci` polls 55 min then
        fails. That means the trigger invariant broke: the `paths-ignore` set of `ci.yml` must
        stay a SUBSET of `deploy.yml`'s, and the pushed branch must be covered by `ci.yml`'s
        `branches` filter. Guard-rail test that fails the build before this can reach `main`:
        `apps/go-api/internal/archlint/ci_deploy_triggers_test.go`.
- [ ] **Emergency valve, urgent hotfix ONLY** — Actions > "Deploy to VPS" > *Run workflow* on
      `main`, tick **`ignorer_attente_ci`**. It skips ONLY the CI wait: `pre-check` still runs and
      the deploy job stays gated on `refs/heads/main` (a dispatch from any other branch deploys
      nothing). Reserved for "production is down and the fix cannot wait ~30 min". The CI verdict
      remains DUE: read the CI run of that commit afterwards and treat a red as a rollback
      candidate.
- [ ] `scripts/deploy.sh` runs on the VPS (via `deploy.yml`): `git reset --hard origin/main`,
      **`docker compose build` while the old containers are still up**, then only on build
      success `docker compose down` + `up -d` (no `--build` — reuses the images just built),
      image prune, BuildKit cache bound to 12 GB (source: `scripts/deploy.sh` — build-before-down
      ordering fixes the 2026-07-23 incident where a failed build left prod down until manual
      intervention; the cap prevents the separate 2026-06-27 disk-fill incident).
- [ ] **Do not lower the BuildKit cache cap below one build's working set** (~5.7 GB measured).
      It was 5 GB from 2026-07-24 to 2026-07-26: below the working set, every deploy evicted the
      next one's cache, so every deploy became a cold CGO build whose memory peak froze the VPS
      three times on 25-26/07. Same reasoning bans `docker system prune -a` on a schedule: it
      deletes the tagged base images (`golang`, `node`, `debian`) because `until` filters on
      image CREATION date and no container ever references them (source: `scripts/deploy.sh`
      step 3b, `scripts/systemd/levelup-docker-prune.service`).
- [ ] **Demo regen is NON-destructive**: it does NOT `rm` `data/demo/warehouse|players`
      before seeding (incident 2026-06-05 left the demo empty on seed failure). The ONLY
      `rm -rf` is for **phantom-directory JSON stubs** (`data/demo/db_profiles.json`,
      `app_settings.json`) that Docker creates as directories at bind-mount time — remove
      those before the real files are written by `seed-demo` (source: `scripts/deploy.sh`
      step 2a, `deploy.yml` job `deploy-demo`).
- [ ] **Demo regen publishes only a checked generation**: `seed-demo` writes into
      `data/demo.generation`, runs the anonymization value check, and only then swaps the
      generated items into `data/demo` (`runtime/` and `auth/` untouched). A failed regen
      publishes nothing; the previous demo stays online (source: `ops/seed_demo_publish.go`).
- [ ] **Demo replay films are provisioned (once)**: the frozen demo replays are re-cooked from
      their films when the artifact schema moves up, and the web VPS keeps no film cache. The
      films live in the persistent store `data/demo_films/<title>/` (outside `data/demo`, never
      regenerated). Provision it once from a workstation that has the films (run `seed-demo`
      there first: it fills the store from `data/cache`):

      ```bash
      rsync -a data/demo_films/ deploy@<vps>:/opt/levelup/data/demo_films/
      ```

      Without it, a demo replay whose production artifact is missing or outdated keeps its
      previous artifact (or is not served) — never a crash. A re-cook runs one film per child
      process under an explicit memory cap (`ops/seed_demo_replay_cook.go`,
      `demoReplaySoftLimit`), while prod and demo are stopped for the seed.

## Post-deploy

- [ ] **`GET /health` returns 200** on `127.0.0.1:8000` — the healthcheck opens metadata +
      shared read-only and returns match count + DuckDB version, so 200 confirms both the
      binary is up and the DBs open (source: `scripts/deploy.sh` step 4; deploy fails if it
      does not respond within 90 s). While it boots, the server already listens: `/health`
      and `/api/*` answer **503 `server_starting`** (step in `details.step`) and the web page
      is served and waits. `deploy.sh` (`curl -sf`) and the Docker healthcheck
      (`-health-check`, `start_period` 20 s) only accept 200, so both keep waiting for the
      real "ready" — a 503 during the first seconds is expected, not a failure.
- [ ] **Boot logs show migrations OK, no FATAL.** Logs are per-category files under
      `/opt/levelup/data/logs/*.log` — grep ALL of them, not just one:
      `grep -riE 'FATAL|panic' /opt/levelup/data/logs/*.log`. Check `migration.log`,
      `duckdb.log`, `provider.log`, `server.crash.log` in particular (source: memory
      `auth_logs_per_category_file`; log set verified on the VPS).
- [ ] **Media tooling present (ffmpeg/ffprobe + codecs).** Run
      `docker compose exec levelup levelup check-env` and confirm ffmpeg AND ffprobe both
      resolve, and the required encoders (`libwebp`, `libx264`, `aac`) and muxers (`hls`,
      `mp4`) are all `[OK]`. A missing component silently breaks WebP thumbnails, HLS
      transcode or live remux at the first media upload (source:
      `internal/ops/media_tooling.go`; the server also logs this non-blocking at boot —
      grep `media tooling` in the console/boot logs). NOTE: the live VPS nginx config must
      mirror `packaging/nginx/levelup.conf` — in particular `client_max_body_size 2g;` on
      `location /api/`, or video uploads fail with HTTP 413.
- [ ] **`/debug/vars` reachable admin-only.** Mounted behind `RequireAuth` +
      `RequireAdmin`, exposes the `levelup` expvar namespace (source:
      `internal/api/server_apiv1.go` `r.Mount("/debug/vars", http.DefaultServeMux)` inside
      the admin group). Confirm an anonymous request is rejected and an admin request
      returns JSON.
- [ ] **No legacy auth reader runs at boot** (ADR 0023 Phase 5, closed 2026-09-13).
      The legacy auth fallbacks were removed on 2026-08-25, and the one-shot boot
      migration, their last consumer, on 2026-09-13 (`7fd6d0fcb`), once its criterion
      held in prod: `auth_migration: scan terminé` with `rt_migrated=0` at every boot
      since 2026-06-14 (re-checked on 2026-10-02 in `auth.log`: 499 scans, the only two
      non-zero ones on 2026-06-13). On the first deploy that ships this removal,
      `auth.log` must show NO new `auth_migration:` line after the boot — one still
      appearing means the previous binary is running. Refresh tokens come solely from
      `data/auth/watcher_tokens/{xuid}.json` (source: ADR 0023, Phase 5 closure section;
      anti-resurrection ratchets in `internal/platform/auth/sentinel_test.go`).

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
