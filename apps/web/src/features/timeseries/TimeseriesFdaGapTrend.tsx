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
 * `divergentZeroGradient`, PAS de visualMap) + markLine 0 sur l'axe PRIMAIRE,
 * PLUS 2 courbes fines « FDA réel (cumulé) » / « FDA attendu (cumulé) » sur un
 * axe SECONDAIRE droit (`yAxisIndex: 1`, pattern dual-axis de
 * `TimeseriesKdaBars`) — lisent `cumulativeReal`/`cumulativeExpected` du helper
 * canonique `cumulativeFdaGap` (source unique — CLAUDE.md n°6), qui appliquent
 * le MÊME skip conjoint que le cumul d'écart (identité
 * cumulativeReal − cumulativeExpected === cumulative).
 *
 * D5 : un match sans attendu (`kda`/`kda_expected` null ou non-fini) ne fait PAS
 * avancer le cumul (report de la dernière valeur), mais figure quand même sur
 * l'axe. Masqué par `useCapability('expected_stats')` (Halo 5 = pas d'attendu →
 * null). Tooltip : cumul écart / réel cumulé / attendu cumulé / réel du match /
 * attendu du match / écart du match (« — » si absent).
 */
import { useMemo, type ReactNode } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken } from '@/lib/accessibility'
import {
  getEChartsThemeColors,
  getAxisBase,
  getLegendBase,
  getTooltipBase,
  CHART_BG,
  escapeHtml,
} from '@/components/charts/_utils'
import { cumulativeFdaGap, meanFdaGap } from '@/lib/charts/cumulativeFdaGap'
import { divergentZeroGradient } from '@/lib/charts/divergentZeroGradient'
import { useCapability } from '@/lib/capabilities/capabilities'
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'
import type { TimeseriesMatchRow } from '@/lib/api/types'
import { buildMatchCategories } from './matchLabels'
import { ChartFromOption } from './ChartFromOption'

export interface FdaGapCumulativeLabels {
  /** Libellé de la série cumulée (tooltip + légende). */
  series: string
  /** Libellé « réel » du match (tooltip). */
  real: string
  /** Libellé « attendu » du match (tooltip). */
  expected: string
  /** Libellé de l'écart du match (tooltip). */
  gap: string
  /** Libellé de la courbe « FDA réel (cumulé) » (légende + tooltip, axe secondaire). */
  realCumulative: string
  /** Libellé de la courbe « FDA attendu (cumulé) » (légende + tooltip, axe secondaire). */
  expectedCumulative: string
  /** Caption du KPI de pied de carte (« Écart moyen par match »). */
  avgCaption: string
  /** Suffixe du KPI (« /match »). */
  perMatch: string
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
  const realCumValues = points.map((p) => p.cumulativeReal)
  const expectedCumValues = points.map((p) => p.cumulativeExpected)
  const gradient = divergentZeroGradient(values)
  const realColor = resolveToken('chart-series-1')
  const expectedColor = resolveToken('chart-series-2')

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 64, left: 48, right: 56 },
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
          `${escapeHtml(labels.realCumulative)}: ${fmtGap(p.cumulativeReal, true)}<br/>` +
          `${escapeHtml(labels.expectedCumulative)}: ${fmtGap(p.cumulativeExpected, true)}<br/>` +
          `${escapeHtml(labels.real)}: ${fmtGap(p.real)}<br/>` +
          `${escapeHtml(labels.expected)}: ${fmtGap(p.expected)}<br/>` +
          `${escapeHtml(labels.gap)}: ${fmtGap(p.gap, true)}`
        )
      },
    },
    legend: {
      ...getLegendBase(tc),
      data: [labels.series, labels.realCumulative, labels.expectedCumulative],
    },
    xAxis: {
      ...axis,
      type: 'category',
      boundaryGap: false,
      data: categories,
      axisLabel: { ...(axis.axisLabel as Record<string, unknown>), interval },
    },
    // Axe primaire (écart cumulé, gauche) + axe secondaire (FDA réel/attendu
    // cumulés, droite — pattern dual-axis de TimeseriesKdaBars).
    yAxis: [
      { ...axis, type: 'value' },
      { ...axis, type: 'value', position: 'right' },
    ],
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
      {
        type: 'line',
        name: labels.realCumulative,
        yAxisIndex: 1,
        data: realCumValues,
        symbol: 'none',
        lineStyle: { width: 1, color: realColor },
      },
      {
        type: 'line',
        name: labels.expectedCumulative,
        yAxisIndex: 1,
        data: expectedCumValues,
        symbol: 'none',
        lineStyle: { width: 1, color: expectedColor, type: 'dashed' },
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
  /** Locale d'affichage ('fr' | 'en') pour le formatage du KPI (séparateur décimal). */
  locale: string
}

export function TimeseriesFdaGapTrend({
  rows,
  title,
  emptyMessage,
  height = 320,
  labels,
  locale,
}: TimeseriesFdaGapTrendProps) {
  const hasExpectedStats = useCapability('expected_stats')
  const themeVersion = useThemeVersion()

  const option = useMemo<EChartsCoreOption | null>(
    () => buildFdaGapCumulativeOption(rows, labels),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [rows, labels, themeVersion],
  )

  // Écart moyen par match du joueur suivi (helper canonique meanFdaGap), sur les
  // matchs AVEC attendu uniquement (D3). null → « — » (aucun match exploitable).
  const meanGap = useMemo(
    () => meanFdaGap(rows.map((r) => ({ real: r.kda ?? null, expected: r.kda_expected ?? null }))),
    [rows],
  )

  // Titre sans attendu (ex. Halo 5) → masquage silencieux (pas de carte vide).
  if (!hasExpectedStats) return null

  const nf = new Intl.NumberFormat(locale, {
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
    signDisplay: 'always',
  })
  const footer = (
    <div className="flex items-center gap-2 border-t border-border px-3 py-2" data-testid="fda-gap-avg">
      <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        {labels.avgCaption}
      </span>
      <span className="tabular-nums text-xs font-semibold text-foreground">
        {meanGap == null ? '—' : `${nf.format(meanGap)}${labels.perMatch}`}
      </span>
    </div>
  )

  return (
    <ChartFromOption
      title={title}
      option={option}
      height={height}
      emptyMessage={emptyMessage}
      footer={footer}
    />
  )
}
