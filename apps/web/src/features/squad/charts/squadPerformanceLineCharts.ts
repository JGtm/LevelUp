/**
 * squadPerformanceLineCharts — teammates.16 : 8 sous-charts performance escouade.
 *
 * Spec : .ai/charts_specs/teammates/16_trio_performance_charts.yaml (étendue
 *        à 8 charts par demande utilisateur — ajout MaxKillingSpree + HS+Perfect).
 *
 * Builders disponibles :
 *   - buildPerformanceLineOption        : 1 line par joueur sur une métrique simple.
 *   - buildKillsDeathsButterflyOption   : sous-chart spécial #1 (kills positifs / deaths négatifs).
 *   - buildHsPerfectOption              : sous-chart #8 — HS line normale + perfect_kills emphasés.
 *
 * Tous consomment `Record<gamertag, SquadPerformanceSeriesPoint[]>` aligné côté
 * serveur sur les matchs partagés (intersection). xAxis = match_order 0..N-1.
 */
import type { EChartsCoreOption } from 'echarts/core'
import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
  stackedAxisExtent,
} from '@/components/charts/_utils'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import { hexComplement, resolveToken } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility'

/** Convertit un hex #RRGGBB résolu par resolveToken en rgba avec alpha. */
function withAlpha(hex: string, alpha: number): string {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  if (isNaN(r) || isNaN(g) || isNaN(b)) return hex
  return `rgba(${r},${g},${b},${alpha})` // color-allow: 2026-09-06 (revue R1, C5) — CONVERSION d une couleur DEJA resolue (hex du theme) en rgba pour son alpha : ce helper ne nomme aucune couleur
}

/** 5 zones Y de 20 pts chacune, du pire (tier-5) au meilleur (tier-1). */
const PERF_ZONES: Array<{ yMin: number; yMax: number; token: SemanticToken }> = [
  { yMin: 0,  yMax: 20,  token: 'perf-tier-5' },
  { yMin: 20, yMax: 40,  token: 'perf-tier-4' },
  { yMin: 40, yMax: 60,  token: 'perf-tier-3' },
  { yMin: 60, yMax: 80,  token: 'perf-tier-2' },
  { yMin: 80, yMax: 100, token: 'perf-tier-1' },
]

export type PerformanceMetricKey =
  | 'kda'
  | 'assists'
  | 'accuracy'
  | 'avg_life_seconds'
  | 'performance_score'
  | 'max_killing_spree'

export interface CommonOpts {
  /** Mapping gamertag → couleur hex. */
  colorByPlayer: Record<string, string>
  /** Ordre des séries (sinon ordre alphabétique). */
  playerOrder?: string[]
  /** Labels personnalisés pour l'axe X (ex: "#1 Tribord"). Sinon xAxisLabels(n). */
  xLabels?: string[]
  /** Joueurs masqués via la légende React (séries vidées). Cf. SquadToggleLegendChart. */
  hiddenPlayers?: Set<string>
  /** Types masqués via la légende (la clé est le label de la série, ex. killsLabel). */
  hiddenTypes?: Set<string>
}

export interface PerformanceLineOpts extends CommonOpts {
  metric: PerformanceMetricKey
  /** Format `value.toFixed(decimals)` pour le tooltip. */
  decimals?: number
  /** Suffixe pour le label de l'axe Y et le tooltip (ex: " %", " s"). */
  unitSuffix?: string
  /** Multiplicateur appliqué avant affichage (ex: 100 pour accuracy 0..1 → %). */
  scale?: number
  /** 'bar' pour des batons groupés, 'line' (défaut) pour une ligne. */
  chartType?: 'line' | 'bar'
  /** Si défini, les barres dont la valeur est < ce seuil sont colorées avec hexComplement. */
  complementBelowValue?: number
  /** Affiche des zones de fond colorées via perf-tier-1..5 (pour metric performance_score). */
  showPerformanceZones?: boolean
  /** Si défini, trace une barre horizontale rouge gras à cette valeur Y (ex: seuil FDA = 1). */
  redReferenceLineAt?: number
}

