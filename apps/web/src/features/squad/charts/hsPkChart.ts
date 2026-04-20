/**
 * hsPkChart — Headshot kills / Perfect kills par partie.
 * Un groupe de barres par coéquipier sélectionné, barmode overlay.
 */
import type { TeammateRow, PlotlyFigurePayload } from '@/lib/api/types'

const CHART_COLORS = ['#7C3AED', '#F59E0B', '#10B981']

export function buildHsPkChart(rows: TeammateRow[]): PlotlyFigurePayload | null {
  if (rows.length === 0) return null

  const labels = rows.map((r) => r.gamertag)
  const hsValues = rows.map((r) => r.with_kpis.headshot_kills_per_game ?? 0)
  const pkValues = rows.map((r) => r.with_kpis.perfect_kills_per_game ?? 0)

  const traces: PlotlyFigurePayload['data'] = [
    {
      type: 'bar',
      name: 'Headshot kills/partie',
      x: labels,
      y: hsValues,
      marker: {
        color: rows.map((_, i) => CHART_COLORS[i % CHART_COLORS.length]),
        opacity: 0.85,
      },
    },
    {
      type: 'bar',
      name: 'Perfect kills/partie',
      x: labels,
      y: pkValues,
      marker: {
        color: rows.map((_, i) => CHART_COLORS[i % CHART_COLORS.length]),
        opacity: 0.45,
        line: {
          color: rows.map((_, i) => CHART_COLORS[i % CHART_COLORS.length]),
          width: rows.map((r) =>
            (r.with_kpis.perfect_kills_per_game ?? 0) > 0 ? 2.5 : 0,
          ),
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
      title: { text: 'Headshot & Perfect kills / partie', font: { size: 13 } },
    },
  }
}
