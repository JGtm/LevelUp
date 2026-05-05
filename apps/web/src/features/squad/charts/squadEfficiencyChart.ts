/**
 * squadEfficiencyChart — teammates : rendement & résistance par match.
 *
 * Chaque track (1 par joueur) affiche 2 lignes brutes :
 *   - trait plein     : rendement offensif  = 225 × (kills + ass/3) / dégâts infligés
 *   - trait pointillé : résistance défensive = dégâts subis / (225 × morts)
 *
 * La ligne de référence à y=1.0 correspond au seuil physique :
 *   - rendement = 1.0 → 0 tir gaspillé (225 dégâts exacts par kill effectif)
 *   - résistance = 1.0 → chaque mort a coûté exactement 225 dégâts à l'ennemi
 */
import type { EChartsCoreOption } from 'echarts/core'
import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
} from '@/components/charts/_utils'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'

export interface EfficiencyTrackOpts {
  /** Couleur hex résolue depuis semantic tokens via getSquadPlayerColors. */
  color: string
  rendementLabel: string
  resistanceLabel: string
  refLabel: string
  showXAxis: boolean
  /** Valeur max Y partagée entre tous les joueurs (échelle uniforme au switch). */
  yMax?: number
}

export function buildSquadEfficiencyTrackOption(
  pts: SquadPerformanceSeriesPoint[],
  opts: EfficiencyTrackOpts,
): EChartsCoreOption {
  if (pts.length === 0) return { backgroundColor: CHART_BG }

  const n = pts.reduce((max, p) => Math.max(max, p.match_order), 0) + 1
  const rendData = new Array<number | null>(n).fill(null)
  const resistData = new Array<number | null>(n).fill(null)

  for (const p of pts) {
    const idx = p.match_order
    if (p.rendement_offensif !== undefined) {
      rendData[idx] = Number(p.rendement_offensif.toFixed(2))
    }
    if (p.resistance_defensive !== undefined) {
      resistData[idx] = Number(p.resistance_defensive.toFixed(2))
    }
  }

  const xLabels = Array.from({ length: n }, (_, i) => `#${i + 1}`)
  // color-allow: hex résolu depuis semantic tokens via getSquadPlayerColors
  const color = opts.color
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  return {
    backgroundColor: CHART_BG,
    grid: {
      top: 8,
      bottom: opts.showXAxis ? 32 : 8,
      left: 8,
      right: 40,
      containLabel: true,
    },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'line' },
      formatter: (params: unknown) => {
        const items = params as Array<{ seriesName: string; value: number | null; marker: string; dataIndex: number }>
        if (!Array.isArray(items) || items.length === 0) return ''
        const label = xLabels[items[0].dataIndex] ?? `#${items[0].dataIndex + 1}`
        const lines = items
          .filter((it) => it.value !== null)
          .map((it) => `${it.marker} ${it.seriesName}: <b>${(it.value as number).toFixed(2)}</b>`)
        return `<div style="font-size:11px">${label}<br/>${lines.join('<br/>')}</div>`
      },
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: xLabels,
      show: opts.showXAxis,
      axisLabel: {
        ...axis.axisLabel,
        interval: n > 30 ? Math.floor(n / 12) : 0,
      },
    },
    yAxis: {
      ...axis,
      type: 'value',
      min: 0,
      max: opts.yMax,
      axisLabel: {
        ...axis.axisLabel,
        formatter: (v: number) => v.toFixed(1),
      },
    },
    series: [
      {
        name: opts.rendementLabel,
        type: 'line',
        data: rendData,
        lineStyle: { color, width: 2 },
        itemStyle: { color },
        symbol: 'circle',
        symbolSize: 4,
        connectNulls: true,
        markLine: {
          silent: true,
          symbol: 'none',
          lineStyle: { color: tc.axisLabel, type: 'dashed', width: 1 },
          data: [{ yAxis: 1.0 }],
          label: {
            formatter: opts.refLabel,
            position: 'end',
            color: tc.axisLabel,
            fontSize: 9,
          },
        },
      },
      {
        name: opts.resistanceLabel,
        type: 'line',
        data: resistData,
        lineStyle: { color, width: 2, type: 'dashed', opacity: 0.55 },
        itemStyle: { color, opacity: 0.55 },
        symbol: 'none',
        connectNulls: true,
      },
    ],
  }
}
