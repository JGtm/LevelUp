/**
 * TimeseriesSquadAdapted — wrappers solo des charts initialement squad.
 *
 * Adaptent en mode 1 joueur (joueur actif uniquement) :
 *   - Performance par session (teammates.04 → solo)
 *   - Rendement & Résistance par match (teammates squad efficiency → solo)
 *
 * Source : data.match_rows uniquement (pas de backend dédié — agrégation /
 * calcul côté front).
 */
import { useMemo, type ReactNode } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
} from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'
import { ChartFromOption } from './ChartFromOption'
import type {
  TimeseriesMatchRow,
  IntensityMatchRow,
  SquadIntensityMatchRow,
  SoloSessionPerfPoint,
} from '@/lib/api/types'
import { buildSquadIntensityHeatmapOption } from '@/features/squad/charts/squadIntensityHeatmapChart'
import {
  damageAxisBounds,
  damagePerDeath,
  damagePerKill,
  defensiveDamageGradient,
  offensiveDamageGradient,
} from '@/lib/charts/oneLifeDamageGradient'
import {
  useEffectiveHpToKill,
  substituteHpToken,
  useProvidesDamageTaken,
} from '@/lib/damage/effectiveHp'
import { buildMatchCategories } from './matchLabels'

interface RenderProps {
  height: number
  option: EChartsCoreOption | null
  title?: ReactNode
  emptyMessage?: string
}
function ChartRender({ option, height, title, emptyMessage }: RenderProps) {
  return (
    <ChartFromOption title={title} option={option} height={height} emptyMessage={emptyMessage} />
  )
}

// ─── Performance par session ─────────────────────────────────────────────────
//
// Source : data.solo_session_perf — agrégat backend sur la population solo
// complète (ignore filtres period/sessions/cascade) pour permettre la
// comparaison cross-session. Bars perf (Y1) + ligne winrate (Y2 0..100%) +
// ligne MMR (Y2 séparé droite).

export interface TimeseriesSessionPerformanceProps {
  points: SoloSessionPerfPoint[]
  granularity: 'session' | 'week' | 'month'
  height?: number
  title?: ReactNode
  emptyMessage?: string
  perfLabel: string
  winRateLabel: string
  mmrLabel: string
  /** `false` (titre sans `team_mmr`, ex. Halo 5) → la série MMR, son entrée de
   *  légende et le suffixe « / MMR équipe » de l'axe Y2 sont retirés (pas de
   *  ligne fantôme). Défaut `true` (Halo Infinite inchangé). */
  showMmr?: boolean
}

