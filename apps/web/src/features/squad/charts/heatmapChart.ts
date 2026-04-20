/**
 * heatmapChart — Win rate par carte (heatmap horizontale).
 */
import type { MapBreakdownRow, PlotlyFigurePayload } from '@/lib/api/types'

const HEATMAP_COLORSCALE = [
  [0, '#EF4444'],
  [0.35, '#F97316'],
  [0.5, '#F59E0B'],
  [0.65, '#06B6D4'],
  [1, '#10B981'],
]

export function buildHeatmapChart(
  rows: MapBreakdownRow[],
): PlotlyFigurePayload | null {
  if (rows.length === 0) return null

  // Trier par win rate décroissant
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
        colorscale: HEATMAP_COLORSCALE as unknown as string,
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
