/**
 * squadFragBreakdownChart — « Répartition des frags » par joueur (barres empilées).
 *
 * Pendant escouade du donut « Répartition des frags » de Synthesis : 1 barre
 * horizontale par joueur, segments = type de frag (mêlée / arme lourde / grenade
 * / autres), longueur = total des frags. Garde la sémantique part-d'un-tout du
 * donut tout en alignant les joueurs pour comparer d'un coup d'œil.
 *
 * Agrège les points per-match de `SquadPerformanceSeriesPoint` par joueur ;
 * `other = max(0, Σkills − (Σmêlée + Σlourde + Σgrenade))` (même règle que le
 * donut). Couleurs PAR TYPE via tokens sémantiques chart-series 1/6/7/8 — mêmes
 * que le donut → cohérence inter-pages. Aucun hex en dur.
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
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import { resolveToken } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility'

export interface FragBreakdownLabels {
  melee: string
  powerWeapon: string
  grenade: string
  other: string
}

export interface FragBreakdownOpts {
  /** Ordre stable des joueurs (main d'abord). Sinon ordre alphabétique. */
  playerOrder?: string[]
  labels: FragBreakdownLabels
}

type SegmentKey = 'melee' | 'powerWeapon' | 'grenade' | 'other'

/** 4 segments mutuellement exclusifs — tokens DISTINCTS, alignés sur le donut. */
const SEGMENTS: Array<{ key: SegmentKey; token: SemanticToken }> = [
  { key: 'melee', token: 'chart-series-1' },
  { key: 'powerWeapon', token: 'chart-series-6' },
  { key: 'grenade', token: 'chart-series-7' },
  { key: 'other', token: 'chart-series-8' },
]

type Breakdown = Record<SegmentKey, number>

function aggregate(points: SquadPerformanceSeriesPoint[]): Breakdown {
  let melee = 0
  let powerWeapon = 0
  let grenade = 0
  let kills = 0
  for (const p of points) {
    melee += p.melee_kills ?? 0
    powerWeapon += p.power_weapon_kills ?? 0
    grenade += p.grenade_kills ?? 0
    kills += p.kills
  }
  return { melee, powerWeapon, grenade, other: Math.max(0, kills - (melee + powerWeapon + grenade)) }
}

function orderedPlayers(rows: Record<string, SquadPerformanceSeriesPoint[]>, playerOrder?: string[]): string[] {
  if (playerOrder && playerOrder.length > 0) return playerOrder.filter((p) => rows[p] !== undefined)
  return Object.keys(rows).sort()
}

export function buildFragBreakdownOption(
  rows: Record<string, SquadPerformanceSeriesPoint[]>,
  opts: FragBreakdownOpts,
): EChartsCoreOption {
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const players = orderedPlayers(rows, opts.playerOrder)
  if (players.length === 0) return { backgroundColor: CHART_BG }

  const byPlayer = new Map<string, Breakdown>()
  for (const player of players) byPlayer.set(player, aggregate(rows[player] ?? []))

  const labelOf: Record<SegmentKey, string> = {
    melee: opts.labels.melee,
    powerWeapon: opts.labels.powerWeapon,
    grenade: opts.labels.grenade,
    other: opts.labels.other,
  }

  const series = SEGMENTS.map(({ key, token }) => ({
    name: labelOf[key],
    type: 'bar' as const,
    stack: 'frags',
    barMaxWidth: 18,
    itemStyle: { color: resolveToken(token) },
    data: players.map((p) => byPlayer.get(p)?.[key] ?? 0),
  }))

  return {
    backgroundColor: CHART_BG,
    grid: { top: 32, bottom: 24, left: 8, right: 24, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => {
        const arr = (Array.isArray(params) ? params : [params]) as Array<{
          name: string
          seriesName: string
          value: number
          marker: string
        }>
        if (arr.length === 0) return ''
        let total = 0
        const lines = arr.map((p) => {
          const v = typeof p.value === 'number' ? p.value : 0
          total += v
          return `${p.marker} ${escapeHtml(p.seriesName ?? '')} : <b>${v}</b>`
        })
        return `${escapeHtml(arr[0].name ?? '')}<br/>${lines.join('<br/>')}<br/>Total : <b>${total}</b>`
      },
    },
    legend: { ...getLegendBase(tc), data: SEGMENTS.map((s) => labelOf[s.key]) },
    xAxis: { ...axis, type: 'value', minInterval: 1 },
    yAxis: {
      ...axis,
      type: 'category',
      data: players,
      inverse: true, // main player en haut (category[0] en haut)
    },
    series,
  }
}
