# ADR 0050 — Match kickoff: one source order, one write, one catch-up

Status: accepted (2026-11-18), write side and readers implemented on the same date.
Relates to: ADR 0019 and ADR 0030 (writes go through `internal/persist`), ADR 0013 and ADR 0016 (single writer per
database, short writer bursts), ADR 0026 (ART-safe write forms), ADR 0047 (revision replayed at startup), ADR 0006
(per-minute rates read the time played).

## Problem

The kickoff of a match (`match_registry.real_start_time`, the instant the game clock of the first round starts) was
written by five writers: an estimate at sync time (computed twice, once for the registry and once for the time played),
a report from the replay document on the local cooking path only, and three offline commands with direct `UPDATE`s
(`cmd/backfill_t0`, `cmd/backfill_t0_film`, `cmd/backfill_time_played`). A seven-value `t0_quality` column decided
between them, without any reader. The time played of every player (`match_participants.time_played_seconds`) was
computed once, at sync, on the estimate, and never again when the replay brought a better kickoff.

## Decision

- **D-1 — One source order, decided in one place.** `timeline.ResolveKickoff` (`internal/analysis/timeline/kickoff.go`)
  is the only function that decides a kickoff: `film` (the `t0FilmMs` the replay document publishes, taken as given,
  no detector replayed), else `api` (the existing estimate from the arrivals of the players present at the start, one
  call to `ComputeT0`, refused when negative or above its 120 s threshold; its thresholds stay private to it), else
  unknown. Halo 5 has neither a replay nor arrivals: its kickoff is unknown, with no title branch. The replay value is
  checked against the match window itself (no invented threshold): a value below zero, or beyond the match duration
  when it is known, is refused like a replay that publishes none, logged and counted (`kickoff_film_hors_bornes_total`).
- **D-1b — What the replay says, in three states** (`sync/kickoff.Film`). PRESENT and publishing a kickoff: source
  `film`, the document's value. PRESENT and publishing none (or an out-of-window one): the film says there is none, the
  next source decides. ABSENT (never generated, purged by retention) or UNREADABLE (locked, being replaced, damaged):
  nothing new is known; a match whose stored source is `film` keeps its stored kickoff and source (the registry is not
  rewritten) and its players' time played is still computed on that stored kickoff, so a formula change reaches it;
  any other match resolves normally. An UNREADABLE replay on a match stored `film` or unknown also counts as a failed
  match for the revision (not stored, retried); an ABSENT one does not. During a catch-up the stored state is read under
  the writer of the batch, so a publication that wrote `film` between the file read and the burst is seen and kept.
- **D-2 — The source is stored next to the value.** New column `match_registry.real_start_source` (`film`, `api`, NULL =
  unknown), added by migration `shared_registry_real_start_source_v1` (column added, table not rebuilt). The column is
  born right: the same step sets `film` on the rows already reported from a replay (`t0_quality = 'film_movement'`), one
  parameterised `UPDATE` per match, so that the first catch-up keeps them when their replay is gone; it then recreates
  `v_match_full`, whose `SELECT mr.*` is frozen at creation. The estimate outcome (single arrival, spread arrivals,
  rejection) is a log line and a counter (`kickoff_source_*_total`, `kickoff_estimation_*_total`), never stored. These
  counters are WRITE flows: they move on a sync insertion or a plan the persister wrote, never on a dry run nor on a
  catch-up that finds the database already right.
- **D-3 — The time played follows the stored kickoff.** `sync/kickoff.Resolve` resolves the kickoff, then computes each
  player's time played with the unchanged formula (`timeline.ComputeTimePlayedFor`, truncation included) on that
  kickoff (`timeline.GameplayTimeline`: unknown kickoff = the window starts at the match start). A player whose arrival
  is unknown, or whose window is empty, keeps its last stored value (for an unknown arrival, the API value, never
  recomputed).
