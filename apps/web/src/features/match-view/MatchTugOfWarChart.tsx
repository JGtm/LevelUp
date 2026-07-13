/**
 * MatchTugOfWarChart — match_view.10 (Dominance : histogramme momentum)
 *
 * Histogramme divergent par tranche temporelle (référence : Squeeze Momentum
 * TradingView) :
 *   - une barre signée par tranche, `delta = kills alliés − kills ennemis` ;
 *   - au-dessus de zéro : couleur `team-ally` ; en-dessous : `team-enemy`
 *     (tokens configurables par l'utilisateur, résolus live via CSS var) ;
 *   - intensité (DEC-4) : opacité pleine quand le momentum se renforce
 *     (`trend: 'up'`), atténuée quand il s'essouffle (`'down'`) — alpha-mix
 *     structurel via `hexToRgba` (source unique `components/charts/_utils`) ;
 *   - `delta = 0` → pas de barre (DEC-5) ; l'axe de symétrie est `y = 0` (DEC-7)
 *     et l'échelle Y est symétrique dynamique (DEC-6).
 *
 * Kill feed conservé (DEC-2) sur le même axe X, dans une grille double :
 *   Grille haute (histogramme + lane alliée + kills alliés + vagues alliées)
 *   Grille basse (lane ennemie + kills ennemis + vagues ennemies)
 *
 * Le calcul par tranche (delta, counts, cumuls, trend) et l'affectation des
 * kills vivent dans `_momentum.ts` (pur, testé). Source de données :
 * `combat_tab.tug_of_war` (bornes de bins), `combat_tab.highlight_events`
 * (kill feed + recompute des counts par équipe), scoreboard (team_side).
 */
import type { EChartsCoreOption } from 'echarts/core'
import { useCallback } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import {
  CHART_BG,
  escapeHtml,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
  hexToRgba,
  type EChartsThemeColors,
} from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { displayPlayerName } from '@/lib/players/displayName'
import type {
  MatchHighlightEvent,
  MatchScoreboardRow,
  MatchTugOfWarBin,
} from '@/lib/api/types'
import { computeMomentumBins, type MomentumBin, type MomentumKill } from './_momentum'
import type { MatchViewText } from './i18n'

interface Props {
  bins: MatchTugOfWarBin[] | null | undefined
  events: MatchHighlightEvent[] | null | undefined
  scoreboard: MatchScoreboardRow[] | null | undefined
  meXUID: string | null
  t: MatchViewText
}

/** Fenêtre max (ms) entre deux kills consécutifs pour rester dans la même vague. */
const WAVE_WINDOW_MS = 8_000
/** Nombre minimum de kills pour qu'une vague soit considérée notable. */
const WAVE_MIN_KILLS = 3
/** Opacités DEC-4 : momentum qui se renforce (up) vs qui s'essouffle (down). */
const OP_STRONG = 0.9
const OP_WEAK = 0.45
/** Gap inter-catégories : histogramme serré (effet Squeeze Momentum). */
const BAR_GAP = '20%'
/** Facteurs de layout dérivés de yMax (échelle symétrique dynamique, DEC-6). */
const LANE_FACTOR = 1.5 // lane des kills alliés, au-dessus des barres positives
const TOP_FACTOR = 1.95 // headroom pour les labels ×N des vagues alliées
const BOTTOM_FACTOR = 1.15 // juste sous la barre négative la plus profonde

type XuidMeta = Map<string, { gamertag: string; ally: boolean }>

interface TeamWave {
  xStart: number; xEnd: number; count: number
  tStartMs: number; tEndMs: number
  waveKills: MomentumKill[]
}

/** Détecte les vagues collectives d'une équipe depuis ses kills triés par temps. */
function detectTeamWaves(sorted: MomentumKill[]): TeamWave[] {
  const out: TeamWave[] = []
  let i = 0
  while (i < sorted.length) {
    let j = i + 1
    while (j < sorted.length && sorted[j].tMs - sorted[j - 1].tMs <= WAVE_WINDOW_MS) j++
    if (j - i >= WAVE_MIN_KILLS) {
      out.push({
        xStart: sorted[i].binIdx - 0.5 + sorted[i].fracInBin,
        xEnd:   sorted[j - 1].binIdx - 0.5 + sorted[j - 1].fracInBin,
        count:  j - i,
        tStartMs: sorted[i].tMs,
        tEndMs:   sorted[j - 1].tMs,
        waveKills: sorted.slice(i, j),
      })
    }
    i = j
  }
  return out
}

