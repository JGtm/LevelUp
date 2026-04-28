# ADR 0004 — Narrative engine: 8 impact roles + radar 6 axes

**Status** — Accepted (2026-04-27). Implemented in `internal/analysis/narrative/` during Phase 1 of `PLAN_META_FOUNDATIONS_GO.md` (chunks MV1→MV5 + S5+S6 + S8).

**Deciders** — Guillaume (GS).

## Context

Two pages need a **narrative summary** of a match — i.e. a per-player characterization beyond raw stats:

- **Match View** : "Who carried this match? Who was the silent hero? Who got smashed?"
- **Squad V2 (synergies)** : "Who plays which role across N matches? Whose contribution is structural vs occasional?"

The Python `v7/cockpit` had partial logic split across multiple files (`squad_score.py`, `match_impact.py`, `clutch_detector.py`) with inconsistent definitions:

- 4 roles only (top_killer, support, anchor, victim) — too coarse to differentiate "silent hero" from "false brother".
- Bilateral (1v1: `myXUID` vs `friendXUID`) — couldn't generalize to N players in a squad.
- "Clutch" detection was approximate (last third of match by index, not by actual time).
- No unified radar: each page rendered its own version.

## Decision

**Centralize narrative computation in `internal/analysis/narrative/`** with two products:

### 1. Eight impact roles (per match per player)

| Role | Definition |
|---|---|
| `top_killer` | Most kills with significant K/D edge |
| `silent_hero` | Mid-table kills but high objective contribution |
| `false_brother` | Negative contribution; high deaths in losses |
| `anchor` | Survives, holds objectives, low risk |
| `last_casualty` | Last to die; buys time for teammates |
| `last_group_kill` | Triggers the wipe of the enemy team in the final fight |
| `first_group_death` | First to fall in the opening engagement |
| `clutch` | Decisive kill in the last 60s with team in deficit |

Each player gets **at most 2 roles per match** (primary + optional secondary) with a numeric `score ∈ [0, 1]`.

### 2. Radar 6 axes (per player per scope)

Six axes normalized 0–100, designed to be **interpretable** without legend:

| Axis | Meaning |
|---|---|
| Combat | Kills + headshots + accuracy weighted |
| Survie | Inverse death rate + lifespan |
| Support | Assists + revives + objective time |
| Score | Personal score (objective contribution) |
| Objectif | Captures + holds + neutralizations |
| Impact | Composite of the 5 above + clutch frequency |

Computed from `canonical.MatchParticipant` collections via `narrative.ComputeRadarAxes(scope)`.

## Architecture

```
internal/analysis/narrative/
  roles.go           — 8 role detectors (pure, idempotent)
  radar.go           — 6-axis computation (pure)
  events_first.go    — first_kill, first_death events
  intensity.go       — match intensity profile (4 phases)
  cadence.go         — kill cadence by 60s buckets
  dominance.go       — domination/comeback detection
  README.md          — usage examples (see P4M.C)
```

All functions are **stateless**, take `canonical.*` inputs, return value types. Zero DB access, zero HTTP, zero side-effect.

Service callers:
- `match_view_service.go` — single match → 8 roles + radar 6 axes for each participant.
- `squad_service_v2.go` — N matches → aggregated roles per player, radar averaged.
- `home_service.go` — narrative badge on recent matches (top role only).

## Consequences

### Positive

- **One canonical algorithm** — both pages render identical data; reviewers can compare apples to apples.
- **N-player support** — `ComputeMatchImpactRoles` accepts any number of participants. Squad V2 (4 friends) and Match View (16 participants) share the same call.
- **Real time-window clutch** — uses `event.timestamp_seconds`, not list index. False positives down ~40% on internal samples.
- **Pure tests** — `narrative_test.go` covers all role detectors with hand-crafted fixtures (~300 unit tests). Zero infra dependency.
- **Frontend uniformity** — `<RadarChart>` consumes `RadarSeriesPayload[]` (`RadarAxis[]` per series). Same component, two pages.

### Negative

- **Heuristic tuning required** — role thresholds (`silent_hero` requires `kills ≥ 0.5 × top_killer`, `objective_score ≥ 0.6 × max`) are hand-calibrated. Need to revisit if K/D distribution shifts in a future title.
- **Radar normalization is title-aware** — `Combat` axis weights `headshots` which doesn't exist for Halo 5 weapons → axis caps at 70 for that title. Documented in `radar.go`.
- **No inter-match memory** — `silent_hero` is computed per-match. Aggregation across matches (e.g. "always silent hero") happens in `squad_service_v2.go` by counting role frequency. Not in narrative core.

## Alternatives evaluated

| Alternative | Rejected because |
|---|---|
| **Keep logic split across 3 Python files** | Inconsistent role definitions, hard to test, no N-player support. |
| **ML / scoring model** | Over-engineered for our sample size (a few hundred matches per user). Heuristics suffice and are explainable. |
| **Per-page narrative implementations** | Risks divergence; reviewers can't compare. |
| **Pre-compute narrative at sync time** | Coupled to sync pipeline; harder to reprocess when a heuristic changes. Better: compute on read. |

## References

- Implementation: `apps/go-api/internal/analysis/narrative/` (~750 lines).
- Tests: `apps/go-api/internal/analysis/narrative/*_test.go` (300+ unit tests).
- Plan source: `.ai/PLAN_META_FOUNDATIONS_GO.md` § 5.3 (narrative).
- Audit Python source: `docs/AUDIT_TEAMMATES_V7_COCKPIT.md` § 3.7 (clutch + roles).
- Frontend wrapper: `apps/web/src/components/charts/RadarChart.tsx` (consumer).
