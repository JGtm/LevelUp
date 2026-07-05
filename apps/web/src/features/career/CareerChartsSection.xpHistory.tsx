/**
 * CareerChartsSection — career.03 — XP history + projections (joueur + amis).
 *
 * Découpé depuis CareerChartsSection.tsx (audit #6 god-file split).
 * Inclut : real XP + estimé pré-sync + projection Héros + projection optimiste.
 *
 * Constantes métier issues de career_logic.py (src Python branche main) :
 *   CAREER_XP_LAUNCH_DATE = 2023-06-20 (CU32, introduction du système de rangs)
 *   WEEKLY_CHALLENGE_XP = 950 XP/semaine
 *   DAILY_CHALLENGE_XP = 500 XP/jour
 *   XP_BOOST_MULTIPLIER = 2.0 (consommable Double XP)
 */
import type { EChartsCoreOption } from 'echarts/core'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { intlLocale as toIntlLocale } from '@/lib/formatters'
import {
  getEChartsThemeColors,
  getAxisBase,
  getTooltipBase,
  getLegendBase,
  CHART_BG,
  escapeHtml,
} from '@/components/charts/_utils'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import { careerManifest } from '@/lib/i18n/generated/career'
import type { ManifestLocale } from '@/lib/i18n/format'
import type {
  CareerHistoryPoint,
  CareerProjections,
  FriendXPHistory,
} from '@/lib/api/types'

// ── Constantes métier ──────────────────────────────────────────────────────

const HERO_XP_TOTAL = 9_319_350
const CAREER_XP_LAUNCH_DATE = '2023-06-20'
const WEEKLY_CHALLENGE_XP = 950
const DAILY_CHALLENGE_XP = 500
const XP_BOOST_MULTIPLIER = 2.0

// Tokens couleur par joueur — mêmes que la page Escouade (squad/colors.ts).
// playerIdx 0 = joueur principal, 1..N = amis (cycle si > 4).
const XP_PLAYER_COLOR_TOKENS: SemanticToken[] = [
  'compare-a',
  'narrative-dominant',
  'perf-tier-3',
  'divergent-pos',
]

function xpPlayerColor(playerIdx: number): string {
  return resolveToken(XP_PLAYER_COLOR_TOKENS[playerIdx % XP_PLAYER_COLOR_TOKENS.length])
}

// ── Types internes ─────────────────────────────────────────────────────────

// playerName = clé de groupage ECharts : toutes les séries d'un joueur partagent
// le même name → un seul clic légende les affiche/masque toutes.
// lineType contrôle le motif visuel ; playerIdx contrôle la couleur.
interface XpSeriesMeta {
  playerName: string
  playerIdx: number
  lineType: 'real' | 'estimated' | 'proj-normal' | 'proj-optimiste'
}

// Label court du type de courbe affiché dans le tooltip.
const XP_TYPE_LABELS: Record<XpSeriesMeta['lineType'], string> = {
  'real': 'réel',
  'estimated': 'estimé',
  'proj-normal': 'Héros',
  'proj-optimiste': 'optimiste',
}

// ── Séries ─────────────────────────────────────────────────────────────────

function buildXpSeries(
  history: CareerHistoryPoint[],
  projections: CareerProjections | null,
  friendsXpHistory: FriendXPHistory[],
): ChartSeries<[string, number]>[] {
  if (history.length === 0) return []
  const series: ChartSeries<[string, number]>[] = []

  const push = (
    key: string,
    playerName: string,
    playerIdx: number,
    lineType: XpSeriesMeta['lineType'],
    datapoints: [string, number][],
  ) => series.push({ key, meta: { playerName, playerIdx, lineType } satisfies XpSeriesMeta, datapoints })

  // ── Joueur principal (playerIdx = 0, playerName = 'Vous') ──
  push('career.xp.history', 'Vous', 0, 'real', history.map(p => [p.recorded_at.slice(0, 10), p.xp_total]))

  const estimated = buildEstimatedXpPoints(history)
  if (estimated.length > 0) push('career.xp.estimated', 'Vous', 0, 'estimated', estimated)

  if (projections && projections.xp_per_day_active > 0) {
    const lastXp = history[history.length - 1].xp_total
    if (lastXp < HERO_XP_TOTAL) {
      const normalCurve = buildNormalProjection(history, projections)
      if (normalCurve.length > 0) push('career.xp.proj.hero', 'Vous', 0, 'proj-normal', normalCurve)

      const optCurve = buildOptimisticProjection(history, projections)
      if (optCurve.length > 0) push('career.xp.proj.opt', 'Vous', 0, 'proj-optimiste', optCurve)
    }
  }

  // ── Amis (playerIdx = 1..N) ──
  // Toutes les séries d'un ami partagent le même playerName (= gamertag).
  // Un clic sur leur entrée légende affiche/masque réel + estimé + projections.
  friendsXpHistory.forEach((friend, i) => {
    // Le contrat autorise `history: null` (le backend Go peut renvoyer null).
    const friendHistory = friend.history ?? []
    if (friendHistory.length === 0) return
    const idx = i + 1
    const gt = friend.gamertag

    push(`career.xp.friend.${gt}`, gt, idx, 'real', friendHistory.map(p => [p.recorded_at.slice(0, 10), p.xp_total]))

    const friendEst = buildEstimatedXpPoints(friendHistory)
    if (friendEst.length > 0) push(`career.xp.friend.${gt}.est`, gt, idx, 'estimated', friendEst)

    const friendProjs = deriveFriendProjections(friendHistory)
    if (friendProjs) {
      const lastXp = friendHistory[friendHistory.length - 1].xp_total
      if (lastXp < HERO_XP_TOTAL) {
        const nc = buildNormalProjection(friendHistory, friendProjs)
        if (nc.length > 0) push(`career.xp.friend.${gt}.proj.normal`, gt, idx, 'proj-normal', nc)

        const oc = buildOptimisticProjection(friendHistory, friendProjs)
        if (oc.length > 0) push(`career.xp.friend.${gt}.proj.opt`, gt, idx, 'proj-optimiste', oc)
      }
    }
  })

  return series
}

