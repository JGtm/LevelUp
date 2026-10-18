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
      - **Latency**: production is updated ~15-25 min after the push (CI wall time, bounded by
        the longest `go-coverage` lot, then the `go-coverage-merge` job) plus the usual deploy time. It is no longer immediate.
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
      shared read-only and answers `{"status":"ok"}` only, so 200 confirms both the
      binary is up and the DBs open (source: `scripts/deploy.sh` step 4; deploy fails if it
      does not respond within 90 s). While it boots, the server already listens: `/health`
      and `/api/*` answer **503 `server_starting`** (step in `details.step`) and the web page
      is served and waits. `deploy.sh` (`curl -sf`) and the Docker healthcheck
      (`-health-check`, `start_period` 20 s) only accept 200, so both keep waiting for the
      real "ready" — a 503 during the first seconds is expected, not a failure. Versions,
      counts and media tooling are not public: an admin reads them at `GET /api/v1/admin/health`.
      `deploy.sh` reads the served version from the container environment, not from `/health`.
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
      mirror `packaging/nginx/levelup.conf` (and `packaging/nginx/rate_limit.conf` in
      `/etc/nginx/conf.d/`, `packaging/nginx/snippets/levelup-assets-v.conf` in
      `/etc/nginx/snippets/`, included by the `/api/v1/assets/` location of both sites) — in
      particular the body-size locations: `/api/` is capped at
      `1m`, and only the media upload (`600m`), OpenSpartan import (`1100m`) and worker
      (`20m`) locations accept more; a missing upload location makes video uploads fail
      with HTTP 413.
- [ ] **`/debug/vars` reachable admin-only.** Mounted behind `RequireAuth` +
      `RequireAdmin`, exposes the `levelup` expvar namespace (source:
      `internal/api/server_apiv1.go` `r.Mount("/debug/vars", http.DefaultServeMux)` inside
      the admin group). Confirm an anonymous request is rejected and an admin request
      returns JSON. On the public demo (no accounts) it is rejected for everyone with
      `403 demo_mode_forbidden`: in demo mode `RequireAuth`/`RequireAdmin` refuse every route
      except the named reads of `middleware/demo_guarded_reads.go`.
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

- [ ] **Post-deploy recomputes (v7.5).** The v7.5 release changes derived values that the deploy does not
      recompute (Halo 5 derived values counted Campaign matches, LUSR v1 rows, film decoder
      revisions, the skill bit of 157 Infinite matches, the token-pool consent). They are an
      operator gesture, ordered, in the section **Post-deploy recomputes (v7.5)** below. Nothing in
      it runs by itself.

## Post-deploy recomputes (v7.5)

**Operator gesture — nothing here runs by itself** (backlog 2026-10-16 E9, deploy decision D-14).
The deploy ships code; it does not recompute stored values. Several derived values written by
earlier builds are stale and are only rewritten by the steps below, in this order:

- Halo 5 derived values (engagement, performance score, assists model, LUSR, sessions) were
  computed with Campaign matches counted; the loaders now exclude them but never recompute on
  their own.
- LUSR v1 rows written by older exclusions are still served by `match_skill_rank_latest_by_type`
  until a v2 replay overwrites them, and the 8 LUSR components have not been written since
  2026-07-18 (E4: the v1 path was removed, the v2 replay writes them again).
- Film-derived tables (kill positions, lives, death context, replay artifacts) moved revisions
  (E8).
- The 157 Infinite matches without the skill bit (E5).
- The 9 token-pool consents (ADR 0038).
- The demo copies JGtm's production databases, so it is re-seeded last.

**Conventions** (all commands from `/opt/levelup` on the VPS):

- `LU` = `docker compose run --rm --no-deps levelup levelup` — the CLI shipped in the image (the
  same form `deploy.yml` uses). Define it once per shell:
  `LU() { docker compose run --rm --no-deps levelup levelup "$@"; }`. A CLI pass writes the
  shared database, so **the server is stopped** (single writer, ADR 0013); the `--dry-run` of a
  pass that runs migrations needs it stopped too.
- "Stopped" = `docker compose stop levelup` before, `docker compose up -d levelup` after the window.
  `levelup-demo` is left running until R10.
- "Admin route" = the browser console of the production site, signed in as admin (same-origin
  `fetch` with `credentials: 'include'`; the CSRF check is origin-based). Jobs are polled with
  `GET /api/v1/jobs/<job_id>` until `status` is `succeeded`.
- A pass prints one line per player (`backfill <pass> OK: title=… gamertag=…`) and a `batch` line
  with `failed=N`; a non-zero `failed` makes the command exit non-zero. A profile without a player
  database for the title is `SKIP`ped, listed, not created.

**Preconditions.** The whole `deploy.yml` run is finished (jobs `deploy`, `deploy-demo` AND
`deploy-worker` — `deploy-demo` stops and restarts `levelup` itself, so a window opened too early
collides with it), `GET /health` is 200, boot logs are clean, and a fresh restic snapshot exists
(`scripts/RESTIC_BACKUP.md`). No host-side `backfill` process is running (`pgrep -f '[b]ackfill'`
is empty).

