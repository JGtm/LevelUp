/**
 * squadNetLivesChart — « Balance des dégâts cumulée » par joueur (onglet Dynamique).
 *
 * Pour chaque joueur : cumul de la balance `netLives = (dégâts infligés − dégâts
 * subis) / PV-pour-tuer` par `match_order` CROISSANT. 1 série `line` par joueur,
 * couleur via `colorByPlayer` (getSquadPlayerColors), markLine 0 (équilibre), PAS
 * d'aire (multi-séries).
 *
 * Report D5 (délégué au helper générique `cumulativeSigned`) : un match sans
 * `damage_taken` (ou `damage_dealt`) ne fait pas avancer le cumul — la courbe
 * REPORTE la dernière valeur. Les trous d'intersection (aucune ligne à un
 * `match_order`) restent `null` et sont pontés par `connectNulls`.
 *
 * Calqué sur `squadFdaGapChart.ts` : réutilise ses helpers partagés
 * (`orderedPlayers` / `maxLength` / `xAxisLabels`).
 */
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
} from '@/components/charts/_utils'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import { cumulativeSigned } from '@/lib/charts/cumulativeSeries'
import { netLives } from '@/lib/charts/netLives'

import {
  maxLength,
  orderedPlayers,
  xAxisLabels,
  type CommonOpts,
} from './squadPerformanceLineCharts'

/**
 * Cumul de la balance des dégâts par `match_order` croissant pour UN joueur,
 * délégué au helper générique `cumulativeSigned` (source unique du cumul,
 * CLAUDE.md n°6). Un point sans `damage_taken`/`damage_dealt` reporte le cumul
 * (D5) mais figure à la valeur courante ; les indices sans point restent `null`.
 */
export function cumulativeNetLivesSeries(
  points: SquadPerformanceSeriesPoint[],
  n: number,
  hp: number,
): Array<number | null> {
  const ordered = points
    .filter((p) => p.match_order >= 0 && p.match_order < n)
    .sort((a, b) => a.match_order - b.match_order)
  const cum = cumulativeSigned(ordered.map((p) => netLives(p.damage_dealt, p.damage_taken, hp)))
  const data = new Array<number | null>(n).fill(null)
  ordered.forEach((p, i) => {
    data[p.match_order] = cum[i].cumulative
  })
  return data
}

export interface NetLivesCumulativeOpts extends CommonOpts {
  /** PV-pour-tuer du titre (barème de conversion dégâts → vies). */
  hp: number
  /** Décimales affichées (tooltip + axe Y). Défaut 1. */
  decimals?: number
}

/** Formate une valeur de balance signée (`+1.2` / `-0.4` / `+0.0`). */
function fmtSigned(v: number, decimals: number): string {
  const fixed = v.toFixed(decimals)
  return v >= 0 ? `+${fixed}` : fixed
}

export function buildNetLivesCumulativeOption(
  rows: Record<string, SquadPerformanceSeriesPoint[]>,
  opts: NetLivesCumulativeOpts,
): EChartsCoreOption {
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const players = orderedPlayers(rows, opts.playerOrder)
  if (players.length === 0) return { backgroundColor: CHART_BG }

  const n = maxLength(rows, players)
  if (n === 0) return { backgroundColor: CHART_BG }

  const xLabels = opts.xLabels ?? xAxisLabels(n)
  const decimals = opts.decimals ?? 1
  const hiddenPlayers = opts.hiddenPlayers ?? new Set<string>()
  const emptyData = new Array<number | null>(n).fill(null)

  const series = players.map((player, idx) => {
    const color = opts.colorByPlayer[player] ?? '#888' // color-allow: gris structurel pour joueur sans couleur attribuée
    const data = hiddenPlayers.has(player)
      ? emptyData
      : cumulativeNetLivesSeries(rows[player], n, opts.hp)
    return {
      name: player,
      type: 'line' as const,
      data,
      lineStyle: { color, width: 2 },
      itemStyle: { color },
      symbol: 'circle' as const,
      symbolSize: 4,
      connectNulls: true,
      // markLine 0 (équilibre : dégâts infligés = dégâts subis) rendue une seule
      // fois, attachée au premier joueur.
      ...(idx === 0
        ? {
            markLine: {
              silent: true,
              symbol: 'none',
              lineStyle: { color: tc.axisLabel, type: 'dashed' as const, width: 1 },
              label: { show: false },
              data: [{ yAxis: 0 }],
            },
          }
        : {}),
    }
  })

  return {
    backgroundColor: CHART_BG,
    grid: { top: 28, bottom: 36, left: 8, right: 24, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'line' },
      valueFormatter: (v: unknown) => (typeof v === 'number' ? fmtSigned(v, decimals) : '-'),
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
      axisLabel: {
        ...axis.axisLabel,
        formatter: (v: number) => fmtSigned(v, decimals),
      },
    },
    series,
  }
}
