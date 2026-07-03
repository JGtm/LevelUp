# temporal — time-window helpers

Pure Go helpers for time-bucketing, period filtering, rolling means, and LOWESS smoothing. Stateless, no DB, no I/O.

## Exports

| Function | Purpose |
|---|---|
| `BucketByGranularity[T](rows, gran, period)` | Group rows into buckets (Day / Week / Month) within a period |
| `ResolveAdaptive(period) Granularity` | Pick the right granularity for a given period (1w → Day, 1y → Week, 5y → Month) |
| `FilterByPeriod[T](rows, period, now)` | Filter rows whose `StartTime()` falls within the period |
| `RollingMean[T](points, window, minPoints)` | Centered rolling mean, fixed window, with minimum-points guard |
| `RollingMeanAdaptive[T](points, minWindow, pct)` | Rolling mean with adaptive window (% of N, clamped to minWindow) |
| `LowessSmooth(points, alpha) []float64` | LOWESS regression smoothing (locally weighted, alpha = bandwidth) |

## Types

```go
type Period string  // "1w" | "1m" | "1y" | "all"
type Granularity string // "day" | "week" | "month"
type Bucket[T] struct { Start, End time.Time; Items []T; Count int }
type HasStartTime interface { StartTime() time.Time }
type Numeric interface { ~int | ~int64 | ~float64 }
```

## Engagement (score + coefficients)

Le package couvre AUSSI le sous-système d'engagement (fichiers `engagement_*.go`),
au-delà du bucketing/smoothing ci-dessus.

| Function | Purpose |
|---|---|
| `ComputeEngagementScore(EngagementScoreInput) (domain.EngagementScoreResult, error)` | Score d'engagement d'un match (pace joueur/équipe/lobby + baseline history) |
| `ComputeEngagementCoefficient([]RatioSample) (*CoefficientResult, error)` | Coefficient d'engagement à partir d'échantillons de ratio |

Types : `EngagementScoreInput`, `CoefficientResult`, `RatioSample` (+ `domain.EngagementScoreResult`).

Sentinel errors : `ErrInsufficientData`, `ErrMatchTooShort`, `ErrInvalidBoundaries`
(`engagement_score.go`) ; `ErrInsufficientCoefHistory` (`engagement_coefficients.go`).

Files: `engagement_curve.go`, `engagement_weights.go`, `engagement_score.go`,
`engagement_coefficients.go`. Consumers: `internal/sync/engagement.go` (per-match
persistence) and `internal/service/engagement_player_service.go`.

## Examples

### Bucket match rows by week over 1 month

```go
import "levelup/go-api/internal/analysis/temporal"

period := temporal.Period("1m")
gran := temporal.ResolveAdaptive(period) // → Day
buckets := temporal.BucketByGranularity(matchRows, gran, period)
// buckets[i].Start, buckets[i].End, buckets[i].Items
```

### Rolling K/D over the last 20 matches

```go
kdValues := []float64{1.2, 0.8, 1.5, 2.0, ...}
rolling := temporal.RollingMean(kdValues, 20, 5)
// rolling[i] = average of kdValues[i-9..i+10] when ≥ 5 points are available
```

### LOWESS smoothing for "form score" timeline

```go
form := []float64{...}            // raw per-match scores
smoothed := temporal.LowessSmooth(form, 0.3) // 30% bandwidth
// smoothed[i] = locally weighted regression at index i
```

## Tests

`bucket_test.go`, `lowess_test.go`, `period_test.go`, `rolling_test.go` — unit tests, no fixtures, no DB.

## When NOT to use this package

- If your bucketing needs DB-side aggregation (volume too large to load in memory) → write SQL in `platform/duckdb/`.
- If your "smoothing" is just a moving average with N=window → `RollingMean` is enough; LOWESS is only needed when the curve has heteroskedastic noise.

## Consumers (audit history)

- `service/timeseries_service.go` — cumul, rolling K/D, EWMA, score/min
- `service/squad_service_v2.go` — Form Score (LOWESS) per player
- `service/synthesis_service.go` — heatmap day×hour bucketing
