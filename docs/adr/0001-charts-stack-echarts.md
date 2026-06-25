# ADR 0001 — Charts stack: ECharts

**Status** — Accepted (2026-04-28). Implemented across Phase 0 → Phase 3 of `PLAN_META_FOUNDATIONS_GO.md`.

**Deciders** — Guillaume (GS), with confirmation from review of 11 wrapper migrations.

## Context

The Python `v7/cockpit` app rendered all charts with **Plotly** (server-side figure construction, JSON payload, `react-plotly.js` rendering). The Go + React migration started with a mix:

- **Plotly** for legacy parity (figures emitted server-side as `*PlotlyFigurePayload`).
- **Recharts** for one early prototype on Match View (TimeseriesLineChart Recharts).

Constraints surfaced during Phase 0–2:

1. **Bundle size** — `react-plotly.js` + `plotly.js-basic-dist` weighs ~2.5 MB gzipped. We render at most 6 charts per page; the cost of loading Plotly upfront is disproportionate.
2. **Server-side coupling** — Plotly figures forced the Go service to know layout details (margins, axes, colors). Bad separation: data shape leaked into presentation.
3. **Recharts limits** — no native heatmap, no radar with multiple series, no custom series for `OutcomeSequenceTape` (RLE bands with brackets). Many charts couldn't be implemented without forking the lib.
4. **Theming and accessibility** — no consistent token-based color system; Plotly hex codes were duplicated everywhere.

## Decision

**Use ECharts (`echarts-core` + `echarts-for-react`) as the single chart library** for all client-side visualizations.

- Chart options are built **client-side** from raw data points sent by Go (no `*PlotlyFigurePayload` in DTOs anymore).
- Reusable wrappers in `apps/web/src/components/charts/` (11 wrappers shipped).
- Each wrapper exposes a pure `buildXxxOption` function (testable without React tree).
- Color tokens via `tokenCssVar()` / `resolveToken()` (cf. ADR 0003).

## Consequences

### Positive

- **Bundle gain** — `react-plotly.js` + `plotly.js` removed (commit `16fb335e`). Net diff: **−684 lines** of dead Plotly wrappers across Phase 3.
- **Coverage** — 11 wrappers cover all current chart needs: `TimeseriesLineChart`, `BarStackedChart`, `BarGroupedChart`, `Heatmap2DChart`, `HistogramChart`, `ScatterChart`, `DonutChart`, `RadarChart`, `OutcomeSequenceTape`, `TimeseriesCombatYield`, `TimeseriesKdaBars`. The last three are page-specific composites.
- **Testability** — pure builders: 8 to 17 unit tests per wrapper (`buildHistogramOption`, `buildDonutOption`, `buildKdaBarsOption`, etc.) without mounting React.
- **Server simplification** — Go DTOs lost 14 dead `*PlotlyFigurePayload` fields (cleanup Option B, commit `4b79a35d`). Services emit only `[]CumulativePoint`, `[]DistributionBucket`, `[]CorrelationDataPair`, etc.
- **Visual sandbox** — `/lab/charts` route renders all 11 wrappers with sample data (commit `345f6b32`). Living documentation.

### Negative

- **ECharts learning curve** — option API is verbose (`tooltip.formatter`, `series[].itemStyle`, etc.). Mitigated by `_utils.ts` shared base styles (`axisBase`, `tooltipBase`, `legendBase`).
- **Lazy loading required** — `echarts-for-react` is lazy-loaded via Suspense (`ChartCard.tsx`) to keep the initial JS bundle small. Adds a small layout flash on first render.
- **Custom series for OutcomeSequenceTape** — implementing the I-beam brackets required dipping into ECharts' custom series API (~150 lines). Validated and shipped, cf. PUNCHLIST decision GS retour.

## Alternatives evaluated

| Alternative | Rejected because |
|---|---|
| **Plotly** | Bundle size, server-side coupling, harder to theme. |
| **Recharts** | No heatmap, no radar with N series, no custom series for OutcomeSequenceTape. |
| **Visx** | Mature but lower-level (D3 wrapped) — required writing every chart from scratch. |
| **Chart.js** | No heatmap or radar with proper accessibility. Less customizable than ECharts. |
| **D3 raw** | Maintenance cost too high for the team size. |

## References

- Migration commits: P3.B `b655d0f2` (Histogram + Scatter), P3.D `9363f2b2` (Donut), P4.A `20563701` (Synthesis), P4.B `edfd1342` (Squad Contributions), P4.C `6b62ffab` (Squad Synergies), P4.D `16fb335e` (Plotly removal).
- Sandbox: `/lab/charts` (`apps/web/src/features/lab/ChartsShowcasePage.tsx`).
- Plan: `.ai/V7/PLAN_META_FOUNDATIONS_GO.md` § 5 (architecture cible).
