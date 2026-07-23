/**
 * SessionEngagementCumulative — « Écart d'engagement cumulé » sur les matchs
 * d'une session (P4). Somme CUMULÉE de la contribution `engagement_score ×
 * (duration_seconds / 60)` (en ÉVÉNEMENTS), match après match dans l'ordre
 * chronologique.
 *
 * `engagement_score` (SessionMatchPoint) est déjà un résidu de rythme
 * (évén./min) ; pondéré par la durée, il donne un nombre d'événements en excès
 * (positif) ou déficit (négatif) vs l'attendu. Report D5 (délégué à
 * `cumulativeSigned` / `engagementGapEvents`) : un match sans score ou sans durée
 * ne fait pas avancer le cumul.
 *
 * Même pattern visuel que `SessionFdaGapCumulative` : aire signée divergente
 * ancrée à 0 (`divergentZeroGradient`) + markLine 0. Zip `entry.match_series`
 * (trié par index) ↔ `matches` (triés par start_time), comme SessionEngagementChart.
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
import { cumulativeSigned } from '@/lib/charts/cumulativeSeries'
import { engagementGapEvents } from '@/lib/charts/engagementGap'
import { divergentZeroGradient } from '@/lib/charts/divergentZeroGradient'
import type { SessionCompareEntry, SessionDetailMatchRow } from '@/lib/api/types'

import { sessionMatchAxisLabel, useSessionT } from './_shared'

export interface EngagementGapPoint {
  /** Étiquette d'axe X du match (`#N` + carte/mode). */
  label: string
  /** Écart du match en événements (null si score/durée absent — D5). */
  value: number | null
  /** Écart cumulé (reporte la dernière valeur si le match n'a pas de score). */
  cumulative: number
}

/**
 * Cumul de l'écart d'engagement sur les matchs. Zip `match_series` (trié par
 * index) ↔ `matches` (triés par start_time) pour aligner score et durée, puis
 * délégué au helper générique `cumulativeSigned`.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function computeCumulativeEngagementGap(
  matches: SessionDetailMatchRow[],
  entry: SessionCompareEntry | null,
): EngagementGapPoint[] {
  const ms = [...(entry?.match_series ?? [])].sort((a, b) => a.index - b.index)
  if (ms.length === 0) return []
  const sorted = [...matches].sort((a, b) => a.start_time.localeCompare(b.start_time))
  const cum = cumulativeSigned(
    ms.map((p, i) => engagementGapEvents(p.engagement_score ?? null, sorted[i]?.duration_seconds ?? null)),
  )
  return ms.map((_, i) => ({
    label: sessionMatchAxisLabel(i, sorted[i]?.map_name, sorted[i]?.pair_name),
    value: cum[i].value,
    cumulative: cum[i].cumulative,
  }))
}

export interface EngagementGapLabels {
  seriesLabel: string
  matchLabel: string
  axisLabel: string
  yDomain?: [number, number]
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildSessionEngagementGapOption(
  series: ChartSeries<EngagementGapPoint>[],
  opts: EngagementGapLabels,
): EChartsCoreOption {
  const points = series[0]?.datapoints ?? []
  if (points.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const interval = points.length > 30 ? Math.floor(points.length / 12) : 0

  const values = points.map((p) => p.cumulative)
  const divergentColor = divergentZeroGradient(values)
  const fmt = (v: number | null, signed = false) =>
    v == null ? '—' : signed && v >= 0 ? `+${Math.round(v)}` : `${Math.round(v)}`

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
          `${escapeHtml(opts.seriesLabel)}: <b>${fmt(p.cumulative, true)}</b> ${escapeHtml(opts.axisLabel)}<br/>` +
          `${escapeHtml(opts.matchLabel)}: ${fmt(p.value, true)}`
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
        lineStyle: { width: 2, color: divergentColor },
        areaStyle: { color: divergentColor, opacity: 0.18, origin: 0 },
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
  entry: SessionCompareEntry | null
  height?: number
  /** Domaine Y [min, max] partagé A/B en mode comparaison (sinon auto-scale). */
  yDomain?: [number, number]
}

export function SessionEngagementCumulative({ title, matches, entry, height = 280, yDomain }: Props) {
  const t = useSessionT()

  const series = useMemo<ChartSeries<EngagementGapPoint>[]>(() => {
    const points = computeCumulativeEngagementGap(matches, entry)
    return points.length === 0 ? [] : [{ key: 'engagement_gap', datapoints: points }]
  }, [matches, entry])

  return (
    <ChartCard
      title={title}
      series={series}
      height={height}
      buildOption={(s) =>
        buildSessionEngagementGapOption(s, {
          seriesLabel: t('session.detail.engagement_cumulative_series'),
          matchLabel: t('session.detail.engagement_cumulative_match'),
          axisLabel: t('session.detail.engagement_cumulative_axis'),
          yDomain,
        })
      }
    />
  )
}
