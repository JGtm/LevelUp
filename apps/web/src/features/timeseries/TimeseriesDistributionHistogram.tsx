/**
 * TimeseriesDistributionHistogram — histogramme + médiane verticale.
 *
 * Wrapper local (page Cumul) : reprend la logique de HistogramChart mais
 * ajoute une markLine verticale à la médiane interpolée depuis les buckets.
 */
import { useMemo, type ReactNode } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
} from '@/components/charts/_utils'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'
import type { DistributionBucket } from '@/lib/api/types'
import { ChartFromOption } from './ChartFromOption'

export interface TimeseriesDistributionHistogramProps {
  buckets: DistributionBucket[]
  height?: number
  title?: ReactNode
  emptyMessage?: string
  colorToken: SemanticToken
  xAxisLabel: string
  medianLabel: string
  /** Optionnel : retourne un token sémantique par bucket (perf-tier-1..5 par
   *  exemple). Si présent, prend précédence sur `colorToken`. */
  colorTokenByBucket?: (b: DistributionBucket) => SemanticToken
  /** Clé de tournée de revue (l'histogramme est monté 6× : la clé vient du caller). */
  reviewKey?: string
}

/** Médiane interpolée à partir des buckets [lower, upper). Null si total nul. */
function bucketMedian(buckets: DistributionBucket[]): number | null {
  const total = buckets.reduce((s, b) => s + b.count, 0)
  if (total === 0) return null
  const target = total / 2
  let cumul = 0
  for (const b of buckets) {
    if (cumul + b.count >= target) {
      const ratio = b.count > 0 ? (target - cumul) / b.count : 0
      return b.bucket_lower + ratio * (b.bucket_upper - b.bucket_lower)
    }
    cumul += b.count
  }
  return buckets[buckets.length - 1].bucket_upper
}

export function TimeseriesDistributionHistogram({
  buckets,
  height = 240,
  title,
  emptyMessage,
  colorToken,
  xAxisLabel,
  medianLabel,
  colorTokenByBucket,
  reviewKey,
}: TimeseriesDistributionHistogramProps) {
  const themeVersion = useThemeVersion()


  const option = useMemo<EChartsCoreOption | null>(() => {
    if (buckets.length === 0) return null
    const tc = getEChartsThemeColors()
    const fill = resolveToken(colorToken)
    const medianColor = resolveToken('outcome-loss')

    const categories = buckets.map((b) =>
      `${b.bucket_lower.toFixed(b.bucket_lower % 1 === 0 ? 0 : 1)}`,
    )
    // Per-bar coloring : si colorTokenByBucket est fourni, chaque barre porte
    // son propre itemStyle (sinon série uniforme via fill).
    const values = colorTokenByBucket
      ? buckets.map((b) => ({
          value: b.count,
          itemStyle: { color: resolveToken(colorTokenByBucket(b)), opacity: 0.95 },
        }))
      : buckets.map((b) => b.count)
    const median = bucketMedian(buckets)
    const medianRounded = median != null ? Math.round(median * 100) / 100 : null

    // markLine est indexée sur l'axe catégorie : on cherche l'index du bucket
    // contenant la médiane, puis on interpole sa fraction interne pour
    // positionner précisément la ligne entre 2 ticks.
    let medianXAxisValue: number | null = null
    if (median != null) {
      for (let i = 0; i < buckets.length; i++) {
        const b = buckets[i]
        if (median >= b.bucket_lower && median < b.bucket_upper) {
          const frac = (median - b.bucket_lower) / (b.bucket_upper - b.bucket_lower)
          medianXAxisValue = i + frac
          break
        }
      }
      if (medianXAxisValue == null) medianXAxisValue = buckets.length - 1
    }

    return {
      backgroundColor: CHART_BG,
      grid: { top: 16, right: 16, bottom: 36, left: 40 },
      tooltip: {
        ...getTooltipBase(tc),
        trigger: 'axis',
      },
      xAxis: {
        ...getAxisBase(tc),
        type: 'category',
        data: categories,
        name: xAxisLabel,
        nameLocation: 'middle',
        nameGap: 24,
        nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
        axisLabel: { ...getAxisBase(tc).axisLabel, interval: Math.max(0, Math.floor(categories.length / 8) - 1) },
      },
      yAxis: {
        ...getAxisBase(tc),
        type: 'value',
      },
      series: [
        {
          type: 'bar',
          name: xAxisLabel,
          data: values,
          // itemStyle de série utilisé seulement quand pas de coloring per-bar.
          itemStyle: colorTokenByBucket ? undefined : { color: fill, opacity: 0.85 },
          barCategoryGap: '5%',
          markLine:
            medianXAxisValue != null && medianRounded != null
              ? {
                  silent: true,
                  symbol: 'none',
                  lineStyle: { color: medianColor, width: 1.5, type: 'dashed' },
                  label: {
                    formatter: `${medianLabel} ${medianRounded}`,
                    color: medianColor,
                    fontSize: 10,
                    fontWeight: 600,
                    backgroundColor: tc.tooltipBg,
                    padding: [2, 4],
                    borderRadius: 2,
                    position: 'insideEndTop',
                  },
                  data: [{ xAxis: medianXAxisValue }],
                }
              : undefined,
        },
      ],
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [buckets, colorToken, xAxisLabel, medianLabel, themeVersion])

  return (
    <ChartFromOption
      title={title}
      option={option}
      height={height}
      emptyMessage={emptyMessage}
      reviewKey={reviewKey}
    />
  )
}
