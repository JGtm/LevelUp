/**
 * seriesAdapters — convertit les structures DTO timeseries (CumulativePoint,
 * HeatmapCell) vers les types canoniques des wrappers ECharts.
 *
 * Phase 2 P2.E : extrait pour testabilité, isole la couche d'adaptation entre
 * le DTO Go et les wrappers visuels.
 */
import type {
  CorrelationDataPair,
  CumulativePoint,
  DistributionBucket,
  IntensityHeatmapPoint,
} from '@/lib/api/types'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { ChartPoint2D } from '@/components/charts/TimeseriesLineChart'
import type { ChartPointHeatmap } from '@/components/charts/Heatmap2DChart'
import type { ChartPointHistogram } from '@/components/charts/HistogramChart'
import type { ChartPointScatter } from '@/components/charts/ScatterChart'

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
 * Convertit un tableau de DistributionBucket en ChartSeries<ChartPointHistogram>[]
 * mono-série. Préserve l'ordre des buckets.
 */
export function distributionBucketsToSeries(
  buckets: DistributionBucket[],
  options: { key: string; name: string },
): ChartSeries<ChartPointHistogram>[] {
  if (!buckets || buckets.length === 0) {
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
      datapoints: buckets.map((b) => ({
        binStart: b.bin_start,
        binEnd: b.bin_end,
        count: b.count,
      })),
    },
  ]
}

/** Outcome codes Halo (mirror analysis.OutcomeWin / OutcomeLoss). */
const OUTCOME_WIN = 2
const OUTCOME_LOSS = 3

export interface ScatterOutcomeLabels {
  win: string
  loss: string
  unknown: string
}

/**
 * Convertit un tableau de CorrelationDataPair filtré sur un label en
 * ChartSeries<ChartPointScatter>[] tri-séries (win/loss/unknown).
 * Les séries vides sont conservées (key stable pour mapper colorTokens).
 */
export function correlationPointsToSeries(
  points: CorrelationDataPair[],
  activeLabel: string,
  outcomeLabels: ScatterOutcomeLabels,
): ChartSeries<ChartPointScatter>[] {
  const filtered = points.filter((p) => p.label === activeLabel)
  const wins = filtered.filter((p) => p.outcome === OUTCOME_WIN)
  const losses = filtered.filter((p) => p.outcome === OUTCOME_LOSS)
  const unknowns = filtered.filter(
    (p) => p.outcome !== OUTCOME_WIN && p.outcome !== OUTCOME_LOSS,
  )
  const series: ChartSeries<ChartPointScatter>[] = []
  if (wins.length > 0) {
    series.push({
      key: 'outcome.win',
      meta: { gamertag: outcomeLabels.win },
      datapoints: wins.map((p) => ({ x: p.x, y: p.y })),
    })
  }
  if (losses.length > 0) {
    series.push({
      key: 'outcome.loss',
      meta: { gamertag: outcomeLabels.loss },
      datapoints: losses.map((p) => ({ x: p.x, y: p.y })),
    })
  }
  if (unknowns.length > 0) {
    series.push({
      key: 'outcome.unknown',
      meta: { gamertag: outcomeLabels.unknown },
      datapoints: unknowns.map((p) => ({ x: p.x, y: p.y })),
    })
  }
  return series
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
