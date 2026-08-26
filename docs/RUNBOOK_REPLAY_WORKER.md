# Runbook — Remote replay build worker

Status as of 2026-08-25: **provisioned, NOT activated.** The machine is ready, the unit is
installed but disabled, and no production behaviour has changed. This runbook describes the
architecture, what is already in place, and the exact sequence that turns the worker on at
the v7.5 release.

Related files:

- `apps/go-api/cmd/replay-worker/` — the worker binary (`main.go`, `job.go`, `memlimit.go`).
- `scripts/deploy-worker.sh` — build + install, run on the worker host by CI.
- `packaging/systemd/levelup-worker.service` — the versioned unit.
- `.github/workflows/deploy.yml`, job `deploy-worker` — the CI trigger.
- `.ai/V7.5/PLAN_OUVRIER_DISTANT.md` — the plan this deployment closes.

---

## 1. Architecture — who talks to whom

```
  csstat (compute VPS)                          lvelup (production web VPS)
  +-------------------------+                   +----------------------------------+
  | levelup-worker.service  |  HTTPS, outbound  | nginx :443 -> Go server :8000    |
  |  /opt/levelup/bin/      | ----------------> |  POST /api/v1/internal/          |
  |    replay-worker        |   Bearer token    |       build-queue/{claim,        |
  |  --work /var/lib/...    |                   |        artifact,complete,        |
  +-------------------------+                   |        heartbeat}                |
             |                                  +----------------------------------+
             | HTTPS, outbound, NO auth
             v
     Azure film CDN (pre-signed chunk URLs handed out inside the job)
```

**The worker only ever pulls.** No inbound port, no shared filesystem, no queue broker:
three POSTs and an artifact upload are the whole protocol
(`apps/go-api/internal/api/handlers/build_worker.go`, header comment).

**What the worker token opens — and nothing else.** `LEVELUP_BUILD_WORKER_TOKEN` guards the
four routes under `/api/v1/internal/build-queue/` (`claim`, `artifact`, `complete`,
`heartbeat`), mounted behind the `RequireWorkerToken` middleware
(`build_worker.go:88-103` and `:254-271`, constant-time comparison). It grants no Halo
token, no database access, nothing beyond "take resolved work and report the result". The
web server has already resolved the manifest and put **pre-signed CDN URLs** in the job, so
the worker needs no credentials of its own.

Responses to expect from those routes:

| Situation | Response |
|---|---|
| No token configured on the server | `503 build_queue_disabled` |
| Token configured, none or wrong one presented | `401 invalid_worker_token` |
| Valid token, empty queue | `200` with an empty body (a resting worker is nominal) |

That token is the **only** authentication on these routes: they accept no cookie and read no
session. That is also why they are exempt from the server-wide CSRF origin check — see
section 3, "The CSRF blocker". A `403 csrf_rejected` here is a deployment symptom, not an
authentication one.

**Where the work comes from.** `replaybuild.DecidePlacement` (`internal/replaybuild/placement.go`)
is the single decision point. In production the default is `worker`: the web VPS never
decodes a film itself (`ErrLocalBuildInProduction`, `placement.go:43-44`). If the worker
token is absent, `worker` degrades to `off` with an explanatory error that
`LogPlacement` writes as a WARN (`placement.go:72-77` and `:104-110`) — the queue is not
filled by something nobody will drain.

---

## 2. Provisioned state (2026-08-25)

**csstat — compute VPS** (Debian 12 bookworm, 6 vCPU, 7.7 GiB RAM, 129 GB free):

- `deploy` account; `/opt/levelup` is a public HTTPS clone of the repository owned by it.
- Go toolchain in `/usr/local/go` (the build is CGO: DuckDB is a transitive dependency of
  the `go-api` module), plus gcc 12.2, git 2.39, systemd 252.
- `/etc/levelup-worker.env`, root-owned, mode 0600, carrying `LEVELUP_BUILD_WORKER_TOKEN`.
  systemd reads it as root before dropping to `deploy` — the service account never needs
  read access to the secret itself.
- `levelup-worker.service` **linked and disabled**:
  `systemctl link /opt/levelup/packaging/systemd/levelup-worker.service`.
  Linking (not copying) is what lets CI refresh the unit: `deploy` has a sudoers rule
  limited to `systemctl daemon-reload` and `systemctl restart levelup-worker`, so it cannot
  copy a file into `/etc/systemd/system/`.
- GitHub secrets `VPS2_HOST` / `VPS2_SSH_KEY`, distinct from the production ones.

**lvelup — production web VPS**: unchanged. A worker token is staged, inert, at
`~/levelup-worker-token.env.pending`. It is not wired into any service and the server
therefore still answers `503` on the worker routes.

**What CI does today.** On every push to `main`, after the web deployment, the
`deploy-worker` job runs `scripts/deploy-worker.sh` on csstat. Because the unit is disabled,
the script updates the binary, prints "service desactive : binaire mis a jour, pas de
restart", and exits 0. That is the nominal pre-activation path, not a failure.

---

## 3. Pre-flight check: is `/api/v1/internal` reachable from the outside?