- **D-4 — One write.** Outside the sync insertions, only `persist.KickoffPersister` writes the kickoff, its source and
  the time played: one match per transaction, one `UPDATE … WHERE match_id = ?` for the registry and one per changed
  player, under the shared writer the caller holds, and only for values that differ from the database (a second pass
  writes nothing). The callers are:
  - the sync, which sets the values on the rows BEFORE their insertion (`kickoff.ApplyToRows`, no `UPDATE`), and the
    OpenSpartan import through the same function;
  - every replay publication through `replayartifacts.Deriver` (local cooking, worker deposit, desktop regeneration
    queue, on-demand generation and the admin "build the replay" action, which go through the worker deposit entry, and
    the derivations catch-up): the match state is read in the read segment, and the writer is acquired only when
    something changes. The offline `levelup backfill-replay` pass does not derive: `levelup backfill-kickoff` follows it.
- **D-5 — One catch-up, gated by a revision of the shared database.** `kickoff.Backfill` recomputes every match of a
  shared database in batches, each batch in one short writer burst; it is idempotent, so an interrupted pass resumes by
  running again. At startup, `kickoff.ApplyRevision` runs it when the value stored under `kickoff_revision` in the
  shared `sync_meta` differs from `timeline.KickoffRev`, and stores the new value only after a complete pass without
  failure. It runs inside the startup scoring task (`service/scoringrevision`), before the player notes, for each
  title that task walks (titles with the LUSR capability). A PARTIAL catch-up (some matches failed,
  `port.ErrSharedRevisionPartial`) leaves the revision unstored, logs an error with the count and one example match,
  counts it (`scoring_revision_base_partagee_partielle_total`), and the player notes still run: one failed match does not
  hold a whole title. Only a catch-up that could not run (database or writer unavailable, a batch that could not be read)
  holds the player notes of that title for that startup. A replay whose match is not in the registry, or a match that
  vanished between the list and its batch, is not a failure (logged, counted, the derivation is marked). The command
  `levelup backfill-kickoff` runs the same function, with a `--dry-run` that lists the gaps without writing.
- **D-6 — The notes replay once, and the revisions are bound.** `PerformanceScoreRev` and `LUSRRev` move to
  `2026-11-18`: on the first startup, the kickoff catch-up runs, then every player's notes are replayed once. Both
  scoring fingerprints hash `timeline.KickoffRev` (`TestScoringParametersMatchRevision`): changing the kickoff revision
  without moving the two scoring revisions fails.
  Amendment (2026-11-22): the kickoff the replay publishes is read on the game clock (film decoder v1.5.0), so the
  stored value of a replayed match changes. `KickoffRev` moves to `kickoff-2026-11-22`, `PerformanceScoreRev` to
  `perf-2026-11-22` and `LUSRRev` to `lusr-2026-11-22` (same two-fingerprint rule, no parameter of their own changes):
  the first startup runs the kickoff catch-up, then replays every player's notes once.

