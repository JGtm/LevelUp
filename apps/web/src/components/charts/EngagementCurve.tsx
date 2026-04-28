/**
 * EngagementCurve — Mock 10 (intra-match) et Mock 11 (session/periode).
 *
 * Reference visuelle : .ai/mockups/engagement/engagement_visualizations.html
 * Plan : .ai/PLAN_ENGAGEMENT_IMPLEMENTATION.md §6.6.2 (Match View) + §6.6.3 (Timeseries)
 *
 * Trois courbes superposees :
 *   - Equipe alliee (gris fonce, fine)        — pace per_player
 *   - Attendu (gris medium, pointille)         — coef × team
 *   - Joueur (couleur saturee, epaisse 4px)   — pace observed
 *
 * Exigences §8.6 du doc reflexion (obligatoires) :
 *   - Auto-zoom Y dynamique (range affiche dans label)
 *   - Hierarchie visuelle marquee (joueur 4px sature, attendu 1.5px pointille,
 *     equipe 1.5px effacee)
 *
 * Rejetes explicitement : gap shading, pastille FormScore in-chart.
 *
 * Reutilise pour Match View intra-match (1 point = 10s, smooth=true) et pour
 * Session/Periode (1 point = 1 match, smooth=false avec markers visibles).
 */
import { useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken } from '@/lib/accessibility'

import { ChartCard, type ChartSeries } from './ChartCard'
import { CHART_BG, axisBase, legendBase, tooltipBase } from './_utils'

export interface EngagementPoint {
  /** Timestamp en ms (Match View) ou index match (Session) */
  x: number
  /** Pace equipe alliee per_player */
  paceTeam: number
  /** Pace attendu = coef × team */
  paceAttendu: number
  /** Pace joueur observe */
  paceJoueur: number
  /** Optionnel : pace lobby per_player (Mock 15 v2 / Match View context) */
  paceLobby?: number
  /** Mort post-creux > 30s (annotation triangle rouge) */
  isPassiveDeath?: boolean
  /** Periode post-mort active (bande rouge translucide) */
  postDeath?: boolean
}

export interface EngagementCurveProps {
  /** Titre affiche dans la ChartCard. */
  title?: string
  /** Subtitle / sous-titre optionnel. */
  subtitle?: string
  /** Donnees ordonnees chronologiquement. */
  points: EngagementPoint[]
  /** Format des labels X axis. */
  xFormatter?: (x: number) => string
  /** Mode de granularite : 'intra' (smooth) ou 'session' (lineaire avec markers). */
  granularity?: 'intra' | 'session'
  /** Etat externe (loading/error/empty). Cf ChartCard. */
  state?: 'loading' | 'error' | 'empty' | 'ready'
  /** Hauteur en px. Default 280 (chart-tall). */
  height?: number
}

/**
 * EngagementCurve — composant generique reutilisable pour Match View
 * (intra-match) et Session (1 point = 1 match).
 */
export function EngagementCurve(props: EngagementCurveProps) {
  const {
    title,
    subtitle,
    points,
    xFormatter,
    granularity = 'intra',
    state = 'ready',
    height = 280,
  } = props

  const option = useMemo<EChartsCoreOption>(
    () => buildEngagementOption(points, granularity, xFormatter),
    [points, granularity, xFormatter],
  )

  // Series virtuelle pour ChartCard (qui exige une series typee).
  const series: ChartSeries<EngagementPoint>[] = [
    { id: 'engagement', name: title ?? 'Engagement', points },
  ]

  return (
    <ChartCard
      title={title}
      subtitle={subtitle}
      series={series}
      state={state}
      height={height}
      option={option}
    />
  )
}

// ---------------------------------------------------------------------------
// Builders ECharts
// ---------------------------------------------------------------------------

function buildEngagementOption(
  points: EngagementPoint[],
  granularity: 'intra' | 'session',
  xFormatter?: (x: number) => string,
): EChartsCoreOption {
  if (points.length === 0) {
    return {} as EChartsCoreOption
  }

  // Auto-zoom Y : couvre toutes les traces avec padding ±1.
  const allY = points.flatMap((p) => [p.paceTeam, p.paceAttendu, p.paceJoueur])
  const yMin = Math.floor(Math.min(...allY) - 1)
  const yMax = Math.ceil(Math.max(...allY) + 1)

  const xData = points.map((p) => (xFormatter ? xFormatter(p.x) : String(p.x)))
  const teamColor = resolveToken('chart-series-1') // gris fonce / atone
  const expectedColor = resolveToken('chart-series-2') // gris medium pointille
  const playerColor = resolveToken('accent-info') // saturated blue

  // Smooth=true en intra (echantillonnage 10s), false en session (matchs discrets).
  const smooth = granularity === 'intra'

  return {
    backgroundColor: CHART_BG,
    grid: { left: 50, right: 24, top: 18, bottom: 38 },
    tooltip: { ...tooltipBase, trigger: 'axis' },
    legend: { ...legendBase, top: 0, bottom: 'auto' },
    xAxis: {
      ...axisBase,
      type: 'category',
      data: xData,
      axisLabel: { ...axisBase.axisLabel, interval: computeXInterval(xData.length) },
    },
    yAxis: {
      ...axisBase,
      type: 'value',
      min: yMin,
      max: yMax,
      name: `events / min (auto-zoom ${yMin}..${yMax})`,
      nameLocation: 'middle',
      nameGap: 36,
      nameTextStyle: { color: 'rgba(255,255,255,0.45)', fontSize: 10 },
    },
    series: [
      // Equipe alliee — fine effacee
      {
        name: 'Equipe alliee',
        type: 'line',
        data: points.map((p) => p.paceTeam),
        smooth,
        symbol: granularity === 'session' ? 'circle' : 'none',
        symbolSize: 5,
        lineStyle: { color: teamColor, width: 1.5 },
        itemStyle: { color: teamColor },
        z: 1,
      },
      // Attendu — pointille fin
      {
        name: 'Attendu',
        type: 'line',
        data: points.map((p) => p.paceAttendu),
        smooth,
        symbol: granularity === 'session' ? 'circle' : 'none',
        symbolSize: 5,
        lineStyle: { color: expectedColor, width: 1.5, type: 'dashed' },
        itemStyle: { color: expectedColor },
        z: 2,
      },
      // Joueur — epais sature (hierarchie visuelle marquee §8.6)
      {
        name: 'Joueur',
        type: 'line',
        data: points.map((p) => p.paceJoueur),
        smooth,
        symbol: granularity === 'session' ? 'circle' : 'none',
        symbolSize: granularity === 'session' ? 8 : 0,
        lineStyle: { color: playerColor, width: 4 },
        itemStyle: { color: playerColor },
        z: 5,
      },
    ],
  }
}

function computeXInterval(n: number): number {
  if (n <= 12) return 0
  if (n <= 30) return 1
  return Math.ceil(n / 15) - 1
}
