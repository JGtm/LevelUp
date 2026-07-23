/**
 * TimeseriesFdaGapTrend — « Écart cumulé au FDA attendu » (onglet Résumé, à
 * DROITE du FDA). Décision D1 du plan PLAN_EXPECTED_FDA_2026-07 + retouche UX
 * 2026-07-23 : passage à la MÊME forme que Sessions (SessionFdaGapCumulative).
 *
 * Somme CUMULÉE signée du différentiel `kda − kda_expected` (FDA réel natif
 * ADR 0006 moins FDA attendu projeté backend), match après match DANS L'ORDRE
 * DONNÉ : les `match_rows` arrivent déjà triés chronologiquement ASC côté
 * service → PAS de re-tri (contrairement à Sessions qui trie par start_time).
 * Rendu : aire signée divergente ancrée à 0 (helper canonique
 * `divergentZeroGradient`, PAS de visualMap) + markLine 0. Cumul via le helper
 * canonique `cumulativeFdaGap` (source unique — CLAUDE.md n°6).
 *
 * D5 : un match sans attendu (`kda`/`kda_expected` null ou non-fini) ne fait PAS
 * avancer le cumul (report de la dernière valeur), mais figure quand même sur
 * l'axe. Masqué par `useCapability('expected_stats')` (Halo 5 = pas d'attendu →
 * null). Tooltip : cumul / réel / attendu / écart du match (« — » si absent).
 */
import { useMemo, type ReactNode } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import {
  getEChartsThemeColors,
  getAxisBase,
  getTooltipBase,
  CHART_BG,
  escapeHtml,
} from '@/components/charts/_utils'
import { cumulativeFdaGap } from '@/lib/charts/cumulativeFdaGap'
import { divergentZeroGradient } from '@/lib/charts/divergentZeroGradient'
import { useCapability } from '@/lib/capabilities/capabilities'
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'
import type { TimeseriesMatchRow } from '@/lib/api/types'
import { buildMatchCategories } from './matchLabels'
import { ChartFromOption } from './ChartFromOption'

export interface FdaGapCumulativeLabels {
  /** Libellé de la série cumulée (tooltip). */
  series: string
  /** Libellé « réel » (tooltip). */
  real: string
  /** Libellé « attendu » (tooltip). */
  expected: string
  /** Libellé de l'écart du match (tooltip). */
  gap: string
}

/** Formate un écart : « — » si absent, préfixe « + » si signé positif. */
function fmtGap(v: number | null, signed = false): string {
  if (v == null) return '—'
  return signed && v >= 0 ? `+${v}` : `${v}`
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildFdaGapCumulativeOption(
  rows: TimeseriesMatchRow[],
  labels: FdaGapCumulativeLabels,
): EChartsCoreOption | null {
  if (rows.length === 0) return null
  // Ordre du service conservé (déjà trié ASC) — pas de re-tri.
  const points = cumulativeFdaGap(
    rows.map((r) => ({ real: r.kda ?? null, expected: r.kda_expected ?? null })),
  )
  const categories = buildMatchCategories(rows)
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  // Interval ADAPTATIF (Timeseries peut avoir des centaines de matchs → labels
  // illisibles avec interval:0), aligné sur SessionFdaGapCumulative.
  const interval = points.length > 30 ? Math.floor(points.length / 12) : 0
  const values = points.map((p) => p.cumulative)
  const gradient = divergentZeroGradient(values)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 64, left: 48, right: 24 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      formatter: (params: Array<{ dataIndex?: number }>) => {
        if (!Array.isArray(params) || params.length === 0) return ''
        const idx = params[0]?.dataIndex ?? 0
        const p = points[idx]
        if (!p) return ''
        const cat = escapeHtml((categories[idx] ?? '').replace(/\n/g, ' · '))
        return (
          `<strong>${cat}</strong><br/>` +
          `${escapeHtml(labels.series)}: <b>${fmtGap(p.cumulative, true)}</b><br/>` +
          `${escapeHtml(labels.real)}: ${fmtGap(p.real)}<br/>` +
          `${escapeHtml(labels.expected)}: ${fmtGap(p.expected)}<br/>` +
          `${escapeHtml(labels.gap)}: ${fmtGap(p.gap, true)}`
        )
      },
    },
    xAxis: {
      ...axis,
      type: 'category',
      boundaryGap: false,
      data: categories,
      axisLabel: { ...(axis.axisLabel as Record<string, unknown>), interval },
    },
    yAxis: { ...axis, type: 'value' },
    series: [
      {
        type: 'line',
        name: labels.series,
        data: values,
        showSymbol: false,
        // Ligne + aire divergentes (vert = cumul au-dessus de l'attendu, rouge
        // en dessous), aire ancrée à 0 (même dégradé, bascule pile sur 0).
        lineStyle: { width: 2, color: gradient },
        areaStyle: { color: gradient, opacity: 0.18, origin: 0 },
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

export interface TimeseriesFdaGapTrendProps {
  rows: TimeseriesMatchRow[]
  height?: number
  title?: ReactNode
  emptyMessage?: string
  labels: FdaGapCumulativeLabels
}

export function TimeseriesFdaGapTrend({
  rows,
  title,
  emptyMessage,
  height = 320,
  labels,
}: TimeseriesFdaGapTrendProps) {
  const hasExpectedStats = useCapability('expected_stats')
  const themeVersion = useThemeVersion()

  const option = useMemo<EChartsCoreOption | null>(
    () => buildFdaGapCumulativeOption(rows, labels),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [rows, labels, themeVersion],
  )

  // Titre sans attendu (ex. Halo 5) → masquage silencieux (pas de carte vide).
  if (!hasExpectedStats) return null
  return <ChartFromOption title={title} option={option} height={height} emptyMessage={emptyMessage} />
}
