/**
 * SessionNetScoreArea — évolution du NET SCORE CUMULÉ (Frags − Morts) match après match.
 *
 * Courbe en aire, base 0, remplissage divergent (vert au-dessus de 0 = solde positif,
 * rouge en dessous = solde négatif) via visualMap. Axe X = "#N + carte" comme les autres
 * graphes chronologiques. Une instance par session (vue single + drawer côte-à-côte).
 *
 * IMPORTANT : les données de la série sont des PAIRES [index, cumul] (2D), pas des
 * scalaires. Avec des scalaires sur un axe catégoriel, `visualMap.dimension: 1` pointe
 * une dimension hors-portée → le mapping de couleur échoue silencieusement → la ligne
 * (qui n'a pas de couleur propre, elle vient du visualMap) devient INVISIBLE. Les paires
 * lèvent l'ambiguïté (dim 0 = x, dim 1 = cumul). Une couleur de repli explicite garantit
 * en plus la visibilité même si le visualMap venait à ne pas s'appliquer.
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
  const posColor = resolveToken('divergent-pos')
  const negColor = resolveToken('divergent-neg')
  const interval = points.length > 30 ? Math.floor(points.length / 12) : 0

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 64, left: 48, right: 24 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      formatter: (params: Array<{ name: string; value: number[]; marker: string }>) => {
        if (!Array.isArray(params) || params.length === 0) return ''
        const p = params[0]
        // value est désormais une paire [index, cumul] → on lit le cumul (dim 1).
        const v = Array.isArray(p.value) ? p.value[1] : Number(p.value)
        const vStr = v >= 0 ? `+${v}` : `${v}`
        return `${p.name.replace('\n', ' · ')}<br/>${p.marker} ${opts.seriesLabel}: <b>${vStr}</b>`
      },
    },
    // Colore la courbe + l'aire selon le signe du cumul (vert > 0, rouge ≤ 0).
    // dimension: 1 = le cumul (dim 0 = l'index x), non ambigu grâce aux paires.
    visualMap: {
      show: false,
      type: 'piecewise',
      seriesIndex: 0,
      dimension: 1,
      pieces: [
        { gt: 0, color: posColor },
        { lte: 0, color: negColor },
      ],
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
        // Paires [index, cumul] : dim 1 explicite pour le visualMap (cf. en-tête).
        data: points.map((p, i) => [i, p.cumulative]),
        symbol: 'none',
        // Couleur de repli : si le visualMap ne s'applique pas, la ligne reste visible.
        lineStyle: { width: 2, color: posColor },
        areaStyle: { origin: 'auto', opacity: 0.25, color: posColor },
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
      running += m.kills - m.deaths
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
