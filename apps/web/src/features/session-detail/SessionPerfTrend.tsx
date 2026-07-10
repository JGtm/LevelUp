/**
 * SessionPerfTrend — "Score de performance" : barres du performance_score par match,
 * colorées par TIER de grading (perf-tier-1..5), avec une ligne de MOYENNE (markLine)
 * dans la couleur de grading de la moyenne + une étiquette voyante à droite.
 *
 * Graphe custom (le wrapper TimeseriesLineChart est ligne-only) : ChartCard +
 * buildOption ECharts. Axe X catégoriel "#N\nCarte" (façon page Escouade), aligné
 * sur le graphe FDA. Couleurs via `resolveToken` (valeurs hex pour le canvas ECharts).
 */
import { useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, escapeHtml, getAxisBase, getEChartsThemeColors, getTooltipBase } from '@/components/charts/_utils'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import type { SessionDetailMatchRow } from '@/lib/api/types'

import { perfTierFromScore, sessionMatchAxisLabel, useSessionT } from './_shared'

interface Props {
  title: string
  matches: SessionDetailMatchRow[]
  height?: number
}

/** Point par match : étiquette d'axe + score + tier (1..5) pour la couleur de barre. */
interface PerfPoint {
  label: string
  score: number
  tier: number
}

interface PerfOpts {
  scoreLabel: string
  meanLabel: string
}

const round1 = (n: number) => Math.round(n * 10) / 10

// eslint-disable-next-line react-refresh/only-export-components
export function buildSessionPerfOption(series: ChartSeries<PerfPoint>[], opts: PerfOpts): EChartsCoreOption {
  const points = series[0]?.datapoints ?? []
  if (points.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  const mean = round1(points.reduce((acc, p) => acc + p.score, 0) / points.length)
  // Moyenne colorée par SA propre couleur de grading (cf. demande utilisateur) :
  // distincte des barres et indique directement le tier moyen de la session.
  const meanColor = resolveToken(`perf-tier-${perfTierFromScore(mean)}` as SemanticToken)

  const barData = points.map((p) => ({
    value: round1(p.score),
    itemStyle: { color: resolveToken(`perf-tier-${p.tier}` as SemanticToken) },
  }))

  const interval = points.length > 30 ? Math.floor(points.length / 12) : 0

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 64, left: 48, right: 72 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      formatter: (params: Array<{ name: string; value: number; marker: string }>) => {
        if (!Array.isArray(params) || params.length === 0) return ''
        const p = params[0]
        return `${escapeHtml(p.name.replace('\n', ' · '))}<br/>${p.marker} ${opts.scoreLabel}: <b>${p.value}</b>`
      },
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: points.map((p) => p.label),
      axisLabel: { ...(axis.axisLabel as Record<string, unknown>), interval },
    },
    // Échelle FIXE 0..100 : le score de perf est sur 100 → max toujours affiché, et A vs B
    // restent directement comparables (mêmes bornes des deux côtés).
    yAxis: { ...axis, type: 'value', min: 0, max: 100 },
    series: [
      {
        name: opts.scoreLabel,
        type: 'bar',
        data: barData,
        barMaxWidth: 28,
        markLine: {
          silent: true,
          symbol: 'none',
          lineStyle: { color: meanColor, type: 'dashed', width: 2 },
          label: {
            show: true,
            position: 'end',
            formatter: `${opts.meanLabel} ${mean}`,
            color: meanColor,
            fontWeight: 'bold',
            fontSize: 12,
            // Halo couleur fond pour rester lisible par-dessus les barres.
            textBorderColor: CHART_BG,
            textBorderWidth: 3,
          },
          data: [{ yAxis: mean }],
        },
      },
    ],
  }
}

export function SessionPerfTrend({ title, matches, height = 280 }: Props) {
  const t = useSessionT()

  const series = useMemo<ChartSeries<PerfPoint>[]>(() => {
    const sorted = [...matches]
      .filter((m) => m.performance_score != null)
      .sort((a, b) => a.start_time.localeCompare(b.start_time))
    if (sorted.length === 0) return []
    return [
      {
        key: 'perf',
        datapoints: sorted.map((m, i) => ({
          label: sessionMatchAxisLabel(i, m.map_name, m.pair_name),
          score: m.performance_score as number,
          tier: m.perf_tier ?? perfTierFromScore(m.performance_score as number),
        })),
      },
    ]
  }, [matches])

  return (
    <ChartCard
      title={title}
      series={series}
      height={height}
      buildOption={(s) =>
        buildSessionPerfOption(s, {
          scoreLabel: t('session.detail.chart_perf_series'),
          meanLabel: t('session.detail.chart_perf_mean'),
        })
      }
    />
  )
}
