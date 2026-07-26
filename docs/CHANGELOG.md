# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

> French version: [FR/CHANGELOG.md](FR/CHANGELOG.md)

## [7.3.0] - 2026-07-26

> Point release built on the "v7.3 Notion backlog" (branch `feat/v7.3-notion-batch`): an overtime flag measured against a declarative regulation table, a first kill / first death chart replacing two histograms that disagreed with each other, size-based log rotation, and a sweep of fixes on Prestige objectives, the demo, the Halo 5 Achievements page, the language switch and the DuckDB write path. It also folds in the quick-win batch **deployed ahead of the release on 2026-07-26** (`1b18ae609`: `public/` assets in the prod image, objective view in the snapshot, match-view layout, Objectives participation axis), which never carried a version number of its own. **Post-deploy actions are required — see *Ops*.**

### Added

- **Overtime flag (`is_overtime` / `overtime_seconds`)** — no API field carries it (checked on 56 real payloads), so the rule is measured: `elapsed` = median `time_played` over the players present from start to finish (fallback `MAX`), overtime when `elapsed > T_reg(variant) + 40 s`. Calibrated over 1,793 matches (proof of concept on `bc918a5a`: 990 s played against 720 s of regulation time), it yields **0 false positives on 724 control Slayer matches** and flags 761 matches. Regulation times are declarative — `config/titles/{slug}/mappings/regulation.toml`, 9 Arena variants at 720 s on Halo Infinite, **empty on Halo 5** — so an undeclared variant is never flagged (safe degradation, zero slug comparison). `internal/analysis/overtime.go` holds the pure helper and the documented `OvertimeMarginSeconds = 40`, with a guard-rail against redefining the margin elsewhere; `platform/duckdb/elapsed_seconds.go` is the best-effort estimator (structured WARN before any degradation). Computed at read time: **no migration, no backfill, retroactive over the whole history**. Front: `lib/narrative/overtime.ts` as the single source, `narrative.overtime.*` FR/EN, header badge plus an Explorer pill through a `NarrativePill` extracted for the purpose (coexists with the dominance marker).
- **`FirstBloodLanes` chart** — one lane per player: cloud of first kills above, first deaths below, median markers, and a window bar stretched between the two medians (linear interpolation, nulls excluded), with rich labels on the left (player, `med. X → Y`, signed advance) and lanes sorted by median first kill. Outcome tokens only, zero hex; a custom series for the window bar (stretched between two X values, height in pixels) and scatters for the clouds and the medians. Pure builder tested without a locale, shared `first_blood.toml` manifest (FR/EN), showcased on `/lab/charts`.
- **Size-based log rotation** — there was nothing to extend: the `app.log.1-3` files in production are leftovers of the Python `RotatingFileHandler` deleted at the cutover, and the package README assumed as much. Home-grown `rotatingWriter` (~200 lines, zero dependency — `lumberjack` is not multi-process safe) plugged onto the category writer of `MultiModuleHandler`: **100 MB × 3 archives per category** by default, tunable without a redeployment through `LEVELUP_LOGS_MAX_SIZE_MB` / `LEVELUP_LOGS_MAX_BACKUPS`. Multi-process hardening: close before rename (Windows), `SameFile` detection for the loser of a race (re-open instead of a double shift), one-minute cooldown after a failure, and a guard against rotating on every line. The `.log.N` archives are invisible to the admin viewer and to `logtail` by construction (existing `.log` filter).
- **429 responses are observable at last** — `rate_limit_429_total` expvar counter plus a WARN sampled every 30 s through `WithLimitHandler`; `httprate` logged nothing at all, which is the class of the 2026-05-06 image incident.
- **`/docs` (Stoplight) on the root router** — gated on `IsProduction` **or** `DemoMode` (the public demo does not set `LEVELUP_ENV`), alongside `securitySchemes.sessionCookie` declared in the contract with its name read from its source (`session_platform.CookieName`). No `security` requirement is posted: the `bare_routes` ratchet remains the authority on which route is guarded.
- **Objectives participation axis** *(deployed ahead of the release)* — an opportunity index computed on `match_objective_stats_latest`, replacing a PSA sum diluted by the total number of matches against thresholds (350×N) that had never been calibrated. 7 families (`ctf`, `zones_koth`, `zones_strongholds`, `oddball`, `stockpile`, `extraction`, `vip`), weights aligned 1:1 on `awards.toml`, `r = min(1.25, 0.65 × actions/P80 + 0.35 × hold/P80)`, aggregation weighted by `n_f`. Injected into Session Compare/Detail, the squad synergy radar, the Match View radar and the Ascension profile; the axis is dropped when no objective match is in scope or the capability is absent (Halo 5), and untracked teammates are covered (shared source). The `ObjectiveScores` (PSA) reader, now dead, is removed.
- **Two title-scoped asset packs** — Spartan abilities (11 × 2 variants plus an FR/EN index, `static/abilities-assets/halo_infinite/`) and grenades (4 types × 2 variants plus an index, `static/grenades-assets/halo_infinite/`), same convention as `weapons-assets`. No page consumes them yet.

### Changed

- **First kill / first death — one core, no server-side bucketing** — the delay was computed **twice** (squad in 15-second bins, timeseries in 10-second bins) over the same `highlight_events` → T0 correction → min per (player, match, type) chain, and the two charts told different stories. Single core (`firstEventsAcc` + `ComputeFirstEventsByActor`), **per-match values** exposed in the page responses (`first_blood` and `compare_first_blood`), and no bucketing left on the server. Squad: the chart leaves Contributions for the **Dynamics** tab (after Intensity). Timeseries: replaces the 10-second histogram in the same slot (**Progression** tab). Sessions: new block under Intensity, present in the comparison drawer too. Shared adapter `_shared/firstBlood.ts`; `maxSec` = p99 rounded up to the minute with a 300-second floor, so one endless match no longer crushes the scale. Old code **deleted** (2 Go builders, 4 types, 2 web components, orphaned i18n); the T0-shift and skip-countdown tests are rewritten on the new builder.
- **Static assets are no longer starved by the rate limiter** — the extension list is centralised in `middleware.IsStaticAssetPath` (single source, archlint guard-rail against a second list, detector self-tested); static paths are exempt from the rate limit (never `/api/*`, except `/api/v1/assets`); `Cache-Control: public, max-age=3600` on the non-hashed files of the dist, `index.html` excluded — a new build is always served.
- **Runtime settings written atomically** — new `atomicfile` package (temp + rename, in-place fallback bounded to `EBUSY` and to an impossible temp file — a single `Write` plus `fsync`, limitation documented) applied to the 3 runtime writes of `app_settings.json` (`Save`, `SaveTitleOverlay`, `last_notified_version`). Only the last one was actually broken inside the container; the admin toggles gain the atomicity they never had. Archlint guard-rail against a bare write.
- **Fine-grained writer telemetry no longer overwritten** — the `sync_v2_postsync/{weapons,events,psa_aliases}` breakdown existed, but `acquireSharedRW` (`sync_v2_wiring`) overwrote the fine label right before `AcquireWriter`, so **100 % of the production WARNs landed under the coarse label**. `ctxkeys.WithDBWriterLabelIfAbsent` applied at the single faulty site, with tests on the property. Chunk size, watchdog and threshold are deliberately untouched — they will be tightened after reading the telemetry in production.
- **Release notification — quieter startup** — `app_release: emit` filters `auth_only` accounts declaratively **before** resolution (5 WARN per restart → 1 grouped `DebugContext`) and deduplicates per player; each player used to be resolved once per title, a latent bug.
- **`ApiError.details`** — `oneOf[object, array]` restored through the `SchemaTransformer` with an explicit `additionalProperties`: a bare `{type: object}` generates `Record<string, never>` on the `openapi-typescript` side.
- **Match view layout** *(deployed ahead of the release)* — the Media block moves last, alone and full width; the frag grid is carried by `MatchFragCard` with a predicate mirroring `FragSunburst` (no more orphaned "Tools of destruction" at one third width); the chart row and the medals/citations row become fluid auto-fit grids with no orphan; media thumbnails switch to `auto-fill minmax(200px)` instead of 5 viewport columns (~65 px each).
- **`echarts` 5.6.0 → 6.1.0 deferred again** — the v7.2.1 entry announced the bump for v7.3; the decision taken on 2026-07-26 keeps it deferred and `.ai/BACKLOG.md` records it as such. Stated here so the previous entry does not stand as an unmet promise.

### Fixed

- **Prestige — cumulative objectives could never be completed** — the gauge (`computeCurrentValue`) and the persistence (`evaluateOne`) each measured progress their own way, so a cumulative objective displayed an average and never reached its target. Single measurement point `evaluateChallengeNow` for both, with a non-divergence test that makes the drift inexpressible. `CumulativeSince` is an uncapped SQL sum bounded at `created_at`: the `WindowMatches = 20` cap would have truncated a "220 in total" objective. `MedalEvent` removed — no writer, dead code.
- **Demo — improvement campaigns invisible** — the `improvement_campaign` insert used the player slug while the read filters on the XUID, so the demo profile showed no campaign at all; `plan.XUID` is now written. The `'closed'` status, outside the enum, becomes `StatusCompleted`, and `CampaignStatus.Valid()` guards the seed write: the column is a free `VARCHAR`, so an invalid value produced silent invisibility rather than an SQL error.
- **Demo — objective statistics absent, and XUIDs published in the clear** — `match_objective_stats` is now extracted into the demo **without** the `appendOnly` marker (the table is born in append-only form; flipping the flag produces a proven Binder Error). The `xuid` column is added to the universal anonymisation of the demo corpus — without it the extraction published real XUIDs on the public demo.
- **Scoreboard broken by a missing objective view** — the `LEFT JOIN match_objective_stats_latest` leaves Q12 for an isolated query (Q12b) loaded best-effort with a `slog.WarnContext` before degradation: a missing view now yields a served scoreboard and an empty Objectives section instead of a total failure. No duplication (the 41-column SQL and the Go scan were moved, not copied). Local diagnosis: the A/B desync hypothesis is **disproved** (B-swap operates on a single file); the real cause is a v65 snapshot without the objective parquet, its handwritten DDL never migrated. The live fallback is correct and rearms at the next snapshot cut.
- **Halo 5 — Achievements page returned 500** — `milestone_catalog` for `halo_5` lacked `condition_fr` / `condition_en`, and `CREATE TABLE IF NOT EXISTS` never adds a column to an existing table. Columns added to the `CREATE` plus an idempotent `ALTER` step for existing databases (a recorded step is never replayed), seed aligned with Halo Infinite, and a misleading comment corrected. Front: the 3 states of the Achievements grid keep their section title (FR/EN).
- **Commendation labels did not follow the language switch** — `queryKeys.citations` and `commendationTotals` now carry the locale (same pattern as medals), so the labels baked by the backend follow a language switch made during the session.
- **`headshot_kills` never translated** — the contract's snake_case form fell through to `humanizeMetricKey`; the canonical key is added FR/EN in `metricLabel` and the `FieldHeadshotKills` alias is realigned.
- **Field mappings fetched twice at boot** — `useFieldMappings` is gated on `isBootstrapped`, closing the hydration window in which slug and locale still held their default values.
- **DuckDB reference leak at shutdown** — `MonitoringStore` is closed by `ServiceRegistry.Close`: end of the `shutdown_db_leak rw:monitoring.duckdb refCount 1` reported at every stop, and the WAL is finally checkpointed. `SetCronRunSink(nil)` as a belt, `DumpCachedLeaks` test added.
- **A panic could hold the RW writer until the next restart** — the 4 non-`defer` releases of the shared write path (event, weapon and `psa_aliases` bursts, plus `combined_persister`) move under a closure `defer`. A film-parsing panic caught by the global recover used to leave the writer held. The nominal release moment is strictly unchanged.
- **Production images stripped of the `public/` assets** *(deployed ahead of the release)* — `.dockerignore` re-includes `apps/web/public/`; every PNG/WebP was being stripped from the production image (the Waypoint logo of the match tables, `logo.png`, `og-default.png`, the Halo 5 emblems). An asset with a known extension now returns a plain 404 instead of the silent SPA `index.html` 200, a `verify-public-in-dist.mjs` guard-rail runs at the `web-builder` stage of the Dockerfile, and the `useMemo` dependencies on `currentTitleSlug` are fixed in `ExplorerMatchesTable` and `SquadSynergyHistoryTable`.
- **"Incomplete match" banner on every snapshot match** *(deployed ahead of the release)* — a v7.2 regression: Q12 joined `match_objective_stats_latest`, absent from the shared snapshot → Catalog Error → `scoreboard_empty` → the banner shown over perfectly healthy data. The table is now exported in the cut and the `_latest` view recreated at read time, its DDL centralised (`internal/migration/objective_stats_view.go`, 5 sites migrated) with an archlint guard-rail against copies and a snapshot schema contract test (16 relations); an older snapshot self-heals through `ErrSnapshotIncomplete` and a clean live fallback.

### Ops

**Post-deploy actions (required)**

- **Objective backfill to replay** — the server-side backfill was cut short before this branch was merged, and the P80 references of the Objectives participation axis are measured for `ctf`, `strongholds` and `koth` but **provisional** for `oddball`, `stockpile`, `extraction` and `vip`. Sequence: reset of the `MBitObjectiveStats` bit (exact SQL in the doc comment of `cmd/backfill_objective_stats/main.go`), backfill, then re-measure the provisional P80 values.
- **Demo regeneration required** — the objective statistics and the XUID anonymisation only reach the public demo through a fresh cut, and the improvement campaigns need the same regeneration to appear.
- **Existing production log stock to purge** — rotation applies only from its own writes onwards. The historical `app.log.1-3` files (leftovers of the Python handler) are truncated **in place**, never removed: an open file descriptor survives an `rm`.
- **`milestone_catalog` ALTER runs at boot** — one idempotent DDL step on the existing `halo_5` databases at the first start following the deployment. No data is lost.

**Other**

- **No migration and no backfill for the overtime flag** — it is computed at read time and therefore applies retroactively to the whole history as soon as the version is deployed.
- **Log rotation is tunable without a redeployment** — `LEVELUP_LOGS_MAX_SIZE_MB` (default 100) and `LEVELUP_LOGS_MAX_BACKUPS` (default 3), per category.
- **`/docs` is exposed neither in production nor on the public demo** — the gate covers both, the demo included, which does not set `LEVELUP_ENV`.

## [7.2.5] - 2026-07-26

> Patch release closing the operational tail of v7.2.1: the demo deployment pipeline, the Waypoint logo that rendered on Chrome but not on Firefox, and two inverted documentation comments. No schema change, no migration, no post-deploy action.

### Fixed

- **Waypoint logo invisible in the match tables** — three distinct causes, only one of which was browser-specific. (1) The link was a flex container with no size of its own inside an auto-width table cell; Chrome and Firefox do not resolve intrinsic size identically in that configuration and Firefox collapsed the image to zero. The link now carries a fixed size, so nothing is negotiated. (2) The HTML `width`/`height` attributes were missing, leaving the engine without a size before CSS resolution — the point that mattered to Firefox. (3) The asset is a 360×160 logotype forced into a 16×16 square with no `object-fit`: the default `fill` crushed it into sub-pixel strokes on **every** browser, including the ones where it appeared, so this part was never reported. `object-contain` restored, box set to 20×16, and width reserved per icon column (`w-9`) rather than widening the shared cell padding, which would have detached the whole table from its borders. Applied identically to `ExplorerMatchesTable.tsx` and its twin `SquadSynergyHistoryTable.tsx` — the two feed the Session, Timeseries and Career pages as well.
- **Demo milestones stuck at 0 (deploy pipeline)** — `insertDemoMilestonesEarned` attaches the demo metadata read-only to derive milestones from the real catalog, but the regen job only stopped the prod service; the demo container kept the file locked and the step failed best-effort (`Conflicting lock is held in PID 0`). The job now stops `levelup-demo` as well, with an `EXIT/INT/TERM/HUP` trap that brings both services back whatever happens — a regen that fails mid-way can no longer leave the demo down. Milestones go from 0 to 6 on both titles. Demo DuckDB files are now published atomically (inode swap) with WAL handling, so a reader never observes a half-written database.
- **Inverted documentation** — `discovery_client.go` claimed the UGC discovery API needs no Spartan authentication; it returns 401 today (measured anonymously on 2026-07-25). The comment described a bygone state of the API and misled anyone writing a new call. Alongside it, the anonymous route that does work is now documented: the map-variant blob path, which is what allowed 120 of 123 maps to be retrieved without authentication. The Waypoint capability comment claiming Halo 5 does not declare `waypoint_match_url` is corrected too — it has since 2026-07-24, which is what produced a false diagnosis — and the now-stale mention of Halo 5 is removed from the column hint in both locales.

### Added

- **Capability parity guard-rail** — `capabilities_parity_test.go` and `capabilities_parity_scan_test.go`: the capability list existed in three places (the Go descriptor, a validation set, and a front-side list) with nothing tying them together, which is exactly what let the comments drift. Bite proven by mutation: removing `CapWaypointMatchURL` from `knownCapabilities` made the whole Halo 5 title disappear from the registry. The real gap was `knownCapabilities` — the Go↔TS mirror already existed since v7.2.
- **Demo seed integration tests** — atomic publication, WAL recovery and inode swap covered (`seed_demo_integration_test.go`, `seed_demo_wal_test.go`).

## [7.2.1] - 2026-07-25

> Patch release consolidating the "v7.2.1 Notion backlog" (items V721-01 to V721-15, branch `feat/v7.2.1-notion-batch`): three more objective modes collected end to end, ten new objective citations, the Explorer season breakdown completed, the squad-challenge pool repaired, 35 routes added to the published contract, removal of the `arc_titles` structure, and the application version taken out of the build. Per-domain summary. **Post-deploy actions are required — see *Ops*.**

### Added (Go API)

- **Objective stats — Stockpile, Extraction and VIP (V721-02)** — mapping frozen on 10 real `GetMatchStats` payloads: `StockpileStats` (6 fields, 2 ISO 8601 durations), `ExtractionStats` (5 fields, no duration) and `VipStats` (7 fields, 2 durations) — the last one was **real but undeclared in `StatsBundle`** (`internal/openspartan/halo_api_payload.go`) and has been added. Separate migration step `shared_objective_stats_add_stockpile_extraction` (`TargetShared`): 18 nullable `ALTER TABLE match_objective_stats ADD COLUMN IF NOT EXISTS` **plus `CREATE OR REPLACE VIEW match_objective_stats_latest` in the same step** — DuckDB freezes a view's column list at creation time, so without it the 18 columns would be invisible to every reader. The existing `CREATE` is untouched (migrations are name-keyed: already-migrated DBs would never replay it). `ObjectiveStatsInsert` gains 18 pointer fields, `buildObjectiveRow` gains the 3 blocks, and `persistObjectiveStats` goes to 43 columns / 43 placeholders / 43 arguments — still a **pure INSERT**, no `ON CONFLICT`, no `INSERT OR REPLACE`; the 4 durations go through `objectiveDurationSeconds` (fractions preserved). Read chain extended in parallel (`ObjectiveRaw`, Q12, scoreboard scan, `buildScoreboardObjective`, `MatchScoreboardObjective`). Anonymized fixtures under `internal/sync/objective/testdata/objective_stats/`. `ALTER ADD COLUMN` is pure DDL, outside the ART #23046 perimeter; no ART guard-rail touched, no new capability (`CapMatchObjectiveStats` / `CapObjectiveStats` are already table-generic; Halo 5 stays `not_exposed`, the table simply stays empty for that title).
- **Elimination and Infection deliberately NOT specified (V721-02)** — no payload in the corpus carries those blocks: the two Arena:Attrition matches (`GameVariantCategory` 7) expose `CoreStats` only at both player and team level, and the only Infection candidate is a Firefight UGC (`GameVariantCategory` 41). No schema was inferred by analogy. Resumption condition: a matchmade match of either mode played and synced, then the same pattern replicated (1 ALTER migration, 1 block in `buildObjectiveRow`, 1 UI column set).
- **Ten objective citations (V721-03)** — `flag_captures`, `flag_secures`, `flag_steals`, `returner_takedown`, `unstoppable_carrier`, `aggressive_return`, `zone_defense`, `untouchable_carrier`, `skull_carrier_takedown` and `skull_grabs` seeded in `defaultCitationMappings()` (`seed_citation_data.go`) with 8 tier constants calibrated on `match_objective_stats_latest` (reference player: 304 objective matches) rather than on the generic ladder, which would have made four of them literally unreachable. Full EN parity (`citationDisplayEN` **and** `citationDescriptionEN`, 98/98, zero orphan) and four guard-rails: norm uniqueness, on-disk existence of every `image_path`, EN coverage of the display name, and a fixture pinning the 10 source columns and their tiers. The registry now holds **98** citations.
- **Explorer — complete season breakdown (V721-06)** — `deterministicSeasonPath` (catalog id `^season(\d+)$` → `Seasons/Season{N}.json`, `extra.matchmade_path` override for the intra-season operation) and `seasonCMSPaths` (deduplicated union, deterministic path first) in `explorer_target_seasons.go`. `computeSeasonBreakdown` no longer tries only the seasons present in `rec.SeasonIDs` — the cause of the "5 out of 14" diagnosed earlier. `domain.SeasonMatchCount.Unresolved` (optional, `omitempty`) plus a section status (`ok` / `local_partial` on partial resolution / `failed` when nothing resolves) separate "season not played" from "season not retrieved". The concurrency cap (`seasonBreakdownConcurrency = 6`) is unchanged and the CSR fan-out — the expensive one, one call per engaged ranked playlist — is now requested only for seasons with `matches > 0`, so the CSR budget is identical to before.
- **Guard-rails** — `no_cross_title_aggregation_test.go` (AST scan, 4 detectors, dated allowlist, bite proven against the exact code removed below); `bare_db_recovery_routing_test.go` (forbids flat access on a bare `db *duckdb.DB` field inside `platform/duckdb/prestige/` — the hole both recovery incidents went through, invisible to `player_db_recovery_routing_test.go` which only sees `pdb.Player.Query(` forms); `default_status_ratchet_test.go`; `TestSharedObjectiveStatsStockpileExtractionColumns` (18 columns, nullability, type family, round-trip **through the `_latest` view**).

### Added (React / TypeScript)

- **Three objective column sets** — `ObjectiveMode` gains `stockpile` | `extraction` | `vip`; `detectObjectiveMode` and `objectiveColsFor` extended in `MatchScoreboard.logic.ts`: 4 Stockpile columns, 4 Extraction, 5 VIP out of the 18 exposed fields (the remaining 5 stay in the DTO without a dedicated column, same ratio as CTF, which exposes 11 and displays 4). `longest_time_as_vip_seconds` aggregates with `max`, never a sum (same rule as Oddball). 13 label/tooltip keys added in **both** locales in `match-view/i18n.ts`.
- **`lib/charts/avgLife.ts`** — the single front-side average-lifespan helper (real value, proxy fallback), and `lib/narrative/dominance.ts` as the single source of the dominance tokens.
- **Review tooling** — `ReviewBadge` + `lib/review`, inert outside a review pass, wired through a `reviewKey` prop on the charts under review.

### Changed

