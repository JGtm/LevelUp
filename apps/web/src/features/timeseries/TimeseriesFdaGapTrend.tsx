/**
 * TimeseriesFdaGapTrend — « Écart au FDA attendu » (onglet Résumé, sous la
 * tendance FDA valeur). Décision D1 du plan PLAN_EXPECTED_FDA_2026-07.
 *
 * Différentiel par match `kda − kda_expected` (FDA réel natif ADR 0006 moins FDA
 * attendu projeté côté backend). Rendu :
 *   - une AIRE divergente ancrée à 0 (dégradé vert au-dessus / rouge en dessous,
 *     bascule EXACTE sur 0 via le helper canonique `divergentZeroGradient`, PAS
 *     de visualMap) sur le différentiel BRUT — un match sans attendu = TROU
 *     (null), jamais 0 (D5) ;
 *   - une LIGNE lissée (moyenne mobile fenêtre 5, MÊME mécanique que
 *     `TimeseriesKdaValueTrend`) qui donne la tendance (le brut est bruité).
 *
 * Masqué par `useCapability('expected_stats')` (Halo 5 = pas d'attendu → null).
 * Tooltip : réel / attendu / écart (« — » si l'attendu manque, D5).
 */
import { useMemo, type ReactNode } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import {
  getEChartsThemeColors,
  getAxisBase,
  getGridBase,
  getLegendBase,
  getTooltipBase,
  CHART_BG,
  escapeHtml,
} from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { divergentZeroGradient } from '@/lib/charts/divergentZeroGradient'
import { useCapability } from '@/lib/capabilities/capabilities'
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'
import type { TimeseriesMatchRow } from '@/lib/api/types'
import { buildMatchCategories } from './matchLabels'
import { ChartFromOption } from './ChartFromOption'

const round2 = (v: number) => Math.round(v * 100) / 100

/**
 * Moyenne mobile centrée fenêtre `window`, NaN/null ignorés. Réplique la
 * mécanique de `rollingMean` de TimeseriesFormCharts (D1 : « MÊME lissage,
 * à l'identique ») — 2e occurrence tolérée (CLAUDE.md n°6, seuil = 3).
 */
function rollingMean(values: (number | null | undefined)[], window: number): (number | null)[] {
  const out: (number | null)[] = new Array(values.length).fill(null)
  if (window <= 1) return values.map((v) => (v == null ? null : v))
  const half = Math.floor(window / 2)
  for (let i = 0; i < values.length; i++) {
    const lo = Math.max(0, i - half)
    const hi = Math.min(values.length - 1, i + half)
    let sum = 0
    let n = 0
    for (let k = lo; k <= hi; k++) {
      const v = values[k]
      if (v == null || !Number.isFinite(v)) continue
      sum += v
      n++
    }
    out[i] = n > 0 ? sum / n : null
  }
  return out
}

/** FDA per-match si fini, sinon null. */
function finiteKda(v: number | null | undefined): number | null {
  return v != null && Number.isFinite(v) ? v : null
}

interface GapDetail {
  real: number | null
  expected: number | null
  gap: number | null
}

interface FdaGapSeries {
  categories: string[]
  /** Différentiel brut par match ; null = match sans attendu (D5). */
  rawDiff: (number | null)[]
  /** Différentiel lissé (fenêtre 5), trous préservés. */
  smoothed: (number | null)[]
  /** Valeurs natives par match pour le tooltip. */
  details: GapDetail[]
}

/** Séries dérivées d'un ensemble de matchs (différentiel brut, lissé, détails). */
function computeFdaGapSeries(rows: TimeseriesMatchRow[]): FdaGapSeries {
  const rawDiff: (number | null)[] = rows.map((r) => {
    const real = finiteKda(r.kda)
    const exp = finiteKda(r.kda_expected)
    if (real == null || exp == null) return null
    return round2(real - exp)
  })
  // Lissage fenêtre 5, mais on PRÉSERVE les trous (un match sans attendu ne doit pas
  // être « rempli » par ses voisins) → la tendance saute proprement les ~2 % manquants.
  const smoothed = rollingMean(rawDiff, 5).map((v, i) =>
    rawDiff[i] == null || v == null ? null : round2(v),
  )
  const details = rows.map<GapDetail>((r, i) => ({
    real: finiteKda(r.kda),
    expected: finiteKda(r.kda_expected),
    gap: rawDiff[i],
  }))
  return { categories: buildMatchCategories(rows), rawDiff, smoothed, details }
}

