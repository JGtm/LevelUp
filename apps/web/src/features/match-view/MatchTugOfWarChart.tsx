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
import { useCallback, useMemo } from 'react'
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
  MatchObjectiveEvent,
  MatchScoreboardRow,
  MatchTugOfWarBin,
} from '@/lib/api/types'
import { computeMomentumBins, type MomentumBin, type MomentumKill } from './_momentum'
import { extractCtfCaptures, type CtfCapture } from './_objectiveCaptures'
import { MatchKillFeed } from './MatchKillFeed'
import type { MatchViewText } from './i18n'
// La cascade « allie = meme camp que moi » etait ecrite ici ET dans MatchKDCumulChart, a
// l'identique : elle vit desormais dans xuidMeta.ts, sous garde-rail.
import { resolveXuidMeta, type XuidMeta } from './xuidMeta'

interface Props {
  bins: MatchTugOfWarBin[] | null | undefined
  events: MatchHighlightEvent[] | null | undefined
  scoreboard: MatchScoreboardRow[] | null | undefined
  meXUID: string | null
  /** Événements d'objectif (CTF captures…) — overlay sur la grille haute. */
  objectiveEvents?: MatchObjectiveEvent[] | null
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

/** Résumé de tranche (delta signé, X/Y kills, cumuls) — servi par le tooltip item des barres (remplace DEC-3). */
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

/**
 * Formatter item des barres : rend le résumé de la tranche survolée. En trigger
 * global `'item'` le formatter reçoit un seul param (la barre) ; on garde le
 * fallback tableau par robustesse. `dataIndex` = index de la tranche.
 */
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
    // Trigger global 'item' : le formatter par-série est honoré (ECharts n'a pas
    // de `trigger` par-série — ne pas le remettre, cf. W2/MatchTugOfWarChart).
    tooltip: { formatter: (p: { data?: { _tip?: string } }) => p.data?._tip ?? '' },
    z: 5,
  }))
}

interface BarSeriesOpts {
  momentum: MomentumBin[]
  colorTeam: string
  colorEnemy: string
  t: MatchViewText
  tc: EChartsThemeColors
  /** Résumés de tranche indexés par bin (tooltip item des barres). */
  binTooltips: string[]
}

