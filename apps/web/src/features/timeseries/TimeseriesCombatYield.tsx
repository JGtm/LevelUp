/**
 * TimeseriesCombatYield — 2 courbes (Offensive Conversion + Defensive
 * Resistance) avec lignes de référence p80, rendu via ECharts.
 *
 * Phase 3 P3.F : remplace l'ancien wrapper Plotly `CombatYieldTimeseries`.
 * Construit côté client depuis les données MatchHistoryRow.
 *
 * Lignes de référence p80 OC=0.83 / DR=1.59 — miroir des constantes Go
 * `combat_yield.go`. Pointillé fin coloré comme la série de référence.
 */
import { useCallback, useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { resolveToken } from '@/lib/accessibility'
import {
  CHART_BG,
  axisBase,
  legendBase,
  tooltipBase,
} from '@/components/charts/_utils'
import type { MatchHistoryRow } from '@/lib/api/types'

/** p80 references — miroir des constantes Go combat_yield.go. */
const OC_P80 = 0.83
const DR_P80 = 1.59

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

  const series = useMemo<ChartSeries<CombatYieldPoint>[]>(() => {
    if (filtered.length === 0) return []
    return [
      {
        key: 'combat.oc',
        meta: { gamertag: labels.ocSeries },
        datapoints: filtered.map((r) => ({
          x: r.start_time,
          y: r.offensive_conversion ?? null,
        })),
      },
      {
        key: 'combat.dr',
        meta: { gamertag: labels.drSeries },
        datapoints: filtered.map((r) => ({
          x: r.start_time,
          y: r.defensive_resistance ?? null,
        })),
      },
    ]
  }, [filtered, labels.ocSeries, labels.drSeries])

  const buildOption = useCallback(
    (s: ChartSeries<CombatYieldPoint>[]) => buildCombatYieldOption(s, labels),
    [labels],
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
): EChartsCoreOption {
  if (series.length === 0) {
    return { backgroundColor: CHART_BG }
  }

  const ocColor = resolveToken('divergent-pos')
  const drColor = resolveToken('divergent-neutral')

  const ocSeries = series.find((s) => s.key === 'combat.oc')
  const drSeries = series.find((s) => s.key === 'combat.dr')

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 56, left: 56, right: 16 },
    tooltip: { ...tooltipBase, trigger: 'axis' },
    legend: { ...legendBase, data: [labels.ocSeries, labels.drSeries] },
    xAxis: { ...axisBase, type: 'time' },
    yAxis: { ...axisBase, type: 'value', min: 0 },
    series: [
      {
        type: 'line',
        name: labels.ocSeries,
        data: ocSeries?.datapoints.map((p) => [p.x, p.y]) ?? [],
        connectNulls: false,
        symbol: 'circle',
        symbolSize: 5,
        lineStyle: { color: ocColor, width: 2 },
        itemStyle: { color: ocColor },
        markLine: {
          symbol: 'none',
          silent: true,
          label: { show: false },
          lineStyle: { color: ocColor, type: 'dotted', width: 1 },
          data: [
            {
              yAxis: OC_P80,
              name: `${labels.ocReference} (${OC_P80})`,
              label: {
                show: true,
                position: 'end',
                color: ocColor,
                fontSize: 10,
                formatter: `${labels.ocReference} (${OC_P80})`,
              },
            },
          ],
        },
      },
      {
        type: 'line',
        name: labels.drSeries,
        data: drSeries?.datapoints.map((p) => [p.x, p.y]) ?? [],
        connectNulls: false,
        symbol: 'circle',
        symbolSize: 5,
        lineStyle: { color: drColor, width: 2 },
        itemStyle: { color: drColor },
        markLine: {
          symbol: 'none',
          silent: true,
          label: { show: false },
          lineStyle: { color: drColor, type: 'dotted', width: 1 },
          data: [
            {
              yAxis: DR_P80,
              name: `${labels.drReference} (${DR_P80})`,
              label: {
                show: true,
                position: 'end',
                color: drColor,
                fontSize: 10,
                formatter: `${labels.drReference} (${DR_P80})`,
              },
            },
          ],
        },
      },
    ],
  }
}