- **Published contract completed (V721-04)** — the four gates of the render harness (`BuildDemoRouter`) were loosened so Prestige, catalog, auto-sync diagnostics and assets metadata mount: **+38 operations, −3, i.e. 35 routes newly published** in `api/openapi.yaml`. `EmptyCatalogRepo` (demo fallback, same stance as `EmptyAssetMetadataHandler`) is injected into the **same** `CatalogHandler`, so the 3 catalog routes are declared once and there is no twin handler to keep in sync; it is never used outside demo mode. A **single harness** was kept on purpose — a second one would reopen the drift channel closed in v7.2.
- **Nominal response statuses (V721-04)** — `humacore.DefaultStatus` posted on **23 routes** (11 × 201 Created, 12 × 202 Accepted). The status is declared once at mount time and is the single source for both the runtime and the document; `default_status_ratchet_test.go` locks it. `api/openapi_manual_fragment.yaml` shrinks accordingly.
- **Application version leaves the build (V721-15)** — `ARG VERSION` and `-X main.version` dropped from the `Dockerfile`, `build.args` removed from both `docker-compose.yml` services. The version now travels **only** through the runtime environment (`LEVELUP_APP_VERSION`), which `scripts/deploy.sh` persists into `.env` so compose reads it for every service and every command (`up`, `--force-recreate`, reboot, manual ssh session) — that is what covers the case which motivated baking it in originally (env lost when the demo-regen recreates the container). Two consequences: tagging a release no longer invalidates the CGO build layer, and the prod and demo images can no longer diverge — the cause of the two parallel CGO compilations that exhausted the VPS during the v7.2.0 deployment. Divergence is now impossible by construction instead of avoided by keeping two values aligned by hand. The demo service reports the real version instead of `dev`. `main.version` remains a fallback for a local `go build` with ldflags.
- **Dependabot (V721-01)** — `js-yaml` override raised to `^4.3.0`, `brace-expansion` overrides added (`^5.0.7`, and `^2.1.2` scoped under `@redocly/openapi-core` → `minimatch`): the three `high` alerts are closed. All three are DoS issues in build tooling never loaded by the browser, and all three fixes are patch bumps. **`echarts` 5.6.0 → 6.1.0 deliberately deferred to v7.3** (dated backlog entry, criterion: chart suite green + visual pass) — a major bump on the engine behind every chart in the app is not a patch-release change.
- **Prestige reads routed through recovery** — `prestige_metadata_repo.go` and the five flat single-row reads of `prestige_social_repo.go` (`GetUserPrestige`, `GetUserPrestigeCrossTitle`, `SquadRepo.Get`, `SquadChallengeRepo.Get`, `CountActiveParticipants`) moved to the `*Recovered` variants; file contention is translated into `dblease.ErrDBLocked` by `translateSocialLockErr` (`prestige_social_recovery.go`, second copy of the pattern — centralisation plus guard-rail at the third).
- **Squad challenge pool** — `squadEligibleTemplates` discards `eval_type=threshold` templates, whose displayed rule does not match the evaluation actually performed (cumulative for everyone in V1). An empty eligible set degrades to `200 []` with a structured log separating an empty catalog from a fully filtered one; no slug comparison introduced.
- **Average lifespan** — `avg_life_seconds` (native API value) exposed and preferred over the `time_played/(deaths+1)` proxy. `buildCorrelationPoints` was a **third** consumer computing its own inline proxy — the histogram and the scatter told two stories about the same metric — and is rebased on the shared `matchAvgLifeSeconds` helper with the same Debug telemetry (fallbacks / used / total). It is extracted into `timeseries_service_correlation.go` (adding the telemetry pushed `timeseries_service_tabs.go` to 517 lines, above the 500-line threshold); the `LifeBuckets` doc comment in `domain/timeseries.go` is corrected accordingly.
- **Win rate and MMR on separate axes** — Y2 becomes the win rate bounded to [0,100] with a `%` formatter, Y3 carries the MMR (`offset: 48`, added only when the title provides it, so no ghost axis on Halo 5); the grid right margin widens only when the MMR axis is present. The two shared an auto-scaled Y2 on which the MMR crushed the win-rate curve onto the bottom line. The squad chart was already correct and is untouched.
- **Synergy radar** — the raw value is shown next to the normalized 0-100 score in the tooltip (`SquadSynergyRadarAxis.raw` was already carried by the DTO), with precision adapted per axis magnitude rather than a single rounding that would flatten the ratio axes.
- **Dominance on the outcome tape** — `dominance_flag` added to the two DTOs concerned (`omitempty`, 0 disappears from the JSON) and rendered as a marker on `OutcomeSequenceTape`.
- **Documentation** — ADR 0014 §6.1 amended: the soft-negative coach signal shipped while the ADR still stated "positive alerts only" (inverted doc), with its guard-rails now written down (−0.10 threshold over ≥ 14 days, neutral category, never `outcome-loss`). `docs/COMMENDATIONS_REFERENCE.md` and its FR pointer updated (88 → 98 citations, "PvP — Game mode" table 11 → 21 rows, 4 of them corrected from `award` to `objective_stat`). `internal/openspartan/doc.go` now points at `docs/ARCHITECTURE_V6.md`, where the OpenSpartan import feature is documented for the first time outside the code — the actual cause of the recurring "why is there an openspartan package?" question (743 occurrences, zero dead code, `RequireAuth`-protected, 73 % covered). `CoachFocusCard.tsx` string object migrated to the `profile.coach.*` i18n manifest (ADR 0003), the last coach component with a hardcoded `STR` object.
- **Plans closed and archived** — `PLAN_COACH_V3_GENERATION.md` (delivered at ~95 % on 2026-06-09, still labelled "Proposed", with stale line references) and `PLAN_CROSS_TITLE_ARCS_2026-07.md` moved to `.ai/archive/` with a corrected execution record; `.ai/BACKLOG.md` triaged (4 delivered entries removed, the `seedWeapons` entry corrected — `weapon_labels.name_fr` is a **live third link** in the resolver `COALESCE`, not vestigial plumbing — and a dated `echarts` 5→6 entry added).

### Fixed

- **Security — three catalog routes mounted without an auth guard** — `/titles/{slug}/catalog/{playlists,pairs,maps}` sat outside any guard while accepting `?xuid=` and `only_played`, leaking one player's activity without authentication. `RequireAuth` applied on **both** branches (nominal and demo fallback); no front caller was affected, verified. The ratchet had never seen them because the demo harness did not mount them — mounting them for V721-04 is what exposed the hole.
- **`GET /prestige/me` without `title_slug` returned 500** — `GetUserPrestigeCrossTitle` scanned `MAX(updated_at)` (an aggregate, therefore NULL over an empty set) straight into `prestige.UserPrestige.UpdatedAt` (`time.Time`); no sentinel was recognised by `serviceError`, so the default branch answered 500. Now scanned through `sql.NullTime` then converted: a player with no PP gets an empty prestige, same contract as `GetUserPrestige`. Audit of the file's other aggregates (`COALESCE(SUM(...), 0)`, `COUNT(*)`) found no second site at risk.
- **Squad challenge pool returned a permanent 500** — `prestige_metadata_repo.go` used a flat `r.db.Query`, so `dblease.ErrDBLocked` was never raised and the handler's 503 mapping never fired; a single concurrent invalidation left `POST /squads/{id}/challenges/pool/refresh` failing until the next restart.
- **Joining or abandoning a squad challenge answered 400 instead of 503** — the error chain was re-wrapped into `ErrInvalidInput`; the cause is now preserved through a second `%w`.
- **`stats_canonical.go` comment** — claimed `OffensiveConversion` / `DefensiveResistance` "stay nil" while they are computed by `ComputeCombatYield`; only `MedalExploitScore` is actually left nil in that converter.
- **Misplaced doc block in `seed.go`** — it documented `defaultCitationMappings` while sitting above `SeedMedalDefinitions`, whose `//nolint` it covered by accident; moved above the function it describes, in `seed_citation_data.go`.

### Ops

**Post-deploy actions (required)**

- **Objective backfill, preceded by a targeted reset** — `markObjectiveDone` sets the `MBitObjectiveStats` bit for every fetched match, including those that produced no row — which was the case of every Stockpile / Extraction / VIP match under v7.2 — so `listCandidateMatchIDs` now excludes them. The exact reset SQL is in the doc comment of `cmd/backfill_objective_stats/main.go`. Matches synced after the deployment need no reset: the native sync never sets that bit.
- **Citation re-seed and full recompute** — `levelup seed citation-mappings` (expected: 10 inserted, 88 updated) then `levelup backfill --all --citations-recompute-all`. The incremental `--citations` path is **not** enough: already-processed matches carry a sentinel row and would not be replayed, leaving the ten new citations at zero across the whole history. Both write to DuckDB — stop the server first (one writer process per DB, ADR 0013/0016).
- **`drop_arc_titles` runs at boot on every player DB** — no data is lost (the table mirrored `arc.title_slug` 1:1), but the DDL executes on the first start after the deployment.

**Other**

- **Two of the ten new citations ship with provisional in-house artwork** — `flag_captures` and `flag_steals` point at placeholder SVGs. Replacing them means dropping the final PNGs into `static/commendations/halo_infinite/`, switching `.svg` → `.png` in the two `ImagePath` values and re-seeding; `TestCitationImagePaths_ExistOnDisk` fails if the extension is switched before the files land.
- **The application version now comes from the runtime environment only** — see *Changed*; `scripts/deploy.sh` writes it to `.env`.

### Removed

- **`arc_titles` (V721-09)** — the multi-title arc capability is dropped: `drop_arc_titles` migration step (the historical `create_arc_titles_join` entry is kept — it is a `schema_migrations` record, not executable dead code — with a declared creator → dropper dependency and its `ApplyBackfill` removed), `ArcTitlesRepo` and its implementation, the `Create` invariant, `creditTitlesFor` (callers fall back to `[challenge.TitleSlug]`), `internal/prestige/cross_title.go` and the three dedicated test files. `order_audit_test.go` now asserts the table is **absent**, and a Phase-2 ratchet forbids reintroducing cross-title aggregation. Rationale and the rule itself are recorded in the `arch-rules` skill.
- **`PrestigeRepo.GetLeaderboard(titleSlug=nil)`** — the cross-title branch had no production caller (the "PP leaderboard" front page is a placeholder that fetches nothing) and was forbidden by the ratchet anyway. The parameter goes from `*string` to `string`, so the cross-title path is no longer expressible at compile time, and an empty slug is rejected with `prestige.ErrInvalidInput` instead of being degraded into "all titles".
- **`CountActiveParticipants`** — no production caller: one interface declaration, one implementation, two test stubs and one reopen-test step removed.
- **`weapons.name_fr` column (V721-05)** — dropped from the registry `CREATE` back in V72-06 and read by nothing since, the column nonetheless survived on already-migrated `metadata.duckdb` files: DuckDB refuses `ALTER … DROP COLUMN` while the ART primary key (`title_slug` + `weapon_key`) exists. New `purge_weapons_name_fr_column` step (CTAS-swap rebuild, `TargetMetadata`), ordered after `add_weapon_registry` with the dependency declared in `order_dependency_test.go`. Idempotent (no-op when the column is already gone, when the table is absent, and on a second run), loss-guarded (row counts compared inside the transaction before any `DROP`), composite primary key recreated after the rename. The non-DROP comment in `weapon_registry.go` now points at the migration instead of the backlog. **Not to be confused with `weapon_labels.name_fr`**, which is very much alive — it is the third link of the display-name `COALESCE` and the safety net for weapon ids that have no `weapon_key` in the registry yet.

## [7.2.0] - 2026-07-25

> Point release consolidating the "v7.2 Notion backlog" (items V72-01 to V72-34, branch `feat/v7.2-notion-batch`): objective-mode statistics end to end, `api/openapi.yaml` now **generated** by Huma, eradication of the cross-title leaks, Explorer live reads through the token pool, Discord alert hardening, and roughly twenty UI/i18n fixes. Per-domain summary. **Post-deploy actions are required — see *Ops*.**

### Added (React / TypeScript)

- **Objective stats — 4 surfaces** — `MatchObjectivesSection` (per-team scoreboard section, one column per stat of the mode played, "Team total" row aggregated at read time), Synthesis `objective_stats` block, `SquadObjectiveStatsPanel` (squad cumulative) and `TimeseriesObjectiveCard` (scope KPI, deliberately preferred over per-match `omitempty` fields). All double-gated (title capability `objective_stats` + data-driven), zero dead code; Halo 5 does not declare the capability, so the sections stay hidden.
- **Explorer — live status badges** — `ExplorerLiveStatus` (`ok` / `failed` / `no_auth` / `local_partial`) surfaced on the 5 live fetches, with a discreet per-section badge (end of the silent degradation, plan A3).
- **Gamertag search — explicit Xbox lookup** — `useGamertagSuggestions` gains `liveResults` / `isLiveLoading` / `liveAttempted` / `triggerLiveSearch`; "Search on Xbox" button in `GamertagSearchInput` (Explorer) and `GamertagCombobox` (generic), i18n in `common.toml`; stale live results dropped and an explicit "no Xbox result" state.
- **Admin — initial sync for a single player** — card on `AdminSyncPage` (`GamertagCombobox` max=1, optional cap, inline `AdminActionButton`, `useRunInitialSync`, active-title pill) plus a `SyncActionsHelp` legend documenting the scope of the four sync actions (forced cycle / global manual / convergence / initial).
- **Table-header tooltips** — `SortableTh.tooltip` + `ColumnMeta.headerTooltip`, both table families covered, FR/EN content aligned on ADR 0006.
- **Chart legends at the foot of the block** — `ChartCard.legend` + new `ChartLegend` component; informative blocks migrated (interactive legends documented as staying on the canvas).
- **"In placement" badge** — `PlacementPendingCell` variant `perf` on Perf / ΔPerf in the tables and on the home tiles, fed by the new `perf_placement_done/total` fields; the Note column keeps `placement_*`, and no zero is ever fabricated.
- **Sessions** — `SessionCareerXP` chart (XP per match + cumulative XP, mirror of Timeseries, data-driven gate) and a "View synergies" button in the L3 header of a squad session.
- **Squad menu** — the missing "Dynamics" L1 entry (`navL1Sections.tsx` + `common.nav.tab_dynamique`); the page, route and i18n already existed.
- **Match not synced** — dedicated `PageUnavailable` screen on `match_not_found` (FR/EN), replacing the generic error screen.
- **Contract guard-rails** — `contract-surface.guard` (static snapshot of `generated.ts`: 176 paths / 522 schemas / 30 enums, disappearance = red, bite proven by mutation), hardened `response-types.guard` (274 citations resolved, 20 critical responses re-exported), and a vitest guard locking `generated.ts` to `openapi.yaml` in CI.

### Added (Go API)

- **Objective stats (V72-03)** — mapping frozen on 8 real `GetMatchStats` payloads (CTF 11 fields, Zones 6 shared by Strongholds/KOTH, Oddball 6; KOTH = `ZonesStats` confirmed, hill discriminated by `StrongholdScoringTicks > 0`); table `match_objective_stats` created **directly append-only** (sequence + `written_at` + `_latest` view + `match_id` index, ART guard-rails extended); pure `ExtractObjectiveStats` + INSERT-only `persistObjectiveStats` inside the shared transaction, populated by `engine_fetch` and `engine_v2bridge`; extractor isolated in the `sync/objective` sub-package (god-package ratchet). Anonymized fixtures under `internal/sync/testdata/objective_stats/`.
- **Objective capabilities — two axes** — `CapMatchObjectiveStats = "match.objective.stats"` (server axis, governs the data path; `halo_infinite = supported`, `halo_5 = not_exposed`) and `CapObjectiveStats = "objective_stats"` (title axis, governs the UI, mirrored in TS `TITLE_CAPABILITIES` with its parity guard-rail).
- **Objective backfill CLI** — `cmd/backfill_objective_stats`, append-only via `persist.InsertObjectiveStats`; candidates = matches without the `MBitObjectiveStats` bit (1<<23) and without a `_latest` row, bit always set. Local run: 5,656 rows.
- **Objective citations** — new `objective_stat` citation mapping type + `loadObjectiveStats` loader reading `match_objective_stats_latest` (23 columns mapped, integral best-effort); four citations switched from awards to counters: `charge`→`zone_captures`, `got_you`→`flag_returns`, `stakeholder`→`zone_secures`, `flag_carrier_hunter`→`flag_carriers_killed`.
- **Explorer live through the token pool** — resolution order strict fresh session → profile → **pool** (`PolicyAnyPublic`), structured WARN + expvar counters; validated end to end on a target whose own refresh token is dead (career, identity, medals and seasons served identically to a healthy token).
- **Admin initial sync** — `POST /admin/actions/initial-sync/run {player_slug, title_slug?, max_matches?}` → `RunPlayerInitialSync` (RunFull mirror of `RunPlayerConvergence`, distinct `JobTypeAdminInitialSync` going through the ADR 0023 token pool, `resolveInitialMaxMatches` clamped 1..2000), same 400/409/503 contract as convergence.
- **New-medal notification** — `medal_first_earned` category: post-sync diff of medal totals, silent cold-start seed, recap beyond 3 medals, FR/EN medal names, Discord dedup per aggregation.
- **Halo 5 medal categories** — read-time resolver mirroring Halo Infinite: 215 medals in 11 categories under the 4 super-sections (0 uncategorized), no re-seed and no backfill; completeness guard-rail plus a category→super-section allowlist.
- **Per-title weapon names** — `weapon_names.toml` per title (29 Infinite / 55 Halo 5), validated loader, boot seed of `weapon_name_labels`, resolver `weapon_id` → `weapon_key` → `{en, fr}`; two guard-rails (registry↔TOML completeness, old sources forbidden inside the resolver).
- **Perf placement signal** — `perf_placement_done/total` (nullable, `omitempty`) computed by the single shared `computePerfPlacements` (chain via `GetPerformanceChain`, eligibility mirroring the perf batch, filled only when `1 ≤ chain size ≤ threshold` and the match has no `perf_score`), independent of the LUSR/CSR state.
- **Career XP on Sessions** — `CareerXPEstimated` on `SessionDetailMatchRow`, exact mirror of the Timeseries computation, gated by `games.ProvidesCareerXPEstimate` / `CareerXPErasFor`.
- **Discord sync-cycle notification** — `scheduler/auto_sync_notify.go`: `notifyDiscordSyncCycle` wired on the scheduler's periodic tick (existing `discord_notify_sync` toggle, no-op without a webhook or with zero new matches), `mergeCycleNewMatchPlayers` merging V2-engine players and live/safety-net players, deduplicated by gamertag; coach forwarding guard-rail untouched.

### Changed

