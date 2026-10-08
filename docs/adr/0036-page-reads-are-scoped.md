# ADR 0036 — Page reads are scoped

**Status**: Accepted (2026-09-26)

**Branch**: `feat/perf-perimetre` (worktree `LevelUp-wt-perf-perimetre`, base `feat/v75`
`ea5682373`). The mechanisms named here shipped with the loading-performance campaign of
2026-09-23 (branch `feat/perf-chargements`, merged into `feat/v75` as `4693062d5`); this ADR
turns what that campaign established into named invariants.

**Relates to**: [ADR 0026](0026-append-only-art-eradication.md) (append-only tables are read
through their `_latest` views only; this ADR says how such a read stays proportional to its
scope), [ADR 0016](0016-shared-db-provider-b-swap.md) (taking the shared writer swaps the reader
out and empties its cache), [ADR 0013](0013-leased-writer-enforcement.md) (one writer per
database), [ADR 0024](0024-lusr-v2-trueskill2-with-counts.md) (LUSR v2, whose post-sync step I5
constrains), [ADR 0035](0035-player-directory-single-identity-key.md) (the xuid is the identity
key; a gamertag is a display value). Supersedes nothing.

Paths are relative to `apps/go-api/internal/` unless they start with `docs/` or `.ai/`. Every
figure comes from `.ai/V7.5/ETAT_DES_LIEUX_PERF_CHARGEMENTS_2026-09-23.md` (sections 1 and 6),
`.ai/V7.5/HANDOFF_PERF_CHARGEMENTS_2026-09-24.md` (section 2) or the lot journals of
`.ai/V7.5/PLAN_PERF_CHARGEMENTS_2026-09-23.md`.

---

## Context

### What was measured

Protocol, identical on the morning and the evening of 2026-09-23: local development server
(`air` behind the Vite proxy, HTTP/1.1, React StrictMode) on the real data, player `JGtm`
(1,160 matches), 16-core / 32 GB workstation, DuckDB at the production defaults (`threads=2`,
`memory_limit=512MB`). Server durations are the `duration_ms` of the HTTP middleware; client
durations run to the last API response of the page. SQL timings were taken on a read-only copy
of `shared_matches_v2.duckdb` with the same two settings. Production (VPS, 2 vCPU) has not been
measured yet.

| Page / gesture | Before (morning) | After (evening) |
|---|---|---|
| Squad, first visit (empty local storage) | 194 s; 7 `pages/teammates` requests, 4 of them cut to 502 by the 30 s write timeout and replayed | 2.6 s: light read 197 ms, then one heavy request of 1,256 ms, already on the right session |
| Squad, reload | 26.3 s | 1.6 s (light 133 ms + heavy 858 ms) |
| Squad, "previous session" (11 matches) | 8.2 s for an intermediate request with no match, then 26.8 s | 0.24 s (one request of 236 ms) |
| Synthesis, all periods | 6.2 s | 2.4 s (`synthesis` 2,098 ms) |
| Sessions | 6.1 s | 0.7 s (`sessions/detail` 284 ms) |
| Timeseries | 9.5 s | 0.9 s (`timeseries` 499 ms) |
| Career, encounters / rivals | 10.7 s / 10.1 s (intermediate measurement, same day) | 1,574 / 4,493 / 3,387 ms and 2,105 / 6,472 / 4,090 ms (three passes, outside sync) |
| Home | 4.2 s + 2.6 s | 2.1 s (`pages/home` 1,642 ms, 310 KB) |
| Shared calls (`/filters/resolve`, field mappings, `/bootstrap`) | 0.2 to 0.6 s per call, two or three per page | 1 to 8 ms after the first call of the process |
| Auto-sync cycle (LUSR post-sync) | 1,243 read-only / read-write swaps in 75 s | 11 swaps on the measured cycle |

SQL on the copy, before and after: Squad top teammates (Q29) 1.7-2.4 s to 21 ms; impact events
(Q32, 38 matches) 2.1-2.4 s to 11-17 ms; allied team (Q32b) 2.3 s to 6-7 ms; tactical kill
journal (`KillEvents`, 6 matches) 1.79 s to 59 ms; `MortsAvecContexte` 3.26 s to 80 ms;
`LoadWeaponRange` 1.32 s to 64 ms; Career encounters (Q26) 2.6-3.7 s to 0.8-1.3 s; rivals (Q27,
two reads) 6.6-7.0 s to 1.9-2.2 s; Q10 and Compare 2.3-2.6 s to 9-30 ms.

### Where the time went

Four shapes, measured on the copy at 2 threads / 512 MB:

1. **A name view evaluated whole on every join.** `v_gamertag_lookup` aggregates
   `match_participants`, two passes over `match_kill_events_latest` and two over
   `killer_victim_pairs` in a `FULL OUTER JOIN`; no filter is pushed into it.
   `SELECT count(*)` over it: 3.1-3.3 s; with `WHERE xuid = ?` on a single xuid: 3.0 s. Squad
   top teammates (Q29): 3.0-3.3 s with `LEFT JOIN v_gamertag_lookup`, 32 ms without. The Squad
   request evaluated the view six times. Sixteen threads and 8 GB did not help the joins (Q29
   2.4-3.1 s): the plan was the problem, not the parallelism.
