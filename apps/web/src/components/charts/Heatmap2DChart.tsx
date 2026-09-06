/**
 * Heatmap2DChart — wrapper ECharts pour `ChartSeries<ChartPointHeatmap>`.
 *
 * Consomme :
 *   - 1 série dont les datapoints sont `{ x, y, value, detail? }`.
 *   - Axes X et Y sont des chaînes (déjà résolues côté service en labels
 *     lisibles).
 *
 * Cas d'usage Squad V2 :
 *   - S3 Heatmap player × map (perf score)
 *   - S6 Intensity match × bucket
 *   - S5 Impact heatmap match × player (potentiel future)
 *
 * Le `value` est mappé via `visualMap` ECharts en gradient cold→hot ou
 * divergent low→high selon `paletteMode`.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken } from '@/lib/accessibility'
import { useSettingsDraftStore, type ColorPalette } from '@/stores/settingsDraftStore'

import { ChartCard, type ChartSeries } from './ChartCard'
import { heatmapRampTokens } from './heatmapColors'
import { CHART_BG, escapeHtml, getAxisBase, getEChartsThemeColors, getTooltipBase } from './_utils'

export interface ChartPointHeatmap {
  x: string
  y: string
  value: number
  detail?: Record<string, unknown>
}

export type HeatmapPaletteMode = 'sequential' | 'divergent'

export interface Heatmap2DChartProps {
  title?: string
  series: ChartSeries<ChartPointHeatmap>[]
  loading?: boolean
  error?: Error | null
  emptyMessage?: string
  height?: number
  /** sequential (cold→hot) ou divergent (low→neutral→high). Default 'sequential'. */
  paletteMode?: HeatmapPaletteMode
  /** Min/max forcés du visualMap (default = auto-fit). */
  valueRange?: [number, number]
}

export function Heatmap2DChart({
  title,
  series,
  loading,
  error,
  emptyMessage,
  height,
  paletteMode = 'sequential',
  valueRange,
}: Heatmap2DChartProps) {
  // Palette d'accessibilité active : pilote la rampe CVD-safe (rebuild via
  // useColorPaletteVersion dans ChartCard + ce sélecteur au changement de palette).
  const colorPalette = useSettingsDraftStore((s) => s.localUiPrefs.colorPalette)
  const buildOption = useCallback(
    (s: ChartSeries<ChartPointHeatmap>[]) =>
      buildHeatmap2DOption(s, { paletteMode, valueRange, colorPalette }),
    [paletteMode, valueRange, colorPalette],
  )

  return (
    <ChartCard
      title={title}
      series={series}
      loading={loading}
      error={error}
      emptyMessage={emptyMessage}
      height={height}
      buildOption={buildOption}
    />
  )
}

interface BuildOpts {
  paletteMode?: HeatmapPaletteMode
  valueRange?: [number, number]
  /** Palette d'accessibilité active — pilote la rampe CVD-safe (cf. heatmapColors). */
  colorPalette?: ColorPalette
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildHeatmap2DOption(
  series: ChartSeries<ChartPointHeatmap>[],
  opts: BuildOpts = {},
): EChartsCoreOption {
  const { paletteMode = 'sequential', valueRange, colorPalette = 'default' } = opts
  if (series.length === 0) {
    return { backgroundColor: CHART_BG }
  }
  const main = series[0]
  const dps = main.datapoints

  // Axes X / Y déduits des datapoints (preserve l'ordre d'apparition).
  const xs: string[] = []
  const ys: string[] = []
  const xSet = new Set<string>()
  const ySet = new Set<string>()
  for (const d of dps) {
    if (!xSet.has(d.x)) {
      xSet.add(d.x)
      xs.push(d.x)
    }
    if (!ySet.has(d.y)) {
      ySet.add(d.y)
      ys.push(d.y)
    }
  }

  // ECharts heatmap data : [xIndex, yIndex, value, detail]
  const data = dps.map((d) => [xs.indexOf(d.x), ys.indexOf(d.y), d.value, d.detail])

  const minV = valueRange?.[0] ?? Math.min(...dps.map((d) => d.value))
  const maxV = valueRange?.[1] ?? Math.max(...dps.map((d) => d.value))

  // Rampe centralisée : en palette d'accessibilité, une heatmap séquentielle
  // bascule sur la rampe de fréquence (luminance monotone, CVD-safe).
  const colors = heatmapRampTokens(paletteMode, colorPalette).map(resolveToken)

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 80, left: 96, right: 24 },
    tooltip: {
      ...getTooltipBase(tc),
      position: 'top',
      formatter: (params: { data: [number, number, number, Record<string, unknown>?] }) => {
        const [xi, yi, v, detail] = params.data
        const count = detail?.count ?? 0
        return `${escapeHtml(ys[yi])} × ${escapeHtml(xs[xi])}<br/>Win Rate: <b>${(v * 100).toFixed(1)}%</b><br/>Matchs: <b>${count}</b>`
      },
    },
    xAxis: { ...axis, type: 'category', data: xs, splitArea: { show: true } },
    yAxis: { ...axis, type: 'category', data: ys, splitArea: { show: true } },
    visualMap: {
      min: minV,
      max: maxV,
      calculable: true,
      orient: 'horizontal',
      left: 'center',
      bottom: 8,
      inRange: { color: colors },
      textStyle: { color: tc.axisLabel, fontSize: 10 },
    },
    series: [
      {
        name: main.key,
        type: 'heatmap',
        data,
        label: {
          show: true,
          formatter: (params: { data: [number, number, number, Record<string, unknown>?] }) => {
            const [, , , detail] = params.data
            return String(detail?.count ?? 0)
          },
        },
        emphasis: {
          itemStyle: { shadowBlur: 8, shadowColor: 'rgba(0,0,0,0.5)' }, // color-allow: 2026-09-06 (revue R1, C5) — voile NEUTRE d ombre/fond d infobulle ECharts, pas une couleur de charte ; dette PREEXISTANTE au lot v2 D, a porter sur un token le jour ou un token de voile existera
        },
      },
    ],
  }
}
