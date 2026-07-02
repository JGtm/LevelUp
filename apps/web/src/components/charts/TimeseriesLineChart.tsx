/**
 * TimeseriesLineChart — wrapper ECharts pour `ChartSeries<ChartPoint2D>[]`
 * en mode "1 trace par série, X = chronologie".
 *
 * Cas d'usage Squad V2 :
 *   - S4 Timeline performance multi-joueurs (1 trace/joueur, marker outcome)
 *   - S4 Form Score (1 trace lissée LOWESS)
 *   - S7 5 charts trio (assists, KDA, accuracy, avg life, performance)
 *   - S7 Killing spree max lissé
 *
 * Le composant accepte plusieurs `ChartSeries<ChartPoint2D>` ; chaque série
 * devient une trace ECharts. Les couleurs sont résolues via tokens
 * (chart-series-1..8 cyclées).
 *
 * Le `Label` d'un ChartPoint2D peut porter un outcome ("win"/"loss"/"tie"/
 * "dnf") — dans ce cas, le marker est coloré selon l'outcome.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken, type SemanticToken } from '@/lib/accessibility'

import { ChartCard, type ChartSeries } from './ChartCard'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
  outcomeColor,
  seriesColor,
} from './_utils'

export interface ChartPoint2D {
  x: string | number | Date
  y: number
  label?: string
}

export interface TimeseriesLineChartProps {
  title?: string
  series: ChartSeries<ChartPoint2D>[]
  loading?: boolean
  error?: Error | null
  emptyMessage?: string
  height?: number
  /**
   * Si true, X est traité comme un axe temporel (datetime). Default true.
   *
   * Deprecated: utiliser xAxisType pour un contrôle plus fin.
   */
  timeAxis?: boolean
  /**
   * Type d'axe X explicite. Si fourni, prend précédence sur timeAxis.
   *
   *  - 'time'     : datetime (ECharts type='time')
   *  - 'category' : étiquettes discrètes
   *  - 'value'    : numérique continu (secondes, scores, etc.)
   */
  xAxisType?: 'time' | 'category' | 'value'
  /** Si true, marker outcome utilise outcomeColor. Default true. */
  outcomeMarkers?: boolean
  /** Optionnel : nom d'une série côté front (override LabelKey backend). */
  seriesNameResolver?: (s: ChartSeries<ChartPoint2D>) => string
  /** Si false, masque les symboles (courbes lisses sans marqueurs). Default true. */
  showSymbol?: boolean
  /** Si true, applique un lissage spline sur les courbes. Default false. */
  smooth?: boolean
  /**
   * Formatter optionnel pour les labels de l'axe X (utile en xAxisType='value'
   * pour afficher des secondes en `m:ss`). Reçoit la valeur brute du tick.
   */
  xAxisLabelFormatter?: (value: number | string | Date) => string
  /**
   * Override hex couleur par série (priorité absolue : > `colorToken` > cycle).
   * Reçoit la série + son index ; retourner `undefined` pour fallback.
   * Utile quand l'appelant a déjà résolu les couleurs via tokens et veut
   * éviter la double résolution + le risque de fallback sur la palette
   * cyclée par défaut quand un token CSS n'est pas chargé.
   */
  seriesColorResolver?: (s: ChartSeries<ChartPoint2D>, idx: number) => string | undefined
  /** Rotation (degrés) des étiquettes de l'axe X. Default : aucune. */
  xAxisLabelRotate?: number
  /** ECharts `axisLabel.interval` pour l'axe X (0 = toutes les étiquettes). */
  xAxisLabelInterval?: number
}

