/**
 * SessionCompareBarMetrics — barres groupées (K/D norm., Win%, Précision) A vs B.
 * Chart 08 du mock session_compare.
 *
 * Toutes les valeurs sont normalisées 0-100 pour un axe Y unique :
 *   K/D → kda / 3.0 × 100 (plafonné 100)   Win% et Accuracy% passent directement.
 */
import { useMemo } from 'react'

import { BarGroupedChart } from '@/components/charts/BarGroupedChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { ChartPointStacked } from '@/components/charts/BarStackedChart'
import type { SessionCompareMetricRow } from '@/lib/api/types'

export interface SessionCompareBarMetricsProps {
  metrics: SessionCompareMetricRow[]
  labels: {
    title: string
    sessionA: string
    sessionB: string
    empty: string
    catKD: string
    catWinRate: string
    catAccuracy: string
  }
  height?: number
}

function parseMetricFloat(metrics: SessionCompareMetricRow[], key: string, side: 'a' | 'b'): number {
  const row = metrics.find((m) => m.key === key)
  if (!row) return 0
  const raw = side === 'a' ? row.value_a : row.value_b
  return parseFloat(raw.replace('%', '').trim()) || 0
}

export function SessionCompareBarMetrics({
  metrics,
  labels,
  height = 280,
}: SessionCompareBarMetricsProps) {
  const series = useMemo<ChartSeries<ChartPointStacked>[]>(() => {
    const kdA = parseMetricFloat(metrics, 'kd_ratio', 'a')
    const kdB = parseMetricFloat(metrics, 'kd_ratio', 'b')
    const winA = parseMetricFloat(metrics, 'win_rate', 'a')
    const winB = parseMetricFloat(metrics, 'win_rate', 'b')
    const accA = parseMetricFloat(metrics, 'accuracy', 'a')
    const accB = parseMetricFloat(metrics, 'accuracy', 'b')

    if (kdA === 0 && kdB === 0 && winA === 0 && winB === 0) return []

    const saLabel = labels.sessionA
    const sbLabel = labels.sessionB

    return [
      {
        key: 'compare-bar-metrics',
        meta: {},
        datapoints: [
          {
            category: labels.catKD,
            components: {
              [saLabel]: Math.min(Math.round(kdA / 3.0 * 100), 100),
              [sbLabel]: Math.min(Math.round(kdB / 3.0 * 100), 100),
            },
          },
          {
            category: labels.catWinRate,
            components: { [saLabel]: Math.round(winA), [sbLabel]: Math.round(winB) },
          },
          {
            category: labels.catAccuracy,
            components: { [saLabel]: Math.round(accA), [sbLabel]: Math.round(accB) },
          },
        ],
      },
    ]
  }, [metrics, labels])

  const componentColors = useMemo(
    () => ({
      [labels.sessionA]: 'compare-a' as const,
      [labels.sessionB]: 'compare-b' as const,
    }),
    [labels.sessionA, labels.sessionB],
  )

  return (
    <BarGroupedChart
      title={labels.title}
      series={series}
      emptyMessage={labels.empty}
      height={height}
      componentColors={componentColors}
      componentOrder={[labels.sessionA, labels.sessionB]}
    />
  )
}
