/**
 * SessionPerfTrend — courbe du `performance_score` par match au fil de la session.
 *
 * 2 séries : la valeur par match (marker outcome-coloré via OutcomeMarkers) +
 * une ligne plate "moyenne session" (référence). Pas de hex direct ni libellé hardcodé :
 * tout passe par session manifest (titres) + `useFieldMappings` (libellé série).
 */
import { useMemo } from 'react'

import { TimeseriesLineChart, type ChartPoint2D } from '@/components/charts/TimeseriesLineChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import { resolveToken } from '@/lib/accessibility'
import type { SessionDetailMatchRow } from '@/lib/api/types'

import { outcomeIntToKey, useSessionT } from './_shared'

interface Props {
  title: string
  matches: SessionDetailMatchRow[]
  height?: number
}

export function SessionPerfTrend({ title, matches, height = 280 }: Props) {
  const t = useSessionT()

  const { series, hasData } = useMemo(() => {
    const sorted = [...matches]
      .filter((m) => m.performance_score != null)
      .sort((a, b) => a.start_time.localeCompare(b.start_time))
    if (sorted.length === 0) {
      return { series: [] as ChartSeries<ChartPoint2D>[], hasData: false }
    }
    const sum = sorted.reduce((acc, m) => acc + (m.performance_score ?? 0), 0)
    const mean = sum / sorted.length

    const perfSeries: ChartSeries<ChartPoint2D> = {
      key: 'perf',
      labelKey: t('session.detail.chart_perf_series'),
      colorToken: 'chart-series-1',
      datapoints: sorted.map((m) => ({
        x: m.start_time,
        y: m.performance_score ?? 0,
        label: outcomeIntToKey(m.outcome) ?? undefined,
      })),
    }

    // Ligne "moyenne" : 2 points (premier + dernier) sur l'axe temporel.
    const meanSeries: ChartSeries<ChartPoint2D> = {
      key: 'mean',
      labelKey: t('session.detail.chart_perf_mean'),
      colorToken: 'divergent-neutral',
      datapoints: [
        { x: sorted[0].start_time, y: mean },
        { x: sorted[sorted.length - 1].start_time, y: mean },
      ],
    }

    return { series: [perfSeries, meanSeries], hasData: true }
  }, [matches, t])

  // Couleurs fixes par séries pour éviter le fallback cycle (token mean = muted).
  const colorByKey: Record<string, string> = {
    perf: resolveToken('chart-series-1'),
    mean: resolveToken('divergent-neutral'),
  }

  if (!hasData) return null

  return (
    <TimeseriesLineChart
      title={title}
      series={series}
      height={height}
      timeAxis
      outcomeMarkers
      smooth={false}
      seriesNameResolver={(s) => s.labelKey ?? s.key}
      seriesColorResolver={(s) => colorByKey[s.key]}
    />
  )
}