function extractValue(p: SquadPerformanceSeriesPoint, metric: PerformanceMetricKey): number | null {
  switch (metric) {
    case 'kda':
      return p.kda ?? null
    case 'assists':
      return p.assists
    case 'accuracy':
      return p.accuracy ?? null
    case 'avg_life_seconds':
      return p.avg_life_seconds ?? null
    case 'performance_score':
      return p.performance_score ?? null
    case 'max_killing_spree':
      return p.max_killing_spree ?? null
  }
}

function fmtVal(v: number | null, decimals = 1, suffix = '', scale = 1): string {
  if (v === null) return '-'
  return `${(v * scale).toFixed(decimals)}${suffix}`
}

export function orderedPlayers(rows: Record<string, SquadPerformanceSeriesPoint[]>, playerOrder?: string[]): string[] {
  if (playerOrder && playerOrder.length > 0) {
    return playerOrder.filter((p) => rows[p] !== undefined)
  }
  return Object.keys(rows).sort()
}

export function maxLength(rows: Record<string, SquadPerformanceSeriesPoint[]>, players: string[]): number {
  let max = 0
  for (const p of players) {
    if (rows[p].length > max) max = rows[p].length
  }
  return max
}

export function xAxisLabels(n: number): string[] {
  return Array.from({ length: n }, (_, i) => `#${i + 1}`)
}

export function buildPerformanceLineOption(
  rows: Record<string, SquadPerformanceSeriesPoint[]>,
  opts: PerformanceLineOpts,
): EChartsCoreOption {
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const players = orderedPlayers(rows, opts.playerOrder)
  if (players.length === 0) return { backgroundColor: CHART_BG }

  const n = maxLength(rows, players)
  if (n === 0) return { backgroundColor: CHART_BG }

  const xLabels = opts.xLabels ?? xAxisLabels(n)
  const scale = opts.scale ?? 1
  const decimals = opts.decimals ?? 1
  const suffix = opts.unitSuffix ?? ''
  const isBar = opts.chartType === 'bar'
  const threshold = opts.complementBelowValue
  const showZones = isBar && (opts.showPerformanceZones ?? false)

  const perfZoneMarkArea = showZones
    ? {
        silent: true,
        data: PERF_ZONES.map(({ yMin, yMax, token }) => [
          { yAxis: yMin, itemStyle: { color: withAlpha(resolveToken(token), 0.12) } },
          { yAxis: yMax },
        ]),
      }
    : undefined

  const series = players.map((player, idx) => {
    const color = opts.colorByPlayer[player] ?? '#888' // color-allow: gris structurel pour joueur sans couleur attribuée
    const rawData = new Array<number | null>(n).fill(null)
    for (const p of rows[player]) {
      const idx = p.match_order
      const v = extractValue(p, opts.metric)
      rawData[idx] = v === null ? null : Number((v * scale).toFixed(decimals))
    }

    if (isBar) {
      const barData = rawData.map((v) => {
        const barColor =
          threshold !== undefined && v !== null && v < threshold ? hexComplement(color) : color
        return { value: v, itemStyle: { color: barColor } }
      })
      return {
        name: player,
        type: 'bar' as const,
        barMaxWidth: 12,
        data: barData,
        // markArea + markLine attachés au premier joueur seulement (rendus une fois pour tout le chart).
        ...(idx === 0 && perfZoneMarkArea ? { markArea: perfZoneMarkArea } : {}),
        ...(idx === 0 && opts.redReferenceLineAt !== undefined
          ? {
              markLine: {
                silent: true,
                symbol: 'none',
                lineStyle: { color: resolveToken('outcome-loss'), width: 2, type: 'solid' },
                data: [{ yAxis: opts.redReferenceLineAt }],
              },
            }
          : {}),
      }
    }

    return {
      name: player,
      type: 'line' as const,
      data: rawData,
      lineStyle: { color, width: 2 },
      itemStyle: { color },
      symbol: 'circle' as const,
      symbolSize: 5,
      connectNulls: true,
    }
  })

  return {
    backgroundColor: CHART_BG,
    grid: { top: 28, bottom: 36, left: 8, right: 24, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: isBar ? 'shadow' : 'line' },
      valueFormatter: (v: unknown) => fmtVal(typeof v === 'number' ? v : null, decimals, suffix),
    },
    legend: { ...getLegendBase(tc), data: players },
    xAxis: {
      ...axis,
      type: 'category',
      data: xLabels,
      axisLabel: {
        ...axis.axisLabel,
        interval: n > 30 ? Math.floor(n / 12) : 0,
      },
    },
    yAxis: {
      ...axis,
      type: 'value',
      ...(showZones ? { min: 0, max: 100 } : {}),
      axisLabel: {
        ...axis.axisLabel,
        formatter: (v: number) => `${v.toFixed(decimals)}${suffix}`,
      },
    },
    series,
  }
}