export function TimeseriesSessionPerformance({
  points,
  granularity,
  height = 360,
  title,
  emptyMessage,
  perfLabel,
  winRateLabel,
  mmrLabel,
  showMmr = true,
}: TimeseriesSessionPerformanceProps) {
  const themeVersion = useThemeVersion()

  const option = useMemo<EChartsCoreOption | null>(() => {
    if (points.length === 0) return null
    const tc = getEChartsThemeColors()
    const colPerf = resolveToken('chart-series-2')
    const colWR = resolveToken('outcome-win')
    const colMMR = resolveToken('chart-series-7')

    // Format X selon la granularité : session=DD/MM, week=Sxx, month=Mois yy.
    const formatX = (p: SoloSessionPerfPoint): string => {
      const d = new Date(p.started_at_utc)
      switch (granularity) {
        case 'week':
          return `${p.session_label}\n(${p.match_count})`
        case 'month':
          return `${d.toLocaleDateString('fr-FR', { month: 'short', year: '2-digit' })}\n(${p.match_count})`
        default:
          return `${d.toLocaleDateString('fr-FR', { day: '2-digit', month: '2-digit' })}\n(${p.match_count})`
      }
    }
    const categories = points.map(formatX)
    const perf = points.map((p) => (p.perf_avg ?? null))
    const wr = points.map((p) => Math.round(p.win_rate * 1000) / 10)
    const mmr = points.map((p) =>
      p.team_mmr_avg != null ? Math.round(p.team_mmr_avg) : null,
    )

    return {
      backgroundColor: CHART_BG,
      grid: { top: 24, right: 60, bottom: 64, left: 56, containLabel: true },
      tooltip: { ...getTooltipBase(tc), trigger: 'axis' },
      legend: { ...getLegendBase(tc), bottom: 0 },
      xAxis: {
        ...getAxisBase(tc),
        type: 'category',
        data: categories,
        axisLabel: { ...getAxisBase(tc).axisLabel, interval: 0, fontSize: 9 },
      },
      yAxis: [
        {
          ...getAxisBase(tc),
          type: 'value',
          name: perfLabel,
          min: 0,
          max: 100,
          nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
        },
        {
          ...getAxisBase(tc),
          type: 'value',
          // Sans MMR (Halo 5) : l'axe Y2 ne porte plus que le taux de victoire.
          name: showMmr ? `${winRateLabel} / ${mmrLabel}` : winRateLabel,
          scale: true,
          nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
          splitLine: { show: false },
        },
      ],
      series: [
        {
          type: 'bar',
          name: perfLabel,
          yAxisIndex: 0,
          data: perf,
          itemStyle: { color: colPerf, opacity: 0.8 },
          barMaxWidth: 24,
        },
        {
          type: 'line',
          name: winRateLabel,
          yAxisIndex: 1,
          data: wr,
          showSymbol: true,
          smooth: false,
          connectNulls: true,
          lineStyle: { color: colWR, width: 2 },
          itemStyle: { color: colWR },
          z: 5,
        },
        // Ligne MMR équipe + son entrée de légende retirées quand le titre ne
        // fournit pas de MMR par match (data + axe restant déjà nettoyés).
        ...(showMmr
          ? [
              {
                type: 'line' as const,
                name: mmrLabel,
                yAxisIndex: 1,
                data: mmr,
                showSymbol: false,
                smooth: false,
                connectNulls: true,
                lineStyle: { color: colMMR, width: 1.5, type: 'dashed' as const },
                z: 4,
              },
            ]
          : []),
      ],
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [points, granularity, perfLabel, winRateLabel, mmrLabel, showMmr, themeVersion])
  return <ChartRender option={option} height={height} title={title} emptyMessage={emptyMessage} />
}

// ─── Rendement & Résistance par match ────────────────────────────────────────
//
// Dégâts BRUTS côté front sur match_rows, repère 225 = 1 vie de Spartan :
//   dégâts/frag = damage_dealt / kills
//   dégâts/mort = damage_taken / deaths
//
// 2 lignes (frag plein, mort dashed) + ligne repère à 225, colorées par dégradé
// (frag : proche de 225 = bon ; mort : au-dessus de 225 = bon). Cf. helper
// oneLifeDamageGradient.

export interface TimeseriesEfficiencyProps {
  rows: TimeseriesMatchRow[]
  height?: number
  title?: ReactNode
  emptyMessage?: string
  rendementLabel: string
  resistanceLabel: string
  refLabel: string
}

// ─── Intensity heatmap (frags par phase de match) ────────────────────────────
//
// Réutilise le builder ECharts du squad (squadIntensityHeatmapChart).
// Format `IntensityMatchRow` côté backend = `SquadIntensityMatchRow` côté UI.

export interface TimeseriesIntensityHeatmapProps {
  rows: IntensityMatchRow[]
  height?: number
  title?: ReactNode
  emptyMessage?: string
  zLabel: string
}

export function TimeseriesIntensityHeatmap({
  rows,
  height = 360,
  title,
  emptyMessage,
  zLabel,
}: TimeseriesIntensityHeatmapProps) {
  const themeVersion = useThemeVersion()

  const option = useMemo<EChartsCoreOption | null>(() => {
    if (rows.length === 0) return null
    // Les matchs arrivent en DESC (récent → ancien) ; on inverse pour lire la
    // heatmap du plus ancien (haut) au plus récent (bas), comme les autres graphes.
    const ordered = [...rows].reverse()
    return buildSquadIntensityHeatmapOption(ordered as SquadIntensityMatchRow[], { zLabel })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rows, zLabel, themeVersion])
  return <ChartRender option={option} height={height} title={title} emptyMessage={emptyMessage} />
}

export function TimeseriesEfficiency({
  rows,
  height = 320,
  title,
  emptyMessage,
  rendementLabel,
  resistanceLabel,
  refLabel,
}: TimeseriesEfficiencyProps) {
  const themeVersion = useThemeVersion()
  const hp = useEffectiveHpToKill() // barème PV-pour-tuer du titre courant (225 Infinite, 115 h5)
  // false (Halo 5 : API sans damage_taken) → « Dégâts / mort » non calculable :
  // on retire entièrement la courbe + son entrée de légende (pas de ligne vide).
  const providesDamageTaken = useProvidesDamageTaken()

  const option = useMemo<EChartsCoreOption | null>(() => {
    if (rows.length === 0) return null
    const tc = getEChartsThemeColors()
    const colRef = resolveToken('divergent-neutral')

    const categories = buildMatchCategories(rows)
    const dmgKill = rows.map((r) => damagePerKill(r.damage_dealt, r.kills))
    const dmgDeath = providesDamageTaken
      ? rows.map((r) => damagePerDeath(r.damage_taken, r.deaths))
      : []
    const bounds = damageAxisBounds([...dmgKill, ...dmgDeath], hp)

    return {
      backgroundColor: CHART_BG,
      grid: { top: 16, right: 16, bottom: 64, left: 52, containLabel: true },
      tooltip: {
        ...getTooltipBase(tc),
        trigger: 'axis',
        formatter: (params: Array<{ seriesName: string; value: number | null; marker: string }>) =>
          params
            .filter((p) => p.value != null)
            .map((p) => `${p.marker}${p.seriesName}: <b>${Math.round(p.value as number)}</b>`)
            .join('<br/>'),
      },
      // Pastille de légende élargie (itemWidth 30 vs 12) : à 12px le pointillé
      // de « Dégâts / mort » est invisible ; à 30px on distingue trait plein
      // (Dégâts/frag) vs pointillé (Dégâts/mort).
      legend: { ...getLegendBase(tc), bottom: 0, itemWidth: 30 },
      xAxis: {
        ...getAxisBase(tc),
        type: 'category',
        data: categories,
        axisLabel: { ...getAxisBase(tc).axisLabel, interval: 0, fontSize: 9 },
      },
      yAxis: {
        ...getAxisBase(tc),
        type: 'value',
        min: bounds.min,
        max: bounds.max,
        axisLabel: { ...getAxisBase(tc).axisLabel, formatter: (v: number) => `${Math.round(v)}` },
      },
      series: [
        {
          type: 'line',
          name: rendementLabel,
          data: dmgKill,
          showSymbol: false,
          smooth: false,
          connectNulls: true,
          lineStyle: { color: offensiveDamageGradient(dmgKill, hp), width: 2 },
          markLine: {
            silent: true,
            symbol: 'none',
            lineStyle: { color: colRef, width: 1, type: 'dashed' },
            label: {
              formatter: substituteHpToken(refLabel, hp),
              color: colRef,
              fontSize: 10,
              position: 'insideEndTop',
            },
            data: [{ yAxis: hp }],
          },
        },
        // Série « Résistance » (Dégâts / mort) + son entrée de légende retirées
        // quand le titre ne fournit pas damage_taken (Halo 5).
        ...(providesDamageTaken
          ? [
              {
                type: 'line' as const,
                name: resistanceLabel,
                data: dmgDeath,
                showSymbol: false,
                smooth: false,
                connectNulls: true,
                lineStyle: {
                  color: defensiveDamageGradient(dmgDeath, hp),
                  width: 2,
                  type: 'dashed' as const,
                },
              },
            ]
          : []),
      ],
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rows, rendementLabel, resistanceLabel, refLabel, themeVersion, hp, providesDamageTaken])
  return <ChartRender option={option} height={height} title={title} emptyMessage={emptyMessage} />
}
