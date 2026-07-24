/**
 * SessionFdaGapCumulative — « Écart cumulé au FDA attendu » sur les matchs d'une
 * session. Décision D2 du plan PLAN_EXPECTED_FDA_2026-07.
 *
 * Somme CUMULÉE du différentiel `kda − kda_expected` (FDA réel natif ADR 0006
 * moins FDA attendu projeté backend), match après match dans l'ordre
 * chronologique. Un match sans attendu est SAUTÉ : la courbe reporte la dernière
 * valeur cumulée (D5 — jamais 0, jamais de trou dans le cumul).
 *
 * Même pattern visuel que `SessionNetScoreArea` : aire signée divergente ancrée à
 * 0 (helper canonique `divergentZeroGradient`, PAS de visualMap) + markLine 0 sur
 * l'axe PRIMAIRE, PLUS 2 courbes fines « FDA réel (cumulé) » / « FDA attendu
 * (cumulé) » sur un axe SECONDAIRE droit (`yAxisIndex: 1`, pattern dual-axis de
 * `TimeseriesKdaBars`) — lisent `cumulativeReal`/`cumulativeExpected` du helper
 * canonique `cumulativeFdaGap`, qui appliquent le MÊME skip conjoint que le
 * cumul d'écart (identité cumulativeReal − cumulativeExpected === cumulative).
 * Masqué par `useCapability('expected_stats')` (Halo 5 = pas d'attendu → null).
 */
import { useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken } from '@/lib/accessibility'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
} from '@/components/charts/_utils'
import { cumulativeFdaGap, type FdaGapCumPoint } from '@/lib/charts/cumulativeFdaGap'
import { divergentZeroGradient } from '@/lib/charts/divergentZeroGradient'
import { useCapability } from '@/lib/capabilities/capabilities'
import type { SessionDetailMatchRow } from '@/lib/api/types'

import { sessionMatchAxisLabel, useSessionT } from './_shared'

export interface FdaGapPoint extends FdaGapCumPoint {
  /** Étiquette d'axe X du match (`#N` + carte/mode). */
  label: string
}