**What the versioned nginx configuration shows.** `packaging/nginx/levelup.conf:85-90`
proxies the whole `location /api/` to `http://127.0.0.1:8000`, with `client_max_body_size 2g`
and `proxy_read_timeout 3600s` — large enough for artifact uploads. Neither
`levelup.conf` nor `demo.conf` contains any `deny`/`allow` directive or any mention of
`internal`. On paper, nothing blocks `/api/v1/internal` from the internet, and the worker
protocol goes through.

**Why that is not proof.** Those files are installed by hand (`sudo cp`, see the header of
`levelup.conf`) and certbot rewrites parts of them. The repository copy does not establish
what is actually running.

**The CSRF blocker, measured on 2026-08-25.** A supervisor dry run of `replay-worker --once`
from csstat against production returned, for every worker route:

```
HTTP 403 {"code":"csrf_rejected","message":"origin non autorisée"}
```

nginx was not at fault, and neither was the token. The Go server mounts a CSRF
origin check across the **whole** root router (`applyTransverseMiddlewares` in
`internal/api/server.go`), and it rejects any mutating request that carries no allowed
`Origin`/`Referer` — which is exactly what a `net/http` client such as the worker sends. The
protocol was dead on arrival, *before* any token check. The existing end-to-end proof could
not see it: it mounted the worker routes on a bare `chi.NewRouter()`, outside the transverse
stack.

Fixed by a **targeted CSRF exemption** for the `/api/v1/internal` prefix only
(`internal/api/middleware/csrf.go`, `CSRF(origins, exemptPrefixes...)`). These routes accept
no cookie and read no session — their only authentication is the dedicated Bearer token — so
the origin check protects nothing there. The rest of the transverse stack (security headers,
request id, rate limit, slog, compression) still applies, and CSRF is unchanged everywhere
else. Regression cover: `internal/api/csrf_transverse_stack_cgo_test.go` exercises the real
assembled router, not a bare mount.

**Verify on production before activating.** From any machine, no token:

```bash
curl -sS -i -X POST -H 'Content-Type: application/json' -d '{}' \
  https://lvelup.info/api/v1/internal/build-queue/claim
```

Expected status codes, in order of what they tell you:

| Response | Meaning |
|---|---|
| `403` + `"code":"csrf_rejected"` | The running build **predates the CSRF fix** (the state measured in production on 2026-08-25). No token change will help — deploy a build that contains it. |
| `503` + `"code":"build_queue_disabled"` | Correct answer **before** step 1: the request reached the worker guard, no token is configured server-side yet. |
| `401` + `"code":"invalid_worker_token"` | Correct answer **after** step 1: token configured, the one presented (here: none) does not match. |
| `404` + an nginx HTML body | nginx does not route this path — fix the production nginx configuration before going further, the worker would silently never get any work. |

A JSON body, whatever the status code, means the request reached the Go server. After the
fix is deployed, `403 csrf_rejected` must never appear on these routes again.

Never put the token in the URL. It travels as `Authorization: Bearer <token>`.

---

## 4. Activation sequence (v7.5 release)

Run the steps in order. Steps 1 and 2 concern production; step 3 concerns csstat; step 4 is
a product decision, not a technical formality.

### Step 1 — Wire the staged token into production

On lvelup, as `deploy`:

```bash
cat ~/levelup-worker-token.env.pending          # inspect, do not echo into logs
cat ~/levelup-worker-token.env.pending >> /opt/levelup/.env.local
cd /opt/levelup && docker compose up -d levelup
```

The production service reads its environment from `.env.local` (`docker-compose.yml`,
`env_file:`). Use `up -d`, **not** `restart`: `docker compose restart` does not re-read
`env_file`, so the container would come back without the variable.

The value must be **byte-identical** to the one in `/etc/levelup-worker.env` on csstat; the
server compares them in constant time and answers `401` on any mismatch.

