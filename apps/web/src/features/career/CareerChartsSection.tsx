/**
 * CareerChartsSection — 4 charts ECharts pour la page Carrière.
 *
 * career.01 — jauge progression vers le prochain rang
 * career.02 — jauge progression vers le rang Héros
 * career.03 — XP réel + XP estimé pré-sync + projection héros + projection optimiste
 * career.04 — évolution LUSR / CSR par playlist_group + tier markAreas
 *
 * Constantes métier issues de career_logic.py (src Python branche main) :
 *   CAREER_XP_LAUNCH_DATE = 2023-06-20 (CU32, introduction du système de rangs)
 *   WEEKLY_CHALLENGE_XP = 950 XP/semaine
 *   DAILY_CHALLENGE_XP = 500 XP/jour
 *   XP_BOOST_MULTIPLIER = 2.0 (consommable Double XP)
 */
import type { EChartsCoreOption } from 'echarts/core'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import {
  getEChartsThemeColors,
  getAxisBase,
  getTooltipBase,
  getLegendBase,
  CHART_BG,
} from '@/components/charts/_utils'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import { LUSR_TIERS } from '@/lib/skillTiers'
import { careerManifest } from '@/lib/i18n/generated/career'
import type { ManifestLocale } from '@/lib/i18n/format'
import { useAppShellStore } from '@/stores/appShellStore'
import type {
  CareerHistoryPoint,
  CareerLusrCheckpoint,
  HeroProgress,
  CareerSummary,
  CareerProjections,
  FriendXPHistory,
} from '@/lib/api/types'

// ── Constantes métier ──────────────────────────────────────────────────────

const HERO_XP_TOTAL = 9_319_350
const CAREER_XP_LAUNCH_DATE = '2023-06-20'

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
const WEEKLY_CHALLENGE_XP = 950
const DAILY_CHALLENGE_XP = 500
const XP_BOOST_MULTIPLIER = 2.0

// LUSR_TIERS importé depuis lib/skillTiers.ts (whitelisté lint-no-hardcoded-fields)

// ── Types internes ─────────────────────────────────────────────────────────

interface GaugePoint { value: number; label: string; detail: string }

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

export interface CareerChartsSectionProps {
  xpHistory: CareerHistoryPoint[]
  lusrCheckpoints: CareerLusrCheckpoint[]
  summary: CareerSummary | null
  heroProgress: HeroProgress | null
  projections: CareerProjections | null
  friendsXpHistory?: FriendXPHistory[]
  /** Colonne droite optionnelle affichée à côté des charts XP + LUSR (sidebar achievements). */
  rightSlot?: React.ReactNode
}

// ── Composant ──────────────────────────────────────────────────────────────

