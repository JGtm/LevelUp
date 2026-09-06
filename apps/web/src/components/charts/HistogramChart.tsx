/**
 * HistogramChart — wrapper ECharts pour un histogramme depuis
 * `ChartSeries<ChartPointHistogram>[]`.
 *
 * Consomme :
 *   - 1 série dont les datapoints sont `{ binStart, binEnd, count }`.
 *   - Les buckets sont rendus comme des barres adjacentes (catégories).
 *
 * Cas d'usage Phase 3 :
 *   - Distributions Timeseries : K/D, kills/match, précision, score/min,
 *     win rate glissant.
 *
 * Le ChartCard parent gère les états loading/error/empty.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken, type SemanticToken } from '@/lib/accessibility'

import { ChartCard, type ChartSeries } from './ChartCard'
import { CHART_BG, getAxisBase, getEChartsThemeColors, getTooltipBase, seriesColor } from './_utils'

export interface ChartPointHistogram {
  binStart: number
  binEnd: number
  count: number
}

export interface HistogramChartProps {
  title?: string
  series: ChartSeries<ChartPointHistogram>[]
  loading?: boolean
  error?: Error | null
  emptyMessage?: string
  height?: number
  /** Token couleur pour la barre (default chart-series-1). */
  colorToken?: SemanticToken
  /** Libellé de l'axe X (ex. "K/D", "Kills / match"). */
  xAxisLabel?: string
  /** Libellé de l'axe Y (default = nb de matchs en FR). */
  yAxisLabel?: string
  /**
   * Format des bornes de bucket. Default : "binStart–binEnd" arrondi à 2
   * décimales si non-entier.
   */
  formatBin?: (point: ChartPointHistogram) => string
  /**
   * Barres ATTÉNUÉES : montrées, mais hors du périmètre que le graphe compte
   * (ajout 2026-09-06, distribution du délai d'échange de l'escouade).
   *
   * L'atténuation est une OPACITÉ + un liseré tireté sur la COULEUR DE SÉRIE, jamais
   * une seconde teinte, et c'est mesuré : aucun token sémantique du dépôt n'est
   * achromatique dans les QUATRE palettes d'accessibilité (`divergent-neutral` vaut
   * #60A5FA — blue-400 — dans la palette par défaut, et n'est gris que sous
   * okabe-ito / cividis / tol-bright). Prendre un token « neutre » aurait donc peint
   * ces barres en BLEU plus soutenu que la série qu'elles sont censées accompagner.
   * Une atténuation de la même couleur n'a, elle, aucune dépendance de palette : elle
   * ne peut pas devenir une seconde signification.
   *
   * POURQUOI PAS DEUX SÉRIES : ce wrapper ne peint que `series[0]` (une seconde série
   * serait ignorée EN SILENCE), et deux séries sur les mêmes catégories décaleraient
   * les barres.
   */
  binAttenuated?: (point: ChartPointHistogram, index: number) => boolean
}

export function HistogramChart({
  title,
  series,
  loading,
  error,
  emptyMessage,
  height,
  colorToken,
  xAxisLabel,
  yAxisLabel,
  formatBin,
  binAttenuated,
}: HistogramChartProps) {
  const buildOption = useCallback(
    (s: ChartSeries<ChartPointHistogram>[]) =>
      buildHistogramOption(s, { colorToken, xAxisLabel, yAxisLabel, formatBin, binAttenuated }),
    [colorToken, xAxisLabel, yAxisLabel, formatBin, binAttenuated],
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
  colorToken?: SemanticToken
  xAxisLabel?: string
  yAxisLabel?: string
  formatBin?: (point: ChartPointHistogram) => string
  binAttenuated?: (point: ChartPointHistogram, index: number) => boolean
}

/**
 * Opacité d'une barre ATTÉNUÉE. Assez basse pour se distinguer d'un coup d'œil d'une
 * barre pleine, assez haute pour rester lisible sur les deux thèmes.
 */
const ATTENUATION_OPACITE = 0.35

function defaultFormatBin(point: ChartPointHistogram): string {
  const fmt = (n: number): string => (Number.isInteger(n) ? String(n) : n.toFixed(2))
  return `${fmt(point.binStart)}–${fmt(point.binEnd)}`
}

/**
 * Pure builder — exporté pour tester l'option ECharts sans monter le React tree.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function buildHistogramOption(
  series: ChartSeries<ChartPointHistogram>[],
  opts: BuildOpts = {},
): EChartsCoreOption {
  const { colorToken, xAxisLabel, yAxisLabel: yLabelOpt, formatBin = defaultFormatBin } = opts
  const { binAttenuated } = opts
  const yAxisLabel = yLabelOpt ?? 'Matchs'
  if (series.length === 0) {
    return { backgroundColor: CHART_BG }
  }
  const main = series[0]
  const dps = main.datapoints
  if (dps.length === 0) {
    return { backgroundColor: CHART_BG }
  }

  const categories = dps.map((d) => formatBin(d))
  const color = colorToken ? resolveToken(colorToken) : seriesColor(0)
  // Une barre ne porte un style propre QUE si l'appelant la déclare atténuée : sans
  // `binAttenuated`, chaque valeur reste un nombre nu et ECharts applique la couleur
  // de série — le comportement historique, bit pour bit.
  const counts = dps.map((d, i) =>
    binAttenuated?.(d, i)
      ? {
          value: d.count,
          itemStyle: {
            color,
            opacity: ATTENUATION_OPACITE,
            borderColor: color,
            borderWidth: 1,
            borderType: 'dashed' as const,
            borderRadius: 2,
          },
        }
      : d.count,
  )

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 56, left: 48, right: 12 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: categories,
      name: xAxisLabel,
      nameLocation: 'middle',
      nameGap: 36,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      axisLabel: { ...axis.axisLabel, rotate: -30 },
    },
    yAxis: {
      ...axis,
      type: 'value',
      name: yAxisLabel,
      nameLocation: 'middle',
      nameGap: 32,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
    },
    series: [
      {
        type: 'bar',
        data: counts,
        barCategoryGap: '10%',
        itemStyle: { color, borderRadius: 2 },
      },
    ],
  }
}
