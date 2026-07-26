# narrative — match-level storytelling primitives

Pure Go primitives for **describing** a match beyond raw stats : impact roles, dominance/comeback flags, intensity profiles, kill cadence, encounter badges, first-events, participation radar (6 axes).

ADR : `docs/adr/0004-narrative-engine.md`.

## Exports

| Function | Purpose |
|---|---|
| `IdentifyImpactRoles(participants) []ImpactRoleResult` | 8 roles per match per player (top_killer, silent_hero, …) |
| `ResolveDominanceBadge(flag) *DominanceBadge` | Dominance / comeback / debacle badge from `canonical.DominanceFlag` |
| `ComputeEncounterBadges(stats, ordinal) []EncounterBadge` | Nemesis / souffre-douleur / ally+ ordinal badges |
| `ComputeFirstEventsPerMatch(events, xuid, matchIDs) []FirstEventsRow` | First kill + first death per match for ONE player (canonical events) |
| `ComputeFirstEventsByActor(events, xuids, matchIDs) map[string][]FirstEventsRow` | Same aggregation for N players, from actor-resolved events (`FirstEventActor`) — squad surface |
| `ComputeMatchIntensityProfiles(events) []MatchIntensityProfile` | Match phase intensity (4 buckets : opening / mid / late / final) |
| `ComputeCadenceProfiles(...)` | Kill cadence by 60s buckets per player |
| `ComputeParticipationProfile(...)` | 6-axis radar : Combat / Survie / Support / Score / Objectif / Impact |
| `NormalizeIntensityBuckets([]int) []float64` | Helper : buckets → 0-1 normalized |
| `AllParticipationAxes() []ParticipationAxis` | List of the 6 axes (for UI rendering order) |
| `DefaultThresholds(modeFamily) ParticipationThresholds` | Per-mode thresholds (BTB ≠ Slayer ≠ Firefight) |

## Key types

### ImpactRoleResult — 8 roles

```go
type ImpactRole string

const (
    RoleTopKiller        ImpactRole = "top_killer"
    RoleSilentHero       ImpactRole = "silent_hero"
    RoleFalseBrother     ImpactRole = "false_brother"
    RoleAnchor           ImpactRole = "anchor"
    RoleLastCasualty     ImpactRole = "last_casualty"
    RoleLastGroupKill    ImpactRole = "last_group_kill"
    RoleFirstGroupDeath  ImpactRole = "first_group_death"
    RoleClutch           ImpactRole = "clutch"
)

type ImpactRoleResult struct {
    XUID    string
    Primary ImpactRole // mandatory
    Secondary *ImpactRole // optional
    Score   float64    // 0..1
}
```

Each player gets at most 2 roles per match.

### ParticipationProfile — 6-axis radar

```go
type ParticipationAxis string

const (
    AxisCombat    ParticipationAxis = "combat"
    AxisSurvival  ParticipationAxis = "survival"
    AxisSupport   ParticipationAxis = "support"
    AxisScore     ParticipationAxis = "score"
    AxisObjective ParticipationAxis = "objective"
    AxisImpact    ParticipationAxis = "impact"
)

type ParticipationProfile struct {
    XUID  string
    Axes  map[ParticipationAxis]float64 // each in [0, 100]
    Raw   map[string]float64            // raw values for tooltip debug
}
```

## Examples

### Match View : 8 roles for all participants

```go
import "levelup/go-api/internal/analysis/narrative"

roles := narrative.IdentifyImpactRoles(matchDetail.Participants)
// roles[i] = { XUID, Primary, Secondary?, Score }
// Render in <PlayerDetailPanel> with localized role labels via fields.toml
```

### Squad V2 : 6-axis radar averaged across N matches

```go
profiles := make([]narrative.ParticipationProfile, 0, len(squadMembers))
for _, member := range squadMembers {
    profile := narrative.ComputeParticipationProfile(
        member.MatchParticipations,
        narrative.DefaultThresholds(modeFamily),
    )
    profiles = append(profiles, profile)
}
// Convert to RadarChart payload : 1 series per member, 6 axes.
```

### Home : dominance badge on a recent match

```go
match := matchSummaries[0]
if badge := narrative.ResolveDominanceBadge(match.DominanceFlag); badge != nil {
    // badge.Label, badge.ColorToken → render in <DominanceBadgePill>
}
```

## Intensity 4-phase profile

`ComputeMatchIntensityProfiles` splits a match into 4 chronological buckets :

| Phase | Window | Use |
|---|---|---|
| Opening | 0–25% match duration | Early setup, who pushed first |
| Mid | 25–50% | Trade phase |
| Late | 50–75% | Pressure, comebacks |
| Final | 75–100% | Clutches, last group fights |

Intensity = events per second normalized 0–1. Used by `<TimeseriesIntensityChart>` (Heatmap match × phase) on Timeseries page.

## Tests

`*_test.go` — 300+ unit tests covering each role detector, dominance flag resolution, encounter ordinals, intensity edge cases (empty match, single event, ties), participation thresholds per mode family.

## Title-aware notes

- **Headshot weighting in `Combat` axis** is reduced (×0.7) for titles where headshots aren't a primary stat (ex: future Halo 3 portage with melee-heavy gameplay).
- **`first_group_death` window** is configurable via `DefaultThresholds(modeFamily)` — Firefight uses a wider window than Slayer.
- **`ResolveDominanceBadge`** returns `nil` if the title doesn't supply `DominanceFlag` (graceful degradation, cf. ADR 0004).

## Consumers

- `service/match_view_service.go` — single-match roles + 6-axis radar
- `service/squad_service_v2.go` — N-match aggregated roles + radar averaged
- `service/home_service.go` — top-role badge on recent matches
- `service/timeseries_service.go` — intensity heatmap match × phase
