/**
 * timelineChart — Perf / Winrate / MMR sur les matchs en escouade (série temporelle).
 */
import type { SquadTimeseriesPoint, PlotlyFigurePayload } from '@/lib/api/types'
import { getPerfColor } from '@/lib/perf-color'

export function buildTimelineChart(
  points: SquadTimeseriesPoint[],
): PlotlyFigurePayload | null {
  if (points.length === 0) return null

  const labels = points.map((p) => p.period_label)
  const winrates = points.map((p) => p.win_rate)
  const perfs = points.map((p) => p.avg_performance ?? 0)

  // Couleur des barres par seuil de perf
  const perfColors = perfs.map((v) => getPerfColor(v))

  const traces: PlotlyFigurePayload['data'] = [
    {
      type: 'bar',
      name: 'Perf. moy.',
      x: labels,
      y: perfs,
      marker: { color: perfColors },
      yaxis: 'y',
    },
    {
      type: 'scatter',
      mode: 'lines+markers',
      name: 'Win rate (%)',
      x: labels,
      y: winrates,
      line: { color: '#10B981', width: 2 },
      marker: { color: '#10B981', size: 6 },
      yaxis: 'y2',
    },
  ]

  return {
    data: traces,
    layout: {
      hovermode: 'x unified',
      height: 300,
      margin: { l: 45, r: 45, t: 30, b: 80 },
      legend: { orientation: 'h', x: 0, y: -0.25 },
      plot_bgcolor: 'rgba(0,0,0,0)',
      paper_bgcolor: 'rgba(0,0,0,0)',
      title: { text: 'Évolution des performances en escouade', font: { size: 13 } },
      xaxis: { tickangle: -35 },
      yaxis: { title: 'Score perf.', range: [0, 100] },
      yaxis2: {
        title: 'Win rate',
        overlaying: 'y',
        side: 'right',
        range: [0, 100],
        showgrid: false,
      },
    },
  }
}
