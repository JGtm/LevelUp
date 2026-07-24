/**
 * CumulativeFragGapChart — écart de frags cumulé contre un rival, duel par duel
 * (ancien → récent). Aire + courbe DIVERGENTES colorées par le SIGNE du cumul :
 * dégradé vert (`divergent-pos`) au-dessus de 0 / rouge (`divergent-neg`) en
 * dessous, à bascule EXACTE sur 0 (aire ancrée à 0 via `areaStyle.origin`).
 *
 * PAS de visualMap : sur une série `line` à données scalaires + axe catégoriel,
 * il laissait la courbe invisible (cf. SessionNetScoreArea). Le dégradé divergent
 * ancré à 0 provient du helper canonique `divergentZeroGradient` (CLAUDE.md n°6).
 *
 * Chaque point porte un SYMBOLE coloré par l'issue du duel (win/loss/neutre) →
 * l'issue se lit match par match, indépendamment du signe du cumul.
 * Aucune couleur hex directe : `divergentZeroGradient()` + `outcomeColor()`.
 */
import { Suspense, lazy, useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { divergentZeroGradient } from '@/lib/charts/divergentZeroGradient'
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'

import { CHART_BG, getAxisBase, getEChartsThemeColors, getTooltipBase, outcomeColor, tickInterval } from '@/components/charts/_utils'

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
  /**
   * Affiche l'axe X (numéro de match 1..N). Défaut false : Relations/Palmarès rendent
   * la courbe nue (frise compacte). Opt-in Explorer où l'échelle « n-ième affrontement »
   * apporte du contexte. Réserve alors une marge basse pour les labels.
   */
  showXAxis?: boolean
}

// outcome backend ("win"|"loss"|"other") → clé OutcomeValue pour outcomeColor().
function outcomeKey(o: string): 'win' | 'loss' | 'tie' {
  return o === 'win' ? 'win' : o === 'loss' ? 'loss' : 'tie'
}

export function CumulativeFragGapChart({ points, height = 120, showXAxis = false }: CumulativeFragGapChartProps) {
  const themeVersion = useThemeVersion()

  const option = useMemo((): EChartsCoreOption => {
    if (points.length === 0) return {}
    const tc = getEChartsThemeColors()
    const axisBase = getAxisBase(tc)

    // Dégradé divergent vert/rouge à bascule EXACTE sur 0 (aire ancrée à 0),
    // helper canonique partagé — CLAUDE.md n°6.
    const values = points.map((p) => p.cumulative)
    const divergentColor = divergentZeroGradient(values)

    const data = points.map((p) => ({
      value: p.cumulative,
      itemStyle: { color: outcomeColor(outcomeKey(p.outcome)) },
    }))

    // Labels d'axe = numéro de match (1..N, l'échelle actuelle des points). Intervalle
    // de tick borné (tickInterval) pour ne pas surcharger les longues séries.
    const matchNumbers = points.map((_, i) => i + 1)
    return {
      backgroundColor: CHART_BG,
      grid: { top: 16, bottom: showXAxis ? 28 : 16, left: 28, right: 8 },
      xAxis: showXAxis
        ? {
            ...axisBase,
            type: 'category',
            data: matchNumbers,
            axisLabel: { ...axisBase.axisLabel, interval: tickInterval(points.length) - 1 },
          }
        : { type: 'category', show: false, data: matchNumbers },
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
  }, [points, showXAxis, themeVersion])

  if (points.length === 0) return null

  return (
    <Suspense fallback={null}>
      <ReactECharts option={option} style={{ height, width: '100%' }} notMerge lazyUpdate theme={undefined} />
    </Suspense>
  )
}