2. **A `_latest` window computed over the whole table.** The `_latest` views keep the current
   version of a row with `QUALIFY ... OVER (PARTITION BY match_id ...)` (ADR 0026); counting
   `match_kill_events_latest` (3.95 M rows) costs 0.85 s. A scope passed as a subquery, a
   semi-join or a filter on another relation of the join left that window over the whole table:
   `KillEvents` took 1,794 / 1,867 / 1,901 ms for 6 / 30 / 1,158 matches. At page level the Squad
   request cost 8.2 s with no match, 26.3 s with 7, 26.8 s with 11 and 24.4 s with 38: the scope
   reduced nothing.
3. **The same read N times per request, and again at every request.** 155 to 170 SQL queries per
   Squad page; the impact events (Q32) read four times on the same matches; for three teammates,
   21 loads of a member's complete history; `/filters/resolve` two or three times per page, each
   one reloading the complete history, with no server cache.
4. **A sync that takes a writer to learn that nothing is new.** The LUSR v2 post-sync step took
   a shared writer every three candidates before checking its watermark: 1,243 read-only /
   read-write swaps between 10:32:33 and 10:33:48, writing nothing. Each swap closes the shared
   reader and empties the DuckDB cache (ADR 0016); an HTTP reader waits up to 3 s for it.

Two conditions hid all of this. Successful responses were logged at DEBUG only, below the
default file level: no duration of a successful request existed in the logs. And
`LEVELUP_DUCKDB_THREADS` / `LEVELUP_DUCKDB_MEMORY_LIMIT` were captured in package variables at
init, before `.env.local` is loaded, so a local override was silently ignored. On top of it, the
30 s server write timeout turned the slowest Squad responses into 502s that the client replayed,
tripling the work.

### What the campaign changed

Twelve lots and a closing lot (C4), one executor per lot, each with parity tests and mutations
played red; an adversarial review of the cumulative diff (1 P0, 3 P1, 12 P2) was fully addressed
or recorded. Each invariant below is one of those remedies, written down so that the next page
read does not replay the whole history.

## Decision

**A page read pays for its scope, not for the player's history, and it pays once.** Seven
invariants, each held today by the guardrail named with it; what each guardrail actually blocks,
and what it does not see, is in *Guardrails*.

### I1 — No page read evaluates `v_gamertag_lookup`

The view stays the canonical definition of a display name (its DDL has a single source,
`analysis/identity.go`), but a read on a request path never joins it. A read names its rows
through its own directory: `nommerLignes` and `annuaireDeLecture`
(`platform/duckdb/squad_repo_annuaire.go`) collect the xuids the read met, load once, on the
read's matches only, what each named level of the view gives them (`xuid_aliases`, participant
gamertag, then the kill-feed leg for the xuids still unnamed, built by the same generator as the
view, `gamertagKillFeedSQL`), and apply one cascade, `analysis.AnnuaireGamertags.Resolve`
(`analysis/identity_annuaire.go`): known bot, alias, participant, kill feed, masked label
(`analysis.MaskedXuidLabel`). An aggregated read (one row per player over all matches) uses the
player's history as its matches (`QMatchsDuJoueurTpl`).

Replacing a join must keep the names. Where the directory, restricted to the read's matches,
cannot see a name the view finds elsewhere, the difference is accepted only when it is named,
counted on the production copy and pinned by a test (Career: 8 player pairs out of 58,353, all
for one player, none on a served row; `TestCareerRepo_Annuaire_EcartNomme_NomHorsHistorique`).

**Guardrail**: `platform/duckdb/annuaire_ratchet_test.go` — `TestLecturesDeLaVueDesNoms_Ratchet`.