// ── Projections côté client pour les amis ─────────────────────────────────
// Miroir de computeActiveXPPerDay (Go career_service.go).

const INACTIVITY_GAP_DAYS = 14

function computeXPPerDayActive(history: CareerHistoryPoint[]): number {
  if (history.length < 2) return 0
  const xpDelta = history[history.length - 1].xp_total - history[0].xp_total
  if (xpDelta <= 0) return 0
  let totalActiveDays = 0
  for (let i = 1; i < history.length; i++) {
    const gapDays =
      (new Date(history[i].recorded_at).getTime() - new Date(history[i - 1].recorded_at).getTime()) / 86_400_000
    totalActiveDays += gapDays <= INACTIVITY_GAP_DAYS ? gapDays : INACTIVITY_GAP_DAYS / 2
  }
  return totalActiveDays > 0 ? xpDelta / totalActiveDays : 0
}

function deriveFriendProjections(history: CareerHistoryPoint[]): CareerProjections | null {
  const xpPerDay = computeXPPerDayActive(history)
  if (xpPerDay <= 0) return null
  const last = history[history.length - 1]
  const xpRemaining = HERO_XP_TOTAL - last.xp_total
  if (xpRemaining <= 0) return null
  const daysToHero = Math.min(xpRemaining / xpPerDay, 365 * 10)
  const heroDate = new Date(new Date(last.recorded_at).getTime() + daysToHero * 86_400_000)
  return {
    xp_per_day_active: xpPerDay,
    xp_per_day_fallback: xpPerDay,
    estimated_hero_date: heroDate.toISOString().slice(0, 10),
    estimated_rank_cap_date: null,
  }
}

function buildEstimatedXpPoints(history: CareerHistoryPoint[]): [string, number][] {
  const first = history[0]
  const firstDate = first.recorded_at.slice(0, 10)
  // Si le 1er snapshot est avant ou pile le lancement CU32, pas d'estimation à faire
  if (firstDate <= CAREER_XP_LAUNCH_DATE) return []
  // Ligne simple de (2023-06-20, 0) → (premier snapshot, xp_total)
  return [[CAREER_XP_LAUNCH_DATE, 0], [firstDate, first.xp_total] as [string, number]]
}

function buildNormalProjection(
  history: CareerHistoryPoint[],
  projections: CareerProjections,
): [string, number][] {
  const last = history[history.length - 1]
  const lastDate = new Date(last.recorded_at.slice(0, 10))
  const endDate = projections.estimated_hero_date
    ? new Date(projections.estimated_hero_date)
    : null
  if (!endDate) return []
  return buildWeeklyCurve(lastDate, last.xp_total, projections.xp_per_day_active, endDate)
}

function buildOptimisticProjection(
  history: CareerHistoryPoint[],
  projections: CareerProjections,
): [string, number][] {
  const last = history[history.length - 1]
  const lastDate = new Date(last.recorded_at.slice(0, 10))
  const challengeXpPerDay = WEEKLY_CHALLENGE_XP / 7.0 + DAILY_CHALLENGE_XP
  const optimisticRate = (projections.xp_per_day_active + challengeXpPerDay) * XP_BOOST_MULTIPLIER
  if (optimisticRate <= 0) return []
  const xpRemaining = HERO_XP_TOTAL - last.xp_total
  const daysNeeded = Math.min(xpRemaining / optimisticRate, 365 * 10)
  const heroDate = new Date(lastDate.getTime() + daysNeeded * 86_400_000)
  return buildWeeklyCurve(lastDate, last.xp_total, optimisticRate, heroDate)
}

function buildWeeklyCurve(
  startDate: Date,
  startXp: number,
  xpPerDay: number,
  heroDate: Date,
): [string, number][] {
  const points: [string, number][] = [[startDate.toISOString().slice(0, 10), startXp]]
  const totalDays = (heroDate.getTime() - startDate.getTime()) / 86_400_000
  const weeks = Math.ceil(totalDays / 7)
  for (let w = 1; w <= weeks; w++) {
    const d = new Date(startDate.getTime() + w * 7 * 86_400_000)
    const xp = Math.min(Math.round(startXp + xpPerDay * w * 7), HERO_XP_TOTAL)
    points.push([d.toISOString().slice(0, 10), xp])
    if (xp >= HERO_XP_TOTAL) break
  }
  return points
}