function formatMmSs(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = Math.max(0, Math.floor(seconds % 60))
  return `${m}:${s.toString().padStart(2, '0')}`
}

/** Résout l'appartenance d'équipe (allié/ennemi) et le gamertag par xuid. */
function resolveXuidMeta(scoreboard: MatchScoreboardRow[] | null | undefined, meXUID: string | null): XuidMeta {
  const sb = scoreboard ?? []
  const meRow = meXUID ? sb.find((r) => r.xuid === meXUID) : undefined
  const allyTeam = meRow?.team_side ?? null
  const meta: XuidMeta = new Map()
  for (const r of sb) {
    const ally = r.is_me || (allyTeam != null && r.team_side === allyTeam)
    meta.set(r.xuid, { gamertag: displayPlayerName(r.gamertag, r.xuid), ally })
  }
  return meta
}

/** Tooltip par tranche (axis) : delta signé, X/Y kills, cumuls (remplace DEC-3). */
function buildBinTooltips(momentum: MomentumBin[], categories: string[], t: MatchViewText): string[] {
  return momentum.map((b, i) => {
    const leader = b.delta > 0 ? t.combatTeamLabel : b.delta < 0 ? t.combatEnemyLabel : null
    const sign = b.delta > 0 ? '+' : ''
    const deltaLine = leader
      ? `${t.combatMomentumDelta} : <b>${sign}${b.delta}</b> (${escapeHtml(leader)})`
      : `${t.combatMomentumDelta} : <b>0</b>`
    return [
      `<b>${escapeHtml(categories[i])}</b>`,
      deltaLine,
      `${t.combatTeamLabel} ${b.teamKills} / ${t.combatEnemyLabel} ${b.enemyKills}`,
      `${t.combatMomentumCumul} : ${b.cumTeam} – ${b.cumEnemy}`,
    ].join('<br/>')
  })
}

/** Formatter du tooltip axis : ancre sur la barre (dataIndex = catégorie). */
function binTooltipFormatter(binTooltips: string[]) {
  return (params: unknown): string => {
    const arr = Array.isArray(params) ? params : [params]
    const bar = arr.find(
      (p) => (p as { seriesType?: string }).seriesType === 'bar' && typeof (p as { dataIndex?: number }).dataIndex === 'number',
    )
    const anchor = bar ?? arr.find((p) => typeof (p as { dataIndex?: number }).dataIndex === 'number')
    const i = anchor ? (anchor as { dataIndex: number }).dataIndex : 0
    return binTooltips[i] ?? ''
  }
}

function waveLabel(color: string, count: number, position: 'top' | 'bottom') {
  return { show: true, formatter: `×${count}`, color, fontSize: 10, fontWeight: 'bold' as const, position, distance: 6 }
}

function waveTip(side: string, w: TeamWave, xuidMeta: XuidMeta): string {
  const byPlayer = new Map<string, number>()
  for (const k of w.waveKills) byPlayer.set(k.xuid, (byPlayer.get(k.xuid) ?? 0) + 1)
  const contrib = [...byPlayer.entries()]
    .sort((a, b) => b[1] - a[1])
    .map(([xuid, n]) => `${displayPlayerName(xuidMeta.get(xuid)?.gamertag, xuid)} — ${n} kill${n > 1 ? 's' : ''}`)
    .join('<br/>')
  return `<b>Vague ${side} ×${w.count}</b> — ${formatMmSs(w.tStartMs / 1000)} → ${formatMmSs(w.tEndMs / 1000)}<br/>${contrib}`
}

interface WaveSeriesOpts { color: string; laneY: number; axisIndex: 0 | 1; sideLabel: string; xuidMeta: XuidMeta }

/** Séries de vague : un segment épais par vague, label ×N à l'extrémité. */
function buildWaveSeries(waves: TeamWave[], opts: WaveSeriesOpts): Record<string, unknown>[] {
  const { color, laneY, axisIndex, sideLabel, xuidMeta } = opts
  const position = axisIndex === 0 ? 'top' : 'bottom'
  const side = axisIndex === 0 ? 'ally' : 'enemy'
  return waves.map((w, wi) => ({
    type: 'line',
    name: `wave-${side}-${wi}`,
    data: [
      { value: [w.xStart, laneY], _tip: waveTip(sideLabel, w, xuidMeta) },
      { value: [w.xEnd, laneY], _tip: waveTip(sideLabel, w, xuidMeta), label: waveLabel(color, w.count, position) },
    ],
    lineStyle: { color, width: 4, opacity: 0.9 },
    itemStyle: { color, borderColor: 'rgba(255,255,255,0.6)', borderWidth: 1.5 },
    symbol: ['circle', 'circle'],
    symbolSize: 9,
    label: { show: false },
    legendHoverLink: false,
    xAxisIndex: axisIndex,
    yAxisIndex: axisIndex,
    tooltip: { trigger: 'item', formatter: (p: { data?: { _tip?: string } }) => p.data?._tip ?? '' },
    z: 5,
  }))
}

