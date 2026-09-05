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

Rounds of round-decided modes (ADR 0032) — a column only the API can fill, so a re-sync
never repairs it. **Server stopped** for `--apply` (single writer, ADR 0013):

```bash
# dry run: no write, no write lock taken
go run ./cmd/backfill-team-rounds --gamertag X

# apply — restricted BY DEFAULT to the variants declared in regulation.toml
# [rounds_decide] (26 matches, ~7 s). --all covers the whole corpus (~1 900 API calls).
go run ./cmd/backfill-team-rounds --gamertag X --apply [--all] [--limit N] [--match ID]
```

#### Projecting replay artifacts into the database — RELEASE ORDER MATTERS

Two passes read the already-cooked replay artifacts (`data/cache/replays/{slug}/{short8}.json`)
and project them into shared tables. **Neither decodes a film.** Both are RELEASE tasks, and
both need the **server stopped**: they take `OpenReadWrite` on the shared DB and run the shared
migrations themselves — including under `--dry-run`.

```bash
# 1. RE-COOK the artifacts first — schema 39 makes every earlier artifact stale, and this is
#    the pass that makes `bombStats` exist in them at all.
go run ./cmd/levelup backfill-replay [--dry-run] [--force] [--limit N] [--only-existing]

# 2. Equipment and pad usage -> match_usage_players + match_usage_films.
go run ./cmd/levelup backfill-usage-summary [--dry-run] [--force] [--match ID] [--limit N] [--title S]

# 3. Assault statistics -> match_bomb_stats (append-only) + dated facts in
#    match_objective_events. Dry run FIRST: it prints per-match counters and writes nothing.
go run ./cmd/levelup backfill-bomb-stats --dry-run
go run ./cmd/levelup backfill-bomb-stats [--force] [--match ID] [--limit N] [--title S]
```

Running (3) before (1) is a **silent no-op**: artifacts older than schema 39 carry no
`bombStats`, so nothing is written and every match lands in the `sans calque` counter. Both
passes are resumable — a match already present in the `_latest` view is skipped unless
`--force`. `backfill-usage-summary` additionally re-summarises when the projection revision or
the artifact schema has moved.

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

### Asset production chains (versioned outputs)

Eleven offline chains, all under `apps/go-api/cmd/`, produce files committed to the repo
(`data/titles/{slug}/reference/`, `static/`, or a generated Go file). None is wired into
`cmd/server` — game decoding and GPLv3 code (`internal/himap`, `internal/ooz`, Kraken/Oodle)
stay isolated to these binaries. Run from `apps/go-api` unless noted. `--title`/`-title`
defaults to `halo_infinite` throughout.

#### weapon-icons (build + table)

```bash
go run ./cmd/weapon-icons-build                      # game root auto-detected
go run ./cmd/weapon-icons-build -deploy "D:/SteamLibrary/.../Halo Infinite/deploy"
# flags: -out DIR  -max N (images per atlas)  -probe N (descriptor→resource re-sync depth)
go run ./cmd/weapon-icons-table                      # derives the Go table from index.json
```

- Output: `static/weapons-assets/halo_infinite/jeu/` — 168 PNG (weapon icons in outline and
  silhouette, plus the kill-feed atlas) + `index.json` (build) ;
  `internal/games/halo_infinite/weapon_icons_table.go`, generated — DO NOT EDIT (table).
- Prereq: game installed + cgo (Kraken) for `weapon-icons-build`. `weapon-icons-table` needs
  neither — it only reads the versioned `index.json`, so it runs anywhere, including CI.
- Replay when: a game content update grows the icon set (build) ; after every
  `weapon-icons-build` run, so the table stays in sync (table).
- Full chain, correspondence tables and refuted leads:
  `.ai/V7.5/icones/ETAT_DE_L_ART_ICONES.md`.

#### mapquant-build

```bash
CGO_ENABLED=1 go run ./cmd/mapquant-build [--levels DIR] [--title slug] [--out FILE]
```

- Output: `data/titles/{slug}/reference/map_quant_bounds.json` — per-map world-bounds used to
  turn the film's quantized coordinates into world coordinates.
- Prereq: game installed (unless `--levels` points elsewhere) + cgo. The display-name → module
  link is a hardcoded table in the tool: a map missing from it is absent from the catalog by
  design (refuses to publish a guessed coordinate).
- Replay when: a new map's module link is established, or the game changes its modules/BSP.

