/**
 * hsPkChart — Headshot kills / Perfect kills par partie.
 *
 * Un groupe de barres par coéquipier sélectionné, barmode overlay.
 *
 * Multi-titres : aucun libellé hardcodé. Les noms de traces et le titre
 * sont passés en argument par le caller (qui les compose via
 * useFieldLabel + getSquadText).
 */
import type { TeammateRow, PlotlyFigurePayload } from '@/lib/api/types'
import { getSeriesColors } from '@/lib/accessibility'
import type { SquadMetric } from '../metrics'

const SERIES_TOKENS = ['perf-tier-1', 'perf-tier-2', 'perf-tier-3'] as const

interface HsPkChartArgs {
  rows: TeammateRow[]
  hsMetric: SquadMetric
  pkMetric: SquadMetric
  hsLabel: string
  pkLabel: string
  title: string
}

export function buildHsPkChart({
  rows,
  hsMetric,
  pkMetric,
  hsLabel,
  pkLabel,
  title,
}: HsPkChartArgs): PlotlyFigurePayload | null {
  if (rows.length === 0) return null

  const labels = rows.map((r) => r.gamertag)
  const hsValues = rows.map((r) => hsMetric.extract(r.with_kpis) ?? 0)
  const pkValues = rows.map((r) => pkMetric.extract(r.with_kpis) ?? 0)
  const colors = getSeriesColors(rows.length, [...SERIES_TOKENS])

  const traces: PlotlyFigurePayload['data'] = [
    {
      type: 'bar',
      name: hsLabel,
      x: labels,
      y: hsValues,
      marker: {
        color: colors,
        opacity: 0.85,
      },
    },
    {
      type: 'bar',
      name: pkLabel,
      x: labels,
      y: pkValues,
      marker: {
        color: colors,
        opacity: 0.45,
        line: {
          color: colors,
          width: pkValues.map((v) => (v > 0 ? 2.5 : 0)),
        },
      },
    },
  ]

  return {
    data: traces,
    layout: {
      barmode: 'overlay',
      height: 280,
      margin: { l: 40, r: 20, t: 30, b: 50 },
      legend: { orientation: 'h', x: 0, y: -0.2 },
      plot_bgcolor: 'rgba(0,0,0,0)',
      paper_bgcolor: 'rgba(0,0,0,0)',
      title: { text: title, font: { size: 13 } },
    },
  }
}
