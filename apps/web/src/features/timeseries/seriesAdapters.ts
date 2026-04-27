/**
 * seriesAdapters — convertit les structures DTO timeseries (CumulativePoint,
 * HeatmapCell) vers les types canoniques des wrappers ECharts.
 *
 * Phase 2 P2.E : extrait pour testabilité, isole la couche d'adaptation entre
 * le DTO Go et les wrappers visuels.
 */
import type { CumulativePoint, IntensityHeatmapPoint } from '@/lib/api/types'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { ChartPoint2D } from '@/components/charts/TimeseriesLineChart'
import type { ChartPointHeatmap } from '@/components/charts/Heatmap2DChart'

/** ISO weekday names used by the heatmap. 0 = Monday … 6 = Sunday. */
export const DOW_LABELS_FR = ['Lun', 'Mar', 'Mer', 'Jeu', 'Ven', 'Sam', 'Dim'] as const
export const DOW_LABELS_EN = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'] as const

/**
 * Convertit un tableau de CumulativePoint en ChartSeries<ChartPoint2D>[] mono-série,
 * ordonné par index puis start_time.
 */
export function cumulativePointsToSeries(
  points: CumulativePoint[],
  options: { key: string; name: string },
): ChartSeries<ChartPoint2D>[] {
  if (!points || points.length === 0) {
    return [
      {
        key: options.key,
        meta: { gamertag: options.name },
        datapoints: [],
      },
    ]
  }
  return [
    {
      key: options.key,
      meta: { gamertag: options.name },
      datapoints: points.map((p) => ({
        x: p.start_time,
        y: p.value,
      })),
    },
  ]
}

/**
 * Convertit un tableau de IntensityHeatmapPoint (day_of_week/hour/count) en
 * ChartSeries<ChartPointHeatmap>[] mono-série. day_of_week est résolu en
 * libellé via `dowLabels`. hour reste sous forme "00".."23".
 */
export function heatmapCellsToSeries(
  cells: IntensityHeatmapPoint[],
  options: {
    key: string
    name: string
    dowLabels: readonly string[]
  },
): ChartSeries<ChartPointHeatmap>[] {
  if (!cells || cells.length === 0) {
    return [
      {
        key: options.key,
        meta: { gamertag: options.name },
        datapoints: [],
      },
    ]
  }
  return [
    {
      key: options.key,
      meta: { gamertag: options.name },
      datapoints: cells.map((c) => ({
        x: String(c.hour).padStart(2, '0'),
        y: options.dowLabels[c.day_of_week] ?? String(c.day_of_week),
        value: c.count,
        detail: { avg_kd: c.avg_kd },
      })),
    },
  ]
}
