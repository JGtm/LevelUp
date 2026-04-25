/**
 * heatmapChart — Win rate par carte (heatmap horizontale).
 */
import type { MapBreakdownRow, PlotlyFigurePayload } from '@/lib/api/types'
import { buildOrdinalColorscale } from '@/lib/accessibility'

export function buildHeatmapChart(
  rows: MapBreakdownRow[],
): PlotlyFigurePayload | null {
  if (rows.length === 0) return null

  // Tokens dans l'ordre croissant : 0 % = tier-5 (mauvais), 100 % = tier-1 (excellent)
  const colorscale = buildOrdinalColorscale([
    'perf-tier-5', 'perf-tier-4', 'perf-tier-3', 'perf-tier-2', 'perf-tier-1',
  ])

  const sorted = [...rows].sort((a, b) => b.win_rate - a.win_rate)
  const maps = sorted.map((r) => r.map_ui)
  const winrates = sorted.map((r) => r.win_rate)
  const counts = sorted.map((r) => r.match_count)
  const customdata = counts.map((c, i) => [c, winrates[i]])

  return {
    data: [
      {
        type: 'heatmap',
        z: [winrates],
        x: maps,
        y: ['Win rate (%)'],
        colorscale: colorscale as unknown as string,
        zmin: 0,
        zmax: 100,
        customdata: [customdata],
        hovertemplate:
          '<b>%{x}</b><br>Win rate: %{z:.1f}%<br>Matchs: %{customdata[0]}<extra></extra>',
        showscale: true,
        colorbar: { title: 'Win %', thickness: 14 },
      },
    ],
    layout: {
      height: 160,
      margin: { l: 80, r: 20, t: 30, b: 80 },
      plot_bgcolor: 'rgba(0,0,0,0)',
      paper_bgcolor: 'rgba(0,0,0,0)',
      title: { text: 'Win rate par carte (escouade)', font: { size: 13 } },
      xaxis: { tickangle: -35 },
    },
  }
}
