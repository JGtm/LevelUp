# ADR 0049 — Match kickoff: one source order, one write, one catch-up

Status: accepted (2026-11-18), write side implemented on the same date; the readers move in a later step of the same
work.
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
  unknown. Halo 5 has neither a replay nor arrivals: its kickoff is unknown, with no title branch.
- **D-2 — The source is stored next to the value.** New column `match_registry.real_start_source` (`film`, `api`, NULL =
  unknown), added by migration `shared_registry_real_start_source_v1` (column added, table not rebuilt). The estimate
  outcome (single arrival, spread arrivals, rejection) is a log line and a counter (`kickoff_source_*_total`,
  `kickoff_estimation_*_total`), never stored.
- **D-3 — The time played follows the stored kickoff.** `sync/kickoff.Resolve` resolves the kickoff, then computes each
  player's time played with the unchanged formula (`timeline.ComputeTimePlayedFor`, truncation included) on that
  kickoff (`timeline.GameplayTimeline`: unknown kickoff = the window starts at the match start). A player whose arrival
  is unknown, or whose window is empty, keeps the stored value, which is the API value written once.
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
  title that task walks (titles with the LUSR capability); a title whose catch-up fails does not replay its players at that startup, so no note is stored on a stale
  time played. The command `levelup backfill-kickoff` runs the same function, with a `--dry-run` that lists the gaps
  without writing.
- **D-6 — The notes replay once.** `PerformanceScoreRev` and `LUSRRev` move to `2026-11-18` with unchanged fingerprints
  (logic change only): on the first startup, the kickoff catch-up runs, then every player's notes are replayed once.

## Guard rails

- `archlint/kickoff_write_guard_test.go`: no `UPDATE` of `real_start_time`, `real_start_source` or `time_played_seconds`
  outside `internal/persist/kickoff_persister.go`; no insertion of these columns outside the sync insertion
  (`persist/shared_persister.go`) and a dated exemption list (demo corpus, Halo 5 roster rebuild); the names
  `t0_quality` and `film_match_start_ms` appear nowhere outside the migrations.
- `timeline.TestKickoffParametersMatchRevision`: the estimate thresholds and the source order are fingerprinted per
  `KickoffRev`; changing one without a new revision fails.
- Unchanged and still green: `archlint/no_raw_real_start_offset_test.go`, the anti-ART guards of
  `internal/sync/no_art_patterns_test.go`, `TestScoringParametersMatchRevision`, the dated `OpenReadWrite` allowlist (the
  three removed commands leave it, the new command enters it).

## Known limits

- `t0_quality`, `film_match_start_ms` and `playable_duration_seconds` stay in the table: DuckDB refuses
  `ALTER TABLE … DROP COLUMN` on `match_registry` (primary key, indexes, dependent views), verified in memory and on a
  copy of the real schema, and a rebuild of the shared registry is not taken for a column name. The first two have no
  writer and no reader; `playable_duration_seconds` is still written at sync because readers still read it until they
  move to the match chronology.
- A match without a replay keeps the estimate, which on old matches can sit in the first seconds of a countdown of about
  twenty: their per-minute rates stay slightly low. No typical value is invented.
- The time played keeps its truncation to the second, and a player whose presence is known but whose window is empty
  keeps the stored value: both are the formula of the sync, unchanged.
- A WAL batch written by an older binary and replayed after the upgrade inserts a kickoff without its source; the next
  revision change or `levelup backfill-kickoff` fills it.
