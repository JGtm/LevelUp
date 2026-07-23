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
 * 0 (helper canonique `divergentZeroGradient`, PAS de visualMap) + markLine 0.
 * Masqué par `useCapability('expected_stats')` (Halo 5 = pas d'attendu → null).
 */
import { useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
} from '@/components/charts/_utils'
import { divergentZeroGradient } from '@/lib/charts/divergentZeroGradient'
import { useCapability } from '@/lib/capabilities/capabilities'
import type { SessionDetailMatchRow } from '@/lib/api/types'

import { sessionMatchAxisLabel, useSessionT } from './_shared'

const round2 = (v: number) => Math.round(v * 100) / 100

export interface FdaGapPoint {
  label: string
  /** Écart cumulé (reporte la dernière valeur si le match courant n'a pas d'attendu). */
  cumulative: number
  /** FDA réel du match (null si absent/non-fini). */
  real: number | null
  /** FDA attendu du match (null si absent/non-fini). */
  expected: number | null
  /** Écart du match `réel − attendu` (null si l'attendu manque — D5). */
  gap: number | null
}

function finite(v: number | null | undefined): number | null {
  return v != null && Number.isFinite(v) ? v : null
}

/**
 * Cumul de l'écart au FDA attendu sur les matchs (triés chronologiquement).
 * D5 : un match sans attendu n'ajoute rien au cumul (report de la dernière
 * valeur) mais figure quand même sur l'axe (point porté à la valeur courante).
 */
export function computeCumulativeFdaGap(matches: SessionDetailMatchRow[]): FdaGapPoint[] {
  const sorted = [...matches].sort((a, b) => a.start_time.localeCompare(b.start_time))
  let running = 0
  return sorted.map((m, i) => {
    const real = finite(m.kda)
    const expected = finite(m.kda_expected)
    const gap = real != null && expected != null ? round2(real - expected) : null
    if (gap != null) running = round2(running + gap)
    return {
      label: sessionMatchAxisLabel(i, m.map_name, m.pair_name),
      cumulative: running,
      real: real != null ? round2(real) : null,
      expected: expected != null ? round2(expected) : null,
      gap,
    }
  })
}

export interface FdaGapCumulativeLabels {
  seriesLabel: string
  realLabel: string
  expectedLabel: string
  gapLabel: string
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
  const divergentColor = divergentZeroGradient(values)
  const fmt = (v: number | null, signed = false) =>
    v == null ? '—' : signed && v >= 0 ? `+${v}` : `${v}`

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 64, left: 48, right: 24 },
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
          `${escapeHtml(opts.realLabel)}: ${fmt(p.real)}<br/>` +
          `${escapeHtml(opts.expectedLabel)}: ${fmt(p.expected)}<br/>` +
          `${escapeHtml(opts.gapLabel)}: ${fmt(p.gap, true)}`
        )
      },
    },
    xAxis: {
      ...axis,
      type: 'category',
      boundaryGap: false,
      data: points.map((p) => p.label),
      axisLabel: { ...(axis.axisLabel as Record<string, unknown>), interval },
    },
    yAxis: {
      ...axis,
      type: 'value',
      ...(opts.yDomain ? { min: opts.yDomain[0], max: opts.yDomain[1] } : {}),
    },
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
          yDomain,
        })
      }
    />
  )
}
