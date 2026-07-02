/**
 * squadEfficiencyChart — teammates : Rendement & Résistance par match, en dégâts
 * BRUTS (repère 225 = 1 vie de Spartan, bouclier + santé).
 *
 * Chaque track (1 par joueur, sélectionné via boutons) affiche 2 lignes :
 *   - trait plein     : dégâts / frag = dégâts infligés / frags
 *   - trait pointillé : dégâts / mort = dégâts subis / morts
 *
 * Ligne repère à y=225 (= 1 vie). Les lignes sont colorées par DÉGRADÉ (cf.
 * oneLifeDamageGradient), pas par la couleur du joueur :
 *   - dégâts/frag : au plus proche de 225, au plus efficace (vert) ;
 *   - dégâts/mort : encaisser plus de 225/mort = bonne résistance (vert).
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
  ONE_LIFE_DAMAGE,
  damageAxisBounds,
  damagePerDeath,
  damagePerKill,
  defensiveDamageGradient,
  offensiveDamageGradient,
} from '@/lib/charts/oneLifeDamageGradient'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import { truncateMap } from '@/lib/charts/matchLabels'

export interface EfficiencyTrackOpts {
  rendementLabel: string
  resistanceLabel: string
  refLabel: string
  showXAxis: boolean
  /** PV-pour-tuer du titre (repère « 1 vie » + dégradés). Défaut ONE_LIFE_DAMAGE. */
  oneLife?: number
}

export function buildSquadEfficiencyTrackOption(
  pts: SquadPerformanceSeriesPoint[],
  opts: EfficiencyTrackOpts,
): EChartsCoreOption {
  if (pts.length === 0) return { backgroundColor: CHART_BG }

  const n = pts.reduce((max, p) => Math.max(max, p.match_order), 0) + 1
  const dmgKill = new Array<number | null>(n).fill(null)
  const dmgDeath = new Array<number | null>(n).fill(null)
  const mapByOrder = new Array<string | undefined>(n).fill(undefined)

  for (const p of pts) {
    const idx = p.match_order
    dmgKill[idx] = damagePerKill(p.damage_dealt, p.kills)
    dmgDeath[idx] = damagePerDeath(p.damage_taken, p.deaths)
    if (p.map_name) {
      mapByOrder[idx] = p.map_name
    }
  }

  // Étiquettes X au format `#N\nMap` (aligné sur les autres charts par match).
  const xLabels = Array.from({ length: n }, (_, i) => {
    const m = mapByOrder[i]
    return m ? `#${i + 1}\n${truncateMap(m)}` : `#${i + 1}`
  })
  const oneLife = opts.oneLife ?? ONE_LIFE_DAMAGE
  const bounds = damageAxisBounds([...dmgKill, ...dmgDeath], oneLife)
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
          .map((it) => `${it.marker} ${escapeHtml(it.seriesName ?? '')}: <b>${Math.round(it.value as number)}</b>`)
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
      min: bounds.min,
      max: bounds.max,
      axisLabel: {
        ...axis.axisLabel,
        formatter: (v: number) => `${Math.round(v)}`,
      },
    },
    series: [
      {
        name: opts.rendementLabel,
        type: 'line',
        data: dmgKill,
        lineStyle: { color: offensiveDamageGradient(dmgKill, oneLife), width: 2 },
        symbol: 'none',
        connectNulls: true,
        markLine: {
          silent: true,
          symbol: 'none',
          lineStyle: { color: tc.axisLabel, type: 'dashed', width: 1 },
          data: [{ yAxis: oneLife }],
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
        data: dmgDeath,
        lineStyle: { color: defensiveDamageGradient(dmgDeath, oneLife), width: 2, type: 'dashed' },
        symbol: 'none',
        connectNulls: true,
      },
    ],
  }
}

export interface RendementMultiOpts {
  refLabel: string
  /** PV-pour-tuer du titre (repère ligne « 1 vie »). */
  oneLife: number
  /** gamertag → couleur hex résolue : 1 série/joueur, colorée par joueur. */
  colorByPlayer: Record<string, string>
  showXAxis: boolean
}

/**
 * buildSquadRendementMultiOption — mode mono-métrique (titre SANS résistance, ex.
 * Halo 5 qui ne fournit pas damage_taken) : une courbe « dégâts / frag » PAR
 * joueur sur un seul graphe, colorée par joueur, affichée/masquée via la légende
 * native ECharts (clic). Axe X = ordre de match PARTAGÉ de l'escouade
 * (`match_order`, intersection chronologique commune à tous les joueurs).
 */
export function buildSquadRendementMultiOption(
  rowsByPlayer: Record<string, SquadPerformanceSeriesPoint[]>,
  players: string[],
  opts: RendementMultiOpts,
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
      const v = damagePerKill(pt.damage_dealt, pt.kills)
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
