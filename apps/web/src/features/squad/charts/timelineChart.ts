/**
 * timelineChart — Perf / Winrate / MMR sur les matchs en escouade
 * (série temporelle).
 *
 * Multi-titres : aucun libellé hardcodé. Tous les libellés (titre, traces,
 * axes) sont passés en argument par le caller.
 */
import type { SquadTimeseriesPoint, PlotlyFigurePayload } from '@/lib/api/types'
import { getPerfColor } from '@/lib/perf-color'
import { resolveToken } from '@/lib/accessibility'

interface TimelineChartArgs {
  points: SquadTimeseriesPoint[]
  title: string
  perfName: string
  winRateName: string
  perfAxis: string
  winRateAxis: string
}

export function buildTimelineChart({
  points,
  title,
  perfName,
  winRateName,
  perfAxis,
  winRateAxis,
}: TimelineChartArgs): PlotlyFigurePayload | null {
  if (points.length === 0) return null

  const labels = points.map((p) => p.period_label)
  const winrates = points.map((p) => p.win_rate)
  const perfs = points.map((p) => p.avg_performance ?? 0)

  const perfColors = perfs.map((v) => getPerfColor(v))

  const traces: PlotlyFigurePayload['data'] = [
    {
      type: 'bar',
      name: perfName,
      x: labels,
      y: perfs,
      marker: { color: perfColors },
      yaxis: 'y',
    },
    {
      type: 'scatter',
      mode: 'lines+markers',
      name: winRateName,
      x: labels,
      y: winrates,
      line: { color: resolveToken('divergent-pos'), width: 2 },
      marker: { color: resolveToken('divergent-pos'), size: 6 },
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
      title: { text: title, font: { size: 13 } },
      xaxis: { tickangle: -35 },
      yaxis: { title: perfAxis, range: [0, 100] },
      yaxis2: {
        title: winRateAxis,
        overlaying: 'y',
        side: 'right',
        range: [0, 100],
        showgrid: false,
      },
    },
  }
}
