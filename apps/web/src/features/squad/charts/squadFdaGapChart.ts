/**
 * squadFdaGapChart — « Écart cumulé au FDA attendu » par joueur (onglet Synergies).
 *
 * Décisions D3/D5 du plan PLAN_EXPECTED_FDA_2026-07.
 *
 * Pour chaque joueur : cumul du différentiel `kda − kda_expected` (FDA réel natif
 * ADR 0006 moins FDA attendu projeté backend) par `match_order` CROISSANT. 1 série
 * `line` par joueur, couleur via `colorByPlayer` (getSquadPlayerColors), markLine 0,
 * PAS d'aire (multi-séries).
 *
 * D5 : un match sans attendu (`kda_expected` NULL / non-fini, ou `kda` absent) ne
 * fait pas avancer le cumul — la courbe REPORTE la dernière valeur (jamais 0, jamais
 * de rupture). Les trous d'intersection (aucune ligne à un `match_order`) restent
 * `null` et sont pontés par `connectNulls`.
 *
 * Extrait de `squadPerformanceLineCharts.ts` (déjà > 500 L) : builder dédié plutôt
 * que gonfler le fichier voisin. Réutilise ses helpers partagés
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
import { cumulativeFdaGap, meanFdaGap, type FdaGapPair } from '@/lib/charts/cumulativeFdaGap'

import {
  maxLength,
  orderedPlayers,
  xAxisLabels,
  type CommonOpts,
} from './squadPerformanceLineCharts'

/**
 * Paire FDA réel / attendu d'un point de la série de performance escouade. FDA
 * réel = valeur native (`kda`, ADR 0006) ; FDA attendu = projeté backend
 * (`kda_expected`). Le helper canonique traite null/non-fini comme absent (D5).
 */
function toPair(p: SquadPerformanceSeriesPoint): FdaGapPair {
  return { real: p.kda ?? null, expected: p.kda_expected ?? null }
}

/**
 * Cumul du différentiel par `match_order` croissant pour UN joueur, délégué au
 * helper canonique `cumulativeFdaGap` (source unique du cumul, CLAUDE.md n°6).
 * D5 : un point sans attendu ne fait pas avancer le cumul (report), mais figure
 * quand même à la valeur courante. Les indices sans point (trou d'intersection)
 * restent `null`.
 */
export function cumulativeFdaGapSeries(
  points: SquadPerformanceSeriesPoint[],
  n: number,
): Array<number | null> {
  const ordered = points
    .filter((p) => p.match_order >= 0 && p.match_order < n)
    .sort((a, b) => a.match_order - b.match_order)
  const cum = cumulativeFdaGap(ordered.map(toPair))
  const data = new Array<number | null>(n).fill(null)
  ordered.forEach((p, i) => {
    data[p.match_order] = cum[i].cumulative
  })
  return data
}

/**
 * Écart MOYEN par match d'un joueur, calculé UNIQUEMENT sur les matchs avec attendu
 * (D3, pastille KPI « +0,7/match »), délégué au helper canonique `meanFdaGap`.
 * `null` si aucun match exploitable.
 */
export function meanFdaGapPerMatch(points: SquadPerformanceSeriesPoint[]): number | null {
  return meanFdaGap(points.map(toPair))
}

export interface FdaGapCumulativeOpts extends CommonOpts {
  /** Décimales affichées (tooltip + axe Y). Défaut 1. */
  decimals?: number
}

/** Formate une valeur d'écart signée (`+0.7` / `-0.4` / `+0.0`). */
function fmtSigned(v: number, decimals: number): string {
  const fixed = v.toFixed(decimals)
  return v >= 0 ? `+${fixed}` : fixed
}

export function buildFdaGapCumulativeOption(
  rows: Record<string, SquadPerformanceSeriesPoint[]>,
  opts: FdaGapCumulativeOpts,
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
    const data = hiddenPlayers.has(player) ? emptyData : cumulativeFdaGapSeries(rows[player], n)
    return {
      name: player,
      type: 'line' as const,
      data,
      lineStyle: { color, width: 2 },
      itemStyle: { color },
      symbol: 'circle' as const,
      symbolSize: 4,
      connectNulls: true,
      // markLine 0 (parité cumulée : FDA réel = FDA attendu) rendue une seule fois,
      // attachée au premier joueur.
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