/** Deux séries bar signées (B1) : positifs team-ally, négatifs team-enemy, opacité DEC-4. */
function buildBarSeries(opts: BarSeriesOpts): Record<string, unknown>[] {
  const { momentum, colorTeam, colorEnemy, t, tc, binTooltips } = opts
  const datum = (b: MomentumBin, color: string) => ({
    value: b.delta,
    itemStyle: { color: hexToRgba(color, b.trend === 'up' ? OP_STRONG : OP_WEAK) },
  })
  const allyData = momentum.map((b) => (b.delta > 0 ? datum(b, colorTeam) : { value: 0 }))
  const enemyData = momentum.map((b) => (b.delta < 0 ? datum(b, colorEnemy) : { value: 0 }))
  // Résumé de tranche servi par le tooltip item de CHAQUE barre : une seule des
  // deux est non nulle par tranche → la barre affichée est toujours survolable.
  const binTip = { formatter: binTooltipFormatter(binTooltips) }
  const base = { type: 'bar', stack: 'momentum', barCategoryGap: BAR_GAP, xAxisIndex: 0, yAxisIndex: 0 } as const
  return [
    {
      ...base,
      name: t.combatTeamLabel,
      data: allyData,
      itemStyle: { color: colorTeam, opacity: OP_STRONG },
      tooltip: binTip,
      markLine: {
        silent: true,
        symbol: ['none', 'none'],
        lineStyle: { color: tc.splitLine, width: 1, type: 'dashed', opacity: 0.8 },
        data: [{ yAxis: 0 }],
        label: { show: false },
      },
    },
    { ...base, name: t.combatEnemyLabel, data: enemyData, itemStyle: { color: colorEnemy, opacity: OP_STRONG }, tooltip: binTip },
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

/**
 * Lanes de temps + vagues collectives (DEC-2). Les KILLS eux-mêmes ne sont plus ici :
 * ils sont rendus en DOM par `MatchKillFeed`, sous le graphe, pour porter l'icône de
 * l'arme TEINTÉE à la couleur d'équipe — un symbole image ECharts se dessine tel quel et
 * ne peut pas être teinté. Les lanes restent dans le graphe : ce sont elles qui portent
 * les segments de vague, dont la position en Y n'a de sens que sur l'échelle des barres.
 */
function buildKillFeedSeries(input: KillFeedInput): Record<string, unknown>[] {
  const { bins, allyKills, enemyKills, colorTeam, colorEnemy, xuidMeta, t, allyLaneY } = input
  const lane = (color: string, y: number, axisIndex: 0 | 1, name: string) => ({
    type: 'line', name, data: bins.map((_, i) => [i, y]),
    lineStyle: { color, width: 1.5, opacity: 0.45 }, showSymbol: false, silent: true,
    tooltip: { show: false }, legendHoverLink: false, xAxisIndex: axisIndex, yAxisIndex: axisIndex, z: 2,
  })
  return [
    lane(colorTeam, allyLaneY, 0, 'Lane alliée'),
    ...buildWaveSeries(detectTeamWaves(allyKills), { color: colorTeam, laneY: allyLaneY, axisIndex: 0, sideLabel: t.combatTeamLabel, xuidMeta }),
    lane(colorEnemy, 0, 1, 'Lane ennemie'),
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

/**
 * buildCaptureSeries — overlay des captures CTF sur la grille haute.
 *
 * L'axe X est 'category' (bins) : on convertit chaque instant de capture en index
 * fractionnaire de bin, avec la même formule que les scatters et les vagues
 * (binIdx - 0.5 + fracInBin). Une capture hors fenêtre des bins est ignorée —
 * dégradation propre, pas d'exception. Mode non-CTF → `captures` vide → série absente.
 */
function buildCaptureSeries(input: {
  captures: CtfCapture[]
  bins: MatchTugOfWarBin[]
  colorTeam: string
  colorEnemy: string
  yTopMax: number
  t: MatchViewText
}): Record<string, unknown>[] {
  const { captures, bins, colorTeam, colorEnemy, yTopMax, t } = input
  const captureML: Record<string, unknown>[] = []
  const captureMP: Record<string, unknown>[] = []
  for (const c of captures) {
    const tSec = c.tMs / 1000
    const idx = bins.findIndex((b) => tSec >= b.bin_start && tSec < b.bin_end)
    if (idx < 0) continue
    const bin = bins[idx]
    const span = Math.max(1, bin.bin_end - bin.bin_start)
    const frac = Math.min(0.999, Math.max(0, (tSec - bin.bin_start) / span))
    const xPos = idx - 0.5 + frac
    const color = c.ally ? colorTeam : colorEnemy
    captureML.push({
      xAxis: xPos,
      lineStyle: { color, width: 1, type: 'solid', opacity: 0.7 },
      label: { show: false },
    })
    // markPoint en tête de grille haute : label « Capture » + tooltip scorer.
    captureMP.push({
      coord: [xPos, yTopMax - 4],
      symbol: 'pin',
      symbolSize: 16,
      itemStyle: { color, opacity: 0.9 },
      label: {
        show: true,
        formatter: t.combatCtfCaptureLabel,
        color,
        fontSize: 9,
        fontWeight: 'bold',
        position: 'top',
        distance: 2,
      },
      _tip: t.combatCtfCaptureTooltip(c.scorer, formatMmSs(tSec)),
    })
  }
  if (captureML.length === 0) return []
  // Hors legend (name absent de legend.data) : c'est un repère, pas une série lisible.
  return [{
    type: 'line',
    name: t.combatCtfCaptureLabel,
    data: [] as Array<[number, number]>,
    showSymbol: false,
    legendHoverLink: false,
    xAxisIndex: 0,
    yAxisIndex: 0,
    markLine: { silent: true, symbol: ['none', 'none'], data: captureML },
    markPoint: {
      data: captureMP,
      tooltip: { formatter: (p: { data?: { _tip?: string } }) => p.data?._tip ?? '' },
    },
    z: 6,
  }]
}

export function MatchTugOfWarChart({ bins, events, scoreboard, meXUID, objectiveEvents, t }: Props) {
  // La carte se recompose depuis les events `kill` : sans bin ET sans kill, la
  // dominance n'a pas de sens → EmptyState (matchs servis live-only, tout titre).
  const hasKillEvents =
    !!events && events.some((e) => (e.event_type ?? '').toLowerCase() === 'kill')
  const series: ChartSeries<MatchTugOfWarBin>[] =
    bins && bins.length > 0 && hasKillEvents
      ? [{ key: 'match_view.combat.tug_of_war', datapoints: bins }]
      : []

  // Le binning est calculé UNE fois ici et partagé : le graphe (barres, vagues) et le
  // kill feed DOM doivent poser leurs kills aux mêmes abscisses. Deux calculs, ce serait
  // deux vérités et un décalage visible entre l'histogramme et le feed.
  const xuidMeta = useMemo(() => resolveXuidMeta(scoreboard, meXUID), [scoreboard, meXUID])
  const feedKills = useMemo(
    () => (bins && bins.length > 0 ? computeMomentumBins(bins, events, xuidMeta).kills : []),
    [bins, events, xuidMeta],
  )

  const buildOption = useCallback(
    (s: ChartSeries<MatchTugOfWarBin>[]): EChartsCoreOption => {
      if (s.length === 0 || !bins || bins.length === 0) return { backgroundColor: CHART_BG }

      const colorTeam = resolveToken('team-ally')
      const colorEnemy = resolveToken('team-enemy')
      const tc = getEChartsThemeColors()
      const categories = bins.map((b) => formatMmSs((b.bin_start + b.bin_end) / 2))

      const { momentum } = computeMomentumBins(bins, events, xuidMeta)
      const binTooltips = buildBinTooltips(momentum, categories, t)

      // Échelle Y symétrique dynamique (DEC-6) : plus de mock 0–100.
      const yMax = Math.max(1, ...momentum.map((b) => Math.abs(b.delta)))
      const allyLaneY = yMax * LANE_FACTOR
      const yTopMax = yMax * TOP_FACTOR
      const yTopMin = -yMax * BOTTOM_FACTOR

      const allyKills = feedKills.filter((k) => k.ally).sort((a, b) => a.tMs - b.tMs)
      const enemyKills = feedKills.filter((k) => !k.ally).sort((a, b) => a.tMs - b.tMs)

      return {
        backgroundColor: CHART_BG,
        grid: [
          { left: 14, right: 14, top: 8, height: '68%', containLabel: false },
          { left: 14, right: 14, top: '78%', height: '10%', containLabel: false },
        ],
        // Trigger 'item' (et non 'axis') : ECharts n'honore PAS un `trigger`
        // par-série, donc les tips per-kill / per-vague ne vivent qu'en trigger
        // global 'item'. Le résumé de tranche est servi par le tooltip item des
        // barres (buildBarSeries). Le passage à 'axis' (HEAD) avait tué les tips
        // item des séries kill-feed/vagues (W2, revue 2026-07-17).
        tooltip: { ...getTooltipBase(tc), trigger: 'item' },
        legend: { ...getLegendBase(tc), bottom: 4, data: [t.combatTeamLabel, t.combatEnemyLabel] },
        xAxis: buildXAxes(categories, tc, bins.length),
        yAxis: [
          { gridIndex: 0, type: 'value', min: yTopMin, max: yTopMax, show: false, splitLine: { show: false } },
          { gridIndex: 1, type: 'value', min: -2, max: 2, show: false, splitLine: { show: false } },
        ],
        series: [
          ...buildBarSeries({ momentum, colorTeam, colorEnemy, t, tc, binTooltips }),
          ...buildKillFeedSeries({ bins, allyKills, enemyKills, colorTeam, colorEnemy, xuidMeta, t, allyLaneY }),
          ...buildCaptureSeries({
            captures: extractCtfCaptures(objectiveEvents, scoreboard, meXUID),
            bins, colorTeam, colorEnemy, yTopMax, t,
          }),
        ],
      }
    },
    [bins, events, xuidMeta, feedKills, scoreboard, meXUID, objectiveEvents, t],
  )

  return (
    <ChartCard
      title={t.combatTugOfWarTitle}
      series={series}
      height={360}
      buildOption={buildOption}
      emptyMessage={t.combatNoData}
    >
      {series.length > 0 && bins && (
        <MatchKillFeed bins={bins} kills={feedKills} scoreboard={scoreboard} xuidMeta={xuidMeta} t={t} />
      )}
    </ChartCard>
  )
}
