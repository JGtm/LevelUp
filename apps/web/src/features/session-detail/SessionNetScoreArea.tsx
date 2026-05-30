/**
 * SessionNetScoreArea — évolution du NET SCORE CUMULÉ (Frags − Morts) match après match.
 *
 * Courbe en aire simple : axe X catégoriel "#N + carte", axe Y = solde cumulé, ligne de
 * référence à 0. Une instance par session (vue single + drawer côte-à-côte).
 *
 * IMPLÉMENTATION VOLONTAIREMENT MINIMALE (le chart restait vide via des montages
 * antérieurs) : données 1D (un scalaire par catégorie, ECharts aligne par index) +
 * couleur de ligne/aire EXPLICITE. PAS de visualMap : il colorait la courbe par le
 * signe du cumul mais, sans couleur propre sur la série, la moindre défaillance du
 * mapping rendait la courbe invisible. La séparation positif/négatif reste lisible via
 * la markLine à 0.
 */
import { useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, getAxisBase, getEChartsThemeColors, getTooltipBase } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import type { SessionDetailMatchRow } from '@/lib/api/types'

import { sessionMatchAxisLabel, useSessionT } from './_shared'

interface NetPoint {
  label: string
  cumulative: number
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildSessionNetScoreOption(
  series: ChartSeries<NetPoint>[],
  opts: { seriesLabel: string },
): EChartsCoreOption {
  const points = series[0]?.datapoints ?? []
  if (points.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const lineColor = resolveToken('chart-series-1')
  const interval = points.length > 30 ? Math.floor(points.length / 12) : 0

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 64, left: 48, right: 24 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      formatter: (params: Array<{ name: string; value: number; marker: string }>) => {
        if (!Array.isArray(params) || params.length === 0) return ''
        const p = params[0]
        const v = Number(p.value)
        const vStr = v >= 0 ? `+${v}` : `${v}`
        return `${p.name.replace('\n', ' · ')}<br/>${p.marker} ${opts.seriesLabel}: <b>${vStr}</b>`
      },
    },
    xAxis: {
      ...axis,
      type: 'category',
      boundaryGap: false,
      data: points.map((p) => p.label),
      axisLabel: { ...(axis.axisLabel as Record<string, unknown>), interval },
    },
    yAxis: { ...axis, type: 'value' },
    series: [
      {
        name: opts.seriesLabel,
        type: 'line',
        // Données 1D : un scalaire par catégorie (ECharts aligne par index). Couleur
        // de ligne + aire EXPLICITE pour garantir la visibilité.
        data: points.map((p) => p.cumulative),
        symbol: 'none',
        lineStyle: { width: 2, color: lineColor },
        areaStyle: { color: lineColor, opacity: 0.18 },
        // Ligne de référence à 0 (séparation solde positif / négatif).
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
}

export function SessionNetScoreArea({ title, matches, height = 280 }: Props) {
  const t = useSessionT()

  const series = useMemo<ChartSeries<NetPoint>[]>(() => {
    const sorted = [...matches].sort((a, b) => a.start_time.localeCompare(b.start_time))
    if (sorted.length === 0) return []
    let running = 0
    const datapoints = sorted.map((m, i) => {
      // Garde-fou : un kills/deaths manquant ne doit pas propager un NaN dans le cumul.
      running += (m.kills ?? 0) - (m.deaths ?? 0)
      return { label: sessionMatchAxisLabel(i, m.map_name, m.pair_name), cumulative: running }
    })
    return [{ key: 'net', datapoints }]
  }, [matches])

  return (
    <ChartCard
      title={title}
      series={series}
      height={height}
      buildOption={(s) => buildSessionNetScoreOption(s, { seriesLabel: t('session.detail.net_score_series') })}
    />
  )
}
