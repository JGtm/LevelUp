# charts — ECharts wrappers catalog

12 reusable client-side chart wrappers built on top of `echarts-core` + `echarts-for-react`. Each wrapper exposes a pure builder (`buildXxxOption`) for unit tests + a React component for rendering.

ADR : `docs/adr/0001-charts-stack-echarts.md`. Live sandbox : `/lab/charts`.

## Catalog

| # | Wrapper | Use case | Page consumers |
|---|---|---|---|
| 1 | `<TimeseriesLineChart>` | Multi-series time / category / value lines | TimeseriesPage cumul + EWMA + intensity, MatchView S4, Career xp_history |
| 2 | `<BarStackedChart>` | Stacked bars (outcomes, components) | Synthesis bipolaire, Squad V2 cadence, Squad V2 HS+PK |
| 3 | `<BarGroupedChart>` | Side-by-side bars (filtered vs total) | Citations medals distribution |
| 4 | `<HistogramChart>` | Distribution buckets | TimeseriesPage K/D, kills, accuracy, score/min, rolling WR |
| 5 | `<ScatterChart>` | Multi-series correlation scatter | TimeseriesPage correlations (5 pairs) |
| 6 | `<DonutChart>` | Pie/donut with semantic slice colors | SessionCompare outcomes |
| 7 | `<Heatmap2DChart>` | 2D heatmap (sequential or divergent palette) | TimeseriesPage intensity day×hour, Synthesis activity, Squad V2 player×map |
| 8 | `<RadarChart>` | N-series 6-axis radar | MatchView participation, Squad V2 radar |
| 9 | `<OutcomeSequenceTape>` | RLE narrative band of recent outcomes | HomePage, MatchHistoryPage, SquadV2Page |
| 10 | `<TimeseriesKdaBars>` (page-specific) | Bars K + bars D + line K/D ratio (dual yAxis) | TimeseriesPage summary |
| 11 | `<FirstBloodLanes>` | One lane per player: first-kill / first-death timing clouds + median advance window | Squad "Dynamique" tab, Timeseries "Progression" tab, Session chart stack |

> Wrappers 10–11 are kept in `features/timeseries/` (not in this folder) because they compose `<ChartCard>` directly with custom `buildOption` and aren't reusable elsewhere.

## Common API contract

All generic wrappers (1–9) consume `ChartSeries<T>[]` :

```ts
interface ChartSeries<T> {
  key: string                 // unique series identifier
  meta?: { gamertag?: string; mode_family?: string; ... }
  datapoints: T[]             // shape depends on chart kind
  labelKey?: string           // optional i18n key for series name
  colorToken?: string         // optional override for series color
}
```

Each chart kind has its own datapoint type (`ChartPoint2D`, `ChartPointStacked`, `ChartPointHeatmap`, `ChartPointHistogram`, `ChartPointScatter`, `ChartPointDonut`).

## Wrapper details

### `<TimeseriesLineChart>`

Multi-series line chart with X-axis type `time | category | value`.

```tsx
<TimeseriesLineChart
  series={cumulativePointsToSeries(points, { key: 'kd', name: 'K/D cumulé' })}
  xAxisType="time"
  outcomeMarkers={false}
  height={300}
/>
```

Props : `series`, `xAxisType`, `outcomeMarkers` (color markers by outcome), `seriesNameResolver`, `height`, `loading`, `error`, `emptyMessage`.

### `<BarStackedChart>` + `<BarGroupedChart>`

Stacked bars vs grouped bars from `ChartSeries<ChartPointStacked>[]` :

```ts
type ChartPointStacked = {
  category: string
  components: Record<string, number>  // sub-key → value
}
```

`componentColors` maps sub-key → SemanticToken. `componentOrder` controls bar stack/group order.
`tooltipHideZero` drops zero-valued sub-keys from the tooltip; `tooltipComponentNote(category,
component)` appends a localized note next to a segment value (used by the match assists chart
for "of which stolen: N"). Either one switches the tooltip to the custom formatter.

### `<HistogramChart>`