function buildXpHistoryOption(series: ChartSeries<[string, number]>[], locale: ManifestLocale): EChartsCoreOption {
  const tc = getEChartsThemeColors()
  const axisBase = getAxisBase(tc)
  const intlLocale = toIntlLocale(locale)

  // Map seriesIndex → meta pour le tooltip custom.
  const metaByIdx = new Map<number, XpSeriesMeta>()

  const echartsSeriesList = series.map((s, idx) => {
    const meta = s.meta as XpSeriesMeta | undefined
    if (meta) metaByIdx.set(idx, meta)

    const playerIdx = meta?.playerIdx ?? 0
    const lineType = meta?.lineType ?? 'real'
    const color = xpPlayerColor(playerIdx)
    const isReal = lineType === 'real'

    return {
      type: 'line',
      // Toutes les séries d'un même joueur partagent le même name ECharts :
      // un clic sur la légende montre/masque l'ensemble de leurs courbes.
      name: meta?.playerName ?? s.key,
      data: s.datapoints,
      itemStyle: { color },
      lineStyle: {
        color,
        width: isReal ? 2 : 1.5,
        type: isReal ? 'solid' : lineType === 'estimated' ? 'dotted' : 'dashed',
        opacity: lineType === 'estimated' ? 0.7 : lineType.startsWith('proj') ? 0.8 : 1,
      },
      symbol: isReal ? 'circle' : 'none',
      symbolSize: isReal ? 5 : 0,
      showSymbol: isReal,
      smooth: false,
      ...(s.key === 'career.xp.history' ? { markLine: buildHeroMarkLine() } : {}),
    }
  })

  // Tooltip — affiche le joueur + le type de courbe pour chaque entrée.
  const fmtXp = (v: number) =>
    v >= 1_000_000 ? `${(v / 1_000_000).toFixed(1)}M XP` : `${Math.round(v / 1_000)}k XP`

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const tooltipFormatter = (params: any[]) => {
    if (!Array.isArray(params) || params.length === 0) return ''
    const date = String(params[0]?.axisValueLabel ?? '').slice(0, 10)
    const lines = params
      .filter((p) => p.value != null && p.value[1] != null)
      .map((p) => {
        const m = metaByIdx.get(p.seriesIndex as number)
        const typeLabel = m ? XP_TYPE_LABELS[m.lineType] : ''
        const name = m?.playerName ?? p.seriesName
        return `${p.marker as string} <b>${escapeHtml(name ?? '')}</b> — ${typeLabel} : ${fmtXp(p.value[1] as number)}`
      })
    return `${date}<br/>${lines.join('<br/>')}`
  }

  return {
    backgroundColor: CHART_BG,
    grid: { left: 55, right: 20, top: 30, bottom: 60, containLabel: false },
    tooltip: { trigger: 'axis', ...getTooltipBase(tc), formatter: tooltipFormatter },
    legend: { ...getLegendBase(tc), bottom: 5 },
    xAxis: {
      type: 'time',
      ...axisBase,
      axisLabel: {
        ...axisBase.axisLabel,
        formatter: (value: number) =>
          new Intl.DateTimeFormat(intlLocale, { month: 'short', year: '2-digit' }).format(new Date(value)),
      },
    },
    yAxis: {
      type: 'value',
      name: 'XP',
      ...axisBase,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      axisLabel: {
        ...axisBase.axisLabel,
        formatter: (v: number) =>
          v >= 1_000_000 ? `${(v / 1_000_000).toFixed(1)}M` : `${Math.round(v / 1000)}k`,
      },
    },
    series: echartsSeriesList,
  }
}

function buildHeroMarkLine() {
  return {
    silent: true,
    symbol: 'none',
    data: [{
      yAxis: HERO_XP_TOTAL,
      lineStyle: { color: resolveToken('perf-tier-3') + '60', width: 1, type: 'dotted' },
      label: {
        show: true,
        formatter: 'Rang Héros',
        position: 'insideStartTop',
        color: resolveToken('perf-tier-3') + '99',
        fontSize: 10,
      },
    }],
  }
}

// ── Composant exporté ──────────────────────────────────────────────────────

export interface CareerXpHistoryChartProps {
  xpHistory: CareerHistoryPoint[]
  projections: CareerProjections | null
  friendsXpHistory: FriendXPHistory[]
  locale: ManifestLocale
}

export function CareerXpHistoryChart({
  xpHistory,
  projections,
  friendsXpHistory,
  locale,
}: CareerXpHistoryChartProps) {
  return (
    <ChartCard<[string, number]>
      title={careerManifest['career.charts.xp_history_title'][locale]}
      series={buildXpSeries(xpHistory, projections, friendsXpHistory)}
      height={340}
      buildOption={(series) => buildXpHistoryOption(series, locale)}
      emptyMessage={careerManifest['career.charts.placeholder_unavailable'][locale]}
    />
  )
}
