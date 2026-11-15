# ADR 0047 — Scoring revision and replay at startup

Status: accepted (2026-11-08); implemented in the KPI revision work of the same date (steps K1 and K2, reviewed in K3).
Source: decisions D14 and D15 of the KPI revision plan.
Relates to: ADR 0024 (LUSR v2), ADR 0026 (append-only tables and `_latest` views), ADR 0013 and ADR 0016 (single writer
per player database), ADR 0009 (expvar counters).

## Problem

The performance score and the LUSR are computed from formulas and parameters held in code. When a release changes one of
them, the stored history of every player was computed with the old formula and stays that way until someone reruns the
computation by hand. Before this ADR, that rerun was a manual tool.

## Decision

- **D14 — One stored revision per player.** Two constants name the revisions of the two formulas
  (`PerformanceScoreRev`, `LUSRRev` in `internal/sync/skill/scoring_revision.go`). `skill.ScoringRevision()` composes them
  into one value. Each player database stores the last value whose full recomputation succeeded in `sync_meta`, under
  `skill.SyncMetaScoringRevision`. A stored value different from the binary's value triggers a full recomputation. Nothing
  is noted until that recomputation has succeeded.
- **D15 — No on-screen indicator, one release-note line.** The recomputation runs in seconds and shows nothing in the
  interface. Each release that changes a formula adds one line to its release notes. No compaction is added.

## Mechanism

- `MatchRecomputer.ApplyScoringRevision(ctx, rev)` in `internal/sync/match_recomputer.go`:
  1. reads the stored revision through a read connection (`OpenReadForQuery`); equal means no work;
  2. runs the full recomputation through `exclusionRecomputes.Do`, which guarantees that a recomputation requested
     after the call has finished, even when another request absorbed it;
  3. only on success, takes the player writer lease and stores the new revision with `SetSyncMeta`;
  4. invalidates the player read caches.
- `internal/service/scoringrevision` runs one pass at startup, player by player, for each title that declares the LUSR
  capability. A player whose recomputation fails keeps its old revision and is retried at the next startup; the other
  players are not held back. Counters: `scoring_revision_appliquee_total`, `scoring_revision_a_jour_total`,
  `scoring_revision_erreurs_total`. The task is a background task of the server, excluded in demo mode.
- A player profile with no player database file is skipped with an Info log. The database is not created.

## Partial failure

`batchComputePerformanceScores` returns its write error after the cleanup and the summary, instead of `nil`. A match that
fails during the LUSR replay (state load, computation, score write, state persist) is counted, and the replay returns
`ErrPartialRecompute`. A match whose solver does not converge is not such a failure: the outcome depends on the match
inputs, so a retry would fail the same way and the revision would never be stored. Before reaching that classification the
computation falls back once on the same match without the kills and deaths observations (ADR 0024, amendment 2026-11-09);
only a second non-convergence is logged, counted apart (`skipped_not_converged`) and gets no new score. `ApplyScoringRevision` stores nothing on any error, partial included. In the interactive path
(`RecomputeAfterExclusion`), a partial replay is logged as a warning and the user action still succeeds, as before this ADR;
the pending flag stays set, so a repeated request recomputes.

## Known limits

- The fingerprint test (`TestScoringParametersMatchRevision`, package `sync`) hashes the numeric constants and weight tables
  of the skill configuration and of the performance score, and the default parameters of the TrueSkill 2 engine that are
  exported. Literals inline in other files are not hashed (the victory-score table of the composite rating, the team weight
  floor of the solver, unexported constants of the shadow and shared-access files). **A change of logic without a change of
  number is not detected: the maintainer must bump `PerformanceScoreRev` or `LUSRRev` by hand.**
- A profile that has a player database but no xuid is skipped and counted as an error: the recomputation joins on the xuid, would find no match and would clear every score. Its revision is not stored, so it is reported again at each start.

## Consequences

- The first startup after the release that changes a formula recomputes every player with a database once. Desktop
  users see a few seconds of background work.
- A later startup with the same binary does nothing for a player whose revision matches.
- Changing a formula or a parameter without bumping the revision is a review failure, enforced by the fingerprint test
  for the hashed parameters and by the delivery checklist for the rest.

## Amendment 2026-11-15 — performance score of the objective chains and Spartan life at 229 (release v8.2.0)

**Relates to** — the Decision above (one composed revision, one recomputation) and the fingerprint test.

Both revisions rise in the same release: `PerformanceScoreRev` = `perf-2026-11-15` and `LUSRRev` = `lusr-2026-11-15`. This is
one recomputation, not two: `ApplyScoringRevision` compares the single composed value `skill.ScoringRevision()` with the one
stored per player, and one `recompute` replays the performance scores and the LUSR together. Test
`TestMatchRecomputer_ApplyScoringRevision_DeuxMonteesUnSeulRecalcul` proves it (two components change, the notes are written
once more, not twice).

**What changed.**
- The performance score of the `arena_objectif` and `ranked_objectif` chains. The `objective_participation` metric (award
  points per minute) leaves their weight profile, which keeps the 13 combat metrics (sum 0.92). The objective and the team's
  support graft onto the combat score through the pure package `internal/sync/objectiveblend` (symmetric weighted mean, mid-rank
  percentiles against the player's own history in the same mode family, `α_obj` = 0.12 + 0.38 × share of the team objective,
  `α_sout` = 0.38 × share of the team support, sum capped at 0.50). Rows come from one query per player over both teams
  (`LoadObjectiveTeamRowsForPlayer`), which also serves the full replay; Halo 5 and a match without `match_objective_stats`
  have no facts, the metric is absent and the score is the combat score.
- `games.DefaultEffectiveHpToKill` 225 → 229 (and `effective_hp_to_kill` of the Halo Infinite title), measured on the
  "damage taken" counter of 1 614 films: 229 in 99.6 % of the lives without shield recharge, stable from October 2024 to
  November 2026 and across the playlists. The relative performance score is unchanged by it (its percentiles are scale
  invariant: 2 matches out of 684 differ, by rounding); the LUSR composite is unchanged beyond a player's first matches, where
  its reference is the absolute P80; the displayed Efficiency and Resistance shift by about 2 %.

**Hashed parameters.** The performance fingerprint now covers the objective blend settings (`AlphaMin`, `AlphaMax`,
`SupportSlope`, `MinFamilyMatches`) and `narrative.ObjectiveSupportFragWeight`. The fingerprint does not see a change of logic
without a change of number (the rank rule, the choice of the reference series, the support columns table): the maintainer bumps
the revision by hand.

**Measured effect** (copy of the local parc, 684 objective matches of tracked players): median change +0.5 point, 90th
percentile of the absolute gap 11.8, 31 % of the matches change tier (35 / 50 / 65 / 80); the other chains do not move.
