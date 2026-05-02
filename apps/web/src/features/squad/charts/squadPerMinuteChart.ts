/**
 * squadPerMinuteChart — teammates.14 : K/D/A par minute, bars groupées.
 *
 * Spec : .ai/charts_specs/teammates/14_per_minute_stats.yaml
 *
 *   - 3 catégories X : Frags/min, Morts/min, Assists/min.
 *   - 1 série bar par joueur (main + coéquipiers, 1..4 séries).
 *   - Frags & Assists au-dessus de l'axe X (positifs), Morts en dessous (négatifs).
 *   - Couleur normale du joueur pour Frags/Assists, couleur "négative" (opacity 0.45)
 *     pour les Morts → conserve l'identité visuelle tout en marquant la nature.
 *   - Axe zéro en blanc gras (zerolinewidth=2 ↔ ECharts splitLine[0] custom).
 *   - Pas de légende (le mapping joueur→couleur est dans la pill / combobox de la page).
 *   - Label sur barre = valeur ABSOLUE (frags positifs, morts négatifs vers le bas
 *     mais lus en valeur positive).
 */
import type { EChartsCoreOption } from 'echarts/core'
import { CHART_BG, axisBase, tooltipBase } from '@/components/charts/_utils'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { SquadPerMinuteEntry } from '@/lib/api/types'

const ZERO_LINE_COLOR = 'rgba(255, 255, 255, 0.75)'
const DEATHS_OPACITY = 0.45

export interface SquadPerMinuteOpts {
  /** Mapping gamertag → couleur hex. Doit couvrir tous les players de rows. */
  colorByPlayer: Record<string, string>
  metricLabels: { frags: string; deaths: string; assists: string }
  perMinuteSuffix: string // ex: " /min" pour tooltip
}

function fmt(v: number): string {
  return v.toFixed(2)
}

export function buildSquadPerMinuteOption(
  series: ChartSeries<SquadPerMinuteEntry>[],
  opts: SquadPerMinuteOpts,
): EChartsCoreOption {
  const rows = series[0]?.datapoints ?? []
  if (rows.length === 0) return { backgroundColor: CHART_BG }

  const xLabels = [opts.metricLabels.frags, opts.metricLabels.deaths, opts.metricLabels.assists]

  // 1 série bar par joueur, 3 valeurs (frags, -deaths, assists).
  const echSeries = rows.map((r) => {
    const color = opts.colorByPlayer[r.player] ?? '#888'
    return {
      name: r.player,
      type: 'bar' as const,
      data: [
        { value: r.kills_per_minute, itemStyle: { color } },
        { value: -r.deaths_per_minute, itemStyle: { color, opacity: DEATHS_OPACITY } },
        { value: r.assists_per_minute, itemStyle: { color } },
      ],
      barMaxWidth: 22,
      label: {
        show: true,
        position: 'top' as const,
        color: 'rgba(255,255,255,0.85)',
        fontSize: 10,
        formatter: (p: { value: unknown; dataIndex: number }) => {
          const v = typeof p.value === 'number' ? p.value : 0
          if (p.dataIndex === 1) return fmt(Math.abs(v))
          return fmt(v)
        },
      },
    }
  })

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 36, left: 8, right: 24, containLabel: true },
    tooltip: {
      ...tooltipBase,
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => {
        const arr = Array.isArray(params) ? params : []
        if (arr.length === 0) return ''
        const cat = (arr[0] as { name?: string }).name ?? ''
        const lines = arr.map((p) => {
          const point = p as { seriesName: string; value: number; color: string }
          const v = Math.abs(point.value)
          return `<span style="display:inline-block;width:8px;height:8px;border-radius:50%;background:${point.color};margin-right:6px"></span>${point.seriesName}: ${fmt(v)}${opts.perMinuteSuffix}`
        })
        return `<strong>${cat}</strong><br/>${lines.join('<br/>')}`
      },
    },
    xAxis: {
      ...axisBase,
      type: 'category',
      data: xLabels,
      axisLine: { lineStyle: { color: ZERO_LINE_COLOR, width: 2 } }, // axe zéro en gras blanc
    },
    yAxis: {
      ...axisBase,
      type: 'value',
      axisLabel: { ...axisBase.axisLabel, formatter: (v: number) => fmt(Math.abs(v)) },
    },
    series: echSeries,
  }
}