export function CareerChartsSection({
  xpHistory,
  lusrCheckpoints,
  summary,
  heroProgress,
  projections,
  friendsXpHistory,
  rightSlot,
}: CareerChartsSectionProps) {
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  return (
    <div className="space-y-4" data-testid="career-charts-section">
      {/* career.01 + career.02 — jauges rang + héros */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <ChartCard<GaugePoint>
          title={careerManifest['career.charts.rank_gauge_title'][locale]}
          series={summary ? [rankGaugeSeries(summary)] : []}
          height={280}
          buildOption={buildRankGaugeOption}
          emptyMessage={careerManifest['career.charts.placeholder_unavailable'][locale]}
        />
        <ChartCard<GaugePoint>
          title={careerManifest['career.charts.hero_gauge_title'][locale]}
          series={heroProgress ? [heroGaugeSeries(heroProgress)] : []}
          height={280}
          buildOption={buildHeroGaugeOption}
          emptyMessage={careerManifest['career.charts.placeholder_unavailable'][locale]}
        />
      </div>
      {/* career.03 + career.04 — timeseries XP + LUSR, avec colonne droite optionnelle */}
      <div className={rightSlot ? 'grid grid-cols-1 gap-4 xl:grid-cols-[1fr_288px]' : 'space-y-4'}>
        <div className="min-w-0 space-y-4">
          <ChartCard<[string, number]>
            title={careerManifest['career.charts.xp_history_title'][locale]}
            series={buildXpSeries(xpHistory, projections, friendsXpHistory ?? [])}
            height={340}
            buildOption={(series) => buildXpHistoryOption(series, locale)}
            emptyMessage={careerManifest['career.charts.placeholder_unavailable'][locale]}
          />
          <ChartCard<[string, number]>
            title={careerManifest['career.charts.lusr_evolution_title'][locale]}
            series={buildLusrSeries(lusrCheckpoints)}
            height={320}
            buildOption={(series) => buildLusrEvolutionOption(series, locale)}
            emptyMessage={careerManifest['career.charts.placeholder_unavailable'][locale]}
          />
        </div>
        {rightSlot}
      </div>
    </div>
  )
}

// ── career.01 — jauge rang ─────────────────────────────────────────────────

function rankGaugeSeries(summary: CareerSummary): ChartSeries<GaugePoint> {
  // progress_pct est déjà en 0..100 côté API Go
  const pct = Math.min(100, summary.progress_pct)
  return {
    key: 'career.gauge.rank',
    datapoints: [{
      value: pct,
      label: summary.rank_label,
      detail: `${summary.current_xp.toLocaleString('fr-FR')} / ${summary.xp_for_next_rank.toLocaleString('fr-FR')} XP`,
    }],
  }
}

function buildRankGaugeOption(series: ChartSeries<GaugePoint>[]): EChartsCoreOption {
  const point = series[0]?.datapoints[0]
  if (!point) return {}
  return buildGaugeOption(point, resolveToken('chart-series-1'), getEChartsThemeColors())
}

// ── career.02 — jauge Héros ────────────────────────────────────────────────

function heroGaugeSeries(hero: HeroProgress): ChartSeries<GaugePoint> {
  // percentage est déjà en 0..100 côté API Go (× 100 fait côté service)
  const pct = Math.min(100, hero.percentage)
  const acquired = hero.xp_total_required - hero.xp_remaining
  return {
    key: 'career.gauge.hero',
    datapoints: [{
      value: pct,
      label: 'Progression vers Héros',
      detail: `${acquired.toLocaleString('fr-FR')} / ${HERO_XP_TOTAL.toLocaleString('fr-FR')} XP`,
    }],
  }
}

function buildHeroGaugeOption(series: ChartSeries<GaugePoint>[]): EChartsCoreOption {
  const point = series[0]?.datapoints[0]
  if (!point) return {}
  return buildGaugeOption(point, resolveToken('perf-tier-2'), getEChartsThemeColors())
}

// ── Gauge builder partagé ──────────────────────────────────────────────────

function buildGaugeOption(
  point: GaugePoint,
  progressColor: string,
  tc: ReturnType<typeof getEChartsThemeColors>,
): EChartsCoreOption {
  const trackColors: [number, string][] = [
    [0.25, resolveToken('perf-tier-1') + '30'],
    [0.5,  resolveToken('perf-tier-2') + '30'],
    [0.75, resolveToken('perf-tier-3') + '30'],
    [1,    resolveToken('perf-tier-4') + '30'],
  ]
  return {
    backgroundColor: CHART_BG,
    tooltip: {
      formatter: () => `${point.label}<br/>${point.detail}<br/>${Math.round(point.value)}%`,
      ...getTooltipBase(tc),
    },
    title: {
      text: point.label,
      subtext: point.detail,
      left: 'center',
      top: '82%',
      textStyle: { color: tc.text, fontSize: 13 },
      subtextStyle: { color: tc.axisLabel, fontSize: 11 },
    },
    series: [{
      type: 'gauge',
      min: 0, max: 100,
      startAngle: 215, endAngle: -35,
      radius: '88%', center: ['50%', '58%'],
      data: [{ value: point.value, name: '' }],
      axisLine: { lineStyle: { width: 16, color: trackColors } },
      progress: { show: true, width: 16, roundCap: false, itemStyle: { color: progressColor } },
      pointer: { show: true, length: '60%', width: 4, itemStyle: { color: tc.text } },
      axisTick: { show: false },
      splitLine: { show: true, distance: 0, length: 5, lineStyle: { color: tc.splitLine, width: 1 } },
      axisLabel: { show: true, distance: 20, color: tc.axisLabel, fontSize: 9, formatter: (v: number) => Math.round(v).toString() },
      title: { show: false },
      detail: { show: true, offsetCenter: [0, '22%'], color: tc.text, fontSize: 28, fontWeight: 'bold', formatter: (v: number) => `${Math.round(v)}%` },
    }],
  }
}

// ── career.03 — XP history + projections ──────────────────────────────────

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
    if (friend.history.length === 0) return
    const idx = i + 1
    const gt = friend.gamertag

    push(`career.xp.friend.${gt}`, gt, idx, 'real', friend.history.map(p => [p.recorded_at.slice(0, 10), p.xp_total]))

    const friendEst = buildEstimatedXpPoints(friend.history)
    if (friendEst.length > 0) push(`career.xp.friend.${gt}.est`, gt, idx, 'estimated', friendEst)

    const friendProjs = deriveFriendProjections(friend.history)
    if (friendProjs) {
      const lastXp = friend.history[friend.history.length - 1].xp_total
      if (lastXp < HERO_XP_TOTAL) {
        const nc = buildNormalProjection(friend.history, friendProjs)
        if (nc.length > 0) push(`career.xp.friend.${gt}.proj.normal`, gt, idx, 'proj-normal', nc)

        const oc = buildOptimisticProjection(friend.history, friendProjs)
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
  const intlLocale = locale === 'fr' ? 'fr-FR' : 'en-US'

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
        return `${p.marker as string} <b>${name}</b> — ${typeLabel} : ${fmtXp(p.value[1] as number)}`
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

// ── career.04 — évolution LUSR ─────────────────────────────────────────────

// Labels via manifest i18n (career.toml → generated/career.ts).
// Tokens sémantiques clairement distincts par catégorie.
const LUSR_GROUP_TOKENS: Record<string, SemanticToken> = {
  ranked: 'compare-b',
  arena:  'compare-a',
  btb:    'divergent-pos',
  fun:    'narrative-humiliation',
  social: 'narrative-dominant',
}

function lusrGroupColor(group: string): string {
  const token: SemanticToken = LUSR_GROUP_TOKENS[group] ?? 'chart-series-1'
  return resolveToken(token)
}

function buildLusrSeries(checkpoints: CareerLusrCheckpoint[]): ChartSeries<[string, number]>[] {
  // Clé de groupage : (rating_type, playlist_group) → une série par combinaison.
  const byKey = new Map<string, { group: string; ratingType: string; playlistName: string; pts: Map<string, number> }>()

  for (const cp of checkpoints) {
    if (!cp.recorded_at) continue
    const group = cp.playlist_group ?? 'arena'
    const ratingType = cp.rating_type ?? 'LUSR'
    const seriesKey = `${ratingType}:${group}`
    const date = cp.recorded_at.slice(0, 10)

    if (!byKey.has(seriesKey)) {
      byKey.set(seriesKey, { group, ratingType, playlistName: cp.playlist_name || group, pts: new Map() })
    } else {
      // Mise à jour au dernier nom connu (tri ASC côté Go → le plus récent écrase)
      byKey.get(seriesKey)!.playlistName = cp.playlist_name || group
    }
    byKey.get(seriesKey)!.pts.set(date, cp.rating_value)
  }

  return Array.from(byKey.entries()).map(([seriesKey, { group, ratingType, playlistName, pts }]) => {
    const label = `${playlistName} (${ratingType})`
    return {
      key: `career.lusr.${seriesKey}`,
      meta: { label, groupKey: group, ratingType },
      datapoints: Array.from(pts.entries())
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([date, val]) => [date, val] as [string, number]),
    }
  })
}

function buildLusrEvolutionOption(series: ChartSeries<[string, number]>[], locale: ManifestLocale): EChartsCoreOption {
  const tc = getEChartsThemeColors()
  const axisBase = getAxisBase(tc)
  const intlLocale = locale === 'fr' ? 'fr-FR' : 'en-US'

  const allRatings = series.flatMap(s => s.datapoints.map(p => p[1]))
  const dataMin = allRatings.length > 0 ? Math.min(...allRatings) : 0
  const tierMin = LUSR_TIERS.findLast(t => t.min <= dataMin)?.min ?? 0

  // Fenêtre par défaut = 12 derniers mois depuis le point le plus récent.
  // Si l'historique est plus court, on affiche tout (dataZoom laisse dérouler vers le passé).
  const allDates = series.flatMap(s => s.datapoints.map(p => p[0]))
  const lastDate = allDates.length > 0 ? allDates.reduce((a, b) => (a > b ? a : b)) : null
  const firstDate = allDates.length > 0 ? allDates.reduce((a, b) => (a < b ? a : b)) : null
  const defaultWindowStart = lastDate
    ? new Date(new Date(lastDate).getTime() - 365 * 86_400_000).toISOString().slice(0, 10)
    : null
  // N'active le zoom que si les données dépassent 13 mois d'historique.
  const needsZoom = firstDate !== null && defaultWindowStart !== null && firstDate < defaultWindowStart

  // Légende explicite = les noms des séries réelles seulement.
  // La série fantôme n'est pas incluse dans legend.data → absente de la légende.
  const legendData = series.map(
    s => (s.meta as { label: string } | undefined)?.label ?? s.key,
  )

  // Les bandes de tier sont attachées à une série fantôme (pas de données, pas de légende).
  // Sans ghost, les markArea disparaissent quand la série porteuse est masquée.
  const ghostSeries = {
    type: 'line',
    name: '__lusr_tiers__',
    data: [],
    silent: true,
    legendHoverLink: false,
    markArea: buildLusrTierMarkArea(locale),
  }

  const echartsSeriesList = [
    ghostSeries,
    ...series.map((s) => {
      const meta = s.meta as { label: string; groupKey: string; ratingType: string } | undefined
      const label = meta?.label ?? s.key
      // LUSR = ligne pleine, CSR = ligne pointillée pour distinguer visuellement.
      const isCSR = meta?.ratingType === 'CSR'
      const color = lusrGroupColor(meta?.groupKey ?? '')
      return {
        type: 'line',
        name: label,
        data: s.datapoints,
        itemStyle: { color },
        lineStyle: { color, width: 2, type: isCSR ? ('dashed' as const) : ('solid' as const) },
        symbol: 'circle',
        symbolSize: 5,
        showSymbol: true,
        smooth: false,
      }
    }),
  ]

  return {
    backgroundColor: CHART_BG,
    grid: { left: 50, right: 20, top: 30, bottom: 60, containLabel: false },
    tooltip: { trigger: 'axis', ...getTooltipBase(tc) },
    legend: { ...getLegendBase(tc), bottom: 5, data: legendData },
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
      name: careerManifest['career.charts.lusr_rating_axis_y'][locale],
      min: tierMin,
      ...axisBase,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
    },
    ...(needsZoom ? {
      dataZoom: [{
        type: 'inside',
        startValue: defaultWindowStart,
        endValue: lastDate,
        filterMode: 'none',
        zoomOnMouseWheel: true,
        moveOnMouseMove: true,
      }],
    } : {}),
    series: echartsSeriesList,
  }
}

function buildLusrTierMarkArea(locale: ManifestLocale) {
  return {
    silent: true,
    label: { show: true, position: 'insideTopLeft' as const, fontSize: 10, opacity: 0.6 },
    data: LUSR_TIERS.map(tier => [
      {
        yAxis: tier.min,
        name: locale === 'fr' ? tier.fr : tier.en,
        itemStyle: { color: resolveToken(tier.token) + '30' },
        label: { color: resolveToken(tier.token) },
      },
      { yAxis: tier.max },
    ]),
  }
}