#### mapcallouts-build

```bash
CGO_ENABLED=1 go run ./cmd/mapcallouts-build                          # native pass only
CGO_ENABLED=1 go run ./cmd/mapcallouts-build --forge-only --forge-fetch  # Forge pass only
CGO_ENABLED=1 go run ./cmd/mapcallouts-build --lexique --forge-only      # + string lexicon
```

- Output: `data/titles/{slug}/reference/map_callouts.json` (native + Forge callout zones) ;
  `--lexique` also writes `callouts_lexique.csv` next to it. Reads the versioned
  `callouts_i18n.csv` (816 labels) as an input.
- Prereq: game installed for the native pass and for `--lexique` ; cgo always (to build) ;
  network only with `--forge-fetch` (anonymous UGC blob fetch, no token). A loss guard blocks
  writing a map that would lose vertices vs. the committed file (`--accepte-perte` overrides).
- Replay when: game update (native pass, or `--lexique`, which "only replays on a game
  update" per its own header) ; a new Forge map needs its callouts (`--forge-fetch`).

#### mapfond-build

```bash
CGO_ENABLED=1 go run ./cmd/mapfond-build [--maps "Cliffhanger,Catalyst"] [--title slug] \
  [--out-dir DIR] [--style jeu] [--natives=false] [--forge=false] [--rapport FILE]
```

- Output: `data/titles/{slug}/reference/map_backgrounds/{key}.png` + `{key}.json` sidecar per
  map (218 files today) — the top-down background image and its calibration.
- Prereq: game installed — **always**, no flag bypasses it, even a Forge-only run ; cgo/GPLv3
  chain (`internal/himap` → `internal/himodule` → `internal/ooz`, never linked into
  `cmd/server`) ; requires `map_objectives.json` already built (hard dependency, fails
  without it) ; uses `map_quant_bounds.json` / `map_callouts.json` / `map_positions_jouees.json`
  / `map_fond_reglages.json` when present, degrades with a warning otherwise.
- Replay when: not documented in the tool itself ; in practice, a new native or Forge map
  needs its background cooked.

#### mapobj-build

```bash
go run ./cmd/mapobj-build --player <Gamertag> --map-id <uuid> [--map-id <uuid>...]
go run ./cmd/mapobj-build --player <Gamertag> --all                    # whole match_registry
go run ./cmd/mapobj-build --from-file <path.mvar> --map-id <uuid>      # offline
go run ./cmd/mapobj-build --refresh-from <dir of .mvar>                # offline, whole catalog
```

- Output: `data/titles/{slug}/reference/map_objectives.json`, written atomically
  (temp file + rename). `map_objects.csv` and `forge_object_types.csv`
  (`data/titles/{slug}/reference/map_geometry/`) are **not** produced by this or any other
  tool — verified zero producer in `cmd/`; they were imported manually and have no replay
  command.
- Prereq: game install **not** required ; network required unless `--from-file`/
  `--refresh-from` (Xbox Live/Halo auth per ADR 0023 — never re-capture a token) ; `--all`
  additionally opens `shared_matches_v2.duckdb` read-only ; cgo needed to build (DuckDB driver).
- Replay when: a new map is played in matchmaking (one `--map-id`) ; `--all` to resync the
  whole registry ; `--refresh-from` after a local `.mvar` dump, fully offline.

#### mapopads-build

```bash
go run ./cmd/mapopads-build --from <dir of .mvar> [--title slug] [--dry-run]
go run ./cmd/mapopads-build --from <dir> --refresh-drifted   # re-validate against fresh .mvar
```

- Output: `data/titles/{slug}/reference/map_weapon_pads.json` (weapon/power-up spawn pads),
  written atomically via the same `mapcatalog.WriteAtomic` helper used by the sync runtime's
  own Forge catch-up path into this file (`.ai/PLAN_V2_REJEU_FILM_2026-09-05.md` item A.3 —
  tracked separately, not part of this chain).
- Prereq: no game install, no network, no cgo ; requires `map_objectives.json` (map_id →
  filename link) and a local dump of `.mvar` files (`--from`).
- Replay when: `--refresh-drifted` — a UGC map's `.mvar` no longer matches the committed
  catalog (measured drift; this is the normal re-validation path since the 2026-09-01 decision).

#### mapstruct-build

```bash
CGO_ENABLED=1 go run ./cmd/mapstruct-build [--levels DIR] [--maps "Cliffhanger,Streets"] \
  [--title slug] [--out-dir DIR]
```

- Output: `data/titles/{slug}/reference/map_structure/{module}.json` (2 files today — the
  default `--maps` list covers only the two maps with 100% measured coverage, not "all").
- Prereq: game installed (deploy variant `pc`, not `ds`) unless `--levels` ; cgo ; requires
  `map_quant_bounds.json` (module ↔ display-name link).
- Replay when: another map's mesh-instance decoding reaches full coverage. **Caveat**: the
  artifact's `structure` field is under a deferred-removal decision
  (`.ai/V7.5/REGISTRE_REPORTS.md`) — still read by two web files — check that entry before
  assuming this tool is safe to drop.

#### mappos-build

```bash
go run ./cmd/mappos-build --cle <mapId> [--carte NAME] [--title slug] [--pas M] \
  [--min-matchs N] [--min-occurrences N] <replay.json>...
```

- Output: `data/titles/{slug}/reference/map_positions_jouees.json` (merges into the existing
  catalog, one map key at a time).
- Prereq: no game install, no cgo — pure post-processing over already-decoded replay
  artifacts (`data/cache/replays/{title}/{matchId}.json`), passed as positional arguments.
- Replay when: more or newer matches should refine a map's played-positions mask.

#### mapnav-fetch

```bash
go run ./cmd/mapnav-fetch -toutes [-out-dir DIR] [-rate-ms N] [-refaire]
go run ./cmd/mapnav-fetch -map-id <uuid> [-map-id <uuid>...] [-dry-run]
```

- Output: `<out-dir, default .ai/re_dump/navmesh>/<mapID>.blob` — **not itself a versioned
  asset**: `.ai/re_dump/` is gitignored. It's the local working cache that `mapfond-build`'s
  Forge pass reads (`cuisson.go`) ; listed here because it feeds a versioned chain.
- Prereq: **not** the game install — an anonymous HTTP fetch from halowaypoint.com's public
  UGC pages (two requests, no auth) ; resumable (skips existing blobs) and rate-limited.
- Replay when: a new Forge map needs its navmesh before `mapfond-build` can cook its
  background ; `-refaire` forces a re-fetch.

#### vehicle-sprite

Multi-subcommand CLI (`inventaire`/`render`/`variantes`/`diag`/`assemble`/`compose2d`), not a
single fixed invocation. Verified fragment of the recipe behind the current set (covers 13 of
the 18 vehicles ; later passes added the rest — check `.ai/V7.5/film_re/*.md` for the current
state before re-running):

```bash
go build -o v4tool.exe ./cmd/vehicle-sprite
v4tool.exe render -variant=any -cote=256 -out=<dir> \
  -modules="pc:globals-rtx-new.module,globals-rtx-new.module,common-rtx-new.module,multiplayer-rtx-new.module" \
  -curate="0x00002705:warthog,0x000025aa:mongoose,0x0000d3db:scorpion,0xb65b3b4a:wasp"
```

- Output: `static/vehicles-assets/halo_infinite/replay/` — 20 files (18 PNG + `index.json` +
  `files_list.txt`), consumed by `useReplayVehicles.ts`. None of it goes through
  `PathResolver` — paths are plain `-out`/`-curate` flags.
- Prereq: game installed, cgo/GPLv3 (never linked into `cmd/server`) ; no network.
- Replay when: a new pilotable vehicle ships. Full recipe: `.ai/V7.5/film_re/V4_RAPPORT_SPRITES_2026-08-31.md`
  §9 and later notes in the same directory.

#### weapon-sounds (mode `livrer`, final step of a larger recipe)

```bash
go run ./cmd/weapon-sounds -mode livrer -donnees <chantier>/_donnees [-sons <chantier>] [-depot <repo>]
```

- Output: `static/sounds/halo_infinite/hinf_*.wav` (26 files) +
  `apps/web/src/features/match-replay/weaponSoundVariations.ts`.
- Prereq: the recipe's earlier, still-external steps (extraction, banks analysis, human vote)
  must already have produced `_donnees/*.json` and the per-weapon source/rendered `.wav`
  tree ; no cgo/game-install for this final step alone.
- Replay when: a weapon vote is finalized, or the full recipe is redone (game update, new
  weapon). Full recipe: `.ai/V7.5/RECETTE_SONS_ARMES.md`.

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

### Media paths migration (one-shot, standalone binary)

Converts legacy **absolute** media paths to portable relative `{owner_slug}/{rel}` paths in
`shared_social.duckdb` (`media_files.file_path` / `thumbnail_path`, propagated to the
`media_likes.media_path` PK). Idempotent — already-relative paths are skipped, a broken
thumbnail is nulled out so the next `BackfillThumbnailPaths` repoints it. Run with the
**server stopped** (opens `shared_social.duckdb` RW). Already executed in prod for the
existing titles; kept for future legacy imports that could reintroduce absolute paths.

```bash
go run ./cmd/migrate-media-paths --db data/titles/{slug}/warehouse/shared_social.duckdb [--dry-run]
# flags: --db PATH (required)  --captures-base DIR  --settings app_settings.json  --dry-run
# --captures-base defaults to app_settings.json media_captures_base_dir
```

### 2D replay — where an artifact gets built (`replay_build_location`)

Setting in `app_settings.json`, re-read on **every** sync cycle (a `PATCH /api/v1/settings`
takes effect without a restart). It arbitrates the *service* paths — the post-sync step and
the admin action — never the operator CLI below.

| Value | What the server does | When it applies |
|---|---|---|
| `local` | This process decodes the film itself, in a **bounded child process** (hard memory cap, low CPU priority). | Default in development. **Refused in production**: a film decode takes ~50 s and peaks at hundreds of times the film's own size; the web VPS never decodes. A `PATCH` asking for it in production is rejected with `400 invalid_replay_build_location`. |
| `worker` | This process **queues** the match and never decodes. A remote `cmd/replay-worker` pulls the job, downloads the chunks from pre-signed URLs, decodes, and pushes the artifact back. | Default in production. Requires `LEVELUP_BUILD_WORKER_TOKEN` on the web instance; **without it the placement degrades to `off`** (queueing when nobody empties the queue would resolve one Halo manifest per match, every cycle, for nothing). |
| `off` | Nothing is built. The replay page falls back to whatever artifacts already exist. | Explicit opt-out. It is the only *silent* placement — the two degradations above each log a `WARN`. |

Empty value = the instance default (`worker` in production, `local` in development). The
decision has a single home: `replaybuild.DecidePlacement`.

The post-sync step (1.58) takes the cycle's **inserted** matches first, then catches up on the
most recent matches of the retention window that have no artifact yet — a Theater film is
published *after* the match, so a single attempt at insertion time would never catch a
late-arriving film. Caps: the catch-up tier never adds more than 5 matches per cycle, a local
build never exceeds 5 matches, and either path stops between two matches once the cycle has
spent 5 minutes. The remaining backlog is published as `postsync_replay_backlog_restant` on
`/debug/vars`, together with `postsync_replay_cycles_total` (zero here while syncs run = the
step is off or unwired).

`replay_retention_months` bounds the same window: the step never builds — and the recurring
purge deletes — artifacts older than it. `0` = unlimited.

The operator CLI ignores this setting on purpose (see `cmd/levelup backfill-replay`): whoever
types it has already decided where they build, on their own machine, with their own cached
films.

### 2D replay — build tooling (facts, equivalence, profiling)

Operator tools of the artifact build chain ("cuisson" in `.ai/V7.5/PLAN_CUISSON_PERF.md`). They read
the local film cache; the two offline ones need no DB and decode one film per bounded child process
(hard memory cap, low CPU priority, solo lock).

```bash
cd apps/go-api
go run ./cmd/levelup replay-facts-export --out internal/analysis/replay/testdata/equivalence \
  [--title slug] <short8|match_id>...
```

Writes one `<short8>.facts.json` per match — match rows, both team scores, variant, candidate map
names — in the shape `replay-build --facts` already reads. Without those facts, zones, objective
actions, VIP/skull/bomb, pads and spawn points are short-circuited and an equivalence run would be
vacuous. Read-only (`OpenReadForQuery`); it fails outright rather than writing empty facts, so stop
a server that holds the shared DB in write.

```bash
go run ./cmd/replay-equiv                            # whole corpus (CORPUS.txt), compare only
go run ./cmd/replay-equiv -films 000d5950 -update    # (re-)freeze the references of one film
# flags: -corpus F  -films a,b (replaces the corpus)  -update  -mem-gib N (default 3, 0 = off)
#        -title slug
```

The equivalence harness of the build chain: it hashes the output of **every** scan, not just the
final artifact, so a divergence is located down to the scan. Parent and child share one binary —
the parent plans and decodes nothing, each film is born in a bounded child (solo lock with bounded
wait, sentinel) and dies with its RAM. References live in
`internal/analysis/replay/testdata/equivalence/<short8>.tsv`, each opening with its
`# digest-grammar: N` marker: a reference frozen under another grammar is an infrastructure failure
("re-freeze with `-update`"), never a decoding difference. `-update` rewrites those references
instead of comparing them — for a declared correction only. The `-walkers` mode (divergence of the
packet-splitting grammars over the whole film cache) was **removed** in 2026-09: it carried a copy
of three historical packet walkers whose originals no longer exist, so it only compared against
itself. Its measurement stays frozen in `.ai/V7.5/MESURES_CUISSON_PERF.md` §2 and is replayed in CI
by the mini-reel test of `internal/analysis/filmsource`.

```bash
LEVELUP_LOG_LEVEL=debug go run ./cmd/replay-build --map "<map name>" --facts <f>.facts.json \
  --cpuprofile tmp/<f>.cpu.prof --memprofile tmp/<f>.heap.prof <short8> [filmDir]
```

Measurement of a single build (protocol §6 of the plan): `LEVELUP_LOG_LEVEL=debug` brings out the
per-scan durations (the binary installs an slog handler), `--cpuprofile` and `--memprofile` write
pprof files (`go tool pprof`), the heap one after the build. All three are inert by default, and
the options must precede `<matchId>` — the flag package stops at the first positional argument.

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

#### Map reverse-engineering corpus (`gamefiles` build tag)

The 59 `*_gamefiles_test.go` files in `internal/himap/` decode the **installed game's**
modules and sweep the 26 catalogue maps. They are long by nature — measured 2026-09-05,
`TestBalayageCoquille` alone takes **203 s** for 26 maps (1 246 s before the module reader
switched to memory mapping the same day). They sit behind
`//go:build gamefiles` so that a plain `go test ./internal/himap/` stays usable (2.8 s).

```bash
make go-api-test-gamefiles                       # whole corpus (~6 min, needs the game)
cd apps/go-api && go test -tags=gamefiles -count=1 -timeout 3600s ./internal/himap/ -v

# One map only (much faster):
BALAYAGE_CARTES=aquarius_map go test -tags=gamefiles -timeout 300s \
  ./internal/himap/ -run TestBalayageCoquille -v

# Game installed somewhere else:
LEVELUP_HALO_DEPLOY=/path/to/Halo Infinite/deploy go test -tags=gamefiles ./internal/himap/
```

Without the game installed every test takes its `t.Skip` and the corpus is empty in a
second — which is exactly what happens in CI. CI therefore only **compiles** it
(`go vet -tags=gamefiles ./internal/himap/`, job `go-test`); it never runs it. The tag
itself is enforced by `internal/himap/corpus_tag_test.go`, which runs in the default build.

**Known red test**: `TestBancCliffhanger` fails (accord 64.4 % against a 64.7 % re-based
reference). It is *pre-existing*, not a regression — verified 2026-09-05 by replaying it on
the previous commit, which yields bit-identical numbers. Nobody could see it before: the
corpus never ran to completion, and CI does not execute it. Tracked in
`.ai/V7.5/REGISTRE_REPORTS.md`.

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

### Local merge gate (`gate-push`)

```bash
make gate-push               # golangci-lint ratchet + web typecheck/lint + test baseline (~25 min)
```

On some Windows dev machines, git-bash's linker fails to resolve DuckDB's
`libduckdb_static` when building CGO test binaries (`undefined reference
__emutls_v._ZSt11__once_call`), which breaks the test-baseline step of
`make gate-push` even though the code itself is fine — native PowerShell links
correctly. Validated workaround (documented in
`.ai/HANDOFF_POST_LOT2_V73.md`): run `scripts/gate-push.ps1` instead. It
reproduces the same 4 links (Go lint, Go integration tests, web typecheck, web
lint) but produces the `go test -json` output from native PowerShell, then
hands it to `scripts/check_test_baseline.sh tests --from-jsonl <file>` (consumer
mode — parses the JSONL, does not re-run the suite). CI remains the authority;
this is a local-only fallback for that specific environment quirk.

```powershell
powershell -File scripts/gate-push.ps1
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