export interface FdaGapDiffLabels {
  /** Libellé de l'aire différentielle (légende + tooltip). */
  gap: string
  /** Libellé « réel » (tooltip). */
  real: string
  /** Libellé « attendu » (tooltip). */
  expected: string
  /** Libellé de la ligne lissée (« Tendance »). */
  smoothing: string
}

/** Formateur de tooltip par match : réel / attendu / écart (« — » si absent, D5). */
function makeGapTooltip(details: GapDetail[], labels: FdaGapDiffLabels) {
  const fmt = (v: number | null, signed = false) =>
    v == null ? '—' : signed && v >= 0 ? `+${round2(v)}` : `${round2(v)}`
  return (params: unknown) => {
    const arr = Array.isArray(params) ? params : []
    if (arr.length === 0) return ''
    const idx = (arr[0] as { dataIndex?: number }).dataIndex ?? 0
    const d = details[idx]
    const cat = escapeHtml(((arr[0] as { name?: string }).name ?? '').replace(/\n/g, ' · '))
    if (!d) return `<strong>${cat}</strong>`
    return (
      `<strong>${cat}</strong><br/>` +
      `${escapeHtml(labels.real)}: ${fmt(d.real)}<br/>` +
      `${escapeHtml(labels.expected)}: ${fmt(d.expected)}<br/>` +
      `${escapeHtml(labels.gap)}: <b>${fmt(d.gap, true)}</b>`
    )
  }
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildFdaGapDiffOption(
  rows: TimeseriesMatchRow[],
  labels: FdaGapDiffLabels,
): EChartsCoreOption | null {
  if (rows.length === 0) return null
  const tc = getEChartsThemeColors()
  const { categories, rawDiff, smoothed, details } = computeFdaGapSeries(rows)
  const gradient = divergentZeroGradient(rawDiff.filter((v): v is number => v != null))
  const trendColor = resolveToken('chart-series-1')
  const zeroLineColor = resolveToken('divergent-neutral')

  return {
    backgroundColor: CHART_BG,
    grid: getGridBase(),
    tooltip: { ...getTooltipBase(tc), trigger: 'axis', formatter: makeGapTooltip(details, labels) },
    legend: { ...getLegendBase(tc), bottom: 0 },
    xAxis: {
      ...getAxisBase(tc),
      type: 'category',
      data: categories,
      axisLabel: { ...getAxisBase(tc).axisLabel, interval: 0, fontSize: 9 },
    },
    yAxis: { ...getAxisBase(tc), type: 'value', scale: true },
    series: [
      {
        type: 'line',
        name: labels.gap,
        data: rawDiff,
        // Aire divergente ancrée à 0 (vert = au-dessus de l'attendu, rouge = en dessous).
        areaStyle: { color: gradient, opacity: 0.18, origin: 0 },
        lineStyle: { color: gradient, width: 1.5 },
        showSymbol: false,
        // Trous VISIBLES (un match sans attendu ne se relie pas — D5).
        connectNulls: false,
        // Ligne 0 : parité réel = attendu.
        markLine: {
          silent: true,
          symbol: 'none',
          lineStyle: { color: zeroLineColor, type: 'dashed', width: 1 },
          label: { show: false },
          data: [{ yAxis: 0 }],
        },
      },
      {
        type: 'line',
        name: labels.smoothing,
        data: smoothed,
        showSymbol: false,
        smooth: true,
        connectNulls: true,
        lineStyle: { color: trendColor, width: 2 },
        z: 5,
      },
    ],
  }
}

export interface TimeseriesFdaGapTrendProps {
  rows: TimeseriesMatchRow[]
  height?: number
  title?: ReactNode
  emptyMessage?: string
  labels: FdaGapDiffLabels
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
    () => buildFdaGapDiffOption(rows, labels),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [rows, labels, themeVersion],
  )

  // Titre sans attendu (ex. Halo 5) → masquage silencieux (pas de carte vide).
  if (!hasExpectedStats) return null
  return <ChartFromOption title={title} option={option} height={height} emptyMessage={emptyMessage} />
}
