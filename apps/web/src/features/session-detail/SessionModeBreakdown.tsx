/**
 * SessionModeBreakdown — répartition des modes joués sur la session, en bâtons verticaux.
 *
 * Agrégation front : compte des matchs par mode (mode_ui normalisé/traduit, fallback
 * pair_name). Tri décroissant. 1 barre par mode, couleurs de série pour distinction.
 */
import { useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, escapeHtml, getAxisBase, getEChartsThemeColors, getTooltipBase, seriesColor } from '@/components/charts/_utils'
import type { SessionDetailMatchRow } from '@/lib/api/types'

import { useSessionT } from './_shared'

interface ModePoint {
  mode: string
  count: number
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildSessionModeBreakdownOption(
  series: ChartSeries<ModePoint>[],
  opts: { countLabel: string; yMax?: number },
): EChartsCoreOption {
  const points = series[0]?.datapoints ?? []
  if (points.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  // Rotation des étiquettes si beaucoup de modes (évite le chevauchement).
  const rotate = points.length > 5 ? 30 : 0

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: rotate ? 64 : 36, left: 40, right: 16, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => {
        const arr = Array.isArray(params) ? params : []
        if (arr.length === 0) return ''
        const p = arr[0] as { name: string; value: number }
        return `${escapeHtml(p.name)}: <b>${p.value}</b> ${opts.countLabel}`
      },
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: points.map((p) => p.mode),
      axisLabel: { ...(axis.axisLabel as Record<string, unknown>), interval: 0, rotate },
    },
    // max figé en mode comparaison (compte partagé A/B) → hauteurs de barres comparables.
    yAxis: { ...axis, type: 'value', minInterval: 1, ...(opts.yMax != null ? { max: opts.yMax } : {}) },
    series: [
      {
        type: 'bar',
        data: points.map((p, i) => ({ value: p.count, itemStyle: { color: seriesColor(i) } })),
        barMaxWidth: 48,
        label: { show: true, position: 'top', color: tc.text, formatter: (p: { value: number }) => String(p.value) },
      },
    ],
  }
}

interface Props {
  title: string
  matches: SessionDetailMatchRow[]
  height?: number
  /** Max du compte (axe Y) partagé A/B en mode comparaison (sinon auto-scale). */
  yMax?: number
}

export function SessionModeBreakdown({ title, matches, height = 260, yMax }: Props) {
  const t = useSessionT()

  const series = useMemo<ChartSeries<ModePoint>[]>(() => {
    const counts = new Map<string, number>()
    for (const m of matches) {
      const mode = (m.mode_ui || m.pair_name || '—').trim()
      counts.set(mode, (counts.get(mode) ?? 0) + 1)
    }
    if (counts.size === 0) return []
    const datapoints = [...counts.entries()]
      .map(([mode, count]) => ({ mode, count }))
      .sort((a, b) => b.count - a.count)
    return [{ key: 'modes', datapoints }]
  }, [matches])

  return (
    <ChartCard
      title={title}
      series={series}
      height={height}
      buildOption={(s) => buildSessionModeBreakdownOption(s, { countLabel: t('session.detail.mode_breakdown_count'), yMax })}
    />
  )
}
