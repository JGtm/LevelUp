/**
 * TimeseriesEngagementGapTrend — « Écart d'engagement cumulé » (onglet
 * Progression, adjacent à EngagementTimeseriesSection).
 *
 * Cumul signé du résidu d'engagement pondéré par la durée : par match,
 * contribution = (pace_joueur − pace_attendu) × (duration_seconds / 60), en
 * ÉVÉNEMENTS. Rendu aligné sur TimeseriesFdaGapTrend : aire signée divergente
 * ancrée à 0 (helper canonique `divergentZeroGradient`) + markLine 0. Cumul via
 * les helpers canoniques `cumulativeSigned` / `engagementGapEvents`.
 *
 * Réutilise la MÊME query `useEngagementTimeseries` que EngagementTimeseriesSection
 * (dédup par le cache TanStack Query — pas de second fetch). Report D5 : un point
 * sans résidu ou sans durée ne fait pas avancer le cumul.
 */
import { useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import {
  getEChartsThemeColors,
  getAxisBase,
  getTooltipBase,
  CHART_BG,
  escapeHtml,
} from '@/components/charts/_utils'
import { cumulativeSigned } from '@/lib/charts/cumulativeSeries'
import { engagementGapEvents } from '@/lib/charts/engagementGap'
import { divergentZeroGradient } from '@/lib/charts/divergentZeroGradient'
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'
import { useEngagementTimeseries } from '@/features/engagement/queries'
import { truncateMap } from '@/lib/charts/matchLabels'
import type { EngagementGranularity, EngagementMatchSummaryAPI, FilterContextInput } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { engagementManifest } from '@/lib/i18n/generated/engagement'
import { useAppShellStore } from '@/stores/appShellStore'
import { ChartFromOption } from './ChartFromOption'

interface GapLabels {
  series: string
  axis: string
}

/** Étiquette X d'un point (match → `#N\nMap` ; agrégat → label brut). */
function pointLabel(m: EngagementMatchSummaryAPI, i: number, granularity: EngagementGranularity): string {
  if (granularity === 'match') {
    return m.map_name ? `#${i + 1}\n${truncateMap(m.map_name)}` : `#${i + 1}`
  }
  return m.label
}

function buildEngagementGapOption(
  points: EngagementMatchSummaryAPI[],
  granularity: EngagementGranularity,
  labels: GapLabels,
): EChartsCoreOption | null {
  if (points.length === 0) return null
  // Ordre du service conservé (déjà trié ASC) — pas de re-tri.
  const cum = cumulativeSigned(
    points.map((m) => engagementGapEvents(m.pace_joueur - m.pace_attendu, m.duration_seconds)),
  )
  const values = cum.map((p) => p.cumulative)
  const categories = points.map((m, i) => pointLabel(m, i, granularity))
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const interval = points.length > 30 ? Math.floor(points.length / 12) : 0
  const gradient = divergentZeroGradient(values)
  const fmt = (v: number) => (v >= 0 ? `+${Math.round(v)}` : `${Math.round(v)}`)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 64, left: 48, right: 24 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      formatter: (params: Array<{ dataIndex?: number }>) => {
        if (!Array.isArray(params) || params.length === 0) return ''
        const idx = params[0]?.dataIndex ?? 0
        const cat = escapeHtml((categories[idx] ?? '').replace(/\n/g, ' · '))
        return (
          `<strong>${cat}</strong><br/>` +
          `${escapeHtml(labels.series)}: <b>${fmt(values[idx])}</b> ${escapeHtml(labels.axis)}`
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
        lineStyle: { width: 2, color: gradient },
        areaStyle: { color: gradient, opacity: 0.18, origin: 0 },
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

export interface TimeseriesEngagementGapTrendProps {
  playerSlug: string
  filters: FilterContextInput
  filterHash: string
  limit?: number
  height?: number
}

export function TimeseriesEngagementGapTrend({
  playerSlug,
  filters,
  filterHash,
  limit = 30,
  height = 320,
}: TimeseriesEngagementGapTrendProps) {
  const locale = useAppShellStore((s) => s.locale)
  const themeVersion = useThemeVersion()
  const query = useEngagementTimeseries(playerSlug, filters, filterHash, limit)

  const pointsAPI = useMemo<EngagementMatchSummaryAPI[]>(() => query.data?.points ?? [], [query.data?.points])
  const granularity: EngagementGranularity = (() => {
    const g = query.data?.granularity
    return g === 'session' || g === 'week' || g === 'month' ? g : 'match'
  })()

  const labels: GapLabels = {
    series: formatMessage(engagementManifest, 'engagement.cumulative_gap.series', locale),
    axis: formatMessage(engagementManifest, 'engagement.cumulative_gap.axis', locale),
  }

  const option = useMemo<EChartsCoreOption | null>(
    () => buildEngagementGapOption(pointsAPI, granularity, labels),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [pointsAPI, granularity, themeVersion, locale],
  )

  return (
    <ChartFromOption
      title={formatMessage(engagementManifest, 'engagement.cumulative_gap.title', locale)}
      option={option}
      height={height}
      emptyMessage={formatMessage(engagementManifest, 'engagement.cumulative_gap.empty', locale)}
    />
  )
}
