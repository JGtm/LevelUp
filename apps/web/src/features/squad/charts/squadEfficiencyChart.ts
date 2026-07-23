/**
 * squadEfficiencyChart — teammates : Rendement & Résistance par match, en dégâts
 * BRUTS (repère « 1 vie » = PV-pour-tuer du titre : 225 Infinite, 115 Halo 5).
 *
 * Un builder unique, paramétré par métrique, rend UNE courbe PAR joueur sur un
 * graphe (colorée par joueur, togglable via la légende native ECharts) :
 *   - metric `damagePerKill`  → « dégâts / frag » = dégâts infligés / frags ;
 *   - metric `damagePerDeath` → « dégâts / mort » = dégâts subis / morts.
 *
 * Axe X = ordre de match PARTAGÉ de l'escouade (`match_order`, intersection
 * chronologique commune à tous les joueurs). Ligne repère « 1 vie » sur les deux
 * métriques (série fantôme hors légende, toujours visible).
 */
import type { EChartsCoreOption } from 'echarts/core'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
} from '@/components/charts/_utils'
import {
  damageAxisBounds,
  damagePerDeath,
  damagePerKill,
} from '@/lib/charts/oneLifeDamageGradient'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import { truncateMap } from '@/lib/charts/matchLabels'

/** Métrique d'efficacité rendue : dégâts par frag (offensif) ou par mort (défensif). */
export type EfficiencyMetric = 'damagePerKill' | 'damagePerDeath'

/** Valeur brute du point pour la métrique demandée (null si non calculable). */
function metricValue(metric: EfficiencyMetric, pt: SquadPerformanceSeriesPoint): number | null {
  return metric === 'damagePerKill'
    ? damagePerKill(pt.damage_dealt, pt.kills)
    : damagePerDeath(pt.damage_taken, pt.deaths)
}

export interface EfficiencyMultiOpts {
  /** Métrique projetée sur l'axe Y (1 courbe/joueur). */
  metric: EfficiencyMetric
  refLabel: string
  /** PV-pour-tuer du titre (repère ligne « 1 vie »). */
  oneLife: number
  /** gamertag → couleur hex résolue : 1 série/joueur, colorée par joueur. */
  colorByPlayer: Record<string, string>
  showXAxis: boolean
}

/**
 * buildSquadEfficiencyMultiOption — une courbe par joueur pour la métrique
 * choisie, colorée par joueur, affichée/masquée via la légende native ECharts
 * (clic). Repère « 1 vie » conservé sur les deux métriques.
 */
export function buildSquadEfficiencyMultiOption(
  rowsByPlayer: Record<string, SquadPerformanceSeriesPoint[]>,
  players: string[],
  opts: EfficiencyMultiOpts,
): EChartsCoreOption {
  let n = 0
  const mapByOrder = new Map<number, string>()
  for (const p of players) {
    for (const pt of rowsByPlayer[p] ?? []) {
      n = Math.max(n, pt.match_order + 1)
      if (pt.map_name && !mapByOrder.has(pt.match_order)) mapByOrder.set(pt.match_order, pt.map_name)
    }
  }
  if (n === 0) return { backgroundColor: CHART_BG }

  const xLabels = Array.from({ length: n }, (_, i) => {
    const m = mapByOrder.get(i)
    return m ? `#${i + 1}\n${truncateMap(m)}` : `#${i + 1}`
  })

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  const allValues: Array<number | null> = []
  const playerSeries = players.map((p) => {
    const data = new Array<number | null>(n).fill(null)
    for (const pt of rowsByPlayer[p] ?? []) {
      const v = metricValue(opts.metric, pt)
      data[pt.match_order] = v
      allValues.push(v)
    }
    const color = opts.colorByPlayer[p] ?? tc.axisLabel
    return {
      name: p,
      type: 'line' as const,
      data,
      lineStyle: { color, width: 2 },
      itemStyle: { color },
      symbol: 'circle' as const,
      symbolSize: 4,
      connectNulls: true,
    }
  })

  const bounds = damageAxisBounds(allValues, opts.oneLife)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 36, bottom: opts.showXAxis ? 32 : 8, left: 8, right: 40, containLabel: true },
    legend: { ...getLegendBase(tc), top: 0, type: 'scroll', data: players },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'line' },
      formatter: (params: unknown) => {
        const items = params as Array<{ seriesName: string; value: number | null; marker: string; dataIndex: number }>
        if (!Array.isArray(items) || items.length === 0) return ''
        const label = xLabels[items[0].dataIndex] ?? `#${items[0].dataIndex + 1}`
        const lines = items
          .filter((it) => it.value !== null && it.value !== undefined)
          .map((it) => `${it.marker} ${escapeHtml(it.seriesName ?? '')}: <b>${Math.round(it.value as number)}</b>`)
        return `<div style="font-size:11px">${label}<br/>${lines.join('<br/>')}</div>`
      },
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: xLabels,
      show: opts.showXAxis,
      axisLabel: { ...axis.axisLabel, interval: n > 30 ? Math.floor(n / 12) : 0 },
    },
    yAxis: {
      ...axis,
      type: 'value',
      min: bounds.min,
      max: bounds.max,
      axisLabel: { ...axis.axisLabel, formatter: (v: number) => `${Math.round(v)}` },
    },
    series: [
      ...playerSeries,
      {
        // Repère « 1 vie » : série fantôme hors légende (non listée dans
        // legend.data) → toujours visible, non togglable.
        name: opts.refLabel,
        type: 'line' as const,
        data: [] as Array<number | null>,
        silent: true,
        markLine: {
          silent: true,
          symbol: 'none',
          lineStyle: { color: tc.axisLabel, type: 'dashed' as const, width: 1 },
          data: [{ yAxis: opts.oneLife }],
          label: { formatter: opts.refLabel, position: 'end' as const, color: tc.axisLabel, fontSize: 9 },
        },
      },
    ],
  }
}
