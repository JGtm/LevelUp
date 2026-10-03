# ADR 0032 — Displaying the score of round-decided modes

**Status**: Accepted (2026-08-29)

**Branch**: `feat/v75`

**Relates to**: [ADR 0006](0006-canonical-indicators.md) (canonical indicators), [ADR 0019](0019-collect-persist-anti-art.md) / [ADR 0026](0026-append-only-art-eradication.md) / [ADR 0030](0030-persist-write-aggregates.md) (INSERT-only writes, why a new column needs a backfill), [ADR 0025](0025-title-agnostic-minimal-viable-window.md) (title-agnostic: config tables, never a slug comparison)

---

## Context

In modes that are decided by **rounds**, the team score returned by the Halo API
(`Teams[].Stats.CoreStats.Score`) is a **cumulative point total across every round**. It does
not say who won.

Measured on 2026-08-29 over the whole corpus — 1 942 matches re-fetched from `GetMatchStats`,
0 errors (`.ai/V7.5/RAPPORT_MANCHES_2026-08-29.md`):

- 57 matches (2,9 %) are played over more than one round, across 9 game variants.
- **4 Oddball matches award the win to the team with FEWER points** (e.g. `293a763e`: the
  app displayed `181 - 186` on a victory that was 2 rounds to 1). The product presented a win
  as a loss on the score line.
- `CoreStats.RoundsWon / RoundsLost / RoundsTied` had been declared in the payload structs
  since the Go port and was **read nowhere**: no mapper, no column, no canonical field.

An obvious rule was tried first and **refuted by the same measurement**: "more than one round
therefore show rounds". Arena CTF is played in two **halves** (`rounds_total = 2`) while its
score is the total number of captures; the rule would have replaced `2 - 3` with `0 - 1`.

Two more traps surfaced: 4 abandoned matches credit one round to one side and zero to the
other (reading a single side would call them single-round), and 18 matches contain a **tied
round** — on one of them each side won a round and a winner still exists, so the round tally
does not name anybody.

## Decision

**1. The score displayed for a match is chosen by one rule, in one place** —
`analysis.ReadTeamScore` (`internal/analysis/team_score_display.go`), pure and
title-agnostic. Three cumulative conditions must hold to display rounds:

```
game_variant_name is declared in regulation.toml [rounds_decide]
AND rounds_total (max of both sides) >= 2
AND rounds_won(A) != rounds_won(B)
```

Any condition failing falls back to points — that is, to the behaviour that predates this
ADR. A NULL column, a title whose API does not publish rounds (Halo 5), an undeclared
variant: none of them degrade anything, they simply do not win the new reading.

**2. Detection is a MEASURED declaration, never a heuristic.** `[rounds_decide]` is a new
table in `config/titles/{slug}/mappings/regulation.toml` (schema 3), keyed by
`game_variant_name`, exactly like `[regulation_seconds]` and `[score_target]` above it. A
`false` entry is rejected at load time: the absence of a key already means "no", and two ways
of saying no eventually contradict each other. Initial content: the three Oddball variants —
the only ones where the current display lies. What is deliberately absent is documented in
the file itself (One Flag CTF and friends already display their round count as their score;
Arena CTF is halves).

**3. The data is persisted, and the label is built in one place.** Three additive columns on
`match_registry` (`team_0_rounds_won`, `team_1_rounds_won`, `rounds_total`), filled at INSERT
by the sync. `analysis.TeamScoreLabel` became the single builder of the `X - Y` label,
replacing five copies that had already diverged into two formats; a grep guard-rail forbids
a sixth.

**4. The API carries the reading, the client carries the words.** `score_kind`
(`points` | `rounds`) travels with `score_label`, plus `score_mine` / `score_theirs` and
`score_points_label` on the match view. No French or English word for "rounds" is produced
server-side.

**5. In the replay, live comes from the film and the verdict comes from the API.** The
banner keeps the in-round points and the round pips, and derives its running tally from the
film (it must follow playback). The end-of-match panel — and its twin repainted into the
exported video — take the round tally from the API, so the replay and the match view can
never announce two different results.

## Consequences

- A new column that only the API can fill needs a dedicated backfill: `persistMatchRegistry`
  is a bare INSERT (ADR 0019/0026/0030), so a re-sync never rewrites an existing row.
  `cmd/backfill-team-rounds` does it, restricted by default to the declared variants — 26
  matches instead of 1 942. Future matches are filled at sync time for every variant, so
  declaring a new variant later only needs a second pass of the same tool.
- **A DuckDB view freezes its `SELECT *` at creation.** `v_match_full` is
  `SELECT mr.* FROM match_registry`; the new columns were invisible to it until a second
  migration step recreated it. Any future column added to `match_registry` and read through
  that view needs the same pair of steps.
- Adding a title costs one TOML table. No slug comparison anywhere; the tables are injected
  per title through the wiring, and a title that declares nothing keeps points.
- The unified `X - Y` label format (spaces) is a visible change on the match view and the
  home card, which previously used `X-Y`.

## Alternatives rejected

- **Deriving the rule from the data alone** (`rounds_total >= 2`): refuted by measurement,
  see Context.
- **Deriving the round count from the film** for the whole product: only ~1 % of matches have
  a replay artifact, and the API covers 100 %.
- **Passing the round count through the replay artifact**: would have required re-cooking
  every existing artifact for a number the API already serves, and would have created a
  second source for a value that must read the same everywhere.
