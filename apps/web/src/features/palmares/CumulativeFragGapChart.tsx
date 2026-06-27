/**
 * CumulativeFragGapChart — écart de frags cumulé contre un rival, duel par duel
 * (ancien → récent). Aire + courbe DIVERGENTES colorées par le SIGNE du cumul :
 * dégradé vert (`divergent-pos`) au-dessus de 0 / rouge (`divergent-neg`) en
 * dessous, à bascule EXACTE sur 0 (aire ancrée à 0 via `areaStyle.origin`).
 *
 * PAS de visualMap : sur une série `line` à données scalaires + axe catégoriel,
 * il laissait la courbe invisible (cf. SessionNetScoreArea). Le dégradé est
 * calculé depuis la boîte englobante de l'aire ancrée à 0 (`zeroRatio`).
 *
 * Chaque point porte un SYMBOLE coloré par l'issue du duel (win/loss/neutre) →
 * l'issue se lit match par match, indépendamment du signe du cumul.
 * Aucune couleur hex directe : `resolveToken()` + `outcomeColor()`.
 */
import { Suspense, lazy, useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken } from '@/lib/accessibility'
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'

import { CHART_BG, getEChartsThemeColors, getTooltipBase, outcomeColor } from '@/components/charts/_utils'

const ReactECharts = lazy(() =>
  import('echarts-for-react').then((m) => ({ default: m.default ?? m })),
)

export interface CumulativeFragPoint {
  cumulative: number
  /** "win" | "loss" | "other" — pilote la couleur du symbole. */
  outcome: string
}

interface CumulativeFragGapChartProps {
  points: CumulativeFragPoint[]
  height?: number
}

// outcome backend ("win"|"loss"|"other") → clé OutcomeValue pour outcomeColor().
function outcomeKey(o: string): 'win' | 'loss' | 'tie' {
  return o === 'win' ? 'win' : o === 'loss' ? 'loss' : 'tie'
}

export function CumulativeFragGapChart({ points, height = 120 }: CumulativeFragGapChartProps) {
  const themeVersion = useThemeVersion()

  const option = useMemo((): EChartsCoreOption => {
    if (points.length === 0) return {}
    const tc = getEChartsThemeColors()
    const posColor = resolveToken('divergent-pos')
    const negColor = resolveToken('divergent-neg')

    // Dégradé divergent vert/rouge à bascule EXACTE sur 0, sans visualMap. L'aire
    // ancrée à 0 → boîte [min(données,0), max(données,0)] ; 0 tombe à zeroRatio.
    const values = points.map((p) => p.cumulative)
    const top = Math.max(...values, 0)
    const bot = Math.min(...values, 0)
    const span = top - bot
    const zeroRatio = span > 0 ? Math.min(1, Math.max(0, top / span)) : 1
    const divergentColor = {
      type: 'linear' as const,
      x: 0,
      y: 0,
      x2: 0,
      y2: 1,
      colorStops: [
        { offset: 0, color: posColor },
        { offset: zeroRatio, color: posColor },
        { offset: zeroRatio, color: negColor },
        { offset: 1, color: negColor },
      ],
    }

    const data = points.map((p) => ({
      value: p.cumulative,
      itemStyle: { color: outcomeColor(outcomeKey(p.outcome)) },
    }))

    return {
      backgroundColor: CHART_BG,
      grid: { top: 16, bottom: 16, left: 28, right: 8 },
      xAxis: { type: 'category', show: false, data: points.map((_, i) => i + 1) },
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
      series: [
        {
          type: 'line',
          data,
          smooth: true,
          showSymbol: true,
          symbol: 'circle',
          symbolSize: 6,
          lineStyle: { width: 2, color: divergentColor },
          areaStyle: { color: divergentColor, opacity: 0.14, origin: 0 },
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
  }, [points, themeVersion])

  if (points.length === 0) return null

  return (
    <Suspense fallback={null}>
      <ReactECharts option={option} style={{ height, width: '100%' }} notMerge lazyUpdate theme={undefined} />
    </Suspense>
  )
}