Single-series bar histogram from `ChartSeries<ChartPointHistogram>[]` :

```ts
type ChartPointHistogram = {
  binStart: number
  binEnd: number
  count: number
}
```

Props : `colorToken`, `xAxisLabel`, `yAxisLabel` (default "Matchs"), `formatBin` (custom bin label).

### `<ScatterChart>`

Multi-series scatter from `ChartSeries<ChartPointScatter>[]` (`{ x, y }`). Props : `seriesColorTokens` map key→token, `seriesNameResolver`, `symbolSize`, `xAxisLabel`, `yAxisLabel`.

### `<DonutChart>`

Single-series pie/donut from `ChartSeries<ChartPointDonut>[]` (`{ name, value }`). Props : `sliceColors` (slice name → SemanticToken), `innerRadius`, `outerRadius`, `showPercent`.

### `<Heatmap2DChart>`

Single-series 2D heatmap from `ChartSeries<ChartPointHeatmap>[]` (`{ x, y, value, detail? }`). Props : `paletteMode: 'sequential' | 'divergent'`, `valueRange?: [min, max]`.

### `<RadarChart>`

N-series 6-axis radar. Doesn't consume the standard `ChartSeries<T>` — uses a dedicated `RadarSeriesPayload[]` shape with `axes: RadarAxis[]` (each axis carries `axis`, `value` 0–100, `raw` for tooltip).

Props : `axisLabels` (axis-key → label override), `seriesNameResolver`.

### `<OutcomeSequenceTape>`

Custom ECharts series rendering an RLE band of recent match outcomes with I-beam brackets (wins above, losses below). Specific to LevelUp's narrative storyline.

Matches carrying a `dominance` flag get a vertical **notch** crossing the band (3 px overhang on both sides) with a tooltip-background gutter — the gutter, not the hue, is what carries contrast. The notch has no density floor: below ~2 px per match, neighbouring notches merge (`clusterDominanceNotches` in `outcomeSequence.ts`, a pure helper); a merged group keeps its colour when all flags agree and turns muted when they differ.

```tsx
<OutcomeSequenceTape
  matches={[
    { matchId: '1', outcome: 'win', map: 'Streets' },
    { matchId: '2', outcome: 'loss', map: 'Argyle' },
    // ...
  ]}
  labels={{ win: 'Win', loss: 'Loss', tie: 'Tie', dnf: 'DNF' }}
/>
```

### `<FirstBloodLanes>`

Replaces the unreadable "first kill / first death per 15 s bucket" histogram. One horizontal lane per player, X axis = seconds since match start :

```tsx
<FirstBloodLanes
  data={[{ player: 'Gamertag', matches: [{ matchId: 'm1', firstKillSec: 47, firstDeathSec: 61 }] }]}
  maxSec={300}
/>
```

`null` = event absent from that match : excluded from the points and from the medians, still counted in the displayed coverage ("11/12 matchs"). Props : `data`, `maxSec` (default 300), `title` (defaults to the manifest title, `''` = no card header), `loading`, `error`, `emptyMessage`, `locale` (defaults to the app shell locale).

API payloads (`first_blood` on the Squad / Timeseries / Session page responses) are mapped to `data` — and `maxSec` derived (p99 rounded up to the minute, floor 300 s) — by `features/_shared/firstBlood.ts`. Never re-map a payload inline: that helper is the single adapter for the three surfaces.

Five series : one `custom` (rounded "advance window" bar spanning median first-kill → median first-death, pixel height, so neither `bar` nor `scatter` fits) plus four `scatter` (kill/death clouds via `symbolOffset ±14 px`, then the two median markers). Lanes are sorted by median first kill ascending and the category axis is `inverse`, so the fastest player is on top. Labels (gamertag, `méd. 50s → 1m04`, `+14s d'avance`) live in the left gutter through `axisLabel.formatter` + `rich`, anchored with `margin: 130 / align: 'left'` — nothing is drawn inside the plot area.

