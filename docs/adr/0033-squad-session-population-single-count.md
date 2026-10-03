# ADR 0033 — One population, one count, for a squad session

**Status**: Accepted (2026-09-09)

**Branch**: `wt/escouade-hors-cadre`

**Relates to**: [ADR 0008](0008-title-path-isolation.md) (per-title paths, global xuid),
[ADR 0029](0029-multi-user-player-ownership.md) (player ownership). Supersedes nothing; it
writes down the rule that commit `3862ff083` (2026-08-02) applied to one surface only.

---

## Context

The Squad page shows a session and its numbers. Between 2026-08-02 and 2026-09-09 the same
defect was reported **twice**: the count of matches shown next to the session did not match
the population of the tables and charts below it.

### What was measured (2026-09-09, session of 27 August 2026)

Player `JGtm` (`2533274823110022`), composition `{Madina97294, Chocoboflor}`, direct SQL on
`shared_matches_v2`:

| Surface | Source | Value |
|---|---|---|
| Rail (L2) session pill | `/filters/resolve` → `session_options.all_sessions[].match_count` | 7 |
| Rail (L2) trailing counter | `/filters/resolve` → `counts.total_matches_after_filters` | 7 |
| Session multi-select | `teammates` → `composition_sessions[].match_count` | 4 |
| Tables, charts, KPIs | `allSquadRows` (same population as above) | 4 |

`FiltersService.ResolveFiltersFromRowsAt` computes over the **main player's** rows, filtered
by match context, session and cascade. It has no knowledge of the selected composition, and
none of `filter_exact_composition`. The rail is therefore **structurally >=** the page body
whenever the exclusive option is on. The three matches it counted and the page did not are
`a0c36016` (Nilton410, 4th in the with-friends teammate ranking), `faff9935` and `94a28b8b`
(passivemarquise, 35th) — each excluded because a *known* teammate outside the selection was
on the main player's team.

### The second thing the measurement settled

The report attributed the gap to a player who **crashed out of a match** (`2cf24f30`,
19:39 UTC: `present_at_completion = FALSE` for Chocoboflor, a bot `bid(3.0)` joining in
progress in his place). That match is **kept** by the current engine, and it must be: neither
`Q30SquadMatchesSharedQuery` nor `Q32bMainTeamParticipantsTemplate` filters on presence, and
bots are outside the known-teammate pool (`NOT LIKE 'bid(%'`).

The product rule behind this is explicit (user, 2026-09-09): *leaving a match is not leaving
the session*. A game crash, a PC reboot or a disconnect must never remove a match from the
squad's session. Nothing enforced that rule in code — it held by accident.

## Decision

**1. Membership of a match in a composition's session is defined once, and does not depend on
presence at the end of the match.** A match belongs to the population when:

- the main player played it, **and**
- every selected teammate appears on the main player's allied team,

regardless of `present_at_beginning`, `present_at_completion`, `left_in_progress`,
`joined_in_progress` or `last_leave_time`. Under the optional exclusive filter
(`filter_exact_composition = true`), one further condition applies: no *known* teammate
outside the selection (top-50 with-friends teammates ∪ configured friends) is on that team.
Lobby fills, bots and opponents never break a composition.

**2. `composition_sessions[].match_count` is the single source of a session count in squad
context.** Every surface — session multi-select, rail pill, rail trailing counter, KPIs,
tables, charts, briefing — reads that number and no other. `/filters/resolve` counts remain
a **loading fallback only**, valid until the teammates response arrives, and never a value
displayed alongside the composition count.

**3. A discarded match is shown, not silently dropped.** When the exclusive option removes
matches, the UI states the gap (`4 / 7`) and names the reason per match: which known
teammate was on the team. A number that shrinks without explanation is what produced two
identical bug reports.

**4. Two ratchets keep it that way.**

- `internal/service/teammates/no_presence_filter_test.go` forbids any presence column in the
  squad population queries (`queries_squad.go`, `squad_repo*.go`). Empty, dated allowlist.
- `apps/web/src/features/squad/singleCountSource.guard.test.ts` forbids any file under
  `features/squad/` from deriving a match count from `total_matches_after_filters` or
  `session_options`. Empty, dated allowlist.

## Consequences

- `TeammatesPageResponse.CompositionSessions` becomes `[]CompositionSessionEntry`: the
  post-filter count (`match_count`, unchanged meaning), the pre-filter count
  (`match_count_roster`) and the discarded matches with the responsible teammates named.
- `filterExactComposition` returns kept **and** discarded rows from one pass — the exclusion
  reason cannot drift from the exclusion itself.
- `PeriodSessionRail` gains an optional count injection. Pages that do not inject keep their
  current rendering; only the squad page switches source.
- The default (option off) is unchanged: the population is the roster intersection, and this
  ADR does not revisit the 2026-08-02 product decision that made the exclusive filter opt-in.

## Alternatives rejected

- **Teach `FiltersService` about the composition.** It would put squad semantics into a
  service every page shares, and give two engines the right to answer the same question — the
  exact shape of the defect.
- **Hide the rail counter on the squad page.** Cheapest, and it was rejected by the user:
  the gap between "my session" and "what this composition played" is information, not noise.
- **Count presence-complete matches only.** Directly contradicts decision 1, and would drop
  the crash match the report was filed about.