**Directory scope "base" and the locating read (lot A, 2026-09-26).** The match view, the match
events resolver and Relations read their directory in *base scope* (`nommerLignesPorteeBase`): aliases
and participant gamertags over the whole database (the view's `MAX`, predicates pushed on tables),
the kill-feed leg on the read's matches, then, for the xuids still unnamed, on the whole database.
That last leg is done in two steps so that it never evaluates the whole `_latest` window of the
canonical kill feed (3 to 7 s measured): (1) **locate** the candidate matches in the raw
append-only table `match_kill_events` and in `killer_victim_pairs`, filtered by those xuids with a
non-empty gamertag; (2) **read** the view's kill-feed leg, `_latest` windows bound by `match_id`, on
those candidates only. Every version of a row that ever carried the xuid designates its match, so
the candidates are a superset of the matches where `_latest` shows it, and every value comes from
`_latest`: the names equal the view's by construction. This amends the read rule of
[ADR 0026](0026-append-only-art-eradication.md) (append-only tables are read through their
`<table>_latest` view) with one exception: **a raw read of an append-only table is admitted to
locate `match_id`s, never to read a value.** Single site today:
`platform/duckdb/squad_repo_annuaire.go`, `localiserKillFeed`, on the template
`analysis.AnnuaireKillFeedLocaliserSQL` (it projects `match_id` only; `TestAnnuaireSQL_PorteeBase`
pins that, and `TestAnnuairePorteeBase_RepliLitLaDerniereVersion` fails if step 2 reads the raw
table). The raw-read guard `TestNoRawAppendOnlyReads` (`platform/duckdb/no_raw_rating_reads_test.go`,
scanning `platform/duckdb`, `api`, `service` and `analysis`) allowlists this one file with a dated
justification (`identity_annuaire.go`, 2026-09-26); the allowlist is keyed by file name, so a
second raw read added to that file would pass it: review holds that line.

### I2 — A `_latest` read on a request path binds its scope under the window

DuckDB pushes under a `QUALIFY ... OVER (PARTITION BY match_id ...)` window only a constant
filter on the partition key (an equality `match_id = ?` also propagates through a join). A read
on a request path therefore passes its list of matches as constant parameters on the `match_id`
of every `_latest` view it reads. Never as a subquery (`IN (SELECT ...)`), a semi-join, or a
filter placed on another relation of the join only: each of these leaves the window over the
whole table while every returned number stays the same, so no value test can see it. Only
`match_id` is pushed; a filter on a player (killer, victim) does not cross the window.

**Guardrails**: `platform/duckdb/tactical_repo_fenetres_test.go` —
`TestTacticalRepo_PerimetreRestreint_FenetresBornees`;
`platform/duckdb/weapon_range_repo_fenetres_test.go` — `TestWeaponRange_Perimetre_FenetresBornees`;
`platform/duckdb/career_repo_fenetres_test.go` — `TestLecturesHistorique_FenetresBorneesAuxMatchsDuJoueur`,
`TestLecturesHistorique_CampagneHorsDeLaListe` (lot B, 2026-09-27);
`platform/duckdb/annuaire_fenetres_test.go` — `TestAnnuaire_ListesLieesEnUnParametre` (item B.7);
`platform/duckdb/squad_life_placement_repo_test.go` — `TestSquadLifePlacementRepo_BorneEtDernierePasse` (Emprise life placement, `match_life_placement_latest`, 2026-09-30);
`platform/duckdb/squad_vehicle_repo_test.go` — `TestSquadVehicleRepo_BorneDernierePasseEtFragsParCamp` (Emprise vehicles resource, `match_vehicle_takes_latest` and `match_kill_events_latest` bound under their windows, kill events read only for the matches that have engine frags to split, 2026-09-30);
`platform/duckdb/solo_lives_repo_test.go` — `TestSoloLivesRepo_BorneEtDernierePasse` (Timeseries "Usage" tab, one player's lives near a teammate: `match_lives_latest`, `match_death_context_latest` and `match_kill_events_latest` each bound on their own `match_id`, the player filtered after the window, 2026-10-06), and `TestSoloLivesRepo_ParJoueurs_BorneEtDernierePasse` (match view, "Isolation" for every player of the team in one read — I4 —, the same three views bound on their own `match_id`, the players filtered after the window, 2026-10-07);
`platform/duckdb/tactical_repo_contextes_test.go` — `TestTacticalRepo_ContextesDeMort_BorneEtNull` (Tactical tab, the death context behind the "alone / near" badge of a zone's replay tiles: `match_death_context_latest` bound on its `match_id`, the whitelist required, 2026-10-06).

### I3 — Data re-read from one request to the next goes through a cache invalidated at sync, never filled by a degraded load

The filter rows and the canonical match history of a player live in one process-wide cache
(`platform/duckdb/player_read_cache.go`): key (xuid, title, database path, variant), TTL 60 s,
one load in flight per key, a generation per (xuid, title) so that a load started before an
invalidation never fills the cache, and a copy on every read (a consumer may enrich its rows).
`InvalidatePlayerReadCaches` empties it for a player wherever those rows change: end of the
player's post-sync (`runPostSyncPipeline`), the "with friends" recompute
(`RecomputeIsWithFriends`), a match exclusion (`MatchExclusionRepo.SetExclusion`). A load that
ended degraded (a best-effort step reported its failure through `noteDegraded` or
`bestEffortFailed`, or the request carrying the load ended meanwhile) is returned to its
requester as before, but is neither stored nor shared with the requests waiting on it: they
reload.

**Guardrails**: `archlint/player_read_cache_invalidation_test.go` —
`TestPlayerReadCacheInvalidationPoints`; `platform/duckdb/player_read_cache_test.go` —
`TestPlayerReadCache_DegradedLoadNotStored` (the P0 of the campaign's adversarial review),
`TestPlayerReadCache_RequestEndedDuringLoadNotStored`,
`TestPlayerReadCache_DegradedLoadNotSharedWithWaiters`.

### I4 — One load per request, never N

A read that several blocks of one request consume is performed once, before the blocks, under
its own duration section. The memo is keyed by the set of its inputs, not their order, and lives
for one request only: the next request sees the database as it is. On the Squad page,
`pourLaRequete` (`service/teammates/teammates_service_loads.go`) gives each request a copy of the
service whose shared readers go through that memo: the impact events (Q32) are read once for the
impact matrix, the intensity profile, the performance series and the first frag
(`prechargerImpacts`), and each member's history once per (title, gamertag). The Career resolves
friends through the tracked-profile registry first, then one read for the others, never one read
per friend.

**Guardrails**: `service/teammates/teammates_service_loads_test.go` —
`TestGetPage_LitLesEvenementsDImpactUneSeuleFois`, `TestGetPage_UnLoadForParMembre`,
`TestGetPage_DeuxRequetesDeuxLectures`; `service/career_service_friends_test.go` —
`TestCareerService_ResolveFriendXUIDs_RegistreDAbordPuisUneLecture`.

### I5 — A steady-state sync takes no writer when nothing is new

The LUSR v2 post-sync step reads its watermark, and the eligibility of the candidates above it,
on the shared reader (`selectShadowWorkUnderRead`, `sync/skill/skill_v2_watermark.go`). With
nothing notable above the watermark it returns without asking for a writer and logs one INFO
line per player and cycle, carrying `candidates` and `new`. With news, it takes the writer in
bounded bursts per player (`defaultLUSRBurstLimits`, `sync/skill/skill_v2_shared_access.go`:
50 matches or 2 s of hold). One predicate decides "already processed", `lusrWatermarkCovers`,
for the pre-filter under the reader, the scorer under the writer and the gap detector. This is a
sync rule, not a page read, but every writer acquisition swaps out the shared reader that page
reads use (ADR 0016).

**Guardrails**: `sync/skill/skill_v2_watermark_test.go` —
`TestLUSRV2Shadow_Stationary_NoWriterAndOneInfoLine`,
`TestLUSRV2Shadow_NoCandidate_NoWriterAndOneInfoLine`; `sync/skill/lusr_watermark_guardrail_test.go`
— `TestNoDuplicateLUSRWatermarkPredicate`, `TestLUSRWatermarkCovers_Boundary`.

### I6 — Every page read declares its duration sections

The middleware (`api/middleware/slog_logger.go`) puts a `timing.Timings` (`observability/timing`)
on every request; a read declares `defer timing.FromContext(ctx).Section("<name>")()`, which is
safe without a `Timings`. Sections are leaves: a section never encloses another, or it would be
counted twice in `total_ms`. After the handler, when at least one section exists and the request
is slow (`LEVELUP_SLOW_REQUEST_MS`, default 1000, read once at mount) or DEBUG is on, the
middleware writes one INFO line `http_timings` (`path`, `duration_ms`, `total_ms`, `sections`,
`calls`); a slow successful response is logged at INFO with `slow: true`. The Career reads also
give the directory they load its own section (`top_encounters` and `top_encounters_annuaire`).

**Guardrails**: `api/middleware/slog_logger_test.go` — `TestSlogLogger_TimingsLoggedForSlowRequest`,
`TestSlogLogger_SlowSuccessLoggedAtInfo`; `platform/duckdb/career_repo_annuaire_test.go` —
`TestCareerRepo_Annuaire_SectionsDeDuree`. The obligation itself is **not yet guarded** (see
*Guardrails*).

### I7 — DuckDB resource bounds are read when each connection opens

`threads` and `memory_limit` come from `LEVELUP_DUCKDB_THREADS` and `LEVELUP_DUCKDB_MEMORY_LIMIT`,
read by `duckThreads()` and `duckMemoryLimit()` when `applyDuckSessionInit` sets up a connection
and when `BudgetsSnapshot` reports them (`platform/duckdb/db.go`), never captured in a package
variable at init, which runs before `.env.local` is loaded (`config.BootstrapEnvLocal`). The
process environment still wins over `.env.local`. Defaults stay 2 threads and 512MB, the
calibration of the production VPS; a development machine may raise them. Measured on the Career
page with 8 threads and 4 GB in the process environment: top-encounters 805 / 2,021 ms and rivals
1,118 / 2,031 ms (two passes), against the default-setting figures of the table above: the same
SQL cost, two to four times shorter.

**Guardrail**: `platform/duckdb/db_resource_limits_env_test.go` —
`TestResourceLimits_EnvSetAfterPackageInitIsHonored`.

## Consequences

- A new page read is written against the seven rules: names through the read's directory; each
  `_latest` view bound to the read's matches; a load shared by several blocks memoised for the
  request; a load repeated across requests cached behind an invalidation point; its sections
  declared. Review checks what no test enforces (I4 outside the two pinned pages, the obligation
  of I6).
- Removing a join can change the physical order of rows. An order a response depends on must be
  total (`ORDER BY` with a tie-breaker): removing the join from Q32b moved badges carried by the
  first row of a tie to another player (33 matches across nine scenarios), and the campaign then
  made Q29, Q32, Q32b, Q32c and Q10 total.
- Binding a list is not free. At full history an `IN` list of 1,158 values costs about 0.3 s per
  `_latest` view, and beyond about 800 matches the bound list costs more than the whole window.
  I2 makes session- and period-scoped reads proportional; full-history reads keep the cost of the
  window (SQL on the copy at full history: Synthesis records about 2.1 s, Sessions about 2 to
  2.9 s, Timeseries about 8.5 to 9.3 s). Only a change of design, materialising or compacting the
  `_latest` state along the ADR 0026 recipe, changes that order of magnitude; it touches the
  anti-ART invariants and waits for production evidence. Lot B (2026-09-27) measured that the cost
  of a long `IN` is the binding of its thousands of parameters, not the filter: the same list as
  ONE `VARCHAR[]` parameter (`list_contains(?::VARCHAR[], match_id)`) descends under the window as
  well and, on the compacted copy (kill feed of one player, three runs), costs 64-87 ms instead of
  98-107 ms at 1,160 matches and 192-249 ms instead of 304-341 ms at 7,190 (the whole window:
  223-274 and 195-214 ms). That constant form is evaluated row by row in time proportional to the
  list, so it is kept for the `match_id` lists that must cross a window: 3,000 xuids against the
  raw kill journal cost 38-44 s (4.0-4.4 s compacted), against 0.11-0.12 s for the same single
  parameter in a semi-join (`col IN (SELECT unnest(?::VARCHAR[]))`), which does not descend under a
  window. Both forms have one text source, `analysis/sql_liste.go` (`SQLDansListe`,
  `SQLDansListeParJointure`), used by the directory templates and by
  `platform/duckdb/perimetre_liste.go`; the ratchet `TestListeLiee_TexteEnUnSeulEndroit`
  (`archlint/liste_liee_ratchet_test.go`) forbids a hand-written copy. A short list pays for it:
  with 8 xuids, 3.3 ms in `IN`, 6.9 ms as one parameter (item B.7, 2026-09-27).
- No write changed shape: INSERT-only through `persist`, reads through the `_latest` views, no
  allowlist of `sync/no_art_patterns_test.go` touched.
- Production has not been measured. The instruments exist (`http_timings`,
  `LEVELUP_SLOW_REQUEST_MS`); the structural work listed under *Exceptions* waits for them, except
  lot A.

## Exceptions (state on 2026-09-27)

Every read below is outside an invariant today, with its measured cost (copy, 2 threads /
512 MB, player `JGtm` and his last match, one call, unless stated) and what retires it. Lot A,
step P2 of `.ai/V7.5/PLAN_PERF_LECTURES_PERIMETRE_2026-09-26.md` (reads only, no write, no anti-ART
invariant touched: the one structural lot accepted before the production measurement), was done
on 2026-09-26: its five reads of the view left the I1 table below, and what it leaves behind is
listed under I2. Lot B (step B of `.ai/V7.5/PLAN_PERF_COMPACTION_ET_PERIMETRE_JOUEUR_2026-09-26.md`, after
the compaction of step C) was done on 2026-09-27: three reads left the I2 table below, and its
remeasure of the two I3 pages is written in their rows.
An exception leaves this list together with its read; a new one needs a line here, a measured cost
and a date.

### Reads of `v_gamertag_lookup` (I1), frozen by the ratchet

| Read | File (occurrences allowed by the ratchet) | Measured cost | Retired by |
|---|---|---|---|
| Explorer: player search by name (`ResolveXUIDByGamertag`) | `platform/duckdb/explorer_repo.go` (1) | 2.3 s | Stays: a lookup by name (`ILIKE`) that a directory keyed by xuid cannot answer; retired with its read |
| Media: match lobbies (`loadMatchLobbies`) | `platform/duckdb/media_repo_filters.go` (1) | one evaluation per call (1.7 to 3 s, not measured alone) | Stays; retired with its read |
| World leaderboard file: stat ranking (`GetStatLeaderboard`) | `platform/duckdb/leaderboard_world_repo.go` (1) | one evaluation per call, not measured | Stays; retired with its read |
| Kill collector (sync, not a page): credit-pass directory, match roster fallback | `sync/killcollector/credit_annuaire.go` (1), `sync/killcollector/roster.go` (1) | the view materialised at each pass, not measured at production size | Stays; retired with its read |

Retired on 2026-09-26 by lot A: the match view scoreboard (Q12) and events (Q21), encounters
(Q23) and encounter stats (Q23b), `GamertagRepo.ResolveGamertags`, Relations Q28 and scoped Q28,
and the Relations heatmap (Q29); their five files left the ratchet table. Before, the match view
paid about 10 s of view evaluation per opened match when all its reads were served. After (copy,
`JGtm`, his last match and 20 random matches, runs on a loaded machine): Q12 median 16-33 ms, Q21
7-21 ms, `ResolveGamertags` 5-13 ms, Q23 46-89 ms (its own shared-history query; its directory
≤ 9 ms), Q23b has no name at all and costs its kill-feed window; Relations Q28 directory 241-277 ms
on top of Q28 itself, heatmap Q29 78-94 ms.

Lot A kept the names: restricted to the opened match, a directory lost names the view finds
elsewhere (11 (match, player) pairs out of 93,636, in 10 matches; in Relations, 2 served rows of
one player). The directory therefore reads in *base scope* (see I1, "Directory scope base and the
locating read"): aliases and participant gamertags over the whole database, the kill feed of the
read's matches, then, for the xuids still unnamed, the kill feed of the candidate matches located
in the raw journal and read through `_latest`. On the production copy, zero name differs from the
view over every served row compared (Q12 93,000 rows, Q21 1,311,594 events, Q23 / Q23b 2,625 rows,
`ResolveGamertags` 1,853 xuids, Relations 12,289 rows, heatmap 1,265 rows). The other seven files of
the ratchet table (DDL, migrations, demo seed, schema comment, validation gate: 11 occurrences) are
not reads.

### `_latest` windows over the whole history (I2)

| Read | File | Measured cost | Retired by |
|---|---|---|---|
| Match view: encounter stats (Q23b), kill-feed window (`kv_stats`) | `platform/duckdb/queries_match_detail.go` | about 0.9 s at rest; 1.6 to 4.6 s median alone in lot A's runs on a loaded machine | Not assigned |

Retired on 2026-09-27 by lot B: Career encounters (Q26) and rivals (Q27, now read ONCE: the two
rankings are sorted in Go, `platform/duckdb/career_repo_rivals.go`), Relations Q28 over the
player's history (the unscoped template is gone: the history is read as a list and goes through the
scoped query) and the Tactical tab's deaths per map (`MortsParCarte`, whose list now sits on both
views). Each reads the player's matches (`QMatchsDuJoueurTpl`, Campaign excluded) or the page's
filter perimeter, binds that list as one `VARCHAR[]` parameter on every `_latest` view it reads (see
*Consequences*), and hands the same list to its directory. Measured on the compacted copy (2 threads
/ 512 MB, whole read with its directory, three alternated runs before -> after on a loaded
machine): Q26 `JGtm` 282-301 -> 169-287 ms, `XxDaemonGamerxX` 144-161 -> 42-46 ms; Q27 `JGtm`
460-465 -> 133-247 ms, `Madina97294` 371-500 -> 122-163 ms; Q28 `JGtm` 371-421 -> 253-437 ms (five
runs: 302-391 -> 233-272 ms), `XxDaemonGamerxX` 139-168 -> 43-44 ms; `MortsParCarte` with the
whitelist the tab sends by default (all the player's matches) `JGtm` 371-622 -> 227-383 ms, with a
30-match whitelist 313-480 -> 30-53 ms. `Nuzzles` (7,190 matches, whose kill feed holds 66 % of the
journal, so the list saves little): Q27 388-457 -> 306-354 ms, `MortsParCarte` 356-588 -> 238-260
ms, Q26 and Q28 unchanged within the noise (five runs, medians 519 -> 579 ms and 1,061 -> 1,158 ms;
0.8 to 0.9 s of Q28 is its directory, whose reads were already bound: see the next paragraph). Served rows identical on the
five tracked players (top 10 and rivals byte for byte, 12,130 Relations rows, 26,784 deaths per map
compared as sets: their order was never total). One intended difference, Halo 5 only: the kill-feed
legs of Q26, Q27 and Q28 did not exclude Campaign matches; the list does (the history those rows
belong to already did), so the 118 kill events between two players in the 63 Campaign matches of
the Halo 5 copy no longer count.

Item B.7 (2026-09-27): the directory itself (`platform/duckdb/squad_repo_annuaire.go`, templates
`analysis/identity_annuaire.go`) bound its lists as `IN (?, ?, …)`: up to 14,900, 10,163 and
17,741 parameters per query for `Nuzzles`' Relations. Every list of the directory is now one
`VARCHAR[]` parameter — the `match_id`s of the kill-feed leg as the constant that crosses the
`_latest` window, every other list in a semi-join, the locating read's xuids unfolded in its
`cherches` CTE — with the templates still the single source shared with the view (the view's DDL is
unchanged). Measured on the compacted copy (three alternated runs): `Nuzzles` Relations Q28
1,118-1,339 -> 551-839 ms (directory 731-912 -> 256-286 ms), heatmap 294-340 -> 179-199 ms; Squad
(`JGtm`, a composition of 579 matches) Q32 314-702 -> 192-257 ms, Q32b 103-106 -> 46-62 ms; the
match view of `JGtm`, whose lists are short, pays the fixed cost of the parameter: medians Q12
14-17 -> 19-21 ms, Q21 9-10 -> 13-14 ms, Q23 38-47 -> 48-51 ms, `ResolveGamertags` 4-5 -> 8 ms.
Names identical to the previous code on every read that goes through the directory (Squad Q29, Q32,
Q32b; Career Q26, Q27, Q10; Compare; match view Q12 and Q21 on 322 matches, Q23 and
`ResolveGamertags` on 279; Relations Q28 and heatmap; five players), `ResolveGamertags` still
leaving an unnamed xuid out of its map. Guardrail: `TestAnnuaire_ListesLieesEnUnParametre`
(`platform/duckdb/annuaire_fenetres_test.go`).

Not a window, but the same kind of cost, listed here because it is a whole-table read on a request
path: when the base-scope directory's last fallback fires (a player that no alias, participant
gamertag or kill feed of the read's matches names: 23 of `JGtm`'s 1,160 matches, 4,688 of
`Nuzzles`'s 7,190), its locating read scans the raw `match_kill_events` (3.95 M rows) in two passes
(killer, then victim): the directory then costs 150 to 360 ms, above the 100 ms budget of a match
view read (`platform/duckdb/squad_repo_annuaire.go`, `localiserKillFeed`). A single pass (killer OR
victim) is the lead; not measured. Not assigned.

**The cost of these windows follows the number of passes, and compaction is the maintenance that
bounds it** (step C of `.ai/V7.5/PLAN_PERF_COMPACTION_ET_PERIMETRE_JOUEUR_2026-09-26.md`, 2026-09-26).
Every re-decode of a film appends a full pass to the film tables (INSERT-only, ADR 0026); a
`_latest` view serves only the last one, but its window partitions every pass. On the local
database of 2026-09-26, 90 % of the film-table rows were superseded passes (11.27 M raw rows for
1.12 M served). `levelup compact-passes` rebuilds those tables with the rows their views serve (no
`DELETE`, view output, DDL, indexes and sequences checked before each COMMIT) and is run, server
stopped, after each re-decode campaign — never at boot nor after a sync. Measured on a copy
(2 threads / 512 MB, loaded machine, two runs, before -> after): Career encounters Q26 1.6-2.4 s ->
0.24-0.76 s, rivals Q27 3.1-4.3 s -> 0.28-0.64 s, Relations Q28 1.8-2.9 s -> 0.16-0.51 s (`Nuzzles`
3.3-3.7 s -> 1.6-1.7 s), match view Q23b median 1.9-2.5 s -> 0.23-0.30 s, `MortsParCarte`
2.7-5.0 s -> 0.35-0.64 s, the directory's locating read (`Nuzzles`, 2,973 xuids) 0.19-0.25 s -> 0.06-0.09 s (the raw
journal now holds only the served pass: same candidates). Compaction does not retire these
exceptions — the windows still span the whole history — it keeps their cost proportional to the
history instead of to the number of decodes; lot B then bound three of them to the player's matches
(above).

### Complete-history pages without a cache (I3)

| Read | Measured cost | Retired by |
|---|---|---|
| `GET /pages/home` | 1,642 ms for 310 KB outside sync; 2.5 s during a sync cycle | Not assigned. Lot B (2026-09-27, no cache by decision) remeasured on the compacted copy the one whole-history `_latest` window of the page, the favourite weapon (`LoadFavoriteWeapon`): 0.10-0.19 s, under its 0.3 s threshold; the page itself was not remeasured (it needs the server) |
| Synthesis, `weapon_records` section | 1,640 ms (complete history) | Not assigned. Lot B (2026-09-27): its windows are already bound to the scope (`JGtm`: 102,235 kill-feed and 96,633 position rows seen, exactly those of his 1,160 matches); 0.05-0.68 s on the compacted copy, the cost of the player's own volume, not of a window over the history |

### Writers outside the invalidation points (I3)

After these writes, the cache is refreshed by its 60 s TTL only: the Halo 5 live sync
(`games/halo_5/livesync/runner.go`, which does not go through `runPostSyncPipeline`), the
OpenSpartan post-import enrichment (`service/openspartan_post_import_service.go`), and command-line
processes (backfill, friends recompute), which are other processes. Not assigned.

## Guardrails

Default suite: `cd apps/go-api && go test ./internal/platform/duckdb/ ./internal/archlint/
./internal/service/... ./internal/sync/skill/ ./internal/api/middleware/` (with `CGO_ENABLED=1`,
as for any DuckDB test). Integration: `go test -tags=integration -p 1 ./internal/platform/duckdb/...`.

| Inv. | Test (file) | Runs in | What it actually blocks | What it does not see |
|---|---|---|---|---|
| I1 | `TestLecturesDeLaVueDesNoms_Ratchet` (`platform/duckdb/annuaire_ratchet_test.go`) | default suite | Parses every non-test Go file under `internal/` and counts, per file, the bare identifier `v_gamertag_lookup` in string literals (a Go comment does not count, an SQL comment inside a literal does); fails when a file exceeds its dated allowance and when it falls below it, so the table only descends. | A new page read that calls a reader still in the table (for instance the Explorer's `ResolveXUIDByGamertag`) adds no literal and passes. |
| I2 | `TestTacticalRepo_PerimetreRestreint_FenetresBornees` (`platform/duckdb/tactical_repo_fenetres_test.go`), `TestWeaponRange_Perimetre_FenetresBornees` (`platform/duckdb/weapon_range_repo_fenetres_test.go`), `TestLecturesHistorique_FenetresBorneesAuxMatchsDuJoueur`, `TestLecturesHistorique_CampagneHorsDeLaListe` (`platform/duckdb/career_repo_fenetres_test.go`), `TestAnnuaire_ListesLieesEnUnParametre` (`platform/duckdb/annuaire_fenetres_test.go`), `TestSoloLivesRepo_BorneEtDernierePasse`, `TestSoloLivesRepo_ParJoueurs_BorneEtDernierePasse` (`platform/duckdb/solo_lives_repo_test.go`), `TestTacticalRepo_ContextesDeMort_BorneEtNull` (`platform/duckdb/tactical_repo_contextes_test.go`) | default suite | Seed ten matches, ask for two, record the SQL the real reader sends, replay each query under `EXPLAIN (ANALYZE, FORMAT JSON)` and fail if any `WINDOW` operator saw more rows than the two matches hold, or if fewer queries than expected carry a window (shared helper `exigerFenetresBornees`). Covers `KillEvents`, `MortsAvecContexte`, `KillPositions`, `Univers`, `MortsParCarte`, `ContextesDeMort`, `LoadWeaponRange`, `LoadWeaponOpening`, `LoadMatchRangeKills`. The lot B test seeds four matches of the player among ten and holds the whole-history reads (`GetTopEncountersGlobal`, `GetRivals`, `GetRelations` on the history and on a scope, `MortsParCarte` without a whitelist) to the player's matches; it also fails if `GetRivals` reads its aggregate more than once, and if a Halo 5 Campaign duel enters the rivals or the encounters. The item B.7 test holds the directory's kill-feed leg to the player's matches in read and base scope, requires the name a kill feed alone gives, and fails if a directory query binds more than five arguments (a list bound value by value). | Any other `_latest` reader: nothing checks the windows of a read these tests do not call. |
| I3 | `TestPlayerReadCacheInvalidationPoints` (`archlint/player_read_cache_invalidation_test.go`) | default suite | Parses `runPostSyncPipeline`, `RecomputeIsWithFriends` and `SetExclusion` and fails if one of them no longer calls `InvalidatePlayerReadCaches`, or cannot be found. | A new writer of the cached rows without the call, or a new cache. |
| I3 | `TestPlayerReadCache_DegradedLoadNotStored`, `TestPlayerReadCache_RequestEndedDuringLoadNotStored`, `TestPlayerReadCache_DegradedLoadNotSharedWithWaiters` (`platform/duckdb/player_read_cache_test.go`) | default suite | With fake loaders: a load that reports a degraded step, or whose request ends while it runs, is returned to its requester but not stored; a request waiting on a degraded load reloads for itself. | A best-effort step that swallows its error without reporting it: the cache cannot know. |
| I4 | `TestGetPage_LitLesEvenementsDImpactUneSeuleFois`, `TestGetPage_UnLoadForParMembre`, `TestGetPage_DeuxRequetesDeuxLectures` (`service/teammates/teammates_service_loads_test.go`) | default suite | Run the Squad `GetPage` on a page where every consumer renders its section and count the reads: one `LoadImpactEvents` for the four consumers, one history load per member, and two requests read twice (the memo does not outlive its request). | Pages other than Squad. |
| I4 | `TestCareerService_ResolveFriendXUIDs_RegistreDAbordPuisUneLecture` (`service/career_service_friends_test.go`) | default suite | Tracked friends resolve from the registry with no read; the others in exactly one read. | Other Career reads. |
| I5 | `TestLUSRV2Shadow_Stationary_NoWriterAndOneInfoLine`, `TestLUSRV2Shadow_NoCandidate_NoWriterAndOneInfoLine` (`sync/skill/skill_v2_watermark_test.go`) | default suite (build tag `cgo`) | On a file database, after a first cycle that writes, a second cycle with nothing new makes zero writer calls and one reader call, writes no row and logs one INFO line with the expected counters; a player with no candidate takes no writer. | Other sync steps that take a writer. |
| I5 | `TestNoDuplicateLUSRWatermarkPredicate`, `TestLUSRWatermarkCovers_Boundary` (`sync/skill/lusr_watermark_guardrail_test.go`) | default suite | Fails if a non-test file of the package other than `skill_v2_watermark.go` compares a match start time or a watermark (`After`, `Before`, `Equal`, `Compare`): a copy of the predicate that would let the pre-filter disagree with the scorer. Pins the boundary: a match exactly on the watermark is already processed. | The zero-writer behaviour itself (held by the two tests above). |
| I6 | `TestSlogLogger_TimingsLoggedForSlowRequest`, `TestSlogLogger_SlowSuccessLoggedAtInfo` (`api/middleware/slog_logger_test.go`) | default suite | A slow request with sections yields exactly one INFO `http_timings` line with `path`, `duration_ms`, `total_ms`, `sections` and `calls`; a slow 2xx is logged at INFO with `slow: true`. | Whether a given read declares a section. |
| I6 | `TestCareerRepo_Annuaire_SectionsDeDuree` (`platform/duckdb/career_repo_annuaire_test.go`) | `integration` tag | The Career and Compare reads each emit their own section and their directory's section, with the expected call counts. | **Not yet guarded**: nothing fails when a new page read declares no section. That obligation is a review rule. |
| I7 | `TestResourceLimits_EnvSetAfterPackageInitIsHonored` (`platform/duckdb/db_resource_limits_env_test.go`) | default suite | Sets `LEVELUP_DUCKDB_THREADS=3` and `LEVELUP_DUCKDB_MEMORY_LIMIT=300MB` after package init; fails unless `BudgetsSnapshot` reports them and a newly opened connection runs with 3 threads and a memory limit within 1 % of 300 MB. | Environment variables captured at init outside `platform/duckdb` (one known: `LEVELUP_REPLAY_PUBLIC`, `api/handlers/replay_local_gate.go`). |

A guardrail that moves or is renamed keeps its reference here up to date, in the same commit; a
guardrail retired on purpose leaves the table with its reason.

## Alternatives rejected (by measurement)

- **More DuckDB threads.** With 16 threads and 8 GB, Q29 with the join still took 2.4-3.1 s:
  parallelism does not fix a plan that materialises the whole view.
- **Filtering the view.** `v_gamertag_lookup` with `WHERE xuid = ?` still costs 3.0 s: nothing is
  pushed through its aggregates.
- **Scoping a `_latest` read by subquery or semi-join.** Before the campaign the tactical reads
  already applied the match list in SQL, on the match registry and through a re-selected
  subquery; the windows still ran over the whole table (`KillEvents` 1,794 ms for 6 matches).