// ---------------------------------------------------------------------------
// Sous-chart #1 — Frags / Morts (butterfly)
// ---------------------------------------------------------------------------

export interface KillsDeathsButterflyOpts extends CommonOpts {
  killsLabel: string
  deathsLabel: string
}

export function buildKillsDeathsButterflyOption(
  rows: Record<string, SquadPerformanceSeriesPoint[]>,
  opts: KillsDeathsButterflyOpts,
): EChartsCoreOption {
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const players = orderedPlayers(rows, opts.playerOrder)
  if (players.length === 0) return { backgroundColor: CHART_BG }
  const n = maxLength(rows, players)
  if (n === 0) return { backgroundColor: CHART_BG }
  const xLabels = opts.xLabels ?? xAxisLabels(n)

  const hiddenPlayers = opts.hiddenPlayers ?? new Set<string>()
  const hiddenTypes = opts.hiddenTypes ?? new Set<string>()
  const showBonus = !hiddenTypes.has('Bonus')
  const bonusColor = resolveToken('bonus') // bonus assistances — violet, distinct des couleurs joueurs + opposés (morts)
  const emptyData = new Array<number | null>(n).fill(null)
  const seriesPerPlayer: Array<Record<string, unknown>> = []
  // Étendue d'axe calculée sur le JEU COMPLET (bonus + joueurs masqués inclus),
  // indépendamment des toggles (item 5, DEC-5) : une pile par joueur, kills+bonus
  // côté positif, morts (déjà négatives) côté négatif — cf. `stackedAxisExtent`.
  const positiveStacks: Array<Array<Array<number | null>>> = []
  const negativeStacks: Array<Array<Array<number | null>>> = []
  for (const player of players) {
    const color = opts.colorByPlayer[player] ?? '#888' // color-allow: gris structurel pour joueur sans couleur attribuée
    const negColor = hexComplement(color) // hue +180°, opaque — même convention que squadPerMinuteChart
    const killsHidden = hiddenPlayers.has(player) || hiddenTypes.has(opts.killsLabel)
    const deathsHidden = hiddenPlayers.has(player) || hiddenTypes.has(opts.deathsLabel)
    const playerHidden = hiddenPlayers.has(player)
    const killsData = new Array<number | null>(n).fill(null)
    const deathsData = new Array<number | null>(n).fill(null)
    const bonusData = new Array<number | null>(n).fill(null)
    for (const p of rows[player]) {
      killsData[p.match_order] = p.kills
      deathsData[p.match_order] = -p.deaths
      bonusData[p.match_order] = p.assists / 3
    }
    positiveStacks.push([killsData, bonusData])
    negativeStacks.push([deathsData])
    // stack identique → kills (positif), bonus (positif) et morts (négatif) partagent la même colonne x.
    seriesPerPlayer.push({
      name: `${player} — ${opts.killsLabel}`,
      type: 'bar',
      stack: player,
      barMaxWidth: 14,
      itemStyle: { color },
      data: killsHidden ? emptyData : killsData,
    })
    if (showBonus && !playerHidden) {
      seriesPerPlayer.push({
        name: `${player} — Bonus`,
        type: 'bar',
        stack: player,
        barMaxWidth: 14,
        itemStyle: { color: bonusColor },
        data: bonusData,
      })
    }
    seriesPerPlayer.push({
      name: `${player} — ${opts.deathsLabel}`,
      type: 'bar',
      stack: player,
      barMaxWidth: 14,
      itemStyle: { color: negColor },
      data: deathsHidden ? emptyData : deathsData,
    })
  }
  const { min: yMin, max: yMax } = stackedAxisExtent(positiveStacks, negativeStacks)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 36, bottom: 36, left: 8, right: 24, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      valueFormatter: (v: unknown) => (typeof v === 'number' ? `${Math.abs(v)}` : '-'),
    },
    legend: { show: false }, // légende custom React (cf. SquadToggleLegendChart)
    xAxis: { ...axis, type: 'category', data: xLabels },
    // min/max FIXÉS sur l'extent complet : sans ça, ECharts recalcule l'axe sur
    // les séries visibles à chaque render (`notMerge`) et un toggle Bonus/joueur
    // fait bouger toute l'échelle (item 5).
    yAxis: {
      ...axis,
      type: 'value',
      min: yMin,
      max: yMax,
      axisLine: { lineStyle: { color: tc.text, width: 1.5 } },
      axisLabel: { ...axis.axisLabel, formatter: (v: number) => `${Math.abs(v)}` },
    },
    series: seriesPerPlayer,
  }
}

