/**
 * RadarChart — wrapper ECharts pour le radar 6 axes.
 *
 * Ne consomme PAS `ChartSeries<T>` standard car le payload backend est
 * spécifique (cf. service.RadarChartSeries Go). Chaque trace = 1 joueur,
 * 6 axes alignés avec valeurs 0..100 normalisées.
 *
 * Cas d'usage Squad V2 :
 *   - S8 Radar Participation 6 axes (Combat/Survival/Support/Score/Objective/Impact)
 *
 * Les axes sont passés par le caller (le service amont fournit la liste
 * dans l'ordre canonique). Les valeurs raw sont disponibles dans
 * `meta.raw_by_axis` pour le tooltip.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard } from './ChartCard'
import { CHART_BG, legendBase, seriesColor, tooltipBase } from './_utils'

/** 1 axe radar : libellé + valeur 0..100 + raw debug. */
export interface RadarAxis {
  axis: string
  value: number
  raw: number
}

/** 1 série radar (1 joueur). Mirror service.RadarChartSeries Go. */
export interface RadarSeriesPayload {
  key: string
  labelKey?: string
  axes: RadarAxis[]
  meta?: { gamertag?: string; mode_family?: string; raw_by_axis?: Record<string, number> }
}

export interface RadarChartProps {
  title?: string
  series: RadarSeriesPayload[]
  loading?: boolean
  error?: Error | null
  emptyMessage?: string
  height?: number
  /** Si fournie, override le label de série (par défaut meta.gamertag ou key). */
  seriesNameResolver?: (s: RadarSeriesPayload) => string
  /** Map axis key → label affiché (i18n du caller). Si absent, utilise axis brut. */
  axisLabels?: Record<string, string>
}

export function RadarChart({
  title,
  series,
  loading,
  error,
  emptyMessage,
  height,
  seriesNameResolver,
  axisLabels,
}: RadarChartProps) {
  // Le ChartCard est typé sur ChartSeries<T> mais on passe RadarSeriesPayload.
  // Ce wrapper est volontairement non-strict (cast au call site) pour
  // garder le ChartCard minimal — le radar a une structure trop spécifique.
  const buildOption = useCallback(
    (s: RadarSeriesPayload[]) =>
      buildRadarOption(s, { seriesNameResolver, axisLabels }),
    [seriesNameResolver, axisLabels],
  )

  return (
    <ChartCard
      title={title}
      series={series as unknown as { key: string; datapoints: unknown[] }[]}
      loading={loading}
      error={error}
      emptyMessage={emptyMessage}
      height={height}
      buildOption={(_s) => buildOption(series)}
    />
  )
}

interface BuildOpts {
  seriesNameResolver?: (s: RadarSeriesPayload) => string
  axisLabels?: Record<string, string>
}

export function buildRadarOption(
  series: RadarSeriesPayload[],
  opts: BuildOpts = {},
): EChartsCoreOption {
  const { seriesNameResolver, axisLabels } = opts
  if (series.length === 0) {
    return { backgroundColor: CHART_BG }
  }

  // Axes : du premier joueur (tous alignés côté backend).
  const axes = series[0].axes.map((a) => ({
    name: axisLabels?.[a.axis] ?? a.axis,
    max: 100,
  }))

  const data = series.map((s, idx) => {
    const name =
      seriesNameResolver?.(s) ?? (s.meta?.gamertag ?? s.key)
    return {
      name,
      value: s.axes.map((a) => a.value),
      itemStyle: { color: seriesColor(idx) },
      lineStyle: { color: seriesColor(idx), width: 2 },
      areaStyle: { color: seriesColor(idx), opacity: 0.15 },
    }
  })

  return {
    backgroundColor: CHART_BG,
    tooltip: {
      ...tooltipBase,
      formatter: (params: { name: string; value: number[] }) => {
        const lines = axes.map(
          (a, i) => `${a.name}: <b>${params.value[i].toFixed(0)}</b>`,
        )
        return `<b>${params.name}</b><br/>${lines.join('<br/>')}`
      },
    },
    legend: {
      ...legendBase,
      data: data.map((d) => d.name),
    },
    radar: {
      indicator: axes,
      shape: 'polygon',
      splitNumber: 4,
      axisName: { color: 'rgba(255,255,255,0.65)', fontSize: 10 },
      splitArea: { areaStyle: { color: ['rgba(255,255,255,0.02)', 'rgba(255,255,255,0.05)'] } },
      splitLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } },
      axisLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } },
    },
    series: [
      {
        type: 'radar',
        data,
      },
    ],
  }
}