| Step | What | Server | Needs | Duration |
|------|------|--------|-------|----------|
| R1 | Skill bit of 157 Infinite matches | stopped | — | seconds (SQL update); not measured |
| R2 | Halo 5 engagement scores | stopped | — | not measured |
| R3 | Halo 5 performance scores | stopped | — | not measured |
| R4 | Halo 5 assists model | stopped | — | not measured |
| R5 | LUSR v2 replay, every tracked Infinite player | stopped | — | one player, 1 108 matches: 47 s (E4.6, local copy) |
| R6 | LUSR v2 replay, every tracked Halo 5 player | stopped | — | not measured |
| R7 | Film sequence (replay re-cook, kill source, usage summary, pad tiers) | stopped | the user's explicit go; its own window | hours; `--dry-run` prints the plan, `backfill-killsource --dry-run` an ETA |
| R8 | Start the server, recompute the sessions | up, admin route | R7 finished | not measured |
| R9 | Pool consent grandfathering (9 players) | up | server up | seconds |
| R10 | Demo re-seed | prod + demo stopped for the seed | R1-R8 done (the demo copies the recomputed JGtm data) | not measured; production is down only while `levelup` is stopped |

R1 to R6 are one maintenance window (a single `docker compose stop levelup`); R7 is a second,
longer window on its own; R8 to R10 follow with the server up.

### R1 — skill bit of the 157 Infinite matches (stopped)

157 Halo Infinite matches first synced between 2026-05-25 and 2026-05-30 (the days the batch path
went live without setting the completion bits; JGtm 77, Chocoboflor 70, XxDaemonGamerxX 10 in
prod) carry no skill bit (`4`) in `match_registry.backfill_completed`. The participants bit
(`512`) of the same matches is set by the migration `shared_registry_participants_bit_v1`; that
migration does not touch the skill bit. `levelup reset-bitmasks` sets it only where the skill
data is already there (`match_participants.team_mmr IS NOT NULL`).

```bash
docker compose stop levelup
LU reset-bitmasks --dry-run
#   -> "[skill] match_registry à updater" counts the matches whose bit will be set (<= 157)
LU reset-bitmasks --apply
LU reset-bitmasks --dry-run      # the [skill] counters must now read 0
```

Matches still unmarked afterwards have no skill data. No command re-fetches it today:
`levelup backfill` has no `--skill` flag, and the HTTP backfill job reports the `skill` scope as
"detected but not implemented" (`BackfillOrchestrator.warnUnimplemented`). They keep the bit unset;
nothing else reads it (decision for the user if they should be fetched: it needs a new pass).

### R2 to R4 — Halo 5 derived values (stopped)

`--all --title halo_5` takes the Halo 5 profiles of `db_profiles.json` (`auth_only` profiles
excluded). One command per pass, in the order of the post-sync pipeline:

```bash
LU backfill --all --title halo_5 --engagement-scores --force   # R2
LU backfill --all --title halo_5 --perf --force                # R3
LU backfill --all --title halo_5 --assists-model --force       # R4
```

Citations are not applicable to Halo 5: the title does not declare `citations.engine` (it serves
the native commendations), and `backfill --all --title halo_5 --citations` answers
`non supporte pour le titre halo_5` and runs nothing. Sessions are R8 (no CLI form).

Admin-route equivalent for R2 and R3 (server up, one call per player; `force_rescan` is needed or
the job recomputes only what is missing; the assists model has **no route**, CLI only):

```js
await (await fetch('/api/v1/backfill/start', {method: 'POST', credentials: 'include',
  headers: {'Content-Type': 'application/json'},
  body: JSON.stringify({player_slug: 'JGtm', title: 'halo_5', engagement_scores: true,
                        performance_scores: true, force_rescan: true})})).json()
```

(`title` absent = the default title; a player not tracked on that title answers 404
`player_not_found`. Do not put `lusr` in this body: use R5/R6.)

### R5 and R6 — LUSR v2 replay of every tracked player, both titles (stopped)

Rewrites `match_skill_rank` (v2) and the 8 `lusr_component_history` components for every player,
which also supersedes the stale v1 rows:

```bash
LU backfill --all --lusr                    # R5, default title (halo_infinite)
LU backfill --all --title halo_5 --lusr     # R6
```

Admin-route equivalent (server up), one call per player and title (the report lists the tracked
players of the title; skip any `auth_only` slug):

```js
for (const title of ['halo_infinite', 'halo_5']) {
  const rep = await (await fetch(`/api/v1/admin/monitoring/lusr-gaps?title=${title}`, {credentials: 'include'})).json();
  for (const p of rep.players) {
    const r = await fetch(`/api/v1/admin/monitoring/lusr-gaps/${encodeURIComponent(p.player_slug)}/recompute?title=${title}`,
                          {method: 'POST', credentials: 'include'});
    console.log(title, p.player_slug, r.status, await r.json());   // {ok: true, updated: N}
  }
}
```

### R7 — film sequence (stopped, its own window, on the user's explicit go)

