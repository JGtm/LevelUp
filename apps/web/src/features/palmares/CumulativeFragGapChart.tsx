/**
 * CumulativeFragGapChart — écart de frags cumulé contre un rival, duel par duel
 * (ancien → récent). Ligne colorée PAR LE SIGNE : vert quand le cumul est
 * positif (tu mènes), rouge quand négatif (tu es derrière) — via un visualMap
 * piecewise autour de 0. Référence de lecture : ligne pointillée à y=0.
 *
 * Aucune couleur hex directe : outcomeColor() → resolveToken().
 */
import { Suspense, lazy, useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { useThemeVersion } from '@/lib/echarts/useThemeVersion'

import { CHART_BG, getEChartsThemeColors, getTooltipBase, outcomeColor } from '@/components/charts/_utils'

const ReactECharts = lazy(() =>
  import('echarts-for-react').then((m) => ({ default: m.default ?? m })),
)

interface CumulativeFragGapChartProps {
  /** Cumul net (Σ frags − Σ morts) par duel, ancien → récent. */
  values: number[]
  height?: number
}

export function CumulativeFragGapChart({ values, height = 120 }: CumulativeFragGapChartProps) {
  const themeVersion = useThemeVersion()

  const option = useMemo((): EChartsCoreOption => {
    if (values.length === 0) return {}
    const tc = getEChartsThemeColors()
    const positive = outcomeColor('win')
    const negative = outcomeColor('loss')
    return {
      backgroundColor: CHART_BG,
      grid: { top: 16, bottom: 16, left: 28, right: 8 },
      xAxis: { type: 'category', show: false, data: values.map((_, i) => i + 1) },
      yAxis: { type: 'value', scale: true },
      tooltip: {
        trigger: 'axis',
        ...getTooltipBase(tc),
        formatter: (params: unknown) => {
          const arr = params as Array<{ value: number }>
          const v = arr?.[0]?.value
          if (v == null || !Number.isFinite(v)) return ''
          return `${v > 0 ? '+' : ''}${v}`
        },
      },
      visualMap: {
        show: false,
        type: 'piecewise',
        dimension: 1,
        seriesIndex: 0,
        pieces: [
          { gte: 0, color: positive },
          { lt: 0, color: negative },
        ],
      },
      series: [
        {
          type: 'line',
          data: values,
          smooth: true,
          showSymbol: false,
          lineStyle: { width: 2 },
          areaStyle: { opacity: 0.08 },
          markLine: {
            silent: true,
            symbol: 'none',
            data: [{ yAxis: 0 }],
            lineStyle: { type: 'dashed', opacity: 0.5 },
            label: { show: false },
          },
        },
      ],
    }
    // themeVersion force le recalcul de l'option au changement de thème.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [values, themeVersion])

  if (values.length === 0) return null

  return (
    <Suspense fallback={null}>
      <ReactECharts option={option} style={{ height, width: '100%' }} notMerge lazyUpdate theme={undefined} />
    </Suspense>
  )
}
