/**
 * explorerCadenceChart — barres divergentes K/D/A pour le bloc « Cadence » de
 * l'Explorer (par match / par minute).
 *
 * Style recopié du graphe « Stats par minute » de Squad Contributions
 * (squadPerMinuteChart) — recopié plutôt qu'importé pour respecter
 * lint-cross-feature-imports : Frags & Assistances au-dessus de l'axe zéro
 * (accentué), Morts en dessous (négatif), label = valeur absolue sur la barre,
 * pas de légende. Une seule série (la cible).
 *
 * Couleurs via tokens sémantiques uniquement (resolveToken).
 */
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken } from '@/lib/accessibility'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
} from '@/components/charts/_utils'

export interface CadenceBarsLabels {
  frags: string
  deaths: string
  assists: string
}

export interface CadenceBarsValues {
  kills: number
  deaths: number
  assists: number
}

/**
 * buildCadenceBarsOption — option ECharts pure (testable). `fractionDigits` :
 * 1 pour « par match », 2 pour « par minute ».
 */
export function buildCadenceBarsOption(
  values: CadenceBarsValues,
  labels: CadenceBarsLabels,
  fractionDigits: number,
): EChartsCoreOption {
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const fragsColor = resolveToken('chart-series-1')
  const deathsColor = resolveToken('outcome-loss')
  const assistsColor = resolveToken('chart-series-3')
  const fmt = (v: number) => v.toFixed(fractionDigits)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 28, left: 8, right: 16, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => {
        const arr = Array.isArray(params) ? params : []
        if (arr.length === 0) return ''
        const p = arr[0] as { name?: string; value?: number }
        return `${escapeHtml(p.name ?? '')} : <strong>${fmt(Math.abs(p.value ?? 0))}</strong>`
      },
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: [labels.frags, labels.deaths, labels.assists],
      axisLine: { lineStyle: { color: tc.text, width: 2 } }, // axe zéro accentué
    },
    yAxis: {
      ...axis,
      type: 'value',
      axisLabel: { ...axis.axisLabel, formatter: (v: number) => fmt(Math.abs(v)) },
    },
    series: [
      {
        type: 'bar',
        barMaxWidth: 28,
        data: [
          { value: values.kills, itemStyle: { color: fragsColor } },
          { value: -values.deaths, itemStyle: { color: deathsColor } }, // Morts sous l'axe
          { value: values.assists, itemStyle: { color: assistsColor } },
        ],
        label: {
          show: true,
          position: 'top' as const,
          color: tc.text,
          fontSize: 10,
          formatter: (p: { value: unknown }) =>
            fmt(Math.abs(typeof p.value === 'number' ? p.value : 0)),
        },
      },
    ],
  }
}