Replay cooking is never run in batch without an explicit go (memory: one film = one child process
under a cap, `--mem-limit-gib` to raise it). The film cache must be under `data/cache` (or
`--cache`); if production keeps no film cache, 7a and 7b cannot decode here (see
`docs/RUNBOOK_REPLAY_WORKER.md`). Order is the E8.6 order; 7c and 7d read the artifacts cooked by 7a
and do not depend on 7b:

```bash
# 7a  re-cook the artifacts already served (schema bump): republishes from the persisted film
#     facts when fresh ("republier", fast), re-decodes otherwise ("redecoder", tens of seconds a film)
LU backfill-replay --only-existing --dry-run
LU backfill-replay --only-existing
# 7b  kill source: rewrites kill_positions, kill_openings, match_lives, match_death_context,
#     match_death_mates (append-only, read through _latest). Resumable: relaunch to resume;
#     `LU backfill-killsource --status` from another terminal shows where it stands
LU backfill-killsource --dry-run
LU backfill-killsource
# 7c  equipment and pad usage projected from the artifacts
LU backfill-usage-summary --dry-run
LU backfill-usage-summary
# 7d  weapon tiers of pad pickups; --force: rows already written carry the previous join
LU backfill-pad-tiers --dry-run
LU backfill-pad-tiers --force
```

- Harvest films have no replay artifact yet (the map had no bounds catalogue entry before E8.3), so
  `--only-existing` does not cook them. Cook them one by one, on the user's go,
  with `LU backfill-replay --one <match_id>`, then `LU backfill-usage-summary --match <id>` and
  `LU backfill-pad-tiers --match <id> --force` for those matches.
- To confirm with the pilot: `docs/COMMANDS.md` states that `backfill-vehicle-takes` keys its
  resume on presence and needs `--force` after a re-bake; whether the schema 99 re-bake changes the
  occupation read it projects is not established here.

### R8 — start the server, recompute the sessions (up)

```bash
docker compose up -d levelup
curl -sf http://127.0.0.1:8000/health      # 200 before going on
```

Sessions have no CLI form: admin route, every profile of every title, each on the databases of
its own title (before this release the route recalculated every profile on the Infinite
databases):

```js
await (await fetch('/api/v1/settings/sessions/recalculate', {method: 'POST', credentials: 'include'})).json()
```

### R9 — token-pool consent grandfathering (server may stay up)

`ADR 0038`: a token joins the shared pool only on a recorded consent. Nine players gave theirs
outside the application; record exactly those, **never `XxGdakilla187xX`, never `--all`**. Simulation
first, then `--yes`. The command only writes `data/auth/watcher_tokens/`, never a DuckDB file; the
running pool applies it at its next scan.

```bash
docker compose exec -T levelup levelup pool-sharing list
docker compose exec -T levelup levelup pool-sharing grandfather \
  --players JGtm,Chocoboflor,XxDaemonGamerxX,Madina97294,DankerGlue,GeleJugefi,Trimbutton,QuiteSiren,UppedJoker
docker compose exec -T levelup levelup pool-sharing grandfather \
  --players JGtm,Chocoboflor,XxDaemonGamerxX,Madina97294,DankerGlue,GeleJugefi,Trimbutton,QuiteSiren,UppedJoker --yes
```

### R10 — demo re-seed (after R1-R8)

The `deploy-demo` job already re-seeded the demo right after the deploy, **before** the
recomputes. The demo copies JGtm's production databases and replay artifacts and filters the
Campaign matches of the frozen Halo 5 manifest (`config/demo/JGtm/halo_5.json`) at load, so it
is re-seeded once more now. Either dispatch *Deploy to VPS* on `main` (a full redeploy; its
`deploy-demo` job does it) or replay the job's own steps by hand — the pre-checks and the exit
guarantees of the job (`.github/workflows/deploy.yml`, job `deploy-demo`) apply:

```bash
pgrep -f '[b]ackfill' && echo "host backfill running: stop here"      # the job skips the regen on a hit
docker compose stop levelup levelup-demo
docker compose run --rm levelup levelup index-media --gamertag JGtm --title halo_5 --buffer-min 10 \
  || echo "index-media halo_5 failed (no H5 clip this regen)"
docker compose run --rm levelup levelup seed-demo --gamertag JGtm --service-tag SPTA --out data/demo --max-media 12
docker compose up -d levelup                          # run these two EVEN IF seed-demo failed:
docker compose up -d --force-recreate levelup-demo    # a failed seed publishes nothing, the previous demo stays
```

`data/demo_films/` must already be provisioned (see *Deploy* above). Then smoke the demo (Home,
Sessions, one Halo 5 page) and `GET /health` on both containers.

### Decisions still open

- **Milestone dates (Halo 5 `earned_at`).** The source is `apps/go-api/cmd/backfill_milestone_dates`
  (`ops.BackfillMilestoneDates`; `--data-root data/titles/halo_5 [--apply]`, server stopped,
  dry-run by default). The production image does **not** ship it (the Dockerfile builds
  `levelup-server`, `levelup` and `backfill_objective_stats` only), so no command of this section
  can run it. Either add a build stage like `backfill_objective_stats`, or leave the dates: a date
  that cannot be derived is stored NULL and the front shows none, never a wrong date.

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
