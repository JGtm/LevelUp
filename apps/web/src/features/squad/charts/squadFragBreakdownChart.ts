/**
 * squadFragBreakdownChart — « Répartition des frags » par joueur (barres empilées).
 *
 * Pendant escouade du sunburst « Répartition des frags » v2 (D8) : 1 barre
 * horizontale par joueur, segments = CLASSE d'arme (Épaule / Poing / Lourde /
 * Mêlée / Grenade / Capacités spartanes / Non attribué), longueur = total des
 * frags. Garde la sémantique part-d'un-tout tout en alignant les joueurs pour
 * comparer d'un coup d'œil.
 *
 * Consomme `frag_classes` (map gamertag → FragClassEntry[], agrégat serveur par
 * classe via fragdist.Build). N classes DYNAMIQUES (union des classes présentes,
 * ordre canonique FRAG_CLASS_ORDER). Couleurs = `fragClassColor(class)` (hex fixes
 * CVD-safe, mêmes que le sunburst → cohérence inter-pages). Labels de classe via le
 * manifeste i18n `frags` (injecté par `classLabel`). Aucun hex en dur.
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
import type { FragClassEntry } from '@/lib/api/types'
import { FRAG_CLASS_ORDER, fragClassColor } from '@/lib/accessibility/scales'

export interface FragBreakdownOpts {
  /** Ordre stable des joueurs (main d'abord). Sinon ordre alphabétique. */
  playerOrder?: string[]
  /** Libellé localisé d'une classe (manifeste `frags`). */
  classLabel: (className: string) => string
}

function orderedPlayers(rows: Record<string, FragClassEntry[]>, playerOrder?: string[]): string[] {
  if (playerOrder && playerOrder.length > 0) return playerOrder.filter((p) => rows[p] !== undefined)
  return Object.keys(rows).sort()
}

/** Classes présentes chez au moins un joueur, dans l'ordre canonique FRAG_CLASS_ORDER. */
function presentClasses(byPlayer: Map<string, Map<string, number>>): string[] {
  const present = new Set<string>()
  for (const kills of byPlayer.values()) {
    for (const [cls, v] of kills) if (v > 0) present.add(cls)
  }
  return FRAG_CLASS_ORDER.filter((c) => present.has(c))
}

/** Map class → kills pour un joueur (agrégat serveur déjà au niveau classe). */
function killsByClass(entries: FragClassEntry[]): Map<string, number> {
  const m = new Map<string, number>()
  for (const e of entries) m.set(e.class, (m.get(e.class) ?? 0) + e.kills)
  return m
}

export function buildFragBreakdownOption(
  rows: Record<string, FragClassEntry[]>,
  opts: FragBreakdownOpts,
): EChartsCoreOption {
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const players = orderedPlayers(rows, opts.playerOrder)
  if (players.length === 0) return { backgroundColor: CHART_BG }

  const byPlayer = new Map<string, Map<string, number>>()
  for (const player of players) byPlayer.set(player, killsByClass(rows[player] ?? []))

  const classes = presentClasses(byPlayer)
  if (classes.length === 0) return { backgroundColor: CHART_BG }

  const series = classes.map((cls) => ({
    name: opts.classLabel(cls),
    type: 'bar' as const,
    stack: 'frags',
    barMaxWidth: 18,
    itemStyle: { color: fragClassColor(cls) },
    data: players.map((p) => byPlayer.get(p)?.get(cls) ?? 0),
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
    legend: { ...getLegendBase(tc), data: classes.map((cls) => opts.classLabel(cls)) },
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
