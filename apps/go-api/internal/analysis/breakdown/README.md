# breakdown — aggregation by map / mode / playlist

Pure Go helpers for grouping match rows by structural dimensions (map, mode, playlist) and computing per-group aggregates. Stateless, no DB, no I/O.

## Exports

| Function | Purpose |
|---|---|
| `ByMap(rows) []MapAggregate` | Group by map name; computes count + winrate + avg KDA |
| `ByMode(rows) []ModeAggregate` | Group by mode (full pair_name) |
| `ByModeCategory(rows) []ModeAggregate` | Group by mode category (custom Halo : `Slayer`, `BTB`, `Ranked`, `Fiesta`, `Husky Raid`, `Firefight`, `Other`) |
| `ByPlaylist(rows) []PlaylistAggregate` | Group by playlist (Halo Infinite playlist UUID) |
| `CompareToHistorical(session, historical) []MapDelta` | Per-map delta : current session vs historical baseline (winrate diff, KDA diff) |
| `CompareByKey(session, historical []KeyedAggregate) []KeyedDelta` | Generic per-key delta : session vs historical for ANY dimension (mode, playlist…) where `CompareToHistorical` (map-only) does not apply |

## Types

```go
type Row struct {
    MapName      string
    ModeName     string
    PlaylistName string
    Outcome      canonical.Outcome
    KDA          *float64
}

type MapAggregate struct {
    MapName    string
    Count      int
    Wins       int
    Losses     int
    Ties       int
    DNFs       int
    Winrate    float64
    AvgKDA     *float64
}

type ModeAggregate    // same shape, ModeName instead of MapName
type PlaylistAggregate // same shape, PlaylistName

type MapDelta struct {
    MapName        string
    SessionCount   int
    SessionWinrate float64
    HistWinrate    float64
    DeltaWinrate   float64
    SessionKDA     *float64
    HistKDA        *float64
    DeltaKDA       *float64
}

// Forme pivot générique (map / mode / playlist) identifiée par une clé stable.
type KeyedAggregate struct {
    Key   string
    Label string
    Counts
    AvgPerformanceScore *float64
}

// KeyedDelta : session vs historique pour une même Key (sémantique de MapDelta,
// générique à toute dimension). WinRateDelta = session.WinRate − historical.WinRate.
type KeyedDelta struct {
    Key                      string
    Label                    string
    Session                  Counts
    Historical               Counts
    WinRateDelta             float64
    AvgPerformanceScoreDelta *float64
}
```

## Examples

### Top maps for the synthesis page

```go
import "levelup/go-api/internal/analysis/breakdown"

aggregates := breakdown.ByMap(rows)
sort.Slice(aggregates, func(i, j int) bool {
    return aggregates[i].Count > aggregates[j].Count
})
top5 := aggregates[:min(5, len(aggregates))]
```

### Compare a session to historical baseline (Squad V2)

```go
sessionAggs := breakdown.ByMap(sessionRows)
historicalAggs := breakdown.ByMap(allHistoricalRows)
deltas := breakdown.CompareToHistorical(sessionAggs, historicalAggs)
// deltas[i].DeltaWinrate > 0 → session better on this map than historical
```

### Compare a session to historical baseline on a NON-map dimension (mode, playlist)

```go
// Build KeyedAggregate slices (Key = mode / playlist id), then:
deltas := breakdown.CompareByKey(sessionAggs, historicalAggs)
// Generic form of CompareToHistorical : same delta semantics, any dimension.
// Keys absent from the session are ignored (we only rank what was played in scope).
```

### Mode categories on the home dashboard

```go
modes := breakdown.ByModeCategory(rows)
// modes contains at most 7 buckets : Slayer / BTB / Ranked / Fiesta / Husky Raid / Firefight / Other
```

## Mode category mapping

`ByModeCategory` uses `analysis.InferModeCategoryFromPairName` (cf. `halo-modes` skill) which infers from prefixes :

- `BTB:` → `BTB`
- `Ranked:` → `Ranked`
- `Fiesta:`, `Super Fiesta:` → `Fiesta`
- `Firefight:`, `KOTH Firefight:` → `Firefight`
- `Husky Raid:` → `Husky Raid`
- otherwise → `Slayer` if 4v4, else `Other`

## Tests

`by_map_test.go`, `by_mode_test.go`, `by_playlist_test.go`, `compare_test.go` — unit tests with hand-crafted row fixtures.

## When NOT to use this package

- For 1000+ rows → consider DB-side `GROUP BY` in `platform/duckdb/` to avoid memory pressure.
- For non-categorical breakdowns (e.g. by hour-of-day) → use `temporal/bucket.go`.

## Consumers

- `service/synthesis_service.go` — top maps, top modes
- `service/squad_service_v2.go` — per-map session vs historical comparison
- `service/career_service.go` — playlist preferences