i18n : own shared manifest `lib/i18n/manifests/first_blood.toml` (FR + EN), resolved inside the component (same pattern as `FragSunburst`); the pure builder only receives resolved label callbacks. Pure model (medians by linear interpolation, sorting, time formats, geometry constants) : `firstBloodLanesModel.ts`.

## Primitives SVG pures (non-ECharts)

Not every chart in this folder is an ECharts wrapper. `Sparkline` is a **pure inline-SVG micro-trend** (flat hard-edge) — no axes, no tooltip, no lazy-loaded ECharts instance. It renders a single `<polyline>` plus a dot on the current value, colored by a semantic token. It is intentionally lightweight so it can be embedded per-cell (admin sync matrices) or inside a compact KPI tile (Explorer briefing) without the cost of a chart instance.

```tsx
import { Sparkline } from '@/components/charts/Sparkline'

<Sparkline values={[12, 9, 14, 8, 11]} token="outcome-win" width={120} height={28} ariaLabel="…" />
```

Props : `values: number[]`, `token?: SemanticToken` (default `info`), `width?` (default 96), `height?` (default 24), `ariaLabel?`. Returns `null` for an empty series.

Geometry lives in `sparklineGeometry.ts` (`sparklinePoints` / `lastPoint`) — pure functions, no DOM, unit-tested in `sparklineGeometry.test.ts` (no canvas mock required). Consumers : `features/admin/convergence/PostSyncMatrix`, `features/admin/sync/SyncCycleHistory`, `features/explorer` briefing tile (win-rate trend). **Not** to be confused with `OutcomeSparkline` in `features/palmares/PalmaresRelationsPage` (a categorical W/L outcome band, a different pattern).

## Color tokens

Wrappers consume `SemanticToken` strings (resolved at runtime via `tokenCssVar` / `resolveToken`). Common tokens :

- `outcome-win`, `outcome-loss`, `outcome-draw`, `outcome-dnf`
- `perf-tier-1` … `perf-tier-5`
- `divergent-pos`, `divergent-neutral`, `divergent-neg`
- `chart-series-1` … `chart-series-8` (cyclic palette for multi-series)
- `narrative-dominant`, `narrative-humiliation`, `narrative-remontada`, etc.

See `lib/accessibility/semantic-tokens.ts` for the full list. **Never use hex codes directly** in chart props.

## Testing pure builders

Each wrapper exposes `buildXxxOption(series, options)` for unit tests :

```ts
import { buildHistogramOption } from './HistogramChart'

const opt = buildHistogramOption(series, { colorToken: 'perf-tier-2' })
expect(opt.series?.[0].type).toBe('bar')
expect(opt.xAxis?.data).toEqual(['0–1', '1–2'])
```

Test counts (commit `b655d0f2` and after) : 8 Histogram + 9 Scatter + 9 Donut + 9 KdaBars = **35 unit tests** covering builder logic without React.

## Adding a new wrapper

1. Create `apps/web/src/components/charts/MyChart.tsx`.
2. Export `interface MyChartProps`, `interface ChartPointMyChart`, and `function MyChart(props)`.
3. Implement `buildMyChartOption(series, options): EChartsCoreOption` as a pure exported function (with `// eslint-disable-next-line react-refresh/only-export-components`).
4. Wrap in `<ChartCard series={series} buildOption={buildOption} loading={...} error={...} />`.
5. Add 5+ unit tests in `MyChart.test.ts` covering empty series, basic rendering, axes, color tokens.
6. Add a `<ShowcaseSection>` to `apps/web/src/features/lab/ChartsShowcasePage.tsx`.
7. Update this README's Catalog table.

## Live sandbox

Run `npm run dev` and navigate to `/lab/charts` for visual samples of all 11 wrappers with realistic demo data. Sandbox is hardcoded-strings-allowed (lint exception) — useful for visual regression checks.

## Reference

- ADR : `docs/adr/0001-charts-stack-echarts.md`
- Foundation guide : `docs/FOUNDATIONS_GUIDE.md` § 3.4
- Color tokens skill : `.claude/skills/color-tokens/SKILL.md`
- ECharts docs : https://echarts.apache.org/en/option.html