// ---------------------------------------------------------------------------
// Sous-chart #8 — Tirs à la tête + frags parfaits (perfect mis en valeur)
// ---------------------------------------------------------------------------

export interface HsPerfectOpts extends CommonOpts {
  hsLabel: string
  perfectLabel: string
}

export function buildHsPerfectOption(
  rows: Record<string, SquadPerformanceSeriesPoint[]>,
  opts: HsPerfectOpts,
): EChartsCoreOption {
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const players = orderedPlayers(rows, opts.playerOrder)
  if (players.length === 0) return { backgroundColor: CHART_BG }
  const n = maxLength(rows, players)
  if (n === 0) return { backgroundColor: CHART_BG }
  const xLabels = opts.xLabels ?? xAxisLabels(n)

  // HS : barre opaque couleur joueur.
  // Perfect kills : stackée sur les HS, même couleur + bordure blanche + glow
  // dans la couleur du joueur → événement rare mis en valeur sans couleur externe.
  const hiddenPlayers = opts.hiddenPlayers ?? new Set<string>()
  const hiddenTypes = opts.hiddenTypes ?? new Set<string>()
  const emptyData = new Array<number | null>(n).fill(null)
  const series: Array<Record<string, unknown>> = []
  for (const player of players) {
    const color = opts.colorByPlayer[player] ?? '#888' // color-allow: gris structurel pour joueur sans couleur attribuée
    const hsHidden = hiddenPlayers.has(player) || hiddenTypes.has(opts.hsLabel)
    const perfectHidden = hiddenPlayers.has(player) || hiddenTypes.has(opts.perfectLabel)
    const hsData = new Array<number | null>(n).fill(null)
    const perfectData = new Array<number | null>(n).fill(null)
    for (const p of rows[player]) {
      hsData[p.match_order] = p.headshot_kills ?? null
      perfectData[p.match_order] = p.perfect_kills ?? null
    }
    series.push({
      name: `${player} — ${opts.hsLabel}`,
      type: 'bar',
      stack: player,
      barMaxWidth: 14,
      itemStyle: { color },
      data: hsHidden ? emptyData : hsData,
    })
    series.push({
      name: `${player} — ${opts.perfectLabel}`,
      type: 'bar',
      stack: player,
      barMaxWidth: 14,
      itemStyle: {
        color,
        borderColor: tc.text, // foreground — border d'emphase qui suit le thème
        borderWidth: 1.5,
        shadowBlur: 8,
        shadowColor: color,
      },
      emphasis: { focus: 'series' },
      data: perfectHidden ? emptyData : perfectData,
    })
  }

  return {
    backgroundColor: CHART_BG,
    grid: { top: 36, bottom: 36, left: 8, right: 24, containLabel: true },
    tooltip: { ...getTooltipBase(tc), trigger: 'axis', axisPointer: { type: 'shadow' } },
    legend: { show: false }, // légende custom React (cf. SquadToggleLegendChart)
    xAxis: { ...axis, type: 'category', data: xLabels },
    yAxis: {
      ...axis,
      type: 'value',
      axisLabel: { ...axis.axisLabel },
    },
    series,
  }
}

// ---------------------------------------------------------------------------
// Sous-chart Rang — MMR équipe par match (courbe par joueur)
// ---------------------------------------------------------------------------

export interface TeamMMROpts extends CommonOpts {
  /** Label pour la courbe MMR d'équipe (partagée par tous les joueurs). */
  mmrLabel: string
}