/** Deux séries bar signées (B1) : positifs team-ally, négatifs team-enemy, opacité DEC-4. */
function buildBarSeries(
  momentum: MomentumBin[],
  colorTeam: string,
  colorEnemy: string,
  t: MatchViewText,
  tc: EChartsThemeColors,
): Record<string, unknown>[] {
  const datum = (b: MomentumBin, color: string) => ({
    value: b.delta,
    itemStyle: { color: hexToRgba(color, b.trend === 'up' ? OP_STRONG : OP_WEAK) },
  })
  const allyData = momentum.map((b) => (b.delta > 0 ? datum(b, colorTeam) : { value: 0 }))
  const enemyData = momentum.map((b) => (b.delta < 0 ? datum(b, colorEnemy) : { value: 0 }))
  const base = { type: 'bar', stack: 'momentum', barCategoryGap: BAR_GAP, xAxisIndex: 0, yAxisIndex: 0 } as const
  return [
    {
      ...base,
      name: t.combatTeamLabel,
      data: allyData,
      itemStyle: { color: colorTeam, opacity: OP_STRONG },
      markLine: {
        silent: true,
        symbol: ['none', 'none'],
        lineStyle: { color: tc.splitLine, width: 1, type: 'dashed', opacity: 0.8 },
        data: [{ yAxis: 0 }],
        label: { show: false },
      },
    },
    { ...base, name: t.combatEnemyLabel, data: enemyData, itemStyle: { color: colorEnemy, opacity: OP_STRONG } },
  ]
}

interface KillFeedInput {
  bins: MatchTugOfWarBin[]
  allyKills: MomentumKill[]
  enemyKills: MomentumKill[]
  colorTeam: string
  colorEnemy: string
  xuidMeta: XuidMeta
  t: MatchViewText
  allyLaneY: number
}

/** Kill feed (DEC-2, inchangé) : lanes + scatter par kill + vagues, sur yMax. */
function buildKillFeedSeries(input: KillFeedInput): Record<string, unknown>[] {
  const { bins, allyKills, enemyKills, colorTeam, colorEnemy, xuidMeta, t, allyLaneY } = input
  const tip = (k: MomentumKill) => `${displayPlayerName(xuidMeta.get(k.xuid)?.gamertag, k.xuid)} — ${formatMmSs(k.tMs / 1000)}`
  const scatterFmt = (p: { data?: { _tip?: string } }) => p.data?._tip ?? ''
  const lane = (color: string, y: number, axisIndex: 0 | 1, name: string) => ({
    type: 'line', name, data: bins.map((_, i) => [i, y]),
    lineStyle: { color, width: 1.5, opacity: 0.45 }, showSymbol: false, silent: true,
    tooltip: { show: false }, legendHoverLink: false, xAxisIndex: axisIndex, yAxisIndex: axisIndex, z: 2,
  })
  const scatter = (kk: MomentumKill[], y: number, axisIndex: 0 | 1, name: string, color: string) => ({
    type: 'scatter', name, data: kk.map((k) => ({ value: [k.binIdx - 0.5 + k.fracInBin, y], _tip: tip(k) })),
    symbol: 'circle', symbolSize: 7, itemStyle: { color, opacity: 0.7 }, legendHoverLink: false,
    xAxisIndex: axisIndex, yAxisIndex: axisIndex, tooltip: { trigger: 'item', formatter: scatterFmt }, z: 3,
  })
  return [
    lane(colorTeam, allyLaneY, 0, 'Lane alliée'),
    scatter(allyKills, allyLaneY, 0, 'Mes kills', colorTeam),
    ...buildWaveSeries(detectTeamWaves(allyKills), { color: colorTeam, laneY: allyLaneY, axisIndex: 0, sideLabel: t.combatTeamLabel, xuidMeta }),
    lane(colorEnemy, 0, 1, 'Lane ennemie'),
    scatter(enemyKills, 0, 1, 'Kills ennemis', colorEnemy),
    ...buildWaveSeries(detectTeamWaves(enemyKills), { color: colorEnemy, laneY: 0, axisIndex: 1, sideLabel: t.combatEnemyLabel, xuidMeta }),
  ]
}

