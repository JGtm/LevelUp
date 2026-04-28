/**
 * timelineChart — Perf / Win rate sur les matchs en escouade.
 *
 * Produit 2 ChartSeries<ChartPoint2D> (perf + win rate normalisé 0-100),
 * consommés par TimeseriesLineChart en mode catégorie. Remplace l'ancien
 * builder Plotly dual-axes bar+line.
 *
 * Multi-titres : aucun libellé hardcodé. Noms de séries passés en argument.
 */
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { ChartPoint2D } from '@/components/charts/TimeseriesLineChart'
import type { SquadTimeseriesPoint } from '@/lib/api/types'

export interface TimelineSeriesArgs {
  points: SquadTimeseriesPoint[]
  perfName: string
  winRateName: string
}

export function buildTimelineSeries({
  points,
  perfName,
  winRateName,
}: TimelineSeriesArgs): ChartSeries<ChartPoint2D>[] {
  if (points.length === 0) return []

  return [
    {
      key: 'perf',
      datapoints: points.map((p) => ({ x: p.period_label, y: p.avg_performance ?? 0 })),
      meta: { name: perfName },
    },
    {
      key: 'winrate',
      datapoints: points.map((p) => ({ x: p.period_label, y: p.win_rate * 100 })),
      meta: { name: winRateName },
    },
  ]
}