export function buildTeamMMROption(
  rows: Record<string, SquadPerformanceSeriesPoint[]>,
  opts: TeamMMROpts,
): EChartsCoreOption {
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const players = orderedPlayers(rows, opts.playerOrder)
  if (players.length === 0) return { backgroundColor: CHART_BG }
  const n = maxLength(rows, players)
  if (n === 0) return { backgroundColor: CHART_BG }
  const xLabels = opts.xLabels ?? xAxisLabels(n)

  const allSeries: Array<Record<string, unknown>> = []

  // MMR équipe — valeur identique pour tous les joueurs d'une même escouade sur un match.
  // On agrège la première valeur non-nulle par match_order (toutes identiques).
  const sharedMmrData = new Array<number | null>(n).fill(null)
  for (const player of players) {
    for (const p of rows[player]) {
      if (p.team_mmr != null && sharedMmrData[p.match_order] === null) {
        sharedMmrData[p.match_order] = p.team_mmr
      }
    }
  }

  // CSR et LUSR sont mutuellement exclusifs par match.
  // On ne crée une série que si le joueur a au moins un point de cette nature.
  type BarItem = { value: number | null; itemStyle: { color: string; opacity: number } }
  const nullItem = (c: string): BarItem => ({ value: null, itemStyle: { color: c, opacity: 0 } })
  const legendData: string[] = []

  for (const player of players) {
    const color = opts.colorByPlayer[player] ?? '#888' // color-allow: gris structurel pour joueur sans couleur attribuée
    const negColor = hexComplement(color)

    const csrData: BarItem[] = Array.from({ length: n }, () => nullItem(color))
    const lusrData: BarItem[] = Array.from({ length: n }, () => nullItem(color))
    let hasCsr = false
    let hasLusr = false

    for (const p of rows[player]) {
      const val = p.skill_rating ?? null
      if (val === null) continue
      const barColor = p.skill_delta !== undefined && p.skill_delta < 0 ? negColor : color
      const item: BarItem = { value: val, itemStyle: { color: barColor, opacity: 0.85 } }
      if (p.skill_rating_type === 'csr') { csrData[p.match_order] = item; hasCsr = true }
      else { lusrData[p.match_order] = item; hasLusr = true } // "lusr" ou type inconnu
    }

    if (hasCsr)  { allSeries.push({ name: `${player} — CSR`,  type: 'bar', barMaxWidth: 12, data: csrData,  z: 2 }); legendData.push(`${player} — CSR`) }
    if (hasLusr) { allSeries.push({ name: `${player} — LUSR`, type: 'bar', barMaxWidth: 12, data: lusrData, z: 2 }); legendData.push(`${player} — LUSR`) }
  }

  // Une seule courbe pour le MMR d'équipe (partagé par tous les joueurs de l'escouade).
  const hasMmr = sharedMmrData.some((v) => v !== null)
  if (hasMmr) {
    allSeries.push({
      name: opts.mmrLabel,
      type: 'line',
      data: sharedMmrData,
      lineStyle: { color: tc.text, width: 2, type: 'dashed' }, // color-allow: blanc neutre pour ligne partagée MMR équipe
      itemStyle: { color: tc.text },
      symbol: 'circle',
      symbolSize: 4,
      connectNulls: false,
      z: 3,
    })
    legendData.push(opts.mmrLabel)
  }

  if (allSeries.length === 0) return { backgroundColor: CHART_BG }

  return {
    backgroundColor: CHART_BG,
    grid: { top: 36, bottom: 36, left: 8, right: 24, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      valueFormatter: (v: unknown) => (typeof v === 'number' ? v.toFixed(0) : '-'),
    },
    legend: { ...getLegendBase(tc), data: legendData, type: 'scroll' },
    xAxis: {
      ...axis,
      type: 'category',
      data: xLabels,
      axisLabel: {
        ...axis.axisLabel,
        interval: n > 30 ? Math.floor(n / 12) : 0,
      },
    },
    yAxis: {
      ...axis,
      type: 'value',
      axisLabel: { ...axis.axisLabel, formatter: (v: number) => v.toFixed(0) },
    },
    series: allSeries,
  }
}