/**
 * Cumul de l'écart au FDA attendu sur les matchs, TRIÉS chronologiquement puis
 * délégué au helper canonique `cumulativeFdaGap` (source unique du cumul,
 * CLAUDE.md n°6). D5 : un match sans attendu n'ajoute rien au cumul (report de
 * la dernière valeur) mais figure quand même sur l'axe.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function computeCumulativeFdaGap(matches: SessionDetailMatchRow[]): FdaGapPoint[] {
  const sorted = [...matches].sort((a, b) => a.start_time.localeCompare(b.start_time))
  const cum = cumulativeFdaGap(
    sorted.map((m) => ({ real: m.kda ?? null, expected: m.kda_expected ?? null })),
  )
  return sorted.map((m, i) => ({
    label: sessionMatchAxisLabel(i, m.map_name, m.pair_name),
    ...cum[i],
  }))
}

export interface FdaGapCumulativeLabels {
  seriesLabel: string
  realLabel: string
  expectedLabel: string
  gapLabel: string
  /** Libellé de la courbe « FDA réel (cumulé) » (légende + tooltip, axe secondaire). */
  realCumulativeLabel: string
  /** Libellé de la courbe « FDA attendu (cumulé) » (légende + tooltip, axe secondaire). */
  expectedCumulativeLabel: string
  yDomain?: [number, number]
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildSessionFdaGapOption(
  series: ChartSeries<FdaGapPoint>[],
  opts: FdaGapCumulativeLabels,
): EChartsCoreOption {
  const points = series[0]?.datapoints ?? []
  if (points.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const interval = points.length > 30 ? Math.floor(points.length / 12) : 0

  const values = points.map((p) => p.cumulative)
  const realCumValues = points.map((p) => p.cumulativeReal)
  const expectedCumValues = points.map((p) => p.cumulativeExpected)
  const divergentColor = divergentZeroGradient(values)
  const realColor = resolveToken('chart-series-1')
  const expectedColor = resolveToken('chart-series-2')
  const fmt = (v: number | null, signed = false) =>
    v == null ? '—' : signed && v >= 0 ? `+${v}` : `${v}`

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 64, left: 48, right: 56 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      formatter: (params: Array<{ dataIndex?: number }>) => {
        if (!Array.isArray(params) || params.length === 0) return ''
        const p = points[params[0]?.dataIndex ?? 0]
        if (!p) return ''
        const cat = escapeHtml(p.label.replace('\n', ' · '))
        return (
          `${cat}<br/>` +
          `${escapeHtml(opts.seriesLabel)}: <b>${fmt(p.cumulative, true)}</b><br/>` +
          `${escapeHtml(opts.realCumulativeLabel)}: ${fmt(p.cumulativeReal, true)}<br/>` +
          `${escapeHtml(opts.expectedCumulativeLabel)}: ${fmt(p.cumulativeExpected, true)}<br/>` +
          `${escapeHtml(opts.realLabel)}: ${fmt(p.real)}<br/>` +
          `${escapeHtml(opts.expectedLabel)}: ${fmt(p.expected)}<br/>` +
          `${escapeHtml(opts.gapLabel)}: ${fmt(p.gap, true)}`
        )
      },
    },
    legend: {
      ...getLegendBase(tc),
      data: [opts.seriesLabel, opts.realCumulativeLabel, opts.expectedCumulativeLabel],
    },
    xAxis: {
      ...axis,
      type: 'category',
      boundaryGap: false,
      data: points.map((p) => p.label),
      axisLabel: { ...(axis.axisLabel as Record<string, unknown>), interval },
    },
    // Axe primaire (écart cumulé, gauche) + axe secondaire (FDA réel/attendu
    // cumulés, droite — pattern dual-axis de TimeseriesKdaBars).
    yAxis: [
      {
        ...axis,
        type: 'value',
        ...(opts.yDomain ? { min: opts.yDomain[0], max: opts.yDomain[1] } : {}),
      },
      { ...axis, type: 'value', position: 'right' },
    ],
    series: [
      {
        name: opts.seriesLabel,
        type: 'line',
        data: values,
        showSymbol: false,
        // Ligne + aire divergentes (vert = cumul au-dessus de l'attendu, rouge en dessous),
        // aire ancrée à 0 (même dégradé, bascule pile sur 0).
        lineStyle: { width: 2, color: divergentColor },
        areaStyle: { color: divergentColor, opacity: 0.18, origin: 0 },
        // Ligne de référence à 0 (parité cumulée réel = attendu).
        markLine: {
          silent: true,
          symbol: 'none',
          lineStyle: { color: tc.axisLabel, type: 'dashed', width: 1 },
          label: { show: false },
          data: [{ yAxis: 0 }],
        },
      },
      {
        name: opts.realCumulativeLabel,
        type: 'line',
        yAxisIndex: 1,
        data: realCumValues,
        symbol: 'none',
        lineStyle: { width: 1, color: realColor },
      },
      {
        name: opts.expectedCumulativeLabel,
        type: 'line',
        yAxisIndex: 1,
        data: expectedCumValues,
        symbol: 'none',
        lineStyle: { width: 1, color: expectedColor, type: 'dashed' },
      },
    ],
  }
}

interface Props {
  title: string
  matches: SessionDetailMatchRow[]
  height?: number
  /** Domaine Y [min, max] partagé A/B en mode comparaison (sinon auto-scale). */
  yDomain?: [number, number]
}

export function SessionFdaGapCumulative({ title, matches, height = 280, yDomain }: Props) {
  const hasExpectedStats = useCapability('expected_stats')
  const t = useSessionT()

  const series = useMemo<ChartSeries<FdaGapPoint>[]>(() => {
    if (matches.length === 0) return []
    return [{ key: 'fda_gap', datapoints: computeCumulativeFdaGap(matches) }]
  }, [matches])

  // Titre sans attendu (ex. Halo 5) → masquage silencieux (pas de carte vide).
  if (!hasExpectedStats) return null

  return (
    <ChartCard
      title={title}
      series={series}
      height={height}
      buildOption={(s) =>
        buildSessionFdaGapOption(s, {
          seriesLabel: t('session.detail.fda_gap_series'),
          realLabel: t('session.detail.fda_gap_real'),
          expectedLabel: t('session.detail.fda_gap_expected'),
          gapLabel: t('session.detail.fda_gap_gap'),
          realCumulativeLabel: t('session.detail.fda_gap_real_cumulative'),
          expectedCumulativeLabel: t('session.detail.fda_gap_expected_cumulative'),
          yDomain,
        })
      }
    />
  )
}
