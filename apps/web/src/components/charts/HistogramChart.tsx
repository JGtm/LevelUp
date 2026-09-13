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
   * L'atténuation est une OPACITÉ sur la COULEUR DE SÉRIE — UN seul indice visuel, et
   * c'est délibéré (correction R2 du 2026-09-06 : la version précédente ajoutait un
   * liseré tireté que PERSONNE ne voyait, sa couleur étant celle du remplissage et
   * l'opacité de 0,35 s'appliquant à l'élément entier ; la doc promettait deux indices,
   * l'écran n'en montrait qu'un). Le SECOND indice n'est pas graphique : c'est le mot,
   * porté par l'étiquette d'axe et le pied de carte de l'appelant.
   *
   * Jamais une seconde teinte, et c'est mesuré : aucun token sémantique du dépôt n'est
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
  /**
   * SEUILS TRACÉS EN POINTILLÉ sur l'axe des catégories (ajout 2026-09-13, distribution
   * des distances à l'équipier de l'onglet Tactique).
   *
   * `at` est une position en INDICE DE CATÉGORIE, fractionnaire : un seuil de 18 m sur
   * des intervalles de 10 m tombe à 1,8 — entre la deuxième et la troisième barre, là où
   * il est vraiment. L'arrondir à une frontière de barre déplacerait la règle du jeu.
   *
   * PLUSIEURS SEUILS SONT LE CAS NORMAL : un filtre qui mélange Arène (18 m) et BTB
   * (24 m) mélange deux règles du jeu, et leur moyenne n'est la règle d'aucun match.
   */
  thresholds?: readonly { at: number; label: string }[]
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
  thresholds,
}: HistogramChartProps) {
  const buildOption = useCallback(
    (s: ChartSeries<ChartPointHistogram>[]) =>
      buildHistogramOption(s, {
        colorToken,
        xAxisLabel,
        yAxisLabel,
        formatBin,
        binAttenuated,
        thresholds,
      }),
    [colorToken, xAxisLabel, yAxisLabel, formatBin, binAttenuated, thresholds],
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
  thresholds?: readonly { at: number; label: string }[]
}

/**
 * Opacité d'une barre ATTÉNUÉE — le SEUL indice graphique de l'atténuation. Assez basse
 * pour se distinguer d'un coup d'œil d'une barre pleine, assez haute pour rester lisible
 * sur les deux thèmes.
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
  const { binAttenuated, thresholds } = opts
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
          itemStyle: { color, opacity: ATTENUATION_OPACITE },
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
        ...(thresholds && thresholds.length > 0 ? { markLine: markLineSeuils(thresholds, tc) } : {}),
      },
    ],
  }
}

/** Les seuils en pointillé — jeton `warning`, étiquette en haut de la ligne. */
function markLineSeuils(
  thresholds: readonly { at: number; label: string }[],
  tc: ReturnType<typeof getEChartsThemeColors>,
): Record<string, unknown> {
  return {
    silent: true,
    symbol: 'none',
    lineStyle: { color: resolveToken('warning'), type: 'dashed', width: 1.5 },
    label: {
      show: true,
      position: 'insideEndTop',
      color: resolveToken('warning'),
      fontSize: 10,
      backgroundColor: tc.tooltipBg,
      padding: [1, 3],
      formatter: (p: { name?: string }) => p.name ?? '',
    },
    data: thresholds.map((t) => ({ name: t.label, xAxis: t.at })),
  }
}
