/**
 * TimeseriesCombatYield — 2 courbes (Offensive Conversion + Defensive
 * Resistance) avec lignes de référence p80, rendu via ECharts.
 *
 * Phase 3 P3.F : remplace l'ancien wrapper Plotly `CombatYieldTimeseries`.
 * Construit côté client depuis les données MatchHistoryRow.
 *
 * Lignes de référence (frontière élite) OC=0.90 / DR=1.65 — miroir des constantes
 * Go `combat_yield.go`. Pointillé fin coloré comme la série de référence.
 */
import { useCallback, useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { resolveToken } from '@/lib/accessibility'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
} from '@/components/charts/_utils'
import type { MatchHistoryRow } from '@/lib/api/types'
import { useOffensiveConversionP80, useProvidesDamageTaken } from '@/lib/damage/effectiveHp'

/** Repère barre (frontière élite mondiale) — miroir des constantes Go combat_yield.go. */
const OC_P80 = 0.90
const DR_P80 = 1.65
/** DR affiché normalisé depuis 1.0 : (DR_P80 - 1.0) = 0.65 */
const DR_DISPLAY_P80 = DR_P80 - 1.0

interface CombatYieldPoint {
  x: string
  y: number | null
}

export interface TimeseriesCombatYieldProps {
  rows: MatchHistoryRow[]
  height?: number
  /** Libellés résolus i18n. */
  labels: {
    ocSeries: string
    drSeries: string
    ocReference: string
    drReference: string
    emptyTitle: string
    emptyDescription: string
  }
}

export function TimeseriesCombatYield({
  rows,
  height = 320,
  labels,
}: TimeseriesCombatYieldProps) {
  const filtered = useMemo(
    () =>
      rows.filter(
        (r) => r.offensive_conversion != null || r.defensive_resistance != null,
      ),
    [rows],
  )

  // false (Halo 5 : API sans damage_taken) → la Résistance n'est pas calculable :
  // on retire entièrement la courbe + la ligne de référence DR (pas de tracé plat
  // à −100% trompeur). Défaut true (Infinite) → graphe à deux courbes inchangé.
  const providesDamageTaken = useProvidesDamageTaken()

  const series = useMemo<ChartSeries<CombatYieldPoint>[]>(() => {
    if (filtered.length === 0) return []
    const ocSeries: ChartSeries<CombatYieldPoint> = {
      key: 'combat.oc',
      meta: { gamertag: labels.ocSeries },
      datapoints: filtered.map((r) => ({
        x: r.start_time,
        y: r.offensive_conversion ?? null,
      })),
    }
    if (!providesDamageTaken) return [ocSeries]
    return [
      ocSeries,
      {
        key: 'combat.dr',
        meta: { gamertag: labels.drSeries },
        datapoints: filtered.map((r) => ({
          x: r.start_time,
          // Normalise depuis 1.0 pour aligner l'axe Y avec OC (0..N%) plutôt que 100..N%
          y: r.defensive_resistance != null && r.defensive_resistance >= 0
            ? r.defensive_resistance - 1
            : null,
        })),
      },
    ]
  }, [filtered, labels.ocSeries, labels.drSeries, providesDamageTaken])

  const ocP80 = useOffensiveConversionP80() // 0.90 Infinite / 1.264 h5 (titre courant)
  const buildOption = useCallback(
    (s: ChartSeries<CombatYieldPoint>[]) => buildCombatYieldOption(s, labels, ocP80),
    [labels, ocP80],
  )

  if (filtered.length === 0) {
    return (
      <EmptyStateNotice title={labels.emptyTitle} description={labels.emptyDescription} />
    )
  }

  return (
    <ChartCard
      series={series}
      buildOption={buildOption}
      height={height}
    />
  )
}

interface BuildLabels {
  ocSeries: string
  drSeries: string
  ocReference: string
  drReference: string
}

/**
 * Pure builder — exporté pour test unitaire.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function buildCombatYieldOption(
  series: ChartSeries<CombatYieldPoint>[],
  labels: BuildLabels,
  ocP80: number = OC_P80,
): EChartsCoreOption {
  if (series.length === 0) {
    return { backgroundColor: CHART_BG }
  }

  const ocColor = resolveToken('divergent-pos')
  const drColor = resolveToken('divergent-neutral')
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  const ocSeries = series.find((s) => s.key === 'combat.oc')
  const drSeries = series.find((s) => s.key === 'combat.dr')
  // DR absent (titre sans damage_taken, ex. Halo 5) → courbe + référence + entrée
  // de légende DR retirées. Le graphe n'affiche que le Rendement.
  const showDr = drSeries != null

  const pctAxis = (v: number) => `${Math.round(v * 100)}%`
  const ocLineSeries = {
    type: 'line' as const,
    name: labels.ocSeries,
    data: ocSeries?.datapoints?.map((p) => [p.x, p.y]) ?? [],
    connectNulls: false,
    symbol: 'circle' as const,
    symbolSize: 5,
    lineStyle: { color: ocColor, width: 2 },
    itemStyle: { color: ocColor },
    markLine: {
      symbol: 'none' as const,
      silent: true,
      label: { show: false },
      lineStyle: { color: ocColor, type: 'dotted' as const, width: 1 },
      data: [
        {
          yAxis: ocP80,
          name: `${labels.ocReference} (${Math.round(ocP80 * 100)}%)`,
          label: {
            show: true,
            position: 'end' as const,
            color: ocColor,
            fontSize: 10,
            formatter: `${labels.ocReference} (${Math.round(ocP80 * 100)}%)`,
          },
        },
      ],
    },
  }
  const drLineSeries = {
    type: 'line' as const,
    name: labels.drSeries,
    data: drSeries?.datapoints?.map((p) => [p.x, p.y]) ?? [],
    connectNulls: false,
    symbol: 'circle' as const,
    symbolSize: 5,
    lineStyle: { color: drColor, width: 2 },
    itemStyle: { color: drColor },
    markLine: {
      symbol: 'none' as const,
      silent: true,
      label: { show: false },
      lineStyle: { color: drColor, type: 'dotted' as const, width: 1 },
      data: [
        {
          yAxis: DR_DISPLAY_P80,
          name: `${labels.drReference} (+${Math.round(DR_DISPLAY_P80 * 100)}%)`,
          label: {
            show: true,
            position: 'end' as const,
            color: drColor,
            fontSize: 10,
            formatter: `${labels.drReference} (+${Math.round(DR_DISPLAY_P80 * 100)}%)`,
          },
        },
      ],
    },
  }
  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 56, left: 56, right: 16 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      formatter: (params: Array<{ seriesName: string; value: [string, number | null]; marker: string }>) => {
        const lines = params
          .filter((p) => p.value?.[1] != null)
          .map((p) => {
            const val = p.value[1] as number
            const pct = Math.round(val * 100)
            // DR est déjà normalisé (value - 1), OC part de 0 — même échelle
            const formatted = p.seriesName === labels.drSeries && pct >= 0
              ? `+${pct}%`
              : `${pct}%`
            return `${p.marker}${escapeHtml(p.seriesName ?? '')}: <b>${formatted}</b>`
          })
        return lines.join('<br/>')
      },
    },
    legend: { ...getLegendBase(tc), data: showDr ? [labels.ocSeries, labels.drSeries] : [labels.ocSeries] },
    xAxis: { ...axis, type: 'time' },
    yAxis: {
      ...axis,
      type: 'value',
      min: 0,
      axisLabel: { ...axis.axisLabel, formatter: pctAxis },
    },
    series: showDr ? [ocLineSeries, drLineSeries] : [ocLineSeries],
  }
}