- **D-7 — One reading, for every reader.** `timeline.ForMatch` (`internal/analysis/timeline/build.go`) is the only rule
  that reads a stored kickoff: a known kickoff = the chronology starts from it; an unknown one (`nil`: Halo 5, a refused
  estimate, missing data) = the chronology starts at the match start and the gameplay duration equals the match duration.
  The write side (`GameplayTimeline`), the builders of per-match timelines (`BuildTimelinesFromPlayerMatches`,
  `BuildTimelinesFromSquadRows`), the match page (`service.matchViewTimeline`: header, event realignment, Dominance
  binning, "Overtime" flag), the squad history rows and the match events source all go through it. On the SQL side two
  fragments in `platform/duckdb/sqlfrag` carry the same rule: `SQLT0Ms` (the kickoff, NULL = unknown) and
  `SQLGameplayDurationMs` (match duration minus kickoff, floored at 0), read by the squad query, the "Overtime" flag of
  the history and the LUSR loader. No reader replaces an absent kickoff by `0`, no reader chooses between two durations:
  the four competing durations of the audit (registry duration, API playable duration, "duration minus kickoff"
  recomputed on the spot, the median of the players' time played) are one gameplay duration, truncated to the second
  like `MatchTimeline.GameplayDurationSeconds` (so a header can differ by one second from the value it served before, which
  subtracted a truncated kickoff). A time played that is absent stays absent (no fall-back to the match duration in the
  per-minute rates, no fall-back to the API playable duration on the home page).
  The match header serves it as `gameplay_duration_seconds` (it was `playable_duration_seconds`, a misleading name: the
  value already was the duration minus the kickoff). The web reads the header only (`header.t0_ms`, one reading of an
  absent value: `matchKickoffMs`): the replay window no longer prefers the document's kickoff to the match's, since the
  match's already is the replay's when the match has one. The replay publication of the lead-share window refuses a
  document kickoff outside the match window like the resolution does (`timeline.FilmWithinMatch`).
- **D-8 — The API playable duration has no reader and no writer.** `match_registry.playable_duration_seconds` is no
  longer inserted by the sync, the OpenSpartan import or the demo corpus, no longer read, and its backfill scope
  (`--playable-duration`) is gone. The column stays in the table (see the known limits).

## Guard rails

- `archlint/kickoff_write_guard_test.go`: no `UPDATE` of `real_start_time`, `real_start_source` or `time_played_seconds`
  outside `internal/persist/kickoff_persister.go`; no insertion of these columns outside the sync insertion
  (`persist/shared_persister.go`) and a dated exemption list (demo corpus, Halo 5 roster rebuild); the names
  `t0_quality` and `film_match_start_ms` appear nowhere outside the migrations.
- `archlint/kickoff_read_guard_test.go`: `real_start_time`, `playable_duration_seconds`, `t0_quality` and
  `film_match_start_ms` are named only by the two fragments, the resolution (`sync/kickoff`), the persister and the
  migrations, plus a dated exemption list (the table definition of a fresh database and its test fixture);
  `archlint/no_raw_real_start_offset_test.go` keeps the kickoff expression in `SQLT0Ms` alone. Web:
  `lib/replay/matchKickoff.guard.test.ts` (one replacement of an absent kickoff by zero, `matchKickoffMs`).
- `timeline.TestKickoffParametersMatchRevision`: the estimate thresholds, the source order and the replay window rule
  are fingerprinted per `KickoffRev`; changing one without a new revision fails.
- Unchanged and still green: `archlint/no_raw_real_start_offset_test.go`, the anti-ART guards of
  `internal/sync/no_art_patterns_test.go`, `TestScoringParametersMatchRevision`, the dated `OpenReadWrite` allowlist (the
  three removed commands leave it, the new command enters it).

## Known limits

- `t0_quality`, `film_match_start_ms` and `playable_duration_seconds` stay in the table: DuckDB refuses
  `ALTER TABLE … DROP COLUMN` on `match_registry` (primary key, indexes, dependent views), verified in memory and on a
  copy of the real schema, and a rebuild of the shared registry is not taken for a column name. The first two have no
  writer and no reader; `playable_duration_seconds` has no writer and no reader since D-8 (rows written before keep their
  value, new rows leave it NULL). A rebuild of the registry would drop the three: not worth the risk for a column name.
- A match without a replay keeps the estimate, which on old matches can sit in the first seconds of a countdown of about
  twenty: their per-minute rates stay slightly low. No typical value is invented.
- The time played keeps its truncation to the second, and a player whose presence is known but whose window is empty
  keeps the stored value: both are the formula of the sync, unchanged.
- A replay publication that changes a time played by a few seconds after the notes were computed does not replay the
  notes: they catch up at the next scoring revision or the next recomputation of that player.
- A WAL batch written by an older binary and replayed after the upgrade inserts a kickoff without its source; the next
  revision change or `levelup backfill-kickoff` fills it.