Check: the pre-flight curl of section 3 now returns `401`, and
`DecidePlacement` stops degrading (no more "mise en file demandee mais le protocole ouvrier
n'est pas ouvert" WARN in the logs).

### Step 2 — Repair impoverished artifacts BEFORE the first worker pass

Copied verbatim from `.ai/V7.5/PLAN_OUVRIER_DISTANT.md`, section "COMMANDE DE REMEDIATION":

```bash
# 1. Constater (READ-ONLY, aucun decodage, aucune ecriture)
go run ./apps/go-api/cmd/levelup backfill-replay --repair-impoverished --dry-run

# 2. Reparer (un film par processus, plafond memoire 3 GiB par defaut)
go run ./apps/go-api/cmd/levelup backfill-replay --repair-impoverished
```

`--repair-impoverished` is mutually exclusive with `--force` (parse error: the mode is
already a targeted selection). Do **not** substitute `--only-existing`, which skips an
impoverished artifact as "already up to date" because it carries the current schema number.

Witness run of 2026-08-25 (dry-run, read-only, 951 cached films): 2 repairable artifacts
(`24dbb67d...`, 29 chunks; `64e8adfa...`, 45 chunks), 32 already complete, 1 off-schema,
916 without artifact. 74 chunks to decode in total — a few minutes.

### Step 3 — Enable the worker on csstat

```bash
sudo systemctl enable --now levelup-worker
systemctl is-enabled levelup-worker     # expect: enabled
systemctl is-active  levelup-worker     # expect: active
```

From this point, `scripts/deploy-worker.sh` restarts the service on every deployment: it
compares the string returned by `is-enabled` to `enabled` exactly (systemd returns exit 0
for several states, and `1` for `linked` as well as for `disabled`).

### Step 4 — Serve the replay publicly (`LEVELUP_REPLAY_PUBLIC=1`)

**This is a product decision, and its stated criterion is currently NOT met.** Read
`apps/go-api/internal/api/handlers/replay_local_gate.go` before doing it.

What the variable does, exactly: `replayPublic` is read once at start-up
(`replay_local_gate.go:74`) and lifts the `LocalOnlyReplay` middleware
(`:110-119`), which otherwise answers `404 replay_not_available` to every replay request
whose **TCP peer address** is not loopback (headers are deliberately not trusted,
`:78-96`). It is mounted on the replay routes only (`internal/api/server_apiv1.go:687`);
it has no effect whatsoever on the worker protocol — steps 1 to 3 work with the gate in
place, and the artifacts are built either way. This step only decides whether visitors can
*see* the 2D replay.

The gate's removal criterion is shot coverage >= 88 percent on a named seven-film corpus.
As documented in the file, `64e8adfa` sits at 87.39 percent, below the floor. The criterion
"does not renegotiate itself": lifting the gate is an explicit user arbitration between
accuracy and availability, to be recorded when taken.

If taken: add `LEVELUP_REPLAY_PUBLIC=1` to `/opt/levelup/.env.local` on lvelup and
`docker compose up -d levelup` (same reason as step 1).

### Step 5 — Verify

1. **Admin build queue table** — `/admin/sync` in the app (`BuildQueueSection`, fed by
   `GET /api/v1/admin/monitoring/build-queue`). Expect the worker to appear with id
   `csstat`, version `replay-worker/1`, and jobs moving from `queued` to `running` to
   `succeeded`.
2. **Worker journal** — on csstat:
   ```bash
   journalctl -u levelup-worker -n 100 --no-pager
   ```
   Expect `replay-worker: demarre` with `work_dir=/var/lib/levelup-worker/work` and
   `films_conserves=false`, then `job pris` / `artefact transmis` / `job reussi`. After each
   job, `morceaux de film supprimes` confirms the disk is being released — that cleanup only
   happens because `--work` points outside the repository cache (`main.go:99-108`,
   `job.go:282-299`).
3. **Film-bomb path** — if a job hits the 3 GiB ceiling, the queue row shows
   `error_code=memory_exceeded` with the match id and the measured peak, and the journal
   shows `PLAFOND MEMOIRE DEPASSE`. The report is sent to the server **before** the process
   exits with code 3 (`job.go:135-153`), so the job is already `failed` with an explicit
   reason; systemd restarts the worker 30 seconds later and it moves on.

---

## 5. Rollback

Deactivating is two independent moves; either one alone already stops the pipeline.

```bash
# On csstat — stop and disable the worker
sudo systemctl disable --now levelup-worker

# On lvelup — remove the token line from /opt/levelup/.env.local, then
cd /opt/levelup && docker compose up -d levelup
```

Removing the token is the one that matters: `DecidePlacement` sees `WorkerConfigured=false`
and returns `PlacementOff` with the message "mise en file demandee mais le protocole
ouvrier n'est pas ouvert" (`placement.go:72-77`), logged as a WARN by `LogPlacement`
(`:104-110`). Queueing stops, the web server still never decodes a film in production
(`ErrLocalBuildInProduction`), and the replay degrades to "whatever artifacts already
exist" — a documented, non-breaking degradation. Nothing is lost: artifacts already built
stay served.

Stopping only the service, without removing the token, leaves jobs accumulating in the
queue with nobody to drain them. Prefer removing the token if the pause is going to last.

---

## 6. Known limits and gotchas

- **Pre-signed CDN URLs age inside the queue.** The payload is resolved at enqueue time and
  never refreshed (`ops/build_queue.go`). A job that waits longer than the Azure signature
  validity fails at download and burns its three attempts. Validity is not measured; do not
  promise a deep queue.
- **Exit codes** (useful when reading the journal): `0` clean stop, `1` protocol error
  (server unreachable, token refused), `2` missing configuration (token or repository root),
  `3` memory ceiling hit. `2` and `1` retry every 30 seconds by design — once the cause is
  fixed, the worker resumes on its own without manual intervention.
- **The unit must stay linked, not copied.** If someone copies the file into
  `/etc/systemd/system/`, `scripts/deploy-worker.sh` detects the divergence and prints the
  fix, but can no longer refresh it (restricted sudoers).
- **`systemd-analyze verify` has not been run on this unit** from the authoring workstation
  (Windows). Run it once on csstat:
  `systemd-analyze verify /opt/levelup/packaging/systemd/levelup-worker.service`.
  Additional hardening (`ProtectSystem=strict`, `ProtectHome`) was deliberately left out for
  the same reason and can be instructed there.