export function TimeseriesLineChart({
  title,
  series,
  loading,
  error,
  emptyMessage,
  height,
  timeAxis = true,
  xAxisType,
  outcomeMarkers = true,
  seriesNameResolver,
  showSymbol = true,
  smooth = false,
  xAxisLabelFormatter,
  seriesColorResolver,
  xAxisLabelRotate,
  xAxisLabelInterval,
}: TimeseriesLineChartProps) {
  const buildOption = useCallback(
    (s: ChartSeries<ChartPoint2D>[]) =>
      buildTimeseriesLineOption(s, {
        timeAxis,
        xAxisType,
        outcomeMarkers,
        seriesNameResolver,
        showSymbol,
        smooth,
        xAxisLabelFormatter,
        seriesColorResolver,
        xAxisLabelRotate,
        xAxisLabelInterval,
      }),
    [
      timeAxis,
      xAxisType,
      outcomeMarkers,
      seriesNameResolver,
      showSymbol,
      smooth,
      xAxisLabelFormatter,
      seriesColorResolver,
      xAxisLabelRotate,
      xAxisLabelInterval,
    ],
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
  timeAxis?: boolean
  xAxisType?: 'time' | 'category' | 'value'
  outcomeMarkers?: boolean
  seriesNameResolver?: (s: ChartSeries<ChartPoint2D>) => string
  showSymbol?: boolean
  smooth?: boolean
  xAxisLabelFormatter?: (value: number | string | Date) => string
  seriesColorResolver?: (s: ChartSeries<ChartPoint2D>, idx: number) => string | undefined
  xAxisLabelRotate?: number
  xAxisLabelInterval?: number
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildTimeseriesLineOption(
  series: ChartSeries<ChartPoint2D>[],
  opts: BuildOpts = {},
): EChartsCoreOption {
  const {
    timeAxis = true,
    xAxisType,
    outcomeMarkers = true,
    seriesNameResolver,
    showSymbol = true,
    smooth = false,
    xAxisLabelFormatter,
    seriesColorResolver,
    xAxisLabelRotate,
    xAxisLabelInterval,
  } = opts
  if (series.length === 0) {
    return { backgroundColor: CHART_BG }
  }

  // xAxisType prend précédence sur timeAxis (legacy).
  const resolvedAxisType: 'time' | 'category' | 'value' =
    xAxisType ?? (timeAxis ? 'time' : 'category')

  const echartsSeries = series.map((s, idx) => {
    // Priorité de résolution :
    //  1. seriesColorResolver(s, idx) → hex pré-résolu par l'appelant
    //     (privilégié quand le caller utilise déjà une palette dédiée :
    //      évite que `resolveToken` retombe sur '' si le token n'est pas chargé,
    //      ce qui poussait ECharts à appliquer son palette interne — bleu).
    //  2. s.colorToken → résolution via tokens à la volée.
    //  3. seriesColor(idx) → cycle chart-series-1..8.
    const explicitHex = seriesColorResolver?.(s, idx)
    const color =
      explicitHex && explicitHex.length > 0
        ? explicitHex
        : s.colorToken
        ? resolveToken(s.colorToken as SemanticToken) || seriesColor(idx)
        : seriesColor(idx)
    const name =
      seriesNameResolver?.(s) ??
      (s.meta && typeof s.meta.gamertag === 'string' ? s.meta.gamertag : s.key)

    const data = s.datapoints.map((dp) => {
      const x =
        resolvedAxisType === 'time' && !(dp.x instanceof Date)
          ? new Date(dp.x as string)
          : dp.x
      const itemStyle =
        outcomeMarkers && dp.label
          ? { color: outcomeColor(dp.label) }
          : undefined
      return {
        value: [x, dp.y] as [string | number | Date, number],
        itemStyle,
      }
    })

    return {
      name,
      type: 'line' as const,
      smooth,
      showSymbol,
      symbolSize: 6,
      lineStyle: { color, width: 1.5 },
      itemStyle: { color },
      data,
    }
  })

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const axisLabel: Record<string, unknown> = { ...(axis.axisLabel as Record<string, unknown>) }
  if (xAxisLabelFormatter) axisLabel.formatter = xAxisLabelFormatter
  if (xAxisLabelRotate != null) axisLabel.rotate = xAxisLabelRotate
  if (xAxisLabelInterval != null) axisLabel.interval = xAxisLabelInterval
  const xAxis: Record<string, unknown> = { ...axis, type: resolvedAxisType, axisLabel }

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 56, left: 56, right: 16 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: {
        type: 'cross',
        ...(xAxisLabelFormatter ? { label: { formatter: ({ value }: { value: unknown }) => xAxisLabelFormatter(value as number) } } : {}),
      },
      ...(xAxisLabelFormatter ? {
        formatter: (params: Array<{ axisValue: number | string; marker: string; seriesName: string; value: [number, number] }>) => {
          if (!Array.isArray(params) || !params.length) return ''
          const rows = params.map(p => `${p.marker} ${escapeHtml(p.seriesName)}: <b>${p.value[1]}</b>`).join('<br/>')
          return `${xAxisLabelFormatter(params[0].axisValue)}<br/>${rows}`
        },
      } : {}),
    },
    legend: { ...getLegendBase(tc), data: echartsSeries.map((s) => s.name) },
    xAxis,
    yAxis: { ...axis, type: 'value' },
    series: echartsSeries,
  }
}