- **`api/openapi.yaml` is now GENERATED (V72-01)** — `make openapi-gen` renders the shared Huma document in three passes (server paths, opaque-wrapper removal, JSON Merge Patch of the manual fragment) with deterministic serialization and true OpenAPI 3.1. `api/openapi_manual_fragment.yaml` is versioned (9 raw-chi routes, 15 `RawBody`, operation errors and descriptions, 55 non-derivable schemas; the fragment takes precedence). Golden tests `TestOpenAPIYAMLIsUpToDate` and `TestManualFragmentPathsSurviveGeneration` replace the deleted drift test. Semantic diff against the H0 baseline: 597 additions, 16 losses all inventoried (stale yaml), 530 schema names unchanged. **Never edit `openapi.yaml` by hand** — edit the Go handler/DTO, or the fragment.
- **Shared OpenAPI document across the 74 Huma mount points** — `NewAPI` variadic + `WithSharedDoc(cfg, docPrefix)` (legacy unchanged without the option), absolute prefixes at the composition point, `MarkRequestBodyOptional` resolving local→absolute, `OnOperationRegistered` anti-collision hook, fidelity test `TestSharedOpenAPIDocCoversAllHumaRoutes`.
- **Operation metadata on the 204 Huma routes** — `humacore.Op` helper (operationID / summary / tags), verbatim parity with the previous yaml (159 exact, 45 documented supplements), gate `openapi_operation_metadata_test`.
- **Portable DTO semantics as Huma tags** — 19 descriptions, 11 enums and 5 defaults carried by 10 domain types; gate `openapi_schema_semantics_test` (35 semantics verified, 0 loss outside the closed inventory).
- **Unified error model** — a single Go type behind `humacore.apiError`, enriched with `Details any` + `FieldErrors []FieldError` (both `omitempty`) → runtime contract `{code, message, retryable}` unchanged across 458 call sites; `ApiErrorSchema` / `FieldErrorSchema` dropped in favour of `ApiError` / `FieldError`.
- **`groups` routes migrated to typed Huma** — the 7 routes leave manual `writeJSON` / raw chi for the shared document (`RawBody` + in-house decoding preserving the 400 `invalid_body`), identical runtime JSON contract, zero test assertion changed; `Group` / `GroupMember` derived faithfully (`Role` enum, `members` never nil at the source).
- **Cross-title sealing (V72-29)** — `useCapabilityStrict` (fail-closed) on the banner synthesis gates; `titleSlug` added as the 2nd segment of every per-player query-key factory (48 keys, 27 call-site files, prefix invalidations preserved) with a key-completeness guard-rail and an anti-fail-open `spartan_customizer` guard-rail; server caches keyed by title (`HomeMatchesCache`, `CachedRecentMatchesProvider`, `seasonEntries`); `CareerLiveCache` keyed `(xuid, titleSlug)`; appearance store re-keyed `title::player` (v3 migration resets it), emblem and banner colours stored separately.
- **Title-switch race (V72-29bis)** — the Go middleware echoes `X-LevelUp-Title-Resolved` on every response (exposed in CORS); the client rejects any response whose echo diverges from the active title (`title_mismatch`, restricted to GET/HEAD — mutations only warn, as an anti-double-submit); `applyActiveTitle` cancels **before** `POST /session/context`, commits the slug after confirmation, then clears and re-bootstraps, with the bootstrap focus refetch neutralized during the switch.
- **Gamertag search** — new `live` parameter on the endpoint (`ctxkeys.WithGamertagLiveSearch` gating `LiveFallbackGamertagSearch`); **every caller defaults to `live=0`** (local only, ~200 ms) with the Xbox lookup as an explicit action — no surprise latency anywhere, a single arming point.
- **Frag sunburst** — grenades broken down by family (frag / plasma / dynamo / splinter) reconciled against the native total; `mechanic_kills` subtracted from the weapon classes, ending the Halo 5 double count (a melee kill while holding a weapon counted twice). Arc sum now equals the total on Halo 5; the "Frags per weapon" list is unchanged.
- **Match view** — the LIVE fallback is gone: `GetMatchView` returns a typed `match_not_found` 404 as soon as the match is absent from the local substrate (see *Removed*); the pre-existing openapi contract is unchanged.
- **`mode_name_tr` centralized** — 6 copies converged on `duckdb/mode_name_tr.go` with an empty-allowlist guard-rail; two previously swallowed errors are now logged. Squad-composition subtitles get their FR modes from the API (`ModeTranslatorFR` injected into the provider, `analysis.NormalizeModeLabel` for composed labels) and their FR playlists resolved by `playlist_id` (id > `name_fr` > EN).
- **French wording** — "Lobby" → "Partie", the "OC / DR" axes → "Rendement / Résistance" (EN "Yield / Resistance"), "Net score cumulé" → "Solde frags − morts cumulé", and the Meganaut FR description fixed by an idempotent migration (Waypoint's official localization is broken).
- **LeaderboardBlock sorting** — migrated to the shared `SortableTh` (13 headers, `aria-sort` gained), dated exemption removed; bidirectional AST guard-rail on Go `registry.go` ↔ TS `TITLE_CAPABILITIES` parity (dated `weapon_kills` allowlist).

### Fixed

- **Objective stats on live matches** — the watcher derives its options from `DefaultSyncOptions` plus a reflection guard-rail, so extraction flags can no longer be forgotten; without it the feature would have stayed empty on freshly synced matches.
- **Home cache invalidation** — keyed on the player DB's real `(xuid, title)`; it was a no-op keyed on the player slug.
- **Bot XUIDs** — `cleanXUID` handles the canonical `bid(x.y)` form on the objective **and** PvE paths (the bare-digit contract is preserved).
- **Medal snapshot** — `rows.Err` checked; an invalid set now seeds silently instead of emitting false notifications.
- **Explorer** — `navigate(..., { replace: true })` on mode/target selection (no more history stacking on every click); X axis restored on the "Cumulative frag gap" chart (Relations untouched) plus an explicit "no match in common" state; the briefing banner toggle no longer breaks the layout with the long label, and its aria wording matches the visible label.
- **Notifications** — career rank names localized by id (`RankLabelResolver`), `gap` / `current_mu` / `next_tier_mu` parameters rounded defensively.
- **Career page** — Rankings and LUSR/CSR evolution blocks height-aligned (`h-full` + fluid).
- **Squad dropdown** — popover minimum width and two-line subtitle; composite React keys on the player lists (empty, duplicated Halo 5 XUIDs).
- **Halo 5 ghost medals** — 3 `medal_name_id` earned in game but absent from both the official catalog and halopedia (cut/beta content) excluded from the Medals page by a dated list; the underlying data is kept.
- **Perf placement eligibility** — aligned with the batch (`start_time`); top-2 playlists resolved by id.
- **Front-side error reading** — the front was reading a `status.error.message` that does not exist server-side; switched to `error_detail` / `error_code`, with nullability guards. Bodies are now declared on the 7 201/202 responses and 4 missing `requestBody` entries.
- **Squad sorting** — `scoreSquad` compares both sides normalized; residual composed labels no longer break the sort silently.

### Ops

**Post-deploy actions (required)**

- **Objective backfill** — run `cmd/backfill_objective_stats` on prod with the server stopped. Without it, objective statistics only cover matches synced after the deployment.
- **Citation re-seed and recompute** — `levelup seed citation-mappings` then `levelup backfill --all --citations-recompute-all`. The four objective citations are recomputed from `match_objective_stats`: nothing is lost, but their values are revised (source: awards → counters).
- **Contributors** — `api/openapi.yaml` is generated: run `make openapi-gen` then `make generate-types`; `make openapi-check` fails on drift.

**Other**

- **Disk alerts — persistent anti-burst** — `DiskWatchState` persisted outside DuckDB (atomic JSON, `PathResolver.DiskWatchStatePath()` = `data/global/admin_state/disk_watch_state.json`, surviving container recreation thanks to the `data/` bind mount). Entering an alert, worsening, and the single recovery notification now survive restarts; the periodic reminder on a stable overshoot was removed.
- **App version baked into the binary** — `docker-compose.yml` `build.args.VERSION` + `scripts/deploy.sh` computing and exporting `LEVELUP_APP_VERSION` **before** the build. Releases no longer depend on the runtime environment — root cause of the missed v7.0.0 / v7.1.0 release notifications, which returned early on `AppVersion == "dev"`.
- **Dependabot** — #65 (postcss 8.5.16 → 8.5.23, dev dependency) merged, #51 (actions/checkout) closed.

### Removed

- **Match-view LIVE fallback** — the whole chain deleted together with its tests and imports: `tryCanonicalMatchView`, the `dataAdapter` / `viewerGamertag` fields and their `With*` setters, `LoadMatchDetail` on `games.TitleDataAdapter` and its 3 implementations, the Halo 5 `mapping_carnage_detail.go` file, `enrichCommendations` (lifetime totals kept), `ctxkeys.WithViewerGamertag`, the canonical `MatchDetail` / `Commendation` types, plus the orphaned `WithRankedClassifier` and the `ranked_hoppers.toml` boot load.
- **Dead code** — `queryKeys.player()` (0 callers), package `weaponregistry` (0 imports), `CareerPage` and `CareerTopMatchesTable` with their tests (not routed), the weapon registry's invented `name_fr` fallback, and the OpenAPI drift test with its helpers (superseded by the golden tests).

## [7.1.0] - 2026-07-24

> Point release consolidating the "v7.1 backlog" work plus the chantiers merged to `main` since the v7.0.0 tag (median-profile Intensity, Career Medals, Squad Dynamics, Expected-FDA differential, home/scoreboard quick wins, admin appearance diagnostic). Per-domain summary.

### Added (React / TypeScript)

- **Squad — Dynamics tab** — new `/squad/dynamique` tab regrouping the intensity, yield/resistance and engagement sections. Intensity reworked from a match × phase heatmap into a median frag-share profile per phase with a P25–P75 interquartile envelope (`lib/charts/phaseProfile`); Yield and Resistance split into two per-player multi-series charts (`buildSquadEfficiencyMultiOption`); damage-balance in title-lives (`netLives`, 225 Infinite / 115 H5) on Sessions and Dynamics via the generic `lib/charts/cumulativeSeries`; cumulative engagement-gap charts (`TimeseriesEngagementGapTrend`, `SquadEngagementGapChart`, `SessionEngagementCumulative`).
- **Squad objectives (challenges)** — full UI loop: localized labels, join feedback, per-member `SquadProgressList`, abandon/delete (2-click confirm), lifecycle (expiration from template cadence). "Cap d'escouade" renamed "Objectifs d'escouade"; ~40 labels migrated to the `squad.toml` manifest (`[squad.focus.*]`, ICU interpolation, ADR 0003).
- **Career — Medals page** — new `/career/medals` sub-page modeled on Citations: full per-title medal catalog including never-earned medals, grouped super-section → category (SpartanRecord taxonomy), all/earned/not-earned filter and three client-side sorts; multi-title (Halo Infinite rarity pill, generic grouping elsewhere).
- **Career — estimated XP** — cumulative XP curve + XP-per-match on Timeseries, capability-gated (`analytics.career_xp_estimate`), methodology tooltip.
- **Explorer — head-to-head panel** — "Over XX matches together" adds win-rate donuts (together / head-to-head, reusing the Relations donuts) and a cumulative frag-gap-vs-target chart; collapsible, persisted synthesis toggle (`explorerPrefsStore`).
- **Expected-FDA charts** — `TimeseriesFdaGapTrend`, `SessionFdaGapCumulative`, per-member `SquadFdaGapCumulativeCard`; a thin per-match expected-FDA line added on the FDA-gap chart (shared axis); canonical `divergentZeroGradient` helper (two inline copies migrated + guard-rail).
- **Home / Synthesis** — peak-achieved date on best LUSR/CSR cards; longest-loss-streak KPI card.
- **Scoreboard** — per-team identity colours + team logos (data-driven, `/titles/{slug}/teams/{team_id}.png`), shared player-column width.
- **Halo Waypoint column** — optional second action column on match lists (theme-aware logo).
- **Media — per-player audio-track roles** — gear modal in the player gallery to declare voice/game/other roles per track (manual > auto NNLS), FR/EN.
- **Sortable tables** — 16 tables made sortable (TanStack activation on the 4 `ExplorerMatchesTable` consumers, shared `SortableTh` on 8 Career/Admin tables with an anti-redefinition guard-rail, match-view clickable headers, generic `SortingFn<T>`).
- **Admin — Spartan appearance diagnostic** — per-player panel (Data tab) with pure `appearanceDiagDisplay` logic (verdict → status badge + i18n keys), on-click mutation and a re-authentication CTA to the existing SSO.

### Added (Go API)

- **Career XP estimate** — pure `analysis.xp_estimate` (era multiplier × personal_score, TOML-versioned eras: ×1 before 2025-11-18, ×2 after), additive `career_xp_estimated` on `TimeseriesMatchRow`, opt-in capability `analytics.career_xp_estimate` (Halo Infinite); calibrated to ~98.9 % on real data.
- **Path to Hero title-agnostic** — max-rank label resolved server-side (title source → rank-catalog top → generic fallback), carried in `HeroProgress` (`max_rank_name_fr/en`); fixes the Halo 5 XP reference line (9.3M → 50M / SR152).
- **Expected FDA** — `analysis.ExpectedFDA` / `FDADiff` (NaN/Inf guards), `CapExpectedStats` (Halo Infinite only), canonical `MatchParticipant.KillsExpected/DeathsExpected`, per-mode expected-assists batch + per-member resolver; `no_inline_expected_fda` arch-lint guard-rail.
- **Halo 5 vehicles & hijacks** — scoped read of `match_commendations` (natural PK, `INSERT OR IGNORE` → exact `SUM`), bilingual name resolution of the 9 "Destructeur" + "Grand Theft" commendations, `commendations.native` capability; per-title hijack label ("Dépositaire" Infinite / "Vol à la tire" H5).
- **Squad exact-composition** — centralized exact-composition predicate (`filterExactComposition`, pool = friends + top-teammates) across the three intersection sites (`GetPage`, briefing header, Q42 anti-join); `no_raw_squad_intersection` grep guard-rail. Compositions resolved by the current player's absolute XUID (`SquadContext`, `findSquadByRoster` xuid-keyed); append-only `levelup backfill-squad-creators` for legacy squads without a persisted creator.
- **Squad chart chronological order** — `computeMapBreakdown` computes first-appearance order server-side (heatmap pattern, deterministic `mapUI` tie-break); the front no longer re-sorts.
- **Squad challenges lifecycle (Go)** — `SquadChallengeView` (label + participants), `DELETE /squad-challenges/{id}` → `AbandonSquadChallenge` (nullable `archived_at`, non-indexed UPDATE, ART-safe), cascade archive on squad delete, template-cadence expiry.
- **Season pass** — raw `premium_owned` signal (`state.IsOwned`, undiluted) exposed for a reliable Premium badge; server-side locale for Challenges and Battle Pass (`challenge_snapshots` locale column + locale in the dedup key, append-only migration; `preferEN` threaded through the Battle Pass builders).

### Changed

- **URL locale segment** — the active language is now a `{-$lang}` route segment (mechanism was already shipped but dormant), so shared links keep their language.
- **Browser-tab titles** — single locale-aware `resolvePageTitle` mechanism, replacing the concurrent `__root` vs local-effect approach.
- **French wording** — anglicism sweep with an i18n guard-rail (grep backstop) forbidding the old literals.
- **Chart legends** — "Tools of destruction" legend centered below the chart; percentage labels on segments and legends; aligned chart heights across Synthesis, Sessions and Squad.
- **Scoreboard** — Halo 5 team names de-duplicated (`labelHasTeamWord`); MVP/LVP tie-break excludes mechanic kills (assassination / shoulder bash / ground pound) on both server and client.
- **Match view** — assassination/ground-pound/shoulder-bash columns gated by `useCapability('native_kill_mechanics')` (absent on Infinite, where they are stored as `0`, not `null`).

### Fixed

- **Citations (data)** — Firefight "eliminations" now count Firefight victories (`compute_wins_firefight`, tiers 5/10/15/25/50); grenade kills restored; "road trip" remapped to the Splatter medal (`221693153`); "flag defender" disabled (no measuring award); PvE stats query fixed (`total_enemy_kills`, not the non-existent `total_kills`); recompute-force now recreates `match_citations` via the append-only path (`EnsureMatchCitationsAppendOnly`, ADR 0026), no longer the legacy 3-column schema.
- **Halo 5 combat mechanics (data)** — targeted one-shot backfill (`backfill-h5-kill-mechanics`) of `match_participants` rows written as zeros before the mapper activation (2,742 carnages, 4,990 rows across 1,883 matches).
- **Squad** — the "Propose challenges" button no longer swallows its error (error toast front-side, `slog.ErrorContext` on the previously masked 500 branch across the prestige routes).

### Ops

- **Deploy — real Docker purge** — root cause of the 2026-07-23 VPS saturation: deprecated `--keep-storage` (Docker 29) silently remapped to a floor → no-op prunes. Fix: `--max-used-space` real ceiling, failures logged (no masking/abort after cutover), `image prune --filter until=24h` (keeps N-1 for same-day rollback), pre-build < 10 GB guard, versioned daily systemd units (manual VPS install, no cron daemon).

### Removed

- **Dead code** — `winRateVsHistoryChart` builder (0 callers) removed with its orphan i18n key; the 1-player squad-efficiency builder + its toggle/footer legend removed (superseded by the multi-player charts).

## [7.0.0] - 2026-07-23

> Consolidated entry grouping the work delivered since 2026-05-02 — previously labeled "Unreleased" pending the tag; shipped to prod as git `v7.0.0` on 2026-07-23. Per-domain summary, not commit-by-commit. *Version-number note: an unrelated, older `[7.0.0]` entry from the pre-migration Python/Streamlit app (2026-04-12) already exists further below in this file — kept as-is for historical continuity; the two `7.0.0` labels denote different products (Python vs. the current Go/React stack), not a re-release.*

### Added (React / TypeScript)

- **Sessions page — rebuilt** — full UX overhaul: F/D/A per match and per minute, performance score by tier, F/D/A radar, OC/DR cloud and per-match engagement charts with explicit axes and skill-tier bands; A/B compare drawer with shared scales; single-session metrics (Win %, KDR, kills/match, rank delta); adaptive session window; sessions with a single match are no longer listed.
- **Explorer — combat profiles & rivalries** — live read-only combat profile of any non-tracked player (career rank + Spartan grade, cadence charts, short-lived cache); dominance metrics and shared-history encounters; CSV export; cascade-aware filters across five dimensions; per-season match bars with CSR rank badge; partial match-ID search.
- **Explorer — briefing V2** — retrospective briefing refinements (Matches mode): the ranking card is split **per rating type** (CSR / LUSR), each shown as a known-tier progression (e.g. `Or II → Or VI`) plus average points per match — never a raw cumulative delta; new **Streaks** (best win / worst loss streak over the whole filtered scope) and **Highlights** (dominance-flag counters) cards; a **conditional solo/squad** context card (shown only when both sub-groups are relevant); `vs usual` deltas are hidden when the scope is the full history (they reappear as soon as a filter narrows it), with full-history dimensions ordered by win rate; unified section-card headers; date ranges now include the year; the playlist dimension is relabeled `Par sélection` (FR). This replaces the earlier `expected vs actual` win-probability block, removed as unreliable.
- **Explorer — briefing V3 (compaction)** — the Matches-mode briefing banner is compacted so the results table rises above the fold: **Ranking** and **Streaks** become socle KPI tiles, the win-rate trend becomes a bare micro-sparkline inside the **Win rate** tile, the solo/squad split is merged into the `By context` card of the `By…` grid, **Highlights** become a bare pill band, and the outcome-sequence tape is removed; legend `(i)` tooltips on the tiles, the `By…` cards and the band; net KDA is color-coded on every briefing surface.
- **Explorer — briefing V4** — post-review rework of the Matches-mode banner: the **Ranking** module is recomputed **per playlist chain** (one line per `(type, playlist_group)` chain, never a cross-chain arrow) and moves out of the tile socle into a split 4th column under `By context`; the socle tiles are rebuilt around the hero **Win rate** tile (`OutcomeBar` + W-D-T + 4-outcome tooltip) with **colored Perf**, **Total time**, **Peak FDA**, and a priority **cascade** of conditional tiles (Best streak > Peak rank > Peak MMR, at most two of three, never more than eight tiles); the trend sparkline is removed; `(i)` tooltips now render through a **portal** so they are no longer clipped by tile `overflow-hidden`; the results table highlights the **best/worst per key column** (MVP/LVP, reusing the scoreboard style) and uses tighter cell padding.
- **Explorer — briefing V5** — post-review finish of the Matches-mode banner: the **FDA** and **Perf** tiles become **min · average · max** triptychs (server-computed bounds, the highlighted average colored by net-KDA / perf tier), absorbing and removing the standalone **Peak FDA** tile; the conditional cascade now shows all **three** tiles when present (Notable streaks, Peak rank, Peak MMR — Peak MMR visible again, capped at eight); **Ranking** joins `By…` / `By context` as a sibling on a single auto-fit row (no more stacked 4th column); the results table now highlights the **top / bottom decile** (p90/p10 over the whole loaded scope) per key column with a softer tint instead of a single best/worst, and **numeric columns are right-aligned** (header and cells; text stays left); every KPI tile carries an **accent** (neutral token when sentiment-free), first-row values are **centered**, the redundant "N matches" counter is removed and **CSV export moved below the table**; **Streaks** is renamed **Notable streaks**, and dimension/context counters drop the word "matches" (kept as a hover title).
- **Explorer — briefing V6** — post-review finish of the Matches-mode banner: the results-table decile highlight (MVP/LVP) now covers **team MMR** (higher = better) instead of the score column (only its highlight is dropped; the Score column still renders); the **FDA** and **Perf** triptych min/max bounds become more legible (`text-foreground` full contrast, slightly larger) while the colored average stays dominant; every briefing tooltip is rewritten in a **factual, impersonal** register (no second person; the triptych reads "lowest · average · highest"); the **Notable streaks** tile is rendered ~10 % wider and **Peak MMR** ~10 % narrower (flex socle, no gap when conditional tiles are absent).
- **Compare / Face-à-face** — dedicated page with 3-player mirror mode (B vs A vs C), readable career rank + all-time CSR for non-local players, encounter stats and badges.
- **Citations** — dedicated page with composite score, LUSR/CSR badges and `CitationProgressRing`.
- **Ascension** — V3 player profile (radar, style badge, LUSR components, leverage panel), behavioral pattern detection (tilt / fatigue / plateau / ceiling), context grid (by mode/map/squad), campaign tracker with start modal, two-tab layout.
- **Objectives / Prestige** — challenges + squad challenges (collective / competitive), narrative arcs (free creation, presets, deletion, completion bonus), guided/coach-driven mode, PP leaderboard.
- **Squad coach** — squad orientation strip, challenge-pool bias by performance axes, `CoachFocusCard` ("focus of the moment") with soft-negative signal.
- **World leaderboard** — global CSR ranking enriched with native per-player stats, multi-season with cross-season trend indicator, local players surfaced first.
- **Media — HLS player** — in-browser clip playback (`hls.js`) with audio-track selector (game / voice / full mix); manual reassociation modal `MediaMatchPicker` (±15 / ±60 / ±180 min window, map thumbnail, per-team lobby, outcome badge) calling `POST /players/{slug}/media/associate`.
- **Admin dashboard** — full monitoring UI (sync cycles + trend sparklines, convergence, data-integrity invariants, token health, per-player Halo API-call attribution, recurring-error collector, logs, perf).
- **Settings — Backup tab** — restic snapshot status, manual trigger, per-database integrity check.
- **CSR / seasons** — CSR season selector + available seasons, dynamic placement thresholds, ranked badges, season pills with cascade-aware folding.

### Added (Go API)

- **CSR per match & per playlist** — `GetPlaylistCsr`, RankRecap per-match CSR, `season_id` + `is_ranked` at write time, dynamic per-season placement thresholds, authoritative ranked-playlist reference, automatic teammate CSR distribution, CLI `backfill-csr-history`.
- **LUSR v2 (TrueSkill2)** — `internal/analysis/skill_v2/` factor graph + expectation propagation, time-played weighting, quit ordering, pre-match win probability, tier calibration, anti-volatility safeguards; shadow mode then canonical, with offline replay and batch hyperparameter tooling.
- **World CSR leaderboard** — Halo Waypoint scraper, `world_player_season_stats` (append-only) enrichment, multi-token aggregator, dedicated cron + header provider, CLI backfill.
- **Coach advisor & squad coach** — proposal generation/accept orchestration (ADR 0020/0028), post-sync hook, HTTP endpoints, squad orientation + challenge-pool bias.
- **Backup / restore** — `pkg/duckdbbackup` generic restic scheduler + LevelUp adapter, `cmd/restore` point-in-time restore, structured logging.
- **Convergent sync** — autonomous asset-name resolution at primary write, weekly safety-net for stragglers, in-cycle catalog refresh cron, cross-source sync dedup gate, data-quality invariants gate.
- **Match timeline / T0** — `MatchTimeline` + `ComputeT0`, real gameplay duration (pre-match countdown subtracted), `CorrectEvents`/`CorrectImpactEvents` wiring, timezone re-normalization of `first_joined_time`/`last_leave_time`.
- **Achievements (Xbox)** — `sync-achievements` CLI + `RunAchievementsOnly`, cross-DB merge service, HTTP handler, category filter.
- **Access control** — multi-user player ownership + `RequirePlayerOwnership` middleware (ADR 0029), instance lockdown, "page unavailable" gating with `apiErrorCode`.
- **Observability** — `event_id` propagated across sync/auth/watcher flows, expvar concurrency metrics, data-integrity invariants, admin diagnostics endpoints.
- **Explorer briefing DTO — per-type ranking & new sections** — `ExplorerBriefingRanked` reworked to emit one `ExplorerBriefingRankedKind` per rating type (tier start/end labels, placement flags, points-per-match) instead of a raw cumulative delta; additive `KPIStats.RankDeltas []RankDelta` exposing the existing per-`RatingType` buckets; new `ExplorerBriefingContextSplit` (solo/squad), `ExplorerBriefingStreaks` and `ExplorerBriefingDominance` blocks; full-history dimension re-sort by win rate. The `expected_win_prob` computation and its DTO fields were removed. The `outcome_sequence` field and its `ExplorerBriefingOutcome` type were dropped from the briefing DTO (the front-end tape was retired in V3).
- **Explorer briefing DTO — per-chain ranking & socle aggregates (V4)** — `ExplorerBriefingRankedKind` gains `playlist_group`; the ranking is recomputed per `(rating_type, playlist_group)` chain via the pure `analysis.ComputeRankProgressionByChain` (net rating variation per chain, co-signed with the tier progression), replacing the cross-chain `KPIStats.RankDeltas` bucket (now removed). The `trend` sub-DTO (`ExplorerBriefingTrend`/`...TrendPoint`) is dropped; `ExplorerBriefingScope` gains `total_duration_seconds`, `peak_kda`, `peak_team_mmr` and `peak_ranks` (best tier reached per rating system, via `analysis.CSRTierOrdinal`).
- **Explorer briefing DTO — Min/Max scope aggregates (V5)** — `ExplorerBriefingScope` gains `min_kda`, `min_perf` and `max_perf` (`*float64`, `omitempty`), computed in-memory by the symmetric `minScopeFloat`/`maxScopeFloat` service helpers over the already-scanned per-match `KDA`/`PerformanceScore` (no new SQL, no migration); they feed the front-end FDA and Perf min·avg·max triptychs alongside the existing `kda`/`avg_perf`/`peak_kda`.
- **Explorer briefing — relevant ranking chains (V6)** — the ranking module no longer lists every chain: the service keeps only chains with at least `MinRankedChainMatches` (= `MinDimensionGroupMatches`, 10) matches, capped at the `RankedChainMaxCount` (3) most-played (`selectTopByMatches`, restored in canonical order), with a **fallback** that keeps the principal chain (majority rating type) when none reaches the threshold — mirroring the dimensions top/flop policy. In-memory filtering over the unchanged pure `analysis.ComputeRankProgressionByChain`; the `ExplorerBriefingRanked` DTO shape is unchanged (it simply emits fewer entries).

### Changed

- **Auth** — SISU provider by default (MSAL removed from UI); `MultiUserTokenStore` is the single source of truth for credentials (ADR 0023), with dead-refresh-token detection, reconnection banner and opt-in password for fast SSO re-login.
- **DuckDB write safety** — critical tables migrated to append-only / INSERT-only to avoid the ART corruption bug (`match_skill_rank`, `match_csrs`, `player_csr_snapshots`, `pve_match_stats`); shared DB provider (B-swap) enabled by default.
- **Token pool** — honors `Retry-After` (429/503) with exponential backoff, singleflight on the resolver to avoid `invalid_grant` bursts, periodic re-scan to hot-add tokens without reboot.

### Fixed

- **Gamertag display** — single-source lookup view, masked names resolved via `killer_victim_pairs`.
- **Timezone** — `first_joined_time` re-normalization (fixes T0 + quit-ordering offsets).
- **LUSR v2** — watermark-vs-row desync (skipped matches), delta ordered by `start_time`.
- **Media** — HLS audio-track selection on Chrome, HEVC remux on scan, `data/media` fallback when the captures base dir is invalid.
- **Squad** — empty charts with 2+ teammates (deduplicated intersection), displayed session = exact composition.

## [Go-API Phase 14] - 2026-04-29

### Added (Go API)

- **Prestige module — Objectives, Arcs & Squad Challenges** — two migrations: `create_prestige_player_schema` (player `stats.duckdb`) adds tables `arc`, `challenge`, `moment_card`, `prestige_telemetry`, `baseline_state`; `create_prestige_shared_social_schema` (`shared_social.duckdb`) adds `prestige_events`, `user_prestige`, `squad`, `squad_member`, `squad_challenge`, `squad_challenge_participant`. Squad challenges support two modes — **collective** (shared target, each member's contribution counted individually) and **competitive** (members race on the same metric). Repos: `PrestigeMetadataRepo`, `PrestigePlayerRepo`, `PrestigeSocialRepo`, `PrestigeBaselineProvider` in `platform/duckdb`. Lazy service wiring via `prestige_lazy_service.go` + `prestige_setup.go`. REST endpoints: `GET/POST /prestige/challenges`, `PATCH /prestige/challenges/{id}`, `POST /prestige/challenges/{id}/abandon`, `GET /prestige/arcs`, `POST /prestige/arcs`, `GET /prestige/profile`, `GET /prestige/social/leaderboard`.

- **In-app notifications** — migration `steps_player_notifications.go` adds `notifications` table per player `stats.duckdb`. `NotificationsRepo` + helpers in `platform/duckdb`. Handler (`notifications.go`) exposes 9 endpoints: `GET /notifications`, `GET /notifications/unread-count`, `POST /notifications/mark-read`, `POST /notifications/mark-all-read`, `PATCH /notifications/{id}/unread`, `DELETE /notifications/{id}`, `GET /notifications/preferences`, `PATCH /notifications/preferences`, `POST /notifications/test`. Boot wiring via `notifications_boot.go`; routes registered in `registry_notifications.go`.

### Added (React / TypeScript)

- **Objectives page** (`/players/$playerSlug/objectifs`) — two tabs: **Challenges** (active list + `CreateChallengeForm` + guided-mode toggle) and **My Journey** (arc retrospective + Prestige progress). Hooks: `useChallenges`, `useArcs`, `useMyPrestige`, `useAbandonChallenge`. Components: `ChallengeCard`, `CreateChallengeForm`. `lib/prestige.ts` defines contracts: `Challenge`, `Arc`, `UserPrestige`, `Tier`, `Cadence`, `EvalType`, `WindowType`, `ChallengeMode`; `TIER_COLORS` and `TIER_LABELS_FR`.

- **PP Leaderboard page** (`/players/$playerSlug/palmares/prestige`) — community PP ranking across squad and relations; period selector (week / month / all); raw/bonus/total PP breakdown; tier badges. Component: `LeaderboardPP`.

- **Notification center** — `NotificationsBell` in nav bar with unread-count badge, 60 s auto-refresh. Page `/players/$playerSlug/notifications`: cursor-paginated list, category + unread-only filters, day-grouped timeline, multi-select bulk actions (mark read/unread, dismiss, mark all read). Mutations: `useDismiss`, `useMarkAllRead`, `useMarkRead`, `useMarkUnread`. `NotificationsSettingsTab` in Settings.

---

## [Go-API Phase 13] - 2026-05-01 — Sprint 54

### Added (Go API)

- **Volet A — Season Calendars** — `MetadataRepository` interface + DuckDB implementation (`metadata_repo.go`). Tables `season_calendars`, `csr_season_calendars` with ETag/content-hash deduplication. `FetchSeasonCalendar` and `FetchCSRSeasonCalendar` in `platform/halo`. CLI `cmd/refresh-metadata` with subcommands `seasons`, `csr-seasons`, `staging`, `all`. Discord notification on hash change.

- **Volet A — Current Season** — `resolveCurrentSeason` in `CareerService` and `StatsService` via `WithMetadataRepo` optional builder. `syntheticSeasonResult` fallback returns `Synthetic.IsFallback=true` when metadata DB unavailable. `CurrentSeason` field added to `CareerPageResponse` and `StatsPageResponse`.

- **Volet B — Match Privacy** — `MatchPrivacyInfo` + `MatchPrivacyWarning` domain types. `privacy_provider.go` in `platform/halo` calls Waypoint privacy endpoint. `WithPrivacyProvider` optional builder on `BootstrapService`; `fetchPrivacyNonBlocking` with 2s timeout. `Privacy` field in `BootstrapResponse`; `PrivacyWarning` in `MatchHistoryPageResponse` and `MatchViewResponse`.

- **Volet C — Compare joueur vs joueur** — `CompareService` with parallel load via `errgroup` (player A from DuckDB, player B from Waypoint or local fallback). 12 KPIs via `buildMetrics`. `CompareRepository`, `PlayerStatsProvider` interfaces. Handler `POST /players/{slug}/pages/compare`.

- **Volet D — Multi-titre staging** — `EnsureStagingTables` in `metadata_repo.go` creates `waypoint_medals_raw` and `waypoint_assets_raw` tables. CLI `refresh-metadata staging` subcommand.

- **Volet E — CSR Leaderboard** — `LeaderboardService` + `LeaderboardRepository`. Local players (`IsLocal=true`) always ranked first. Handler `GET /players/{slug}/pages/leaderboard`.

- **Volet F — Tests** — `compare_service_test.go` (2 tests), `leaderboard_service_test.go` (2 tests). All pass.

### Added (React / TypeScript)

- `PrivacyBanner` component — shows yellow/red alert for partial/full privacy; integrated in `MatchHistoryPage` and `MatchViewPage`.
- `useCompare` hook (`useMutation`) + `CompareDrawer` with form and 12-KPI table.
- `useLeaderboard` hook (`useQuery`) + `LeaderboardBlock` with season/playlist filters.
- New types: `NormalizedPlayerStats`, `CompareMetricRow`, `CompareRequest/Response`, `MatchPrivacyInfo/Warning`, `LeaderboardEntry/Request/Response`.
- `privacy_warning` field added to `MatchHistoryPageResponse` and `MatchViewResponse` types.

### Changed (OpenAPI)

- New schemas: `MatchPrivacyInfo`, `MatchPrivacyWarning`, `NormalizedPlayerStats`, `CompareMetricRow`, `CompareRequest/Response`, `LeaderboardEntry/Response`, `CurrentSeasonResult`, `SeasonSynthetic`.
- `BootstrapResponse`: added `privacy` field.
- `MatchHistoryPageResponse`: added `privacy_warning` field.
- `MatchViewResponse`: added `privacy_warning` field.



### Added (Go API)

- **HaloClient interface** — `internal/sync.HaloClient` interface extracted from the concrete client. `SyncEngine`, `backfill_weapons`, and `syncCareerRank` now depend on the interface, making all Halo API calls mockable without network access.

- **`mockHaloClient`** — deterministic test double in `internal/sync` (same package). Supports controlled history, stats, film, and career responses, plus call-count assertions. Compile-time check via `var _ HaloClient = (*mockHaloClient)(nil)`.

- **Mock-based unit tests** — `engine_mock_test.go`: 15 new tests covering validation and mock behavior; no `//go:build integration` tag needed — runs on every CI pass.

- **Explorer KDA** — `CommonMatchRaw`, `CommonMatchRow` and the Q19 SQL query now include `kills`, `deaths`, and `kda` columns pulled from `match_participants`. `convertCommonMatches` maps them directly; zero-KDA entries no longer appear.

- **Input validation** — new `Validate()` methods on `FilterContextInput`, `MatchHistoryQueryRequest`, and `SyncOptions`; `xuid` length/digit validation in `syncCareerRank`; both HTTP handlers call Validate() before service dispatch and return 400 on invalid input.

- **Table-driven validation tests** — `internal/domain/validate_test.go`: 27 cases for all three Validate() methods; pure Go, zero CGO.

- **POST /backfill/start** — new endpoint and `BackfillStartRequest` domain type; route registered in `server.go`.

- **`DirMediaIndexer`** — `service.MediaIndexer` interface + concrete `DirMediaIndexer.ResetAndReindex` implementation replaces the old stub; job progress reported via `jobs.Store.Update`.

- **`NewSettingsHandlerWithIndexer`** — additional constructor for `SettingsHandler` that accepts an explicit `MediaIndexer`; original `NewSettingsHandler` delegates to it using `DirMediaIndexer`.

- **Media reset integration test** — `settings_media_test.go`: `TestPostMediaResetIndex_Stub_Replaced` verifies that a POST to `/settings/media/reset-index` reaches `JobStatusSucceeded` with `ProgressPct=100` and no "stub" text in `CurrentStep`.

- **ExplorerMatchRow OpenAPI schema** — `ExplorerMatchRow` in `api/openapi.yaml` now declares `kills`, `deaths`, and `kda` fields.

- **Golden test for KDA** — `TestExplorerService_GetCommonMatches_WithStats` asserts that non-zero kills/deaths/kda from `CommonMatchRaw` are preserved end-to-end through `convertCommonMatches`.

### Changed (Go API)

- **`extractKillsFromLabel` removed** — placeholder function (always returned 0) and its test deleted; KDA data now originates in the service layer.

- **Coverage baseline raised** — `coverage_baseline.txt`: `76.0` → `79.2` (per-package mean after filter); `internal/sync` went from 0% to 13.1%.

---

## [7.0.0] - 2026-04-12

### Added

- **Home V7 — active challenges restored from HaloStats `/decks`** — the Mission Control home now displays the active challenge card again, with deck completion counts, real expiry, localized title/description, Waypoint badge lookup, and the player's actual progress ratio (`x/y`).

- **V7 Synthesis page** — the new top-level section after Squad now works on the full match history by default, with its own local period selector, existing overview charts regrouped in one place, a self-contained solo-vs-squad duel chart, and cleaner long-range top-vs-total bucketing for sparse multi-year histories.

- **Media V2 — persistent likes and richer grouping** — screenshots and videos can now be liked directly from the Media V2 grid, grouped by liked state, session, or solo/squad context, and rendered with the user-provided local heart assets.

- **Challenge persistence layer** — new domain module `src/data/challenges.py` with internal split between `src/data/_challenge_catalog.py` and `src/data/_challenge_snapshots.py`:
  - `challenge_definitions` + `challenge_translations` added to `metadata.duckdb`
  - `challenge_snapshots` added to each player `stats.duckdb`
  - definitions are versioned by `content_hash`
  - player snapshots are append-only and deduplicated on state change via `state_hash`

- **Multi-language challenge catalog** — all challenge title/description translations exposed by the CMS are now persisted locally, normalized to BCP-47, with `en-US` fallback if the requested locale is unavailable.

- **Discord notifications for new indexed media** — a new failsafe module `src/utils/_discord_media.py` sends a rich embed (with GIF/image thumbnail attached) to the Discord webhook whenever new media files are indexed. Anti-spam via a `discord_notified_at` column in `media_files` — each file is notified exactly once regardless of re-scans. New `discord_notify_new_media` toggle in Settings (independent of sync/backfill notifs). Thumbnail is sent as a `multipart/form-data` attachment (≤ 8 MB); graceful JSON fallback if the file exceeds the limit or is unreadable.

- **Invitation-based authentication & registration** — new `/register` route backed by a dedicated `AuthRegisterPage` component; account creation requires a server-issued invitation code passed as a `?code=` query parameter. The Go API validates the code before writing the account (expired and already-used codes return explicit 400 errors). Route integrated in the React routing tree alongside `/login`; `AuthGuard` redirects unauthenticated visitors to the correct entry point based on context.

### Changed

- **Migration loading** — migration steps are now loaded dynamically, so new step modules such as `add_challenge_metadata` and `add_challenge_snapshots` do not depend on a hand-maintained import list.

- **Media V2 rendering path** — the media grid now uses native Streamlit thumbnails instead of a per-card iframe lightbox, plus a shared dialog lightbox with optional auto-advance for videos.

### Fixed

- **Home challenges are now live-first and failsafe** — if `metadata.duckdb` is temporarily locked by another Python process, the Home V7 challenge card still renders from live API data and simply skips persistence for that refresh instead of returning `None`.

- **Media likes survive reruns and reloads** — `data/ui_prefs.json` now preserves structured `media_likes` values during preference merges, auto-repairs legacy stringified likes, and avoids the double-toggle edge case on the heart control.

### Tests

- Added targeted coverage for challenge persistence and Home V7 enrichment in `tests/test_challenges_data.py`, `tests/test_home_mission_control_challenges.py`, and `tests/test_home_mission_control.py`.

- Added targeted coverage for Media V2 thumbnail rendering, like persistence, and button callback fallback behavior in `tests/test_media_components_sprint4.py`, `tests/test_media_v2_grid_interactions.py`, and `tests/test_ui_persistence_v64.py`.

## [6.5.0] - 2026-04-10

### Added

- **Teammates — per-player intensity heatmap** — a new section after "Squad complementarity" renders a match × phase (early/mid/late) heatmap for each squad member's kill profile. A segmented-control toggle (All / per player) switches views without re-querying the DB — data is loaded in a single pass. Reuses `compute_match_intensity_profiles` + `plot_match_intensity_heatmap`.

- **Discord — separate sync/backfill notifications** — a new independent toggle `discord_notify_backfill` is distinct from `discord_notify_sync`. Both checkboxes are now laid out vertically (one per line). `notify_operation_done` routes to the correct flag based on `operation.startswith("backfill")`. The Backfill section in Settings has been moved to last position, under a `st.subheader` with a warning caption and collapsed expander by default.

- **Shared info-layer component** — `_render_note` extracted into a public `render_info_note(key, lang)` function in `src/ui/components/info_note.py`, shared by Teammates and Timeseries pages.
  - 6 new i18n keys in `teammates.py`: `tm_no_data`, `tm_impact_caption`, `tm_weapons_chart_caption`, `tm_metrics_caption`, `tm_note_radar`, `tm_note_cadence`.
  - Captions added: impact heatmap, weapons bar chart, metric bar charts.
  - Post-chart notes added after the synergy radar and cadence map charts.
  - All conditional captions wrapped in `hints_visible()` across Teammates and Timeseries.
  - `EXCLUDED_WEAPON_IDS` import unified from `_weapon_data.py` (replaces the local `_FILM_EXCLUDED_IDS` constant).

### Changed

- **Settings V3 — `frozen=True`, `patch_settings`, atomic write** — major internal overhaul of the settings layer:
  - `AppSettings` is now `frozen=True` — no silent in-memory mutation.
  - `patch_settings(key, value)` replaces direct `save_settings()` calls across the codebase (`streamlit_app.py`, `sidebar.py`, all page sections).
  - `_WRITE_LOCK` ensures thread-safe writes; `_PROCESS_CACHE` deduplicates content to skip needless I/O.
  - Settings UI pages fully migrated to `on_change` callbacks — `_auto_save_show_*` and `_build_settings_from_ui` removed.
  - `_render_backfill_checkboxes` extracted; `_check_settings_consistency` made non-blocking.
  - `save_settings()` kept as a CLI-only wrapper.
  - `path_picker.directory_input` now supports `on_change` / `args`.
  - `__init__.py` exports `patch_settings`.

### Fixed

- **Settings — atomic write + cross-platform recovery cascade** — settings are now written atomically via `os.replace()` after writing to a temporary file. Recovery cascade: valid backup restore → factory reset → empty file protection. Prevents corrupted `app_settings.json` on crash or forced kill.
  - `_atomic_write`: retry `os.replace()` up to 4 attempts (50/100/200/500 ms) for Windows file locking.
  - `save_settings()`: full traceback logged (5 levels) to trace any future accidental overwrite.

- **Settings — `show_records` silently reset to `True`** — default fallback corrected from `True` → `False` in 3 files; `app_settings.json` and its backup updated in place.

- **`plot_map_outcome_timeline` removed** — the chart was disabled via `if False:` in all call sites (`teammates_map_charts.py` ×2, `win_loss.py`). The function and its source file (`_maps_outcome_timeline.py`) have been deleted; orphan i18n keys `tm_map_timeline_title/caption` and `wl_map_timeline_title/caption` also removed. Note: the chart was re-evaluated as option B and remains active in `win_loss.py` via its own implementation.

- **`app_settings.json.bak` excluded from version control** — contains the same sensitive data as `app_settings.json` (Discord webhook, local paths); added to `.gitignore`.

### Tests

- **Settings V3 + atomic write** — 76 new tests (31 + 45); settings coverage: 77.5 % → 87.7 %.
  - `TestGetSettingsPath` (5 cases): `LEVELUP_SETTINGS_PATH` override.
  - `TestApplyLegacyMigrations` (6 cases): `screens/videos` → `captures_base_dir` migration.
  - `TestPatchSettings*` (13 cases): persistence, session_state, no-op dedup, fallback, rollback.
  - `TestOnChangeSetting` / `TestOnChangeShowHints` (5 cases): toast, browser prefs.
  - Recovery cascade: valid backup, factory reset, retry Windows, empty file protection.
- `TestRenderInfoNote` — 10 new cases: `hints_visible`, lists, bold text, paragraphs, empty input.

---

## [6.4.0] - 2026-04-07

### Added

- **Media library — filters & sort** — the Media page now offers a full filter panel:
  - **Owner** — multiselect to show/hide sections (My captures / Teammate(s) / Unmatched)
  - **Map** — filter by map name (from match data)
  - **Mode** — filter by normalized mode label
  - **Outcome** — multiselect (Victory / Defeat / Tie / DNF) derived from match outcome codes
  - **Context** — radio selector to restrict to Solo or Squad matches
  - **Sort** — sort by capture date (default), map, mode, outcome, or owner; ascending or descending toggle
  - Type (image/video) and filename filters preserved from the previous panel
  - Unmatched media (no associated match) are unaffected by match-specific filters and always appear in their own section when selected

- **CSR fanout for squadmates** — when syncing a ranked match, the CSR ranking data for all registered co-players is now automatically collected from the `skill_json` API payload and distributed to each player's DB. Previously required each player to sync their own account to obtain CSR history for shared matches.

- **Comeback badges fanout** — Remontada / Collapse / Contre-Remontada badges are now computed for registered co-players during the sync fanout, in parallel with PSA and CSR distribution.

### Added (continued)

- **Teammates — fixed player legend panel** — a floating panel (bottom-right, `position: fixed`) now shows each squad member's color throughout the entire squad section. It appears from the squad header onwards and stays visible while scrolling. Legends have been removed from all individual charts on the page (kills/deaths, per-minute stats, metrics, killing sprees, HS+PK, first events, weapon kills) since they are fully replaced by the panel. Switch strategy by changing `_PANEL_MODE` in `teammates_legend.py` (`"fixed"` / `"sidebar"` / `"hidden"`).

### Added (continued)

- **DB healthcheck** — a new module (`src/utils/healthcheck_db.py`) verifies the state of all DuckDB databases at every app startup and after deploys:
  - Checks presence of tables, v6 views (`v_gamertag_lookup`, `v_match_full`, `v_killer_victim_full`, `v_weapon_kills`), and critical columns per DB
  - Verifies that `metadata.duckdb` is attachable from `shared_matches`
  - Detects pending migrations
  - Auto-repairs missing or broken v6 views via `ensure_resolution_views()`
  - `--deep` mode adds referential integrity checks (orphan participants/medals, duplicates)
  - CLI: `python scripts/healthcheck_db.py [--verbose] [--deep] [--player GT] [--json]`
  - Integrated into `launcher.py` — runs automatically after migrations at boot, prints ✅/⚠️/❌ to console
  - Integrated into `deploy.sh` — post-deploy smoke test, results appended to `data/logs/healthcheck_deploy.log` (persisted across deploys via the Docker volume)

- **UI state persistence across sessions** — selected player and language are now persisted in the browser via a lightweight `localStorage` component (`levelup.prefs`). On next visit the app automatically restores the last active player without any server-side session.
  - Filter preferences (map/mode/outcome/context) moved from `.streamlit/filter_preferences/` to `data/players/{gamertag}/ui_prefs.json` — inside the Docker data volume, so they survive container rebuilds and image updates
  - Silent migration: existing `.streamlit/` prefs are copied to the new location on first load, then the legacy file is left in place as a fallback
  - `_resolve_db_path` extended with a third priority level (localStorage slug → `data/players/<slug>/stats.duckdb`) between deep-link and SPNKr auto-detect
  - New custom Streamlit component: `src/ui/components/browser_storage/` (`persist_browser_prefs`, `restore_browser_prefs`, `clear_browser_prefs`)

### Added (continued)

- **Reading aids toggle** — a new sidebar checkbox ("Aides à la lecture") lets users show or hide the ~45 contextual help banners scattered across every page. The setting is stored in `ui_prefs.json` (key `show_hints`) and defaults to on. `hints_visible()` is the project-wide predicate used in all page modules.
  - Several `st.expander` help blocks converted to `st.popover` for a lighter inline experience (legend/badge sections in `match_view_players`, `match_view_encounters`, `career_top_matches_render`, `career_encounters_render`, `teammates_impact`)
  - `restore_hints_from_prefs()` restores the persisted value from `ui_prefs.json` on restart

- **Career KPI cards redesigned** — the summary row above the career charts is now a compact 8-card row:
  - **Matches played**, **Total duration**, **Frags**, **Deaths**, **Assists**, **Accuracy**, **Time alive**, **Results**
  - Frags, Deaths, Assists: main value with an inline per-minute sub-value in small type
  - Color coding (green / gold / red) vs. all-time average (threshold ±8%)
  - Results card: segmented bar with W/L/T/DNF raw counts and color legend
  - `render_top_summary()` removed as redundant; `_build_kpi_cards()` extracted to respect 80L limit

- **Win/Loss page merged into Timeseries** — the standalone Win/Loss page has been absorbed into the Timeseries page as a new tab. The `win_loss` route has been removed from `page_router.py`. Timeseries tabs renamed: Résumé · Cartes & Modes · Progression · Avancé.

### Changed

- **Docker image** — `ffmpeg` is now installed in the image, enabling video thumbnail generation in containerized deployments without a manual post-install step.

## [6.3.0] - 2026-04-03

### Added

- **Map & mode names localized in the UI language** — all map names, playlists, game modes, and pairs now display in French or English across every page: sidebar filters, match tables, charts, and the win-rate histogram. Powered by a new `asset_translations` table in `metadata.duckdb` storing 9 674 localized names across 14 BCP-47 languages.
  - New schema: `asset_translations (asset_id, asset_type, lang, name, fetched_at, PK)` and `medal_translations (name_key, lang, name, description, PK)` in `metadata.duckdb`
  - `v_match_full` v6 i18n overhaul: four legacy table JOINs (`meta.maps`, `meta.playlists`, `meta.playlist_map_mode_pairs`, `meta.game_variants`) removed; replaced by 8 `LEFT JOIN meta.asset_translations` (en-US + fr-FR × 4 asset types). New columns: `map_name_fr`, `playlist_name_fr`, `pair_name_fr`, `game_variant_name_fr`
  - `resolve_asset_name()` / `resolve_medal_name()` use the pivot tables for deterministic per-language lookups
  - `MetadataResolver.resolve()` now accepts a `lang` parameter
  - Populate with: `python scripts/populate_asset_translations.py` (supports `--dry-run`, `--force`, `--types map playlist pair game_variant`)

- **Medal description tooltips** — hovering a medal in the Last Match grid or the Citations section shows its description. Fallback to medal name when description is unavailable.

- **Squad all-time records** (Teammates page) — per-player career-best records surfaced on the Escouade page with colored rectangle annotations and per-map breakdowns. Records are fetched from each player's full match history (not only shared sessions).
  - `compute_squad_records()` — pure analysis function returning best all-time value per metric per player
  - Colored per-player rectangle overlays + record-by-map charts
  - Integrated into `render_trio_charts` and `render_metric_bar_charts`

- **Top Killer badge** 🔫 — new badge on the Impact timeline for the first player on the team to reach 10 kills. Added to `_EVENT_TO_EMOJI`. Explicitly excluded from *Héros silencieux* and *Faux-frère* badge detection to prevent conflation. Badge legend added to the expander below the grid.

- **Butterfly histogram — Time to First Kill/Death** (Teammates page) — the first-kill / first-death distribution is now a mirrored butterfly histogram with 15-second bins, vertical bin separators, and tick labels at every boundary. The pre-game countdown is subtracted so the timer reflects actual in-game time; `NULL` preserved when no kill/death occurred.

- **`playable_duration_seconds` + `real_start_time`** in `match_registry` (v6.3 migration) — `playable_duration_seconds` is the match length minus the pre-game countdown; `real_start_time` is the absolute UTC start of gameplay. Backfill: `python scripts/backfill_data.py --playable-duration`.

- **PSA fanout for teammates** — Personal Score Awards earned by squad members are now distributed to all relevant player DBs during post-sync.

- **Win rate histogram enriched** — bar tooltip now shows total match count per map; the map column uses the translated `map_ui` name.

- **Settings page V2** — reorganized into fixed sections (General, Sync, Performance, Display) with a simplified internal function signature. No behavioral changes.

- **VPS Ionos deployment package** — `packaging/nginx/` configuration files, `deploy.sh`, step-by-step guide (`docs/DEPLOY_GUIDE_ETAPES.md`), and Ionos-specific VPS guide (`docs/DEPLOY_VPS_IONOS.md`).

- **Scoreboard — medal & citation grid** — medals and citations in the Last Match scoreboard now display in a centered 4-column grid instead of a flat list, for a cleaner at-a-glance overview.

- **Scoreboard — weapons grid** — weapons are displayed in a centered 2-column grid with corrected 54×18 px thumbnail ratio.

- **Explorer — partial Match ID search** — new text input lets you type 3+ characters of a match ID to filter results; a live counter shows the number of matches found, with clear feedback when no result is found.

- **Sidebar — compact language selector** — the language switcher now uses flag emojis in a compact inline format; the app version is always visible below the logo.

- **Sidebar — inline sync indicator** — the sync status indicator is displayed inline with the player heading, without a match counter.

- **Last Match — performance score placement** — the player's performance score is now shown directly to the left of the LUSR/CSR rating for easier comparison.

- **Kill cadence histogram** (Combat tab, match timeline) — bicolor histogram showing kills-per-segment for your team (blue) vs enemies (vermillion) at 15/30/60s granularity. Two moving-average traces overlay the bars (one per team). Default granularity changed from 30s to 15s.

- **Match intensity heatmap** (Progression / Timeseries tab) — heatmap showing kill intensity by match × game phase (early/mid/late). Colorscale Plasma for high contrast.

- **Squad cadence chart** (Teammates page) — grouped-bar chart synchronizing each squad member's kill tempo across a shared match, with per-player color mapping.

- **Media library — periodic auto-indexing** — the media indexer now runs automatically every N hours (configurable via `media_indexing_interval_hours`, default 4 h; 0 = single-run only) and triggers a one-shot re-index after each successful sync (`media_reindex_after_sync` setting).

### Changed

- **`shared_matches.duckdb` → `shared_matches_v2.duckdb`** — the shared match database file has been renamed. `get_shared_matches_path()` helper introduced; all hardcoded paths updated project-wide; `compute_sessions.py` updated.

- **Radar squad** — complementarity axis thresholds recalibrated to p90 (was p80) for better per-mode distribution; all-time view removed, radar filtered by session only.

- **LUSR incremental seed** — seed cascade bug corrected: `seed_ratings` was being mutated before `existing_states` read it in incremental mode.

- **Sidebar i18n** — `_filters_cascade()` uses `playlist_name_fr` / `map_name_fr` when the UI language is French.

- **`_try_attach_meta_for_views()`** — now checks for `meta.asset_translations` (exists in v6) instead of `meta.maps` (removed in v6).

- **Last Match KPI row redesigned** — map and mode fused into a single card; date format restored; rendered via `render_compact_html_cards`; dominance legend added. MMR, Frags, and Deaths cards enriched with the opponent team's score and a colored gap indicator. `os_card` height unified; impact badges enlarged by +30 %; all KPI borders applied on all 4 sides.

- **i18n — `add_i18n_display_columns()`** — `map_ui`, `mode_ui`, and `playlist_ui` columns now injected at DataFrame load time, propagated into `session_compare` and timeseries pages; missing `match_view` translation keys added (medals, scoreboard detail).

- **Viz (Teammates) — Phase 7 ChartData** — the 5 squad chart functions migrated to the `ChartData` pattern for consistent rendering and reduced boilerplate.

- **Internal refactoring (V1 Phases 1-4 + V2 Axe C)** — `is_uuid_like` unified project-wide; `normalize_mode_label` decoupled; dead code removed; `normalize_mode` / `map_label_fn` callbacks eliminated from 49 call sites.

- **Viz refactoring — Axe G** — Plotly `title=` parameter removed from ~55 viz function signatures across 25 files; chart titles now rendered via `st.subheader()`. `margin_top` reduced from 30→10 in `config.py`. Eliminates Plotly deprecation warnings at runtime.

- **Viz refactoring — Axe H** — `PlotOptions` adopted in 10 viz functions (formerly `lang` + `height` positional args); 4 stale `# noqa: PLR0913` annotations removed. Three silent `title=` bugs fixed in `win_loss.py`, `match_view_players_nemesis.py`, `session_compare.py`.

- **UI refactoring — Plan V3 (axes K/I/L)** — session_state keys migrated to typed `SK` constants in 4 modules; `render_chart_or_info` adopted in 7 pages/components replacing ad-hoc `safe_chart_render` wrapping; 8 C901 complexity violations resolved via sub-function extraction (`match_view_players`, `match_view_players_nemesis`, `match_view_charts`, and others).

- **Sessions — schema fix + performance** — `mv_session_stats.session_id` corrected to `VARCHAR` (was silently stored as `INTEGER`); session row upsert migrated to bulk `conn.register()` (one INSERT instead of N round-trips); `_add_friends_columns` extracted as a vectorized Polars helper; `_refresh_session_stats` now incremental (recalculates only the 3 most recent sessions; full rebuild only when `new_ids=None`).

### Fixed

- `v_match_full` was silently created without i18n — `map_name_fr` was always `NULL` in production because `_try_attach_meta_for_views()` fell back to the no-metadata code path after failing to find `meta.maps` (removed in v6).
- `map_name_fr` / `playlist_name_fr` / `pair_name_fr` propagated in all Polars data paths via `COLUMNS_COMMON`; mismatch between sidebar filter values and DataFrame columns corrected.
- Map/mode names now display in French in: sidebar, bullet + perf charts, heatmaps, match history, Escouade charts, and Records page.
- Timeseries: match countdown subtracted from first-kill and first-death timestamps in `load_first_event_times()`; `NULL` preserved after subtraction.
- Apostrophes escaped in `title=` attributes of medal / citation tooltips (prevented broken HTML).
- `_build_map_id_index` reads `asset_translations` instead of the removed `maps` table.
- Map thumbnail lookup via `map_id` (asset_id) independent of displayed language.
- Squad radar: `shared_match_ids` intersection no longer collapses when a squad member has no matches.
- Squad radar: `compute_participation_profile` called with `ProfileOptions` (fixes `TypeError`).
- Records page: ghost hatched bars removed; `offsetgroup` corrected on bar data layers.
- Records overlay: correctly colored line, exact width, `duration_seconds` fallback.
- `playlist_name_fr` falls back to the EN name in `v_match_full` when FR translation is absent.
- Performance page: team score chart shows bonus or base score according to context.
- Squad stats-per-minute visible for all squad members (not only the focal player).
- Weapon aliases `Mutilator` / `Mutilateur` added to `_scoreboard_asset_urls`.
- `top_killer` added to `_EVENT_TO_EMOJI` (missing entry caused emoji display fallback).

- Last Match: font and background style of KPI cards harmonized across all card types.
- Scoreboard: row hover background removed for a cleaner scoreboard; citation descriptions now displayed inside the tooltip.
- Scoreboard: citations expandable filter correctly aligned with the Citations tab.
- Discord notifications: now use `v_match_full` + `COALESCE(playlist_name_fr, ...)` / `map_name_fr` so correct localized names appear in alerts.
- Citations: `spartan_carnage` score type corrected from `stat` → `medal` (now correctly sums killing-spree medals).
- Hero page: stale adornment cache cleared; hero backdrop no longer overflows onto KPI cards.
- i18n: French agreement corrected — label reads "Recherche par ID de Match"; `mode_ui` removed from `add_i18n_display_columns` to avoid unnormalized `pair_name_fr` values leaking into filters.

- Squad records invisible on Teammates charts — three distinct bugs fixed: `_resolve_record` returned `None` instead of falling back to the global record when `per_map_records` was empty (caused all record shapes to be skipped by Plotly); `map_ui` column was absent from the `d_self` pipeline in `trio_helpers`; import of `xuid` in `match_view_players_nemesis` pointed to wrong module.

- Date filter calendar: lower and upper bounds now span full calendar years, allowing free backward/forward navigation without hitting artificial day-level boundaries.
- Teammates legend panel: visibility now conditioned to the squad section only — panel starts as `display:none`, appears when the `#llp-squad-start` sentinel enters the viewport, and hides again when the Impact section sentinel (`#llp-impact-end`) reaches the top of the screen.
- Native Streamlit chrome hidden — header, toolbar, menu, and decoration bars suppressed via `.streamlit/config.toml` and CSS overrides.
- Explorer deep links: `open_match_button` now uses `session_state` instead of `query_params` to avoid triggering a DB switch on navigation; `_scroll_into_view` scrolls the matched row into view via a same-origin JS `scrollIntoView` call; gamertag deep links also benefit from the auto-scroll.
- KDA/efficiency ratio: value read directly from the API's `mean(ratio)` field instead of being recomputed as `(K + A/3) / D`, fixing systematic divergence for players with many assists.
- Media watcher: process-level guard (`_PERIODIC_LOCK` / `_PERIODIC_STARTED`) moved before the Linux/Windows branching to prevent duplicate `watchdog.Observer` instances on Streamlit rerun; active mode (inotify vs. polling) now logged on startup.
- Sync migrations: DB marked as migrated only after `ensure_resolution_views()` succeeds (success-based guard); bare `duckdb.connect()` calls in `_engine_connections.py` and `launcher.py` replaced by context managers.
- Healthcheck: `'repaired'` status treated as warning-level (was silently ignored); `recompute_status()` added to recalculate the overall result after in-place mutation of individual checks; `deploy.sh` updated to match.
- Migration runner: `metadata` DB processed before `shared` so the v6 i18n views can attach `metadata.duckdb` at creation time; a `logger.warning()` is now emitted when views fall back to NULL-column mode (degraded path made visible).

### Tests

- `test(ui_persistence)`: 9 new tests for `hints_visible()` / `restore_hints_from_prefs()`; 5 930 tests pass total
- `test(remediation)`: non-regression tests for P0.2 (`browser_storage` dead code removal), P0.3 (localStorage comment fixes), P1.2 (4 extra test cases)

- `test(i18n)`: `resolve_map_display_names()` coverage + `map_ui` / `mode_ui` column assertions
- `test(radar)`: `f1_vide` edge case + `shared_match_ids` collapse regression
- Test suite updated for Settings V2 signature, i18n propagation, medal/playlist refactoring
- Size baseline updated (`scripts/size_baseline.txt`)
- Missing coverage for Phases 1-4 refactor added; missing `map_ui` fixtures and orphan tests after Phase 4 corrected
- `test_teammates_helpers` adapted to use the new public `normalize_mode_label`
- `test(cadence)`: 17 unit tests for cadence histogram (97% coverage); `my_kills`/`enemy_kills` field variants; 4-trace output validation (bars × 2 + MA × 2)
- `test(sessions)`: `_add_friends_columns`, `_upsert_session_rows` bulk path, `_refresh_session_stats` incremental, PME/mv_varchar schema migrations

---

## [6.2.1] - 2026-03-29

### Added

- **Badges "Héros silencieux" & "Faux-frère"** (`src/analysis/impact_analysis.py`) — two narrative impact badges detected from `match_participants` + `medals_earned`:
  - *Héros silencieux*: top assists but not top kills on the team (formula B: assists ≥ threshold × medal boost)
  - *Faux-frère*: high kills on a losing team where the rest of the squad underperformed
  - Single-match and multi-match variants; both surface on the Impact timeline and Teammates pages
  - Legend exposed in an expander below the badge grid
- **Impact ranking table** — HTML table (`os-impact-table` CSS class) replaces the previous heatmap; shows ranked impact score per player per match with color-coded rows
- **Combined Headshots + Perfect Kills chart** (teammates page) — `plot_headshots_perfect_kills()` renders a single grouped-bar chart per teammate instead of two separate charts
- **Top Matches — BTB exclusion** — `career_top_exclude_btb` setting in `AppSettings`; when enabled, Big Team Battle matches are filtered out of the career top-matches list for fairer Arena/BTB comparison
- **`--btb-only` / `--arena-only`** backfill options in `scripts/backfill_data.py` — targeted repair of corrupted CTF/TC/KOTH/Assault team scores for BTB or 4v4 matches respectively
- **`monitor_uptime.sh`** — shell bash equivalent of `monitor_uptime.py` for environments without Python

### Changed

- **KDA → Efficiency rename** — all local aggregate variables named `kda` in `src/analysis/` renamed to `efficiency`; public API unchanged, only internal naming
- **Impact scoring** — *Héros silencieux* weight raised to +1.5; missing constants (`SILENT_HERO_MEDAL_BOOST`, `FALSE_BROTHER_LOSS_PENALTY`) centralized
- **Impact page** — heatmap replaced by the HTML ranking table; layout reorganized with impact badges first, teammates section last
- **Teammates page** — medals section moved to last position; individual Headshots and Perfect Kills bar charts replaced by the combined chart
- **Mode label normalization (phase 1 + 2)** — `resolve_display_mode()` added as a unified resolver; `translate_pair_name()` delegates to it; `mode_pair_overrides` expanded with 29 FR/EN overrides; sidebar filter and match tables use normalized labels
- **Top matches score normalization** — score delta divided by the match max for Arena/BTB equity; sorted by `performance_score` (not raw `score_diff`)
- **`waypoint_player` propagated** in top-match Explorer links so deep links open the correct player profile
- **Docker** — `.env.local` mount is now mandatory (removed `required: false`)

### Fixed

- **Corrupt team scores** — `team_score` set to `NULL` when contaminated by `ps_score` (CASTLE WARS, Sentry Defense, and all objective modes); `backfill_fix_score_inversions` extended to Slayer, KOTH, and Assault
- **Score inversions** — Blue/Red team score swap corrected for Slayer + KOTH/Assault; comeback threshold made proportional to match duration
- **Dominance flag** — `WHERE dominance_flag IS NULL` corrected to `WHERE dominance_flag NOT IN (1, 2)` so rows with default value `0` are no longer skipped during backfill
- **Comeback badge order** — `CONTRE_REMONTADA` now evaluated before `REMONTADA`; the previous code path was dead
- **Comeback threshold** — symmetric formula via `_resolve_threshold()`; avoids asymmetric detection between winning and losing teams
- **Navigation deep link** — player DB correctly restored when navigating directly to a match via `gamertag + match_id` URL params
- **`run.sh`** — spurious `nul` file created by Git Bash on Windows cleaned up at startup
- Impact badge display: emoji 🗡️ applied consistently for Faux-frère; "assists" label corrected to French in the badge tooltip

### Tests

- `identify_silent_hero_multi` + `identify_false_brother_multi`: 16 new tests
- `infer_mode_category` + inverted sidebar format: covered
- Top-matches: cases E/F/G — BTB filter, NULL score exclusion, badge priority sort
- `test(analysis)`: logging + cumulative efficiency coverage (v6.2.1 rename)
- **Total: previous suite + ~25 new tests, 0 failures**

---

## [6.2.0] - 2026-03-28

### Added

- **`src/analysis/comeback_analysis.py`** — pure analysis module for comeback badge detection from highlight kill-events (`event_type='kill'`, `time_ms`). Exposes `build_score_snapshot()` and `detect_comeback_badge()`. No DB access.
- **`src/data/comeback_backfill.py`** — service layer that loads events/participants from `shared_matches.duckdb`, calls `comeback_analysis`, and writes the result into `player_match_enrichment.dominance_flag`. Pattern mirrors `dominance_backfill.py`.
- **`DominanceFlag.REMONTADA = 3`**, **`DominanceFlag.DEBANDADE = 4`**, **`DominanceFlag.CONTRE_REMONTADA = 5`** — new enum values in `src/analysis/_medal_verdicts.py`. Stored in the existing `dominance_flag` TINYINT column (no migration required). Exclusive with DOMINATION/HUMILIATION by design.
- **Comeback badge threshold constants** in `src/analysis/_medal_verdicts.py`: `COMEBACK_DEFICIT_THRESHOLD`, `COMEBACK_COUNTER_GAP`, `COMEBACK_EARLY_CUTOFF`, `COMEBACK_COLLAPSE_CUTOFF`. Centralized for easy tuning.
- **`SyncScope.comeback_badges`** + **`SyncScope.force_comeback_badges`** — new fields; `comeback_badges` added to `_ALL_DATA_FIELDS` so `--all-data` activates it.
- **`--comeback-badges` / `--force-comeback-badges`** CLI arguments in `scripts/backfill/cli.py`.
- **Unified squad view** (`src/ui/pages/_teammates_trio.py`) — `f2_xuid` is now optional; `render_trio_view` handles squads of 2, 3, or 4 players. The 1-vs-1 single-teammate view (`render_single_teammate_view`) has been deleted entirely.
- **Combined Kills ↑ / Deaths ↓ chart** (`src/visualization/trio.py`) — `plot_trio_kills_deaths()` renders a mirrored bar+line chart per squad member using `barmode="group"` and symmetric Y-axis ticks via `build_symmetric_abs_ticks()`. Replaces separate kills and deaths charts.

### Changed

- `render_trio_view` now supports 1 friend (duo) without a code path change — `f2_xuid` is `None` when only one friend is selected.
- `_merge_trio_dataframes` returns a two-player join when `f2_df` is absent; downstream code detects the schema (`has_f2_cols`) instead of checking the DataFrame reference.
- `render_trio_synergy_radar`, `_render_per_minute_stats`, `_render_trio_performance_charts`, `_render_trio_medals` all accept `f2_name/f2_df = None`.
- `_TRIO_METRIC_SPECS` no longer includes kills/deaths entries (replaced by combined chart).
- **Game mode label normalization (phase 1+2)** — display names now go through a unified resolver (`resolve_display_mode`), `translate_pair_name` delegates to this resolver, and `mode_pair_overrides` was expanded with 29 FR/EN overrides to keep mode naming consistent across UI filters, tables, and charts.
- Size baseline updated after `render_trio_view` growth (added f2-optional branches).

### Removed

- `render_single_teammate_view()`, `_render_single_teammate_details()`, `_render_single_teammate_weapon_and_map()` from `src/ui/pages/teammates_views.py`.
- `render_comparison_charts()` (dead code) from `src/ui/pages/teammates_charts.py`.
- `render_single_teammate_view` import and routing branch in `src/ui/pages/teammates.py`.
- Test class `TestRenderSingleTeammateView` from `tests/ui/test_teammates_views_page.py`.
- Test class `TestRenderSingleMapSection` from `tests/ui/test_teammates_map_charts.py`.
- `test_render_comparison_charts_exists` from `tests/test_phase6_refactoring.py`.

### Tests

- **5 178 tests total, 0 failures** (4 skipped)

---

## [6.1.0] - 2026-03-21

### Performance

- **Sync 7-axis optimization** — full rework of the sync pipeline throughput:
  - **Axe 1 — Partial parallel post-sync** — citation backfill runs in a background thread while player DB writes are serialized; eliminates citation/write contention
  - **Axe 2 — shared_matches R/O direct** — shared DB opened read-only without ATTACH on the player connection; removes cross-DB locking; total sync time reduced ~30–40 %
  - **Axe 3 — parallel_fetch pipelining** — match history fetch and individual match detail requests overlapped via asyncio semaphore (`fetch_slots` tuned to 15)
  - **Axe 4 — Citations bulk SQL** — citation computation collapsed from N individual queries to 6 bulk SQL statements + `executemany`; sub-second for 50-match batches
  - **Axe 5 — CPU-bound transforms via `run_in_executor`** — match parsing offloaded from the event loop; no longer blocks async I/O
  - **Axe 6 — LUSR batch UPSERT vectorized** — `executemany` batch replaces individual per-match inserts; ~10× faster for large LUSR backlogs
  - **Axe 7 — Adaptive `batch_commit_size`** — commit interval auto-tuned to `max_matches / 10` (floor 20, cap 100) to reduce WAL pressure on large syncs
- **Timing logs** — `post_sync` now logs elapsed time and per-task counts (perf/sessions/citations) at INFO level

### Fixed

- **`refresh_materialized_views` Binder Error** — `GROUP BY mode_category` (alias) replaced by `GROUP BY 1`; DuckDB 1.4.4 couldn't resolve the alias when the CASE expression references a joined column (`mr.pair_name`)
- **shared_matches handle conflict in post-sync** — `batch_compute_performance_scores` now falls back to the player connection (`shared.*` prefix) when `shared_conn` is unavailable due to a file handle conflict; eliminates the "0 performance scores batch" warning; logic split into `_load_matches_for_perf()` + `_compute_perf_updates()` helpers (≤ 80 L each)
- **Career rank name stored incorrectly** — `parse_career_rank()` now reads `rank_name` and `rank_tier` from `metadata.duckdb` (via `career_ranks`) instead of the approximate formula; fixes wrong display (e.g. "Silver 3 (VI)" instead of "Lance Corporal Diamond 1") in logs and `career_progression` table

### Tests

- 6 new tests covering timing log format and post-sync parallel execution (`tests/perf/test_post_sync_parallel.py`, `tests/perf/test_dual_semaphore.py`)
- **4 799 tests total, 0 failures**

---

## [6.0.0] - 2026-03-15

> ⚠️ **Weapon extraction still in beta** — attribution accuracy not guaranteed in all cases (estimated coverage 70–100 % depending on matches); weapon catalog in progress.

### Added

- **ID resolution layer** — three SQL views in `shared_matches.duckdb` replacing all ad-hoc 5-source gamertag cascades:
  - `v_gamertag_lookup` — FULL OUTER JOIN `xuid_aliases` + `match_participants` with deduplication and priority
  - `v_match_full` — `match_registry` enriched with i18n metadata (maps, playlists, game variants)
  - `v_killer_victim_full` — killer/victim pairs with resolved gamertags
  - `ensure_metadata_attached(conn)` helper added to `src/utils/db.py`

- **`weapon_labels` table** in `metadata.duckdb` (`src/data/migration/steps/add_weapon_labels.py`)
  - Schema: `weapon_labels(weapon_id UBIGINT PK, name_en VARCHAR, name_fr VARCHAR)`
  - DB-first resolution: `_resolve_weapon_from_db()` with `@lru_cache` + Python dict fallback
  - Automatic migration `add_weapon_labels` (`target_db="metadata"`) registered in the migration system
  - `src/ui/i18n/weapons.py` cleaned up: `get_all_weapon_ids`, `get_weapon_ids_by_faction`, `translate_weapon_name` removed

- **`src/auth/` package** — new auth layer replacing all manual Azure/env configuration:
  - `LEVELUP_CLIENT_ID` hardcoded — no Azure portal setup required for end users
  - `_msal.py`: `SerializableTokenCache` persisted in DuckDB (`sync_meta`) via MSAL
  - `provider.py`: single entry point — process cache (4 h TTL), MSAL silent refresh, `AuthRequiredError`, `start/complete_device_flow`
  - `_halo_exchange.py`: stateless `access_token → (spartan, clearance)` exchange via spnkr.auth

- **Launcher — SSO auto-detection** (`launcher.py`)
  - Gamertag automatically resolved from Microsoft login via Halo API (no manual entry required)
  - New first-launch flow: Device Code → DuckDB MSAL → sync → Streamlit (zero manual configuration)
  - Recovery menu simplified to Device Code Flow only

- **`resolve_medal_name` helper** (`src/analysis/`) — medal name resolution from `metadata.duckdb`, no hardcoded dicts

- **Last Match — previous/next navigation** — `◀ Previous` / `Next ▶` buttons to navigate between all filtered matches without reloading data

- **`populate_metadata_from_discovery.py` rewritten** for v5.1+
  - Reads `match_registry` from `shared_matches.duckdb` (replaces deprecated `match_stats`)
  - Extended DDL with i18n columns (`name_en`, `name_fr`, `mode_name`, `playlist_canonical_*`)
  - Logic extracted into `scripts/_metadata_db.py` (DDL + i18n enrichment)

- **`WeaponKillsMixin.load_grenade_melee_kills(xuid, match_ids)`** — new repo method querying
  `shared.match_participants` for grenade/melee totals. Replaces all direct `_get_connection()`
  calls in UI code (`_timeseries_weapons.py`, `match_view_weapon_kills.py`, `teammates_weapons.py`).

- **Custom medal: Avenger** — detects revenge kills (you kill the opponent who last killed you) via `killer_victim_pairs`
  - Custom ID `9 000 000 001` (beyond official Halo medal range)
  - Global backfill via correlated subquery on `killer_victim_pairs`: for each kill, checks if the victim is the killer from the player's previous death
  - CLI: `python scripts/backfill_data.py --avenger` (or `--force-avenger` for full recompute)
  - Names (`medals_fr/en.json`) and descriptions (`medals_descriptions_fr/en.json`) in static JSON files
  - `resolve_medal_description()` enhanced with JSON fallback when `metadata.duckdb` has no `medals` table
  - 18 tests (12 backfill + 6 description)

- **Top Gun label** 🔫 — badge on the Impact timeline for the first player on your team to reach 10 kills in a match
  - Constant `TOP_GUN_KILL_THRESHOLD = 10`; `_find_top_gun_event()` scans `highlight_events` chronologically
  - Integrated into the existing impact events pipeline (no UI caller changes needed)
  - Bilingual labels: "As de la gâchette" (FR) / "Top Gun" (EN)

### Changed

- **Gamertag resolver** (`src/data/sync/_gamertag_resolver.py`) — 5-source cascade replaced by a single `v_gamertag_lookup` JOIN; `load_match_player_gamertags()` reduced from 4 sequential queries to 1
- **`match_registry` consumers** migrated to `v_match_full` (asset loader, career encounters, etc.)
- **`killer_victim_repo`** and `career_encounters_data` migrate to `v_killer_victim_full`
- **i18n DuckDB migration** — `modes_fr/en.json` migrated to `metadata.duckdb`; playlist and game_variant JSON dicts removed from source code
- **`get_tokens_from_env()`** (sync) — deprecated wrapper delegating to `src.auth`; internal callers updated
- **Weapon parser — global correlation** — fire_event match rate corrected from 15 % to 95 % after fixing `b2_dispatch` routing; compact COMPLETE logs with sentinel / no_weapon distinction and `b2_dispatch` drop rate exposed
- **Backfill `--weapons --all`** — match_ids deduplicated across all players so each film is downloaded only once
- **`v_weapon_kills` view enforced app-wide** — all read queries now use `shared.v_weapon_kills`
  (exposes `effective_weapon_id = COALESCE(reconciled_as, weapon_id)`) instead of the raw
  `weapon_kills` table. Affected: `match_view_weapon_kills.py`, `citations/_data_loader.py`,
  `_roster_loader.py` (scoreboard top-weapon subquery).
- **`load_weapon_kills_for_player` replaces `load_weapon_kills_for_match`** in
  `match_view_weapon_kills.py` — SQL-level filter by `xuid` instead of Python post-filter.
- **`v_gamertag_lookup` enforced app-wide** — all remaining `LEFT JOIN xuid_aliases` patterns
  replaced; guards `_has_shared_view` / `_has_shared_table` removed (view guaranteed present
  in v6). Affected: `_encounter_loader.py`, `_career_encounters_repo.py`, `_roster_loader.py`,
  `_events_repo.py`, `_discord_queries.py`.
- **`load_match_roster()` simplified** — two redundant Python gamertag enrichment passes
  (inline `xuid_aliases` + `v_gamertag_lookup` queries with dead guards) removed; enrichment
  delegated exclusively to `resolve_gamertags_batch()`. ~45 lines removed.

### Removed

- **`highlight_events.gamertag` column** — migration `drop_highlight_events_gamertag`; gamertag resolved via `v_gamertag_lookup` instead
- **`resolve_xuid_from_input` wrapper** — dead code removed
- **`get_outcome_name_fr`** and `_refdata_outcomes` — replaced by metadata.duckdb lookup
- **14 Azure/OAuth functions** in `launcher.py` (−652 lines net): Azure wizard, `has_client_id`, `config-az`/`paste-id` recovery options, environment variable `SPNKR_AZURE_CLIENT_ID` no longer required
- **`_has_gamertag_column()`** helper in `_weapon_kills_repo.py` — dead code since `drop_highlight_events_gamertag`
- **Dead guards** `_has_shared_view("v_gamertag_lookup")` in `teammates_impact.py` and `_events_repo.py` — `else` branches returning `NULL AS gamertag` removed
- **`_append_grenade_melee()`** helper in `teammates_weapons.py` — replaced by `load_grenade_melee_kills()`

### Tests

- `tests/test_resolution_views.py` — 11 tests: view priority, alias/participant fallback, NULL filter, deduplication, EN columns always populated, FR columns NULL without metadata, idempotence, gamertag resolution for killer/victim
- `tests/test_global_correlation.py` — 19 tests: **100 % coverage** on `_global_correlation.py` (38/38 stmts, 12/12 branches)
- `_parser_logging.py` — **100 % coverage** (57/57 stmts, 10/10 branches)
- **4 719 tests total, 0 failures**

---

## [5.7.0] - 2026-03-13

### Added

- **Traductions FR des rangs Halo** (`src/ui/i18n/ranks.py`)
  - 17 rangs de carrière (Recruit→Recrue, General→Général, Hero→Héros…) + 6 tiers CSR (Silver→Argent, Gold→Or…)
  - Helper `translate_rank()` avec fallback sur le nom anglais original
  - Intégré dans le script de migration metadata (`migrate_metadata_to_duckdb.py`)

- **Launchers bilingues FR/EN** (`LevelUp.sh`, `LevelUp.bat`)
  - Détection automatique de la langue système (POSIX `LC_ALL`/`LANG`, Windows Registry `LocaleName`)
  - ~30 messages localisés dans chaque launcher (premier lancement, erreurs, winget, etc.)

### Changed

- **Pandas→Polars** : suppression de 7 appels `.to_pandas()` dans les modules UI/viz
  - `participation_charts.py` (pie, bars, stacked) : Polars natif bout en bout
  - `participation_charts_extra.py` (sunburst) : `.to_pandas()` conservé uniquement à la frontière `px.sunburst`
  - `objective_analysis.py` (3 tables assist/awards) : `st.dataframe` Polars natif
  - `duckdb_analytics.py` (KDA trend) : `st.line_chart` avec `x=`/`y=` Polars natif

- **CSS-only map thumbnails** : remplacement du script JS sandboxé (non fonctionnel) par un système hover CSS pur
  - Suppression de `_MAP_TOOLTIP_SCRIPT` dans `styles.py` (38 lignes JS)
  - Classes `.map-hover` + `.map-popup` dans `static/styles.css`
  - `match_table_html.py` et `win_loss_table_style.py` mis à jour
  - `_build_map_url_index()` amélioré : `lru_cache(maxsize=None)`, normalisation Unicode

### Removed

- Guard `was_pandas` dans `_performance_relative.py` : `compute_performance_series()` accepte désormais uniquement `pl.DataFrame`

### Tests

- `TestHighlightEventsSequenceIdempotent` ajouté dans `test_migrations.py` (couverture A.4)
- 45/45 tests migrations passants

## [5.6.0-beta] - 2026-03-10

> ⚠️ **Beta** — weapon attribution accuracy not yet guaranteed in all cases (estimated coverage 70–100 % depending on matches); weapon catalog in progress.

### Added

- **Weapon extraction from SPNKr films** (`src/analysis/weapon_parser.py`, `src/data/services/weapon_extraction_service.py`)
  - Parses `REPLICATION_DATA` chunks from match films to identify the weapon used for each POV kill (player_index=1, universal invariant)
  - kill→last fire event correlation within a 2 000 ms window; melee/grenade/vehicle kills detected via medals (`MELEE_API_ID=1`, `GRENADE_API_ID=0`)
  - POV coverage: ~87.5 % of kills
  - Hexagonal architecture: `weapon_parser.py` (pure domain, zero IO), extended `HaloAPIPort`, `WeaponExtractionService` (orchestration), enriched `WeaponKillsMixin` (upsert, backfill bit, queries)
  - Table `weapon_kills (match_id, xuid, weapon_id, kills)` in `shared_matches.duckdb` (PRIMARY KEY `match_id, xuid, weapon_id`) + index `idx_wk_match_xuid`
  - Migration `add_weapon_kills` (`target_db="shared"`) registered in the automatic migrations system
  - Local cache of downloaded chunks in `data/investigation/chunks/<match_id>/`

- **Sync integration** (`src/data/sync/_engine_weapon_kills.py`)
  - Automatic weapon extraction on new matches via `WeaponKillsEngineMixin`
  - Controlled by `SyncOptions.with_weapons`; configurable via `spnkr_refresh_backfill_weapons` in `app_settings.json` and the Settings page checkbox

- **Backfill weapon_kills** (`scripts/backfill_data.py --weapons`)
  - `--weapons [--force-weapons] [--gamertag <GT>]` via the unified backfill CLI
  - Bit `MatchBits.WEAPON_KILLS` (1 << 21) set on `match_registry.backfill_completed` after processing

- **Weapon kills section in Match View** (`src/ui/pages/match_view_weapon_kills.py`)
  - Summary tab: kills-by-weapon table for the POV player

- **Teammates Weapons tab** (`src/ui/pages/teammates_weapons.py`)
  - Per-weapon kill breakdown for all teammates on shared matches

- **MSAL Device Code Flow** (`src/utils/msal_device_flow.py`, `src/ui/xbox_oauth_ui.py`)
  - Replaces the OAuth redirect flow: user enters a short code on xbox.com/activate (no redirect URI or client secret required)
  - Pure MSAL wrapper: `initiate_device_flow()`, `acquire_token_blocking()`, `DeviceCodeResult`, `DeviceFlowError`
  - Streamlit UI component: start / polling / reset (integrated in Setup Wizard step 2 and Settings)
  - `setup_wizard_xbox.py` extracted from `setup_wizard.py` to stay within the 500-line module limit
  - `--device-code` flag added to `scripts/spnkr_get_refresh_token.py` for CLI token acquisition
  - `msal>=1.28.0` added as optional dependency
  - Azure configuration simplified: only `client_id` required (no `client_secret`, no `redirect_uri`)

- **Friends Impact matrix** (`src/visualization/friends_impact_heatmap.py`)
  - Vertical separators (Plotly shapes) between each match column for improved readability
  - Renamed from "Impact Heatmap" to "Impact Matrix" (FR i18n update)

- **Documentation** (`docs/CONFIGURATION.md`)
  - Azure guide simplified for Device Code Flow — `client_secret` and `redirect_uri` steps removed

### Fixed

- **Discord notifier** (`src/utils/discord_notifier.py`) — Lightweight embed restored when all players are idle (was accidentally suppressed in a previous commit)

### Tests

- **51 unit tests** (`tests/test_weapon_parser.py`, `tests/test_weapon_service.py`): constants, `find_frame_positions`, `build_frame_estimator`, `correlate_kills_to_weapons`, `count_kills_by_api_weapon`, `WeaponExtractionService` mocks (no kills, no film, dry-run, upsert, caching, errors), `WeaponKillsMixin` repo (upsert/conflict, bit marking, missing matches, gamertag lookup)
- **28 tests added/rewritten** for Device Code Flow (`tests/test_msal_device_flow.py`, `tests/test_auth.py`, `tests/test_xbox_oauth.py`, `tests/test_setup_wizard_logic.py`, `tests/test_setup_wizard_page.py`): `authorization_pending`, `slow_down`, DC Flow no-secret pattern, `get_spartan_tokens`, `resolve_player_identity`, `complete_device_code_flow`
- **4 041 tests total, 0 failures**

### Removed

- **Xbox OAuth redirect flow** — `build_xbox_auth_url()`, `generate_oauth_state()`, `exchange_code_for_refresh_token()`, `run_xbox_oauth_callback()`, `_handle_xbox_oauth_callback()` removed; replaced by Device Code Flow
- **`client_secret` / `redirect_uri`** — No longer required for token acquisition; `SPNKR_AZURE_CLIENT_SECRET` and `SPNKR_AZURE_REDIRECT_URI` environment variables are deprecated
- **`scripts/backfill/backfill_weapon_kills.py`** — Standalone backfill script removed (violated CLAUDE.md: all backfill must go through `scripts/backfill_data.py`)

---
## [5.5.0] - 2026-03-07

### Added

- **Session Comparison page revamped** (`src/ui/pages/session_compare.py` and related modules)
  - Outcomes distribution: W/L/T/DNF donut charts per session with win-rate in center
  - Match highlights: best/worst match per session (F/D ratio, mode name)
  - F/D + accuracy progression: K/D curve renamed F/D (FR), accuracy on secondary Y-axis (dashed), avg lifespan in hover
  - Modes breakdown: grouped horizontal bar chart of modes played per session
  - Map stats table: wins/losses per map for both sessions
  - Cumulative net score: per-match performance score coloring (green ≥70 / orange ≥45 / red <45) + LUSR or CSR overlay on secondary Y-axis (auto-detected from `match_skill_rank`)
  - Participation profile: replaced opaque stacked radar with grouped horizontal bars; thresholds scaled by number of matches

- **Setup Wizard — Guided initial configuration** (`src/ui/pages/setup_wizard.py` + `setup_wizard_logic.py`)
  - Two flows: **Xbox Express** (recommended, 2 steps) and **Azure manual** (advanced, 3 steps)
  - Custom CSS cards with icons, animated progress bar, numbered steps
  - Logic separated from UI (`SetupStatus`, `validate_azure_credentials()`, `validate_gamertag()`, `create_player_profile()`, `save_azure_credentials()`)
  - Guard in `main()`: the wizard displays automatically when credentials or player are missing
  - FR/EN i18n (~49 keys) in `src/ui/i18n/setup.py`

- **Xbox OAuth — One-click Xbox login** (`src/ui/xbox_oauth.py` + `xbox_oauth_ui.py`)
  - Full flow: Microsoft URL → callback `?code=XXX&state=YYY` → code exchange → refresh_token → spartan/clearance tokens → gamertag+XUID resolution → automatic provisioning
  - `xbox_oauth.py` (436L): pure OAuth logic without Streamlit dependency
  - `xbox_oauth_ui.py` (163L): Streamlit component integrated in Settings (login button, status, logout)
  - CSRF protection with random `state` validated on callback return
  - FR/EN i18n in `src/ui/i18n/pages/xbox.py`

- **Player Provisioning** (`src/app/player_provisioning.py`)
  - `provision_player()`: creates `data/players/{gamertag}/stats.duckdb` + `sync_meta` table + registers in `db_profiles.json` — idempotent

- **Auth Status** (`src/utils/auth.py`)
  - `AuthStatus` dataclass + `get_auth_status()`, `check_credentials()`, `write_env_local()` (writes/updates `.env.local` while preserving comments)

- **macOS / Linux compatibility** — `LevelUp.sh` (new): first-launch launcher equivalent to `LevelUp.bat` for macOS/Linux, written in POSIX sh (no bashisms — compatible with macOS bash 3.2, dash, zsh). Detects Python 3.10+ via versioned binaries (`python3.12` → Homebrew), Homebrew Intel/Apple Silicon paths (`/opt/homebrew`, `/usr/local`), then generic. Distribution-targeted help messages. `run.sh` fixed to detect `.venv/bin/python` (macOS/Linux) or `.venv/Scripts/python.exe` (Windows Git Bash). `launcher.py`: `_find_system_python()` enriched with versioned candidates and Homebrew paths; `_cmd_doctor()` now uses `_preferred_python_executable()` cross-platform.

- **`launcher.py setup`** — Interactive installation command: detects Python (py launcher → PATH → standard locations → installation via winget), creates `.venv`, installs dependencies (`pip install -e ".[spnkr]"`). Supports `--update` to update an existing environment.

- **`launcher.py doctor`** — Full environment diagnostic: OS, Python, venv, critical vs expected package versions, number of configured players, presence of `metadata.duckdb`

- **Portable packaging** (`packaging/build_release.py`)
  - Generates a self-contained zip `LevelUp-v{version}-win64-portable.zip` containing Python Embeddable 3.12 (~15 MB) + the full project
  - First launch: automatic dependency installation via pip

- **GitHub Actions Release** (`.github/workflows/release.yml`)
  - Triggered on push of tag `v*.*.*`
  - Portable zip build + automatic publication as a GitHub Release

- **Portable `%APPDATA%` mode** (`src/utils/paths.py`, `auth.py`, `env.py`)
  - Data stored in `%APPDATA%/LevelUp/` (Windows) or `$XDG_DATA_HOME/levelup/` (Linux) when no `.venv` at the root
  - Developer mode: `./data/` if `.venv` exists
  - Override possible via `LEVELUP_DATA` environment variable
  - `.env.local` looked up in `DATA_DIR` first, then at the repo root

- **Token fallback DB** (`src/ui/profile_api_tokens.py`)
  - Fallback 3: reads the refresh_token from the player DB `sync_meta` if absent from environment variables

- **Documentation**
  - `docs/CONFIGURATION.md`: complete rewrite with TOC, step-by-step Azure guide with 11 annotated screenshots, sections Player Profiles, Environment Variables, App Settings, Security, Troubleshooting
  - `docs/FR/CONFIGURATION.md`: FR version updated
  - `docs/SYNC_GUIDE.md`: rewrite with v5.1 sync architecture, ASCII diagram, detailed commands
  - `docs/FR/SYNC_GUIDE.md`: updated

- **Automatic schema migrations** (`src/data/migration/`) — versioned runner applied automatically at startup (`launcher.py → _run_migrations()`). Each DB (`player`, `shared`, `shared_pve`) tracks migrations in a `schema_migrations` table. 11 initial migrations registered. To add a schema change: create an idempotent `ensure_xxx` function in `src/data/sync/migrations.py`, create the corresponding step in `src/data/migration/steps/` and import it in `steps/__init__.py`.

### Fixed

- **CSRF** (`streamlit_app.py`) — Fixes comparison `_xbox_state != _xbox_state` (self-comparison, always False) → `_xbox_state != _expected_state`
- **`_repo_root` undefined** (`src/ui/profile_api_tokens.py`) — `_repo_root()` was never imported → replaced with `REPO_ROOT` from `src.utils.paths`
- **Expanded DuckDB retry** (`src/data/sync/_engine_connections.py`) — `except duckdb.IOException` → `except duckdb.Error` + retry delay `0.15s → 0.5s`
- **GC sync mode** (`src/ui/_sync_duckdb_ops.py`) — `gc.collect()` + `time.sleep(0.3)` to release DuckDB file handles on Windows
- **OAuth consumed guard** (`streamlit_app.py`) — `_xbox_oauth_consumed` flag to prevent double-processing of the callback on Streamlit rerun
- **Test isolation webhook** (`tests/test_monitor_uptime.py`) — Patches `get_secret` instead of mutating `os.environ` to avoid reloading `.env.local`
- **Deprecated Streamlit API** (`src/ui/pages/setup_wizard.py`) — Replaces the three `use_container_width=True` occurrences with `width="stretch"`: Xbox Express button, Azure manual button, and OAuth `st.link_button`.
- **Missing UI smoke test** (`src/ui/pages/setup_smoke_test.py`) — UI module recreated: 3 phases with progress bars, verification table, and continue/retry buttons.
- **Incorrect `SPNKrAPIClient` test patch** (`tests/test_player_tokens.py`) — Mock target corrected to `src.data.sync._career.create_api_client` to match the API abstraction.

### Tests

- **75 tests added** (1,482 lines) covering all new modules:
  - `test_auth.py` (13 tests): `AuthStatus`, `get_auth_status()`, `write_env_local()`
  - `test_setup_wizard_logic.py` (20+ tests): `SetupStatus`, validation, profile creation, edge cases
  - `test_xbox_oauth.py` (18 tests): OAuth URL, code exchange, store/load token, provisioning
  - `test_xbox_oauth_callback_e2e.py` (9 tests): full code→player flow, errors, CSRF, token cycle
  - `test_setup_wizard_page.py` (15 tests): mocked UI (MockStreamlit), Xbox/Azure modes, progression; `width="stretch"` assertions on widgets
- **3,831 tests, 0 failures**

### Architecture

- **API Abstraction — Ports & Adapters**: decouples the codebase from the SPNKr library to make a future API backend switch easier
  - `api_port.py`: `HaloAPIPort` Protocol — structural contract (`runtime_checkable`) defining the methods every Halo API client must implement
  - `api_factory.py`: `create_api_client(backend="spnkr")` factory — centralized instantiation, extensible to other backends
  - `_auth.py`: authentication facade — UI modules call `refresh_halo_tokens()` without importing SPNKr directly
  - Consumer migration: `engine.py`, `orchestrator.py`, `strategies.py`, `_career.py`, `populate_metadata_from_discovery.py`, `profile_api_tokens.py`, `player_assets.py`, `xbox_oauth.py` — all now use the factory or the auth facade
  - 14 dedicated tests (`test_api_abstraction.py`): Protocol compliance, factory behavior, auth facade, and verification that migrated UI modules no longer import SPNKr

### Removed

- **`scripts/_archive/`** — 89 dead code files deleted (legacy weapon analysis scripts, diagnostics, i18n patches, obsolete utilities)
- **`requirements.txt`** — Removed, replaced by `pyproject.toml` (single source of truth for dependencies)
- **`setup.bat`** — Replaced by `LevelUp.bat` (improved Python detection, installation via winget, use of `pip install -e .`)
- **`scripts/install_dependencies.py`** — MSYS2 SSL workaround, used `requirements.txt`
- **`scripts/setup_env.ps1`**, **`scripts/setup_env.sh`**, **`scripts/activate_env.sh`** — Replaced by `launcher.py setup`
- **`tests/test_spnkr_refactoring.py`** — Tests for deleted archived code

### Chore

- Root cleanup: `ACKNOWLEDGMENTS.md`, `CHANGELOG.md`, and `CONTRIBUTING.md` moved to `docs/`
- Scripts moved: `activate_env.sh`, `run_monitor_hidden.vbs` → `scripts/`
- `LevelUp.bat` replaces `setup.bat` as the Windows entry point
- `Dockerfile` and `e2e-browser-optional.yml` updated to use `pyproject.toml` instead of `requirements.txt`
- `run.sh` now redirects to `launcher.py setup` instead of `activate_env.sh`

### Additional updates (8 March 2026)

- **XP & Hero rank multi-player comparison** — Career page now overlays XP curves and Hero projections for every player with a refresh token:
  - Real XP curve (lines + markers, distinct colour per player)
  - Pre-sync estimated XP curve (dotted, same colour) — linear interpolation over matches played before the first sync
  - "At this pace" → Hero projection (dashed) and optimistic projection (challenges + boost ×2, dash-dot)
  - All secondary curves hidden by default — click the legend to show them
  - **Variable precision** depending on available data: real XP delta between snapshots when enough syncs exist, otherwise falls back to a global average rate (total XP / days since earliest known match, or since Career Rank launch on 20 June 2023). Precision improves automatically with each new sync.

### Additional updates (7 March 2026)

- **Timezone selector** — Choose the display timezone directly in Settings (Europe/Paris by default, ~40 timezones available). Match timestamps adapt automatically throughout the app.
- **Improved Career page** — Better readability of the LUSR ranking section, smoother navigation.
- **Bot xuid migration** — Automatic correction of matches containing misidentified bots in the shared database.
- **Stability** — Fixes on adversary data loading, match queries, UI cache, and synchronisation. Improved reliability on Windows during concurrent DuckDB access.

## [5.4.0] - 2026-03-04

### Added

- **Explorer page — unified match search and navigation** (`src/ui/pages/explorer.py`)
  - Replaces the legacy "Match" page with a 6-module architecture (`explorer`, `explorer_results`, `explorer_enrich`, `explorer_data`, `explorer_logic`, `match_table_html`)
  - **Cascading filters**: date, squad (solo/squad), experience type (ranked/unranked/PvE), playlist, game mode, map
  - **Fuzzy gamertag search** with dynamic suggestions and XUID resolution
  - **OS-style HTML table** (`match_table_html.py`): KDA, kills, deaths, accuracy, score, MMR delta, performance, headshots, spree, average life; cross-page deep links
  - **Deep linking**: `?page=Explorer&gamertag=XXX` or `&match_id=XXX` for direct navigation
  - **Encounter badges**: rival, mentor, prey — computed from cross-player history
  - **Enrichment** (`explorer_enrich.py`): team score, MMR delta, performance, average lifetime, Waypoint URL
  - **Complete FR/EN i18n** (`src/ui/i18n/pages/explorer.py`)
  - **Structured logging**: info (deep links), warning (player not found, missing DB), error (DB exceptions with `exc_info`)
  - **40 unit tests** (`tests/test_explorer_logic.py`) covering logic, enrichment, data access, and HTML rendering

### Tests — previous skips fixed

The following tests were marked `@pytest.mark.skip` or `skipif(True)` and now run normally:

| File | Test(s) | Fix reason |
|------|---------|------------|
| `tests/test_rag.py` | `TestHaloKnowledgeBase` (3 tests) + `test_chunk_overlap` | Removed `skipif(True)` guards and false skip |
| `tests/test_season_archive.py` | `test_get_archive_info_with_archives` | Removed skip + `>= 0` assertion (tiny Parquet file) |
| `tests/test_i18n_refactoring.py` | `test_no_duplicate_keys_in_module[pages]` | Added package support (`pages/` folder instead of `pages.py`) |
| `tests/e2e/test_streamlit_browser_e2e.py` | `test_e2e_004_deeplink_match_query_params` | Regex `exception(?!nel)` excludes "exceptionnel" (French word) |
| `tests/test_cache_integrity.py` | 11 SQLite legacy tests | File **removed** (v3 dead code) |
| `tests/conftest.py` | all `e2e_browser` tests | Removed auto-skip guard + installed Chromium |

To rerun only these tests:

```bash
# RAG
python -m pytest tests/test_rag.py::TestHaloKnowledgeBase tests/test_rag.py::TestTextChunker::test_chunk_overlap -v

# Season archive
python -m pytest tests/test_season_archive.py::TestDuckDBRepositoryArchives::test_get_archive_info_with_archives -v

# i18n (package pages/)
python -m pytest tests/test_i18n_refactoring.py::TestNoInternalDuplicates -v

# E2E deeplink
python -m pytest tests/e2e/test_streamlit_browser_e2e.py::test_e2e_004_deeplink_match_query_params -v

# Full suite without integration
python -m pytest -q --ignore=tests/integration
```

### Added

- **Encounter history — section below the scoreboard** (`src/ui/pages/match_view_encounters.py`)
  - New HTML table displayed directly below the scoreboard on the Match View page
  - For each non-friend player in the match: encounter frequency, ally/enemy split, ally win rate, enemy win rate, cross K/D (from `killer_victim_pairs`), and last encounter date
  - Sorting: enemies first, then allies; within each group by `total_encounters DESC`
  - Compact grey row for first encounters (`total = 1`), full row with metrics beyond that
  - Automatic inline badges: **Hard to Kill** (deaths/kills > 2 and at least 3 deaths), **Ally+** (ally WR ≥ 65% over at least 2 matches), **Tough** (enemy WR ≤ 35% over at least 3 matches)
  - Color coding reuses scoreboard CSS classes (`os-sb-td--best`, `os-sb-td--worst`, amber)
  - Scope: all non-squad, non-friend players

- **Dedicated SQL loader** (`src/data/repositories/_encounter_loader.py`)
  - `load_encounter_stats(self_xuid, target_xuids, db_path)` — 3 CTEs on `shared_matches.duckdb` (`match_participants`, `killer_victim_pairs`, `match_registry`, `xuid_aliases`)
  - Automatically derives the `shared_matches.duckdb` path from `stats.duckdb`
  - Uses a direct `duckdb_read_only()` connection on shared (no ATTACH conflict)

- **Pure testable logic** (`src/ui/pages/match_view_encounters_logic.py`)
  - `EncounterStats` (Pydantic v2), `Badge` (dataclass), `ordinal_fr()`, `build_friends_set()`, `filter_encounter_xuids()`, `compute_encounter_badges()`
  - `build_friends_set`: dual source `player_match_enrichment.friends_xuids` → fallback `friends_defaults.json`
  - 28 unit tests in `tests/test_match_view_encounters.py` (without importing Streamlit)

- **i18n keys** (`src/ui/i18n/pages.py`): `mv_encounter_history`, `col_role`, `col_encounters`, `col_wr_ally`, `col_wr_enemy`, `col_kd_cross`, `col_last_seen`

### Technical

- `match_view.py`: calls `render_encounter_section()` after `render_match_scoreboard()` (+10 lines, zero business logic added to the file)
- SRP architecture preserved: 3 new files under 350 lines each, functions under 50 lines, UI and data logic separated

### Refactoring & Architecture (branch `refactor/cleanup-all`)

> **Massive 6-phase refactor** — 331 files modified, about 30,000 lines rewritten, 72 new submodules, 3,693 passing tests (including 79 dedicated tests added). No user-facing functional changes.

#### Phase 0-4: Infrastructure & initial splits

- **Split `transformers.py` (2,095L → package)** — `src/data/sync/transformers/` split into 7 submodules (`_helpers`, `_match`, `_skill`, `_events`, `_medals`, `_personal_scores`, `_pve`) + `__init__.py` re-exporting everything; no breaking change
- **Split `filters_render.py` (1,460L → 4 modules)** — extracted `_filters_period.py`, `_filters_session.py`, `_filters_cascade.py`; `filters_render.py` reduced to orchestration
- **Split `engine.py` (1,500L → 8 mixins)** — `_shared_writes.py`, `_performance.py`, `_skill_rating.py`, `_career.py`, `_aggregates.py`, `_tokens.py`, `_engine_connections.py`, `_engine_schema.py`
- **Split `duckdb_repo.py` (1,200L → 8 mixins)** — `_match_queries_helpers.py`, `_match_queries_polars.py`, `_archives_repo.py`, `_awards_repo.py`, `_diagnostic_repo.py`, `_events_repo.py`, `_medals_repo.py`, `_schema_introspection.py`
- **Split utility modules** — `media_indexer.py`, `api_client.py`, `batch_insert.py`, `discord_notifier.py`, `cache_loaders.py`, `radar_chart.py`, `teammates_views.py`, `sync.py`, `timeseries_combat.py`
- **`_SyncProtocol`** (`src/data/sync/_protocol.py`) — explicit `Protocol` contract for the 8 `DuckDBSyncEngine` mixins; removes 70+ `# type: ignore[attr-defined]`
- **`PageContext` + `MatchViewParams`** (`src/app/_page_context.py`) — real types instead of 5 `Any` fields in the `NamedTuple`
- **`SessionKeys` / `SK`** (`src/app/session_keys.py`) — 20+ centralized `st.session_state` keys, IDE completion, no more silent typos
- **`_sql_fragments.py`** (`src/data/query/_sql_fragments.py`) — single source of truth for `WIN_RATE_EXPR` (WIN+LOSS denominator, `NULLIF` division), `IS_WIN`, `IS_LOSS`; 7 duplicated occurrences removed from `analytics.py` and `trends.py`
- **v4→v5 technical debt removed**: `_PERF_SCORE_AVAILABLE` guard (always true), dead method `_ensure_performance_score_column()`, magic number `outcome == 4` → `Outcome.DID_NOT_FINISH`

#### Phase 5: Analysis & visualization splits

- **Split `performance_score.py` (950L → 3 modules)** — `_performance_relative.py` (relative match score), `_performance_session.py` (session score v1/v2, `ScoreComponent`); public facade unchanged
- **Split `antagonist_charts.py` (570L → 3 modules)** — `_antagonist_kv.py` (stacked bars, time series, heatmap), `_antagonist_duels.py` (duel history, nemesis summary, indicators); public facade unchanged
- **Split `rag.py` (750L → 4 modules)** — `_rag_models.py` (RAGConfig, Document, SearchResult), `_rag_github.py` (GitHubIndexer), `_rag_chunker.py` (TextChunker); public facade unchanged

#### Phase 6: UI & data splits

- **Split `refdata.py` (880L → 2 modules)** — `_refdata_personal_scores.py` (68-member `PersonalScoreNameId` enum, score/name/ID dictionaries); public facade unchanged
- **Split `_roster_loader.py` (520L → 2 mixins)** — `_gamertag_resolver.py` (`GamertagResolverMixin`, 5-source XUID→gamertag cascade); `_roster_loader.py` now inherits from the mixin
- **Split `cache_filters.py` (740L → 3 modules)** — `_cache_loading.py` (recent matches, pagination, match count), `_cache_sessions.py` (session DB computation); public facade unchanged
- **Split `filters_render.py`** — `_filters_apply.py` (`apply_filters` in 190L, empty-state diagnostic); public facade unchanged
- **Split `session_compare_charts.py` (480L → 2 modules)** — `_session_compare_history.py` (HTML history table); public facade unchanged

#### Quality & coverage

- **79 dedicated unit tests** — `test_submodules_phase5.py` (37 tests) + `test_submodules_phase6.py` (42 tests), covering the 13 submodules directly and verifying public re-exports
- **Logger added to 3 previously silent modules** — `_cache_loading.py` (6 `except` blocks → `logger.debug` with `exc_info`), `_performance_relative.py` (1 catch-all), `_rag_github.py` (1 network error); all submodule `except Exception` blocks are now traced
- **Centralized logging system** (`src/utils/log_config.py`) — `setup_app_logging()`: file-only logs (`data/logs/app.log` 5 MB×3, `data/logs/sync.log` 10 MB×5), no console output; `setup_script_logging()` for CLI scripts; `log_duration()` context manager with configurable millisecond threshold. Wired into app launch, player loading, session selection, filter changes, DataFrame loading, KPIs, match navigation (last match / carnage / previous match buttons), sync UI, backfill CLI, tailscale, and RAG. `data/logs/` is excluded from the repository.
- **`.gitattributes`** — enforces `eol=lf` across the repo; resolves pre-commit mixed line-ending conflicts on Windows (`core.autocrlf=true`)
- **`pyproject.toml`** — `per-file-ignores` for `scripts/*` and `launcher.py` (C901/PLR0912/PLR0913/PLR0915 complexity tolerated in utility scripts)
- **Quality enforcement** — `scripts/check_code_size.py` (ratchet), `tests/test_code_quality.py` (3 structural quality tests), CLAUDE.md rules 13-17 (max file/function size, max args, complexity, SRP)

### Bug fixes (backported from `main`)

- **Post-sync filter auto-invalidation** (`src/app/filters_render.py`) — `_filters_db_key_{player}` replaces the write-once `_filters_loaded_*` boolean; filters now reset automatically when the DB changes (sync, CLI, backfill, profile change)
- **Post-sync citation computation** (`src/data/citations_backfill.py`) — incremental module called by `DuckDBSyncEngine` after each sync; newly inserted matches get their citations immediately
- **SyncLock wired into the UI** (`src/ui/sync.py`) — `SyncLock(timeout=0)` protects against concurrent inter-process syncs; `SyncAlreadyRunning` is surfaced cleanly to the user, and DuckDB WAL is flushed before `end_sync_mode()`
- **Process-level Tailscale guard** (`src/utils/tailscale.py`) — module-level `threading.Event` replaces per-session `st.session_state`; `ensure_funnel_started_once()` guarantees only one startup and one Discord notification per Python process
- **False Discord webhook alert** (`src/utils/startup_check.py`) — skips the check when Doppler is active; loads `.env.local` before validation
- **Missing `_PERF_SCORE_AVAILABLE`** (`src/data/sync/_performance.py`) — module-level variable missing after the `engine.py` split into mixins; added `try/except ImportError` guard with `_PERF_SCORE_AVAILABLE = True/False`; fixes `F821 Undefined name` and runtime `NameError`
- **Fragile NaN check** (`src/ui/pages/match_view.py`) — replaces floating-point NaN idiom `x == x` with `x is not None`
- **i18n** (`src/ui/translations.py`, `src/ui/i18n/widgets.py`) — restored 2 truncated `PAIR_FR` keys, removed duplicate `tm_session_trend`, cleaned 343 redundant entries (399 → 56 useful entries)
- **Per-player backfill detection** (`scripts/backfill/detection.py`) — the 6 per-player flags (`medals`, `personal_scores`, `performance_scores`, `accuracy`, `shots`, `enemy_mmr`) now check the current player's real data instead of the global `backfill_completed` bitmask; fixes a bug where the first synced player masked matches for other players; new `_player_done_guard()` function; 15 new multi-player tests + 9 adapted tests

---

## [5.3.0] - 2026-02-28

### Added

- **LUSR (LevelUp Skill Rank) — TrueSkill 2 per-group rating system** (`src/analysis/`)
  - `skill_rating_config.py`: TrueSkill 2 constants, Bronze→Onyx I-VI tiers, 5-component composite score
  - `playlist_groups.py`: 6 isolated Halo Infinite groups (ranked 1.00, arena 0.80, tactical 0.70, btb 0.60, social 0.40, fun 0.15) with detection by `pair_name` prefix or `playlist_name`
  - `skill_rating.py`: full algorithm — `PlayerState` per group, `compute_composite_score()`, `trueskill_update()`, `compute_enemy_strength()`, per-group inactivity, sequential `compute_skill_ratings_batch()`
  - `skill_rating_calibration.py`: COMPOSITE_WEIGHTS calibration module by comparison with `team_mmr` API (random grid search, MAE or Pearson correlation metric)
  - 68 unit tests covering the algorithm, groups, inactivity, tiers, and calibration

- **Per-group LUSR: independent TrueSkill state per context**
  - `existing_states: dict[str, PlayerState]` replaces `existing_state: PlayerState` — a ranked match no longer affects the arena rating
  - `states.setdefault(group, PlayerState())` creates a state on the first match of each group
  - Inactivity, accuracy history, and σ decay are now per-group

- **LUSR/CSR Backfill** (`scripts/backfill_data.py`, `scripts/backfill/`)
  - `--lusr` / `--force-lusr`: local LUSR computation from `shared.match_participants` (sequential, incremental)
  - `--csr` / `--force-csr`: CSR retrieval from the Halo API for ranked matches
  - `compute_lusr_for_player()` in `strategies.py`: UPSERT into `match_skill_rank` with `rating_delta`, tier, and tier_label
  - `match_skill_rank` table auto-created by `ensure_match_skill_rank_table()` in `migrations.py`
  - Backfill bits: `lusr = 1 << 16` (65536), `csr = 1 << 17` (131072) in `BACKFILL_FLAGS`

- **SyncScope flags**: `lusr`, `force_lusr`, `csr`, `force_csr` in `src/data/sync/scope.py`

- **CSR data model** (`src/data/sync/models.py`, `src/data/sync/transformers.py`)
  - `SkillParticipantUpdate` extended: `pre_match_csr`, `post_match_csr`, `csr_tier`, `csr_sub_tier`
  - `RankRecap.PreMatchCsr` / `PostMatchCsr` extraction in `transform_all_skill_stats()`

- **LUSR visualization** (`src/visualization/timeseries_combat.py`)
  - `plot_lusr_timeseries()`: semi-transparent tier zones, `rating ± deviation` confidence band, 20-match smoothed trend

- **UI — Career page and Match View** (`src/ui/pages/`)
  - `career.py`: visual cards per group (90px centered rank image, LUSR/CSR badge, ▲/▼ delta) + group selector (`st.selectbox`) for the progression graph — replaces the expander table and tabs
  - `match_view.py`: ��� Rank tab with rank badge, colored progress bar, green/red delta

- **Calibration CLI**
  - `python -m src.analysis.skill_rating_calibration --player <GT> [--n-samples 300] [--metric corr]`
  - Grid search over the weight simplex (uniform Dirichlet distribution, reproducible seed)
  - Displays optimal weights ready to copy into `skill_rating_config.py`

- **Post-sync/backfill Discord notifications** (`src/utils/discord_notifier.py`)
  - New failsafe module — no external dependencies (stdlib `urllib.request` only)
  - Sends a Rich Embed to Discord at the end of each `sync.py` and `backfill_data.py` run
  - Embed content: operation, start/end time, total duration, number of players and matches processed
  - Per player: synced matches (or backfill-processed), data completeness (medals + events), last match (map, mode, KDA, result, playlist)
  - Bar color: green ✅ (all OK), yellow ⚠️ (incomplete data), red ❌ (error)
  - `--no-discord` flag on `sync.py` and `backfill_data.py` to disable on demand
  - `notify_operation_done()`: public entrypoint — `disabled=True` short-circuits immediately
  - `fetch_last_match_info(xuid)`: SQL on `shared_matches.duckdb` (JOIN `match_registry` + `match_participants`)
  - `count_new_matches(xuid, gamertag, since)`: counts matches with `first_sync_at >= since`
  - `count_matches_missing_data(xuid)`: counts matches with `medals_loaded=FALSE OR events_loaded=FALSE`

- **Secure Discord webhook configuration**
  - `discord_notifications_enabled: false` toggle in `app_settings.json` (no secrets in this file)
  - Webhook URL read from `DISCORD_WEBHOOK_URL` in `.env.local` (gitignored) via `_load_dotenv_if_present()`
  - Backwards-compatible fallback on the `discord_webhook_url` key in `app_settings.json`
  - Documented section in `.env.local.example`

- **Full FR/EN internationalization (i18n)** (`src/ui/i18n/`)
  - Dedicated i18n package with specialized modules: `common.py`, `pages.py`, `widgets.py`, `viz.py`, `cli.py`
  - Functions: `t(key, lang=None)` (Streamlit UI), `viz_t(key, lang)` (Plotly), `discord_t(key, **kwargs)` (Discord), `ct(key, **kwargs)` (CLI/scripts)
  - Language stored in `st.session_state["lang"]` (Streamlit) or `LEVELUP_LANG` env variable (scripts)
  - ������/������ language selector in the sidebar (`_render_lang_selector()` in `src/app/sidebar.py`)
  - Three fields in `AppSettings`: `lang`, `discord_lang`, `cli_lang` (default `"fr"`)
  - `src/ui/translations.py` bilingual: `translate_playlist_name(name, lang)` and `translate_pair_name(name, lang)` — preserves `" on Map"` grouping and Halo prefixes (Arena, BTB, Ranked)
  - `src/analysis/mode_categories.py`: bilingual `normalize_pair_name_to_mode_ui(pair_name, lang)`
  - `src/utils/discord_notifier.py` fully bilingual: `_format_player_field`, `build_embed_payload`, outcomes (���/���/⚖️/���), KDA (`{k}K / {d}D / {a}A` vs `{k}F / {d}D / {a}A`), operation labels, footer
  - `src/visualization/distributions_outcomes.py` bilingual: Wins/Losses/Ties/Unfinished traces, time buckets (match/hour/day/week/month), win rate heatmap (EN/FR days), `plot_matches_at_top_by_week` (Others/Top Rate)
  - `src/visualization/antagonist_charts.py` bilingual: `plot_duel_history` translates Win/Loss/Tie in duel annotation
  - `src/ui/pages/win_loss.py`: all viz calls pass `lang=get_lang()`

### Changed

- **Sessions filter — Solo / Squad redesign** (`src/app/filters_render.py`, `src/ui/cache_filters.py`, `src/ui/filter_state.py`, `src/ui/pages/teammates.py`)
  - Sidebar Sessions section split into two mutually exclusive subsections: **"En solo"** (sessions where no selected friend was present) and **"Mon escouade"** (sessions where at least one friend was present)
  - Replaces the "Dernière session en trio" single button with two full subsections, each with Last/Previous buttons and a selectbox
  - Classification is dynamic: friend XUIDs come from the Teammates multiselect (max 3), matched against `teammates_signature` via vectorized Polars `str.contains` — O(k×N) in C/SIMD, no Python loops on match rows
  - Friend selection persisted in `FilterPreferences` (`friends_selected_labels`, `picked_solo_session_label`, `picked_squad_session_label`)
  - Shadow keys added for Streamlit 1.54+ widget-state persistence across navigation
  - `teammates_signature` propagated through all return paths of `cached_compute_sessions_db`
  - 7 new i18n keys in `widgets.py` (`filter_solo_title`, `filter_squad_title`, `filter_last_carnage`, `filter_prev_carnage`, `filter_solo_session_label`, `filter_squad_session_label`, `filter_squad_no_friends`)
  - `apply_filters` now uses `filter_state.base_s_ui` directly (single source of truth) instead of re-calling `cached_compute_sessions_db`, with an empty-intersection guard

- **LUSR algorithm — Elo-style update (`K_ELO = 32`)** replaces TrueSkill draw zone
  - Root cause of divergence: `v_draw(t > 0)` gave positive deltas even at composite=0.5, creating infinite drift when `state.mu > INITIAL_MU` or when the player over-fragged their `kills_expected`
  - New mu formula: `delta_mu = K_ELO × (composite − 0.5) × weight_factor` → exact ZERO at composite=0.5, independent of `mu_opp`
  - Sigma retains TrueSkill reduction evaluated at t=0 (symmetric, `mu_opp` only influences `c²`)
  - Result: stabilized ratings — SpartanA (Diamond V) → Platinum IV BTB / Platinum VI Arena / Diamond IV Ranked, SpartanB/SpartanC → Gold II-IV depending on mode
- **Composite score calibrated on 1765 matches** (SpartanA, SpartanC, SpartanB — Silver → Diamond)
  - Target signal: `individual_mmr = team_mmr × (kills_expected / ke_avg_match)`
  - Weighting by `nb_matches × MAE_improvement`: SpartanA 36.7%, SpartanC 40.0%, SpartanB 13.3%
  - New weights: kills_vs_expected=31%, deaths_vs_expected=28%, damage_efficiency=23%, accuracy_delta=13%, win_factor=5%
- **damage_efficiency bias elimination**: `PlayerState.damage_eff_history` per-group — the damage component now uses a delta vs personal history (like accuracy_delta) instead of the raw value
- **mu_opp anchored on `state.mu`**: `compute_enemy_strength` uses `player_mu=state.mu` as the base estimate for opponents (matchmaking pairs players of similar level)
- **Reduced inactivity params**: `INACTIVITY_SIGMA_PER_DAY` 3.5→1.0, `MAX_INACTIVITY_DAYS` 30→14 — avoids ±200 pt swings after a long break
- **Reduced CSR seed sigma**: `PlayerState.from_csr()` starts at `sigma=MIN_SIGMA` (60) instead of `INITIAL_SIGMA × 0.6` (210) — CSR is a strong anchor; starting in a stable state avoids initial volatility

- **Career page — Estimated pre-sync XP curve** (`src/ui/pages/career.py`)
  - Dotted purple trace retroactively estimating XP for the ~561 matches played before the first sync
  - Average XP/match = `first_xp / n_pre_sync_matches` — curve starts near 0 at the oldest match and connects seamlessly to the first real snapshot
  - Visually distinct from the real XP trace (purple `#CE93D8` dotted line)

- **Career page — Projection curves to Hero rank** (`src/ui/pages/career.py`)
  - **Standard projection** (orange dashed): extrapolates from the current active XP/day rate, excluding inactivity gaps > 14 days
  - **Optimistic projection** (green dash-dot): adds weekly challenge XP (950 XP/week = 4×50 + 3×100 + 3×150) plus daily challenge XP (500 XP/day), all with ×2 XP boost — total +4 450 XP/week from challenges
  - Both curves hidden by default — click the legend to reveal them
  - Gold horizontal line at the Hero threshold (9,319,350 XP)
  - Projection capped at 10 years to avoid infinite charts
  - Legend moved to the bottom of the chart (centered)
  - 23 unit tests in `tests/test_career_xp_projection.py`

### Fixed

- **20 pre-existing tests fixed** following the v5.1 migration (shared architecture)
  - Group A (assertions/fixtures): `test_backfill_bitmask`, `test_backfill_detection`, `test_xuid_resolution_regression` (×2), `test_post_refactor_perf_contracts`, `test_data_services_contracts`, `test_media_components_sprint4` (×2), `test_media_improvements`, `test_legacy_free_global`
  - Group B (v4→v5 mocks): `test_lazy_loading` (×5 — `_get_match_source` v5.1), `test_data_contract_sessions` (v5 shared + player_match_enrichment fixture rewrite)
  - Group C (source + mocks): `test_sessions_integration` (production DB fallback hidden by `__file__` patch), `test_duckdb_repository_schema_contract` (`xuid/gamertag` schema in shared fixture), `test_teammates_impact_tab` (×2 — mock `_ensure_shared_attached` + `_load_highlight_events`)

---

## [5.2.0] - 2026-02-20

### Added

- **v5.2 Filters — Intent-based persistence** (`src/ui/filter_state.py`)
  - `FilterPreferences`: dataclass saved as JSON per player
  - Persisted modes: `playlist_mode`, `mode_mode`, `map_mode` (`"exclude"` / `"include"`)
  - Exclusion lists: `excluded_playlists`, `excluded_modes`, `excluded_maps`
  - `_detect_filter_mode()`: 70/30 heuristic — if > 70% of options are checked, use "exclude" mode; otherwise "include"
  - `reconcile_filter_prefs()`: auto-reconciliation when new options appear — new playlists/modes/maps included by default without resetting existing preferences
  - 45 unit tests in `tests/test_filter_state.py`

- **v5.2 Filters — "Experience Type" selector** (`src/app/filters_render.py`)
  - Static pre-filter: "Unranked PVP", "Ranked PVP", "PVE (Firefight)" enabling the `is_firefight` filter
  - Correct cascade deletion: modes/maps computed from full `dropdown_base` (before playlist filter)
  - `FilterPreferences` integrated into cascade filter rendering

- **PvE / Firefight v5.2 stats — Dedicated `shared_pve.duckdb` database**
  - New `data/warehouse/shared_pve.duckdb` database separate from `shared_matches.duckdb` (avoids NULL columns on 90%+ of PvP matches)
  - `pve_match_stats` table: per-player per-match Firefight stats — waves, boss kills, kills by enemy type (Banished: Grunt, Elite, Jackal, Brute, Hunter, Skimmer; Forerunner: Crawler, Soldier, Knight, Warden)
  - `ensure_pve_schema()` in `src/data/sync/migrations.py` — idempotent schema creation
  - `PVE_SCHEMA_DDL`: full DDL + `idx_pve_xuid` + `idx_pve_match_id` indexes

- **PvE stats — Python models** (`src/data/sync/models.py`)
  - `PveMatchStatsRow`: dataclass with 20 columns (waves, boss, enemy by type, pve_bits)

- **PvE stats — Transformer** (`src/data/sync/transformers.py`)
  - `extract_pve_stats(match_json)`: extraction for all players of a Firefight match
  - `_find_pve_stats_dict(player)`: recursive search for the PvE block (EliminationStats / PveStats / FirefightStats / key detection)
  - `_extract_enemy_kills_by_type(pve_dict)`: dual-structure support (direct `GruntKills` fields + `EnemyKillsByType` sub-dict)
  - `_is_firefight_match()` enhanced: 3 criteria — `GameVariantCategory` (IDs 41, 42 validated on real API JSON), `UgcGameVariant.PublicName`, `Playlist.PublicName` (firefight/baptême/survive)

- **PvE stats — Insert pipeline** (`src/data/sync/batch_insert.py`)
  - `batch_insert_pve_stats(conn, rows)`: batch insert with `INSERT OR REPLACE`

- **PvE stats — Bitmask** (`src/data/sync/constants.py`)
  - `PveBits(IntFlag)`: granular bitmask for `pve_match_stats.pve_bits` — TOTAL_KILLS, BOSS_KILLS, GRUNT, ELITE, JACKAL, BRUTE, HUNTER, SKIMMER, SENTINEL, MARINE + ALL_ENEMIES, FULL_PVE combinations
  - `MatchBits.PVE_STATS = 1 << 20`: global guard in `match_registry.backfill_completed` — set for every processed match (Firefight or not) to avoid infinite re-detection

- **PvE stats — Sync Engine** (`src/data/sync/engine.py`)
  - `_pve_connection`: lazy-init connection to `shared_pve.duckdb`
  - `_pve_db_lock`: dedicated asyncio lock
  - `_get_pve_connection()`: lazy init + `ensure_pve_schema` on first access
  - `_try_insert_pve_stats(stats_json, match_id, shared_conn)`: extraction + insert + set `MatchBits.PVE_STATS` bit — called in `_process_new_match` and `_process_known_match`

- **PvE stats — SyncScope** (`src/data/sync/scope.py`)
  - `pve_stats: bool` and `force_pve_stats: bool` fields in `SyncScope`
  - Registered in `_FORCE_MAP` and `_ALL_DATA_FIELDS`

- **PvE stats — Backfill detection** (`scripts/backfill/detection.py`)
  - Double guard: `mr.is_firefight = TRUE AND (COALESCE(mr.backfill_completed, 0) & PVE_STATS) = 0`
  - `force_pve_stats`: ignores the guard, returns all Firefight matches
  - `MatchBits.PVE_STATS` added to `compute_bits_needed_from_scope`

- **PvE stats — Backfill CLI** (`scripts/backfill/cli.py`)
  - `--pve-stats` and `--force-pve-stats` arguments

- **PvE stats — Backfill orchestrator** (`scripts/backfill/orchestrator.py`)
  - `_backfill_pve_for_match()`: opens `shared_pve.duckdb`, `ensure_pve_schema`, `batch_insert_pve_stats`, sets guard bit in `match_registry`
  - `pve_stats_inserted` counter in `_empty_result()`

- **PvE citations** (`src/analysis/citations/engine.py`)
  - `load_match_pve_stats(match_id)`: reads from `shared_pve.duckdb`
  - PvE stats merged into `match_stats` before citation computation
  - `pve_stat` recognized as `mapping_type` (handled identically to `stat`)

- **81 new tests**:
  - `tests/test_filter_state.py`: 45 tests — `FilterPreferences`, `_detect_filter_mode()`, `reconcile_filter_prefs()`, save/load
  - `tests/test_pve_transformers.py`: 36 tests — `_is_firefight_match()`, `_extract_enemy_kills_by_type()`, `extract_pve_stats()`, DuckDB schema, batch insert, `PveMatchStatsRow`, `PveBits`, `SyncScope.pve_stats`

- **"Last match" scoreboard** (`src/ui/pages/match_view_players.py`, `src/data/repositories/_roster_loader.py`)
  - `load_match_scoreboard(match_id)`: DuckDB query joining `match_participants` + `xuid_aliases` + `medals_earned` sub-query (Perfect Kill, ID 1512363953). 20 fields per player, sorted by `(team_id, rank)`.
  - `render_match_scoreboard()`: per-team HTML table with 18 columns — Gamertag, Rank, Score, Kills, Deaths, Assists, KDA, Killing Spree, Headshots, Perfect Kills, Shots, Shots Hit, Accuracy, Melee, Power Weapons, Damage Dealt, Damage Taken, Avg Lifetime
  - Handles N teams + players without `team_id` (NULL → separate group at the end)
  - Okabe-Ito color headers: blue `#0072B2` for the player's team, vermillion `#D55E00` for opponents
  - Player row highlighted (cyan `#00e5ff`)
  - Gamertag resolution via `load_match_gamertags_fn` (same pipeline as the former roster)
  - CSS `.os-scoreboard` / `.os-sb-*` with column wrapping (`max-width: 80px`, `word-break`)
  - Replaces the removed "Players" (roster) section

- **Per-player tokens for player-gated endpoints** (`src/data/sync/api_client.py`, `src/ui/profile_api_tokens.py`)
  - `SPNKR_OAUTH_REFRESH_TOKEN_<NORMALIZED_GT>` in `.env.local` per player (e.g.: `_SPARTANC`, `_MON_GT_2`)
  - Normalization: `re.sub(r"[^A-Za-z0-9]", "_", gt.strip()).upper()`
  - `get_tokens_for_player(gamertag)`: async, returns `Tokens | None` — skip + warning if absent (no global fallback on restricted endpoint)
  - `get_player_token_env_key(gamertag)`: returns the normalized env key
  - `profile_api_tokens.get_tokens()` extended: optional `gamertag` param — priority player token > global token (natural fallback for public endpoints)
  - `profile_api.py`, `get_profile_appearance()`: `gamertag` param propagated to SPNKr fetch
  - `load_profile_api()`: derives the gamertag from the DB and passes it to `get_profile_appearance()` — fixes adornment/career rank for players who do not own the global token

- **Player-gated Career Rank sync** (`src/data/sync/engine.py`)
  - `sync_career_rank()` uses `get_tokens_for_player()` — silent skip + warning if absent
  - Persists `spartan_id` in `career_progression` (column added via `add_spartan_id_to_career_progression()` migration)
  - `CareerRankRow.spartan_id` in `src/data/sync/models.py`

- **Spartan ID in the hero banner** (`src/ui/styles.py`, `src/app/main_helpers.py`)
  - `get_hero_html()`: new `spartan_id` parameter — displayed in the career-rank section under the rank label (`.career-rank__spartan-id`)
  - `render_profile_hero()`: loads `spartan_id` from `career_progression` (DB, source of truth) and passes it to the hero HTML
  - CSS `.career-rank__spartan-id`: compact, semi-transparent, letter-spaced style

- **32 new tests** (`tests/test_player_tokens.py`)

### Changed

- **Colorblind accessibility — Okabe-Ito palette migration** (`src/visualization/`)
  - 7 visualization files updated: `antagonist_charts.py`, `performance.py`, `objective_charts.py`, `participation_charts.py`, `team_dominance_timeline.py`, `match_impact_timeline.py`, `friends_impact_heatmap.py`
  - Replaced saturated neon red/green pairs (incompatible with deuteranopia and protanopia) with the **Okabe-Ito** palette (Wong 2011), the international reference for accessible charts
  - Main mappings: neon green `#00ff00` → blue-green `#009E73` · red `#ff4444` → vermillion `#D55E00` · magenta `#ff66ff` → mauve pink `#CC79A7` · team colors `#3DFFB5`/`#FF4D6D` → blue `#0072B2`/vermillion `#D55E00`
  - Each palette documented with previous hex values and justification in a comment block

- **`_is_firefight_match()`** — Merging of the two duplicated definitions into a single unified function covering all 3 criteria (GameVariantCategory + UgcGameVariant.PublicName + Playlist.PublicName)

### Deprecated

- **`display_name_from_xuid()` and `get_xuid_aliases()`** (`src/ui/aliases.py`) — Marked `.. deprecated::`. Use `load_match_gamertags_fn` for match context. Kept for scripts/migration/export.

### Removed

- **"Players" (roster) section** from the Last Match page — Replaced by the scoreboard. `render_roster_section` is no longer called from `match_view.py`.

### Fixed

- **`_is_firefight_match()` duplication** — Two definitions coexisted in `transformers.py`. The second silently overrode the first, making detection via `UgcGameVariant` inoperative. Merged into a single complete definition.

---

## [5.1.0] - 2026-02-17

### Added

- **`src/data/sync/scope.py` module** — **SyncScope** dataclass centralizing flags
  - Replaces 30+ boolean kwargs copied across 6 files (cli → backfill_data → orchestrator → detection → API)
  - `SyncScope.from_cli_args(args)`: construction from argparse
  - `SyncScope.make_all()`: factory for `--all-data`
  - `resolve()`: automatic implications (`all_data` → fields, `force_X` → X)
  - Properties: `has_any_option()`, `needs_api`, `needs_local_only`, `requested_types`
  - Registries: `_ALL_DATA_FIELDS`, `_FORCE_MAP`, `_REQUESTED_TYPE_MAP`
  - 98 unit tests in `tests/test_sync_scope.py`
  - **Add a new type**: 1 field in SyncScope + 1 CLI arg + business logic implementation
- **`src/ui/streamlit_modern.py` module** — Modern Streamlit compatibility wrappers
  - `fragment_if_available`: graceful-degradation decorator for `@st.fragment`
  - `PLOTLY_CLEAN_CONFIG`: Plotly config without toolbar
  - `plotly_chart`: wrapper with clean config by default
  - `HAS_FRAGMENT`, `HAS_NAVIGATION`: version detection
- **`src/ui/vectorize_helpers.py` module** — Vectorized replacement for `map_elements()`
  - `build_mapping()`: pre-computed dict mapping on distinct values
  - `vectorized_apply()`: vectorized apply via `replace_strict()`
  - `safe_int_format()`, `format_score_pair()`: reusable Polars expressions
- **`get_shared_matches_path()` helpers** — Centralized functions in `src/utils/paths.py`
  - `get_shared_matches_path()`: absolute path to `shared_matches.duckdb`
  - `get_shared_matches_path_from_player()`: deduction from player path
- **`cleanup_legacy_tables.py` script** — Obsolete table removal
  - 9 tables removed: `match_stats`, `medals_earned`, `highlight_events`, `player_stats`, `xuid_aliases`, + 4 `mv_*` views
  - Options: `--dry-run`, `--backup`, `--all`
  - Automatic backups in `backups/pre_cleanup/`
- **`mv_player_matches` materialized view** — v5.1 performance optimization
  - Pre-computed joins on match_participants + match_registry + metadata
  - SQL parsing reduced from 170→10 lines per query
  - Performance gain: -70% SQL parsing
- **Streamlit Repository Cache** — `get_cached_repository_st()` with `@st.cache_resource(ttl=3600)`
  - Persistent DB connection between UI pages
  - Gain: 80ms→<20ms connection
- **DuckDB Performance Indexes** — 16+ indexes created on 9 tables
  - Composite indexes `(xuid, match_id)`, `(match_id, xuid)`
  - Sorted indexes on `start_time`
- **Metadata schema cache** — `_has_column()` and `_has_shared_mp_column()` cached
  - Avoids repeated `information_schema` queries
- **LEGACY banner migration scripts** — 5 scripts flagged + README.md
  - Clear "OUT OF SERVICE POST-V5.1" banner
  - Documentation in `scripts/migration/README.md`

### Changed

- **`backfill_data.py` refactored** — `main()` uses `SyncScope.from_cli_args()` (−90 lines)
  - No longer need to copy 30+ `args.X` twice for `--all` and `--player`
- **`orchestrator.py` refactored** — `backfill_player_data`, `backfill_all_players`, `_backfill_with_api` accept `scope=SyncScope`
  - Old kwargs preserved (marked `LEGACY`) for backward compatibility
  - `requested_types` built via `scope.requested_types` instead of 16 `if/append`
- **`detection.py` refactored** — `find_matches_missing_data` accepts `scope=SyncScope`
  - Old kwargs preserved (marked `LEGACY`) for backward compatibility
- **Bumped Streamlit ≥1.37.0** — Required for `@st.fragment` and future `st.navigation` migration
- **Plotly `config={"displayModeBar": False}`** — Applied to 69 `st.plotly_chart` calls (15 files)
  - Removes Plotly toolbar for a cleaner UI
- **`@fragment_if_available`** — Decorator applied to 5 multi-chart pages
  - timeseries, session_compare, win_loss, objective_analysis, career
  - Reduces re-renders to the fragment only on filter interactions
- **`match_history.py` modernized** — Replaced custom HTML with `st.dataframe` + `column_config`
  - Dead code removed: `_format_score_label`, `_fmt`, `_fmt_mmr_int`
  - Native Streamlit virtualization for wide tables
- **`st.navigation` lazy loading** — 11 page closures in `streamlit_app.py`
  - `build_navigation()` + `render_page_selector_nav()` in `page_router.py`
  - Legacy fallback `dispatch_page()` for Streamlit < 1.36
  - Only visited pages are imported → -60% initial memory
- **Centralized `duckdb_read_only()`** — Context manager in `src/utils/db.py`
  - 7 files migrated (career, cache_loaders, cache_filters, media_library, multiplayer, data_loader)
  - Direct `duckdb.connect` calls: 14 → 4 (remaining: sync engine, legitimate writes)
- **Reduced `st.rerun()`** — 32 → 14 in `src/`
  - `checkbox_filter.py`: 16 reruns → 0 via `on_click`/`on_change` callbacks
  - Trio button filters: `on_click=_apply_trio_filter`
- **`unsafe_allow_html` hardening** — `html.escape()` on dynamic data
  - `kpi.py` and `performance.py`: XSS protection
  - `sidebar.py` brand: HTML → `st.header()` + `st.divider()`
- **Modernization regression tests** — 30 tests in `test_8ter_modernisation.py`
  - Coverage: staticPlot, fragments, st.navigation, duckdb_read_only, st.rerun, html.escape
- **Complete `map_elements()` eradication** — 28 occurrences replaced in 15 files
  - Replaced with `build_mapping()` + `replace_strict()` or native Polars expressions
  - Files: filters.py, filters_render.py, win_loss.py, last_match.py, stats.py,
    match_view_charts.py, media_library.py, teammates_helpers.py, session_compare.py,
    session_compare_charts.py, duckdb_analytics.py, match_view.py, citations.py,
    teammates_service.py, media_indexer.py
- **`xuid_aliases` migration → `shared_matches.duckdb`** — Single centralized source
  - 9 files migrated to read from `shared.xuid_aliases` (13,955 rows)
  - Local `stats.duckdb` fallbacks removed
  - Files: `aliases.py`, `xuid.py`, `multiplayer.py`, `cache_loaders.py`, `engine.py`, `_roster_loader.py`, `sessions_backfill.py`, `sync.py`, `resolve_missing_gamertags.py`
- **`_get_match_source()`** now returns a 3-tuple `(source_sql, params, uses_mv)`
  - Enables skipping redundant joins in v5.1 mode
- **8+ cache_loaders functions** migrated to `get_cached_repository_st()`
  - Redundant new connections removed
- **metadata/MMR joins** skipped in v5.1 mode when `uses_mv=True`
  - RC3/RC4: -3 LEFT JOINs on the critical path

### Fixed

- **Citations tab showed 159 citations instead of 45** — Filtering by `citation_mappings.enabled` re-enabled
  - The `halo5_commendations_fr.json` JSON contains 159 citations (weapons, Spartan Companies, etc.)
  - Filtering had been removed, displaying all citations including those without mapping
  - Fix: JSON items are now filtered by normalized names of enabled citations via `CitationEngine.load_mappings()`
  - File: `src/ui/commendations.py`

### Removed

- **Legacy player DB tables** — 9 tables per player, data centralized
  - `match_stats`, `medals_earned`, `highlight_events`, `player_stats`, `xuid_aliases`
  - Obsolete views: `mv_match_stats_with_context`, `mv_recent_matches`, `mv_team_stats`, `mv_opponent_stats`
  - 38,528 rows freed across 4 players
- **SQLite runtime references** — 0 `import sqlite3` in `src/`
- **`metadata.db` references** — Everything migrated to `metadata.duckdb`
- **Deprecated `attach_sqlite` method** — Removed from duckdb_engine.py

### Performance

| Metric | v5.0 | v5.1 | Gain |
|--------|------|------|------|
| DB connection | 80ms | <20ms | **-75%** |
| load_matches(100) | 200ms | <80ms | **-60%** |
| First UI page | 1500ms | <800ms | **-47%** |
| SQL parsing/query | 170 lines | 10 lines | **-94%** |

---

## [5.0.0] - 2026-02-15

### Added

- **shared_matches.duckdb architecture** — Shared database centralizing matches for all players
  - 6 tables: `match_registry`, `match_participants`, `highlight_events`, `medals_earned`, `xuid_aliases`, `highlight_events_id_seq` sequence
  - 14 optimized indexes (match_id, xuid, start_time, composites)
  - Full DDL schema: `scripts/migration/schema_v5.sql`
  - Documentation: `docs/SHARED_MATCHES_SCHEMA.md`
- **v4 → v5 Migration** — Incremental per-player migration scripts
  - `scripts/migration/create_shared_matches_db.py`: shared DB creation
  - `scripts/migration/migrate_player_to_shared.py`: per-player migration
  - Result: 1289 matches migrated, 285 shared (22.1%), 0 orphans
- **Shared match detection in Sync Engine** — Lightweight sync for already-known matches
  - `_process_known_match()`: personal enrichment only (saves 1-2 API calls/match)
  - `_process_new_match()`: full sync to shared (registry + participants + events + medals)
  - `extract_all_medals()`: medal extraction for ALL players in the match
  - `extract_match_registry_data()`: common match data extraction
- **Multi-DB ATTACH in DuckDBRepository** — Transparent reads from `shared_matches.duckdb`
  - `shared_db_path` auto-detected or configurable
  - Native queries on `shared.match_participants`, `shared.match_registry`, `shared.medals_earned`
  - Propagation in the repository factory
- **`_get_match_source()` sub-query** — Abstraction allowing all UI pages to read from shared without modification
- **v5 Sync API optimizations**
  - Parallelized skill + events API calls (`asyncio.gather`)
  - DB insert batching (commit every 10 matches)
  - Performance scores computed in batch post-sync
  - Optimized rate limit (10 req/s, parallel_matches=5)
- **DuckDB-first citations** — New per-match stored citations system
  - `CitationEngine`: computation and SQL aggregation engine
  - `citation_mappings` table in `metadata.duckdb`: 14 rules (8 existing + 6 reintegrated)
  - `match_citations` table in each player's `stats.duckdb`
  - Backfill CLI: `--citations` / `--force-citations` in `scripts/backfill_data.py`
  - 6 reintegrated objective citations: Flag Defender, Got Your Back!, Flag Stalker, Stake a Claim, Charge!, Forced Annexation
  - `enabled` column in `citation_mappings` for disabling without deletion
  - V5 (shared_matches) support in `CitationEngine` with V4 fallback
  - Documentation: `docs/CITATIONS.md`
- **MockStreamlit test framework** — `MockStreamlit` fixture in `conftest.py` for headless UI page testing
- **+946 tests** added (S1→S7ter) — total 2768 passed, 0 failed, 38 skipped
- **Post-migration cleanup script** — `scripts/cleanup_player_dbs_v5.py`
  - Removes redundant tables from player DBs after v5 migration (match_stats, match_participants, highlight_events, medals_earned)
  - `--dry-run` mode for simulation without modification
  - Optional backup before cleanup
  - Automatic `shared_matches.duckdb` existence validation
  - Automatic VACUUM for disk space recovery (-85% average size reduction)
  - Documentation: `docs/CLEANUP_V5.md`
- **Documentation**: `docs/SHARED_MATCHES_SCHEMA.md`, `docs/SYNC_OPTIMIZATIONS_V5.md`, `docs/TESTING_V5.md`, `docs/ARCHITECTURE_V5.md`, `docs/MIGRATION_V4_TO_V5.md`, `docs/CLEANUP_V5.md`

### Changed

- **`DuckDBSyncEngine`** refactored to write to `shared_matches.duckdb` (matches, participants, events, medals)
- **`DuckDBRepository`** refactored with ATTACH `shared_matches.duckdb` in read-only
  - `load_match_participants()` → reads from `shared.match_participants`
  - `load_highlight_events()` → reads from `shared.highlight_events`
  - `load_medals_for_match()` → reads from `shared.medals_earned`
  - `load_matches()` → JOIN `shared.match_participants` + `shared.match_registry` + `player_match_enrichment`
- **All UI pages** use `_get_match_source()` instead of `match_stats` directly
- **`render_h5g_commendations_section()`** uses `CitationEngine` (SQL aggregation, ~90% faster)
- **`render_citations_page()`** simplified — no longer pre-aggregates medals/stats for citations
- **Citation filtering** driven by `citation_mappings.enabled` (no longer needs the exclusion JSON)
- **`pyproject.toml` version** bumped from 3.0.0 to 5.0.0
- **Project status**: Development Status 4-Beta → 5-Production/Stable

### Removed

- **v4 compatibility VIEWs** removed (`scripts/migration/remove_compat_views.py`)
- **Duplicated data** in player DBs: `match_participants`, `highlight_events`, `medals_earned` centralized in shared
- **`src/db/migrations.py` shim** — deprecated, removed in favor of `src.data.sync.migrations`
- `CUSTOM_CITATION_RULES` dict (old `commendations.py`)
- `_compute_custom_citation_value()` (slow iterations, replaced by SQL)
- `load_h5g_commendations_tracking_rules()` (replaced by `citation_mappings` DuckDB)
- `DEFAULT_H5G_TRACKING_ASSUMED_PATH` / `DEFAULT_H5G_TRACKING_UNMATCHED_PATH` constants
- Dependency on commendation tracking JSON files
- JSON exclusion logic in `render_h5g_commendations_section()`

### Fixed

- **Flaky Windows tests**: `tmp_dir` → `tmp_path` to avoid DuckDB `WinError 32` (file locking)
- **`lazy_loading` tests**: v4 mode forced for compatibility

### Performance

| Metric | v4 | v5 | Gain |
|--------|----|----|------|
| Storage (4 players) | 800 MB | 250 MB | **-69%** |
| DB size per player | 200 MB | 30 MB | **-85%** |
| API calls (sync 4 players) | 12,000 | 3,300 | **-72%** |
| Sync time (100 matches) | 45 min | 12 min | **-73%** |
| Time/match (shared) | 16s | 0.5s | **-97%** |
| Time/match (new) | 16s | 2-3s | **-81%** |