function buildXAxes(categories: string[], tc: EChartsThemeColors, n: number): Record<string, unknown>[] {
  return [
    {
      gridIndex: 0, type: 'category', data: categories,
      axisLine: { show: true, lineStyle: { color: tc.axisLine } },
      axisTick: { show: true, alignWithLabel: true, length: 4, lineStyle: { color: tc.axisLine } },
      axisLabel: { rotate: 0, fontSize: 9, color: tc.axisLabel, interval: Math.max(0, Math.floor(n / 12)), margin: 6 },
      splitLine: { show: false },
    },
    {
      gridIndex: 1, type: 'category', data: categories, show: false,
      axisLine: { show: false }, axisTick: { show: false }, axisLabel: { show: false }, splitLine: { show: false },
    },
  ]
}

export function MatchTugOfWarChart({ bins, events, scoreboard, meXUID, t }: Props) {
  // La carte se recompose depuis les events `kill` : sans bin ET sans kill, la
  // dominance n'a pas de sens → EmptyState (matchs servis live-only, tout titre).
  const hasKillEvents =
    !!events && events.some((e) => (e.event_type ?? '').toLowerCase() === 'kill')
  const series: ChartSeries<MatchTugOfWarBin>[] =
    bins && bins.length > 0 && hasKillEvents
      ? [{ key: 'match_view.combat.tug_of_war', datapoints: bins }]
      : []

  const buildOption = useCallback(
    (s: ChartSeries<MatchTugOfWarBin>[]): EChartsCoreOption => {
      if (s.length === 0 || !bins || bins.length === 0) return { backgroundColor: CHART_BG }

      const xuidMeta = resolveXuidMeta(scoreboard, meXUID)
      const colorTeam = resolveToken('team-ally')
      const colorEnemy = resolveToken('team-enemy')
      const tc = getEChartsThemeColors()
      const categories = bins.map((b) => formatMmSs((b.bin_start + b.bin_end) / 2))

      const { momentum, kills } = computeMomentumBins(bins, events, xuidMeta)
      const binTooltips = buildBinTooltips(momentum, categories, t)

      // Échelle Y symétrique dynamique (DEC-6) : plus de mock 0–100.
      const yMax = Math.max(1, ...momentum.map((b) => Math.abs(b.delta)))
      const allyLaneY = yMax * LANE_FACTOR
      const yTopMax = yMax * TOP_FACTOR
      const yTopMin = -yMax * BOTTOM_FACTOR

      const allyKills = kills.filter((k) => k.ally).sort((a, b) => a.tMs - b.tMs)
      const enemyKills = kills.filter((k) => !k.ally).sort((a, b) => a.tMs - b.tMs)

      return {
        backgroundColor: CHART_BG,
        grid: [
          { left: 14, right: 14, top: 8, height: '68%', containLabel: false },
          { left: 14, right: 14, top: '78%', height: '10%', containLabel: false },
        ],
        tooltip: { ...getTooltipBase(tc), trigger: 'axis', axisPointer: { type: 'shadow' }, formatter: binTooltipFormatter(binTooltips) },
        legend: { ...getLegendBase(tc), bottom: 4, data: [t.combatTeamLabel, t.combatEnemyLabel] },
        xAxis: buildXAxes(categories, tc, bins.length),
        yAxis: [
          { gridIndex: 0, type: 'value', min: yTopMin, max: yTopMax, show: false, splitLine: { show: false } },
          { gridIndex: 1, type: 'value', min: -2, max: 2, show: false, splitLine: { show: false } },
        ],
        series: [
          ...buildBarSeries(momentum, colorTeam, colorEnemy, t, tc),
          ...buildKillFeedSeries({ bins, allyKills, enemyKills, colorTeam, colorEnemy, xuidMeta, t, allyLaneY }),
        ],
      }
    },
    [bins, events, scoreboard, meXUID, t],
  )

  return (
    <ChartCard
      title={t.combatTugOfWarTitle}
      series={series}
      height={360}
      buildOption={buildOption}
      emptyMessage={t.combatNoData}
    />
  )
}
