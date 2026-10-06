/**
 * tendancesLines.logic — le builder PUR des courbes de la section « Évolution » : de courbes
 * déjà découpées par horizon à une option ECharts, sur le modèle de
 * `features/squad/charts/squadSessionTimelineChart` (MMR en pointillé sur un second axe).
 *
 * AXES. X de type `time`, borné à la fenêtre de l'horizon. Y de gauche à l'échelle des données,
 * étiquettes formatées par `formatTrendValue`. Un axe Y de droite (sans lignes de grille,
 * marge droite élargie) n'existe que si une courbe de droite est fournie : le MMR adverse.
 *
 * REPÈRES. `reference` (ex. 0 pour la FDA) se pose en pointillé sur la première courbe ;
 * `prevMean` de chaque courbe, en pointillé de SA couleur. Un repère disparaît avec sa courbe
 * quand la légende la masque.
 *
 * COULEURS. Les courbes portent un JETON (`CurveColor`), résolu ICI à chaque construction :
 * l'option suit donc le thème et la palette d'accessibilité (ChartCard reconstruit l'option à
 * chaque changement). Rien n'est résolu au chargement du module.
 */
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  LEGEND_ITEM_WIDTH_LINE,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getGridBase,
  getLegendBase,
  getTooltipBase,
  legendEntries,
  seriesColor,
} from '@/components/charts/_utils'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import { intlLocale } from '@/lib/formatters'
import type { TrendsPoint } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { formatTrendValue, type Step } from './tendances.logic'

/** Couleur « encre des étiquettes d'axe » : celle du thème, pas un jeton de palette. */
export const AXIS_INK = 'axis-ink'

/** Couleur d'une courbe : un jeton sémantique, ou l'encre des étiquettes d'axe. */
export type CurveColor = SemanticToken | typeof AXIS_INK

/** Une courbe de l'axe de gauche. */
export interface LinesCurve {
  name: string
  color: CurveColor
  points: readonly TrendsPoint[]
  /** Trait pointillé, sans symbole. */
  dashed?: boolean
  /** Moyenne de la période d'avant : repère horizontal de la couleur de la courbe. */
  prevMean?: number
}

/** La courbe de l'axe de droite : le MMR adverse (traitement fixe). */
export interface LinesRightCurve {
  name: string
  points: readonly TrendsPoint[]
}

/** Libellés déjà traduits dont l'infobulle a besoin. */
export interface LinesLabels {
  /** Une ligne d'infobulle : nom de la courbe, valeur formatée, nombre de matchs du point. */
  tooltipLine: (name: string, value: string, matches: number) => string
  /** En-tête d'un point du pas « semaine » : la date de début de semaine, déjà formatée. */
  weekOf: (date: string) => string
}

export interface TendancesLinesInput {
  curves: readonly LinesCurve[]
  right?: LinesRightCurve
  /** Repère horizontal posé sur la première courbe. */
  reference?: number
  /** Fenêtre de l'axe X, en millisecondes. */
  from: number
  to: number
  /** Unité et décimales de l'axe de gauche (celles de l'API). */
  unit: string
  decimals: number
  step: Step
  locale: Locale
  /** Fuseau d'affichage des dates de l'infobulle (celui de la réponse de l'API). */
  timeZone?: string
  labels: LinesLabels
}

/** Marge droite avec un second axe, en pixels (modèle de la frise d'escouade). */
const RIGHT_MARGIN_TWO_AXES = 60
/** Au-delà de cette fenêtre, les étiquettes de l'axe X portent mois et année plutôt que jour et mois. */
const LONG_WINDOW_DAYS = 120
const DAY_MS = 24 * 60 * 60 * 1000

interface TooltipParam {
  seriesIndex: number
  marker: string
  value: [number, number]
  data: { matches: number }
}

function pointData(points: readonly TrendsPoint[]) {
  return points.map((p) => ({ value: [new Date(p.t).getTime(), p.value], matches: p.matches }))
}

function resolveCurveColor(color: CurveColor, axisInk: string): string {
  return color === AXIS_INK ? axisInk : resolveToken(color)
}

/** Date d'un point selon le pas : l'instant pour un match, le jour, le début de semaine, le mois. */
function formatPointDate(
  t: number,
  step: Step,
  locale: Locale,
  timeZone: string | undefined,
  labels: LinesLabels,
): string {
  const loc = intlLocale(locale)
  const date = new Date(t)
  const fmt = (options: Intl.DateTimeFormatOptions) =>
    new Intl.DateTimeFormat(loc, { ...options, timeZone }).format(date)
  switch (step) {
    case 'match':
      return fmt({ dateStyle: 'medium', timeStyle: 'short' })
    case 'week':
      return labels.weekOf(fmt({ dateStyle: 'medium' }))
    case 'month':
      return fmt({ month: 'long', year: 'numeric' })
    default:
      return fmt({ dateStyle: 'medium' })
  }
}

function markLineOf(
  curve: LinesCurve,
  index: number,
  input: TendancesLinesInput,
  colors: { curve: string; axisInk: string },
) {
  const data: object[] = []
  if (index === 0 && input.reference != null) {
    data.push({ yAxis: input.reference, lineStyle: { color: colors.axisInk } })
  }
  if (curve.prevMean != null) {
    data.push({ yAxis: curve.prevMean, lineStyle: { color: colors.curve } })
  }
  if (data.length === 0) return undefined
  return {
    silent: true,
    symbol: 'none',
    lineStyle: { type: 'dashed', width: 1 },
    label: { show: false },
    data,
  }
}

function leftSeries(
  curve: LinesCurve,
  index: number,
  input: TendancesLinesInput,
  axisInk: string,
): object {
  const color = resolveCurveColor(curve.color, axisInk)
  const markLine = markLineOf(curve, index, input, { curve: color, axisInk })
  return {
    name: curve.name,
    type: 'line',
    yAxisIndex: 0,
    smooth: false,
    data: pointData(curve.points),
    symbol: curve.dashed ? 'none' : 'circle',
    symbolSize: 6,
    lineStyle: { width: 2, color, ...(curve.dashed ? { type: 'dashed' } : {}) },
    itemStyle: { color },
    ...(markLine ? { markLine } : {}),
  }
}

/** La courbe de droite : pointillé, losange, `seriesColor(4)`, second axe. */
function rightSeries(right: LinesRightCurve): object {
  const color = seriesColor(4)
  return {
    name: right.name,
    type: 'line',
    yAxisIndex: 1,
    smooth: false,
    data: pointData(right.points),
    lineStyle: { width: 2, type: 'dotted', color },
    itemStyle: { color },
    symbol: 'diamond',
    symbolSize: 7,
  }
}

function xAxisLabelFormatter(input: TendancesLinesInput) {
  const loc = intlLocale(input.locale)
  const long = input.to - input.from > LONG_WINDOW_DAYS * DAY_MS
  const options: Intl.DateTimeFormatOptions = long
    ? { month: 'short', year: '2-digit', timeZone: input.timeZone }
    : { day: 'numeric', month: 'short', timeZone: input.timeZone }
  const format = new Intl.DateTimeFormat(loc, options)
  return (value: number) => format.format(new Date(value))
}

function tooltipFormatter(input: TendancesLinesInput) {
  const nbLeft = input.curves.length
  return (raw: unknown): string => {
    const items = (Array.isArray(raw) ? raw : [raw]) as TooltipParam[]
    if (items.length === 0) return ''
    const header = formatPointDate(
      items[0].value[0],
      input.step,
      input.locale,
      input.timeZone,
      input.labels,
    )
    const lines = items.map((p) => {
      const isRight = input.right != null && p.seriesIndex === nbLeft
      const name = isRight ? input.right!.name : input.curves[p.seriesIndex]?.name ?? ''
      const value = isRight
        ? formatTrendValue(p.value[1], 'number', 0, input.locale)
        : formatTrendValue(p.value[1], input.unit, input.decimals, input.locale)
      return `${p.marker}${escapeHtml(input.labels.tooltipLine(name, value, p.data.matches))}`
    })
    return [`<b>${escapeHtml(header)}</b>`, ...lines].join('<br/>')
  }
}

/**
 * Construit l'option ECharts des courbes d'un graphique d'« Évolution ». Sans courbe :
 * option vide (le cadre affiche alors son état vide).
 */
export function buildTendancesLinesOption(input: TendancesLinesInput): EChartsCoreOption {
  if (input.curves.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const hasRight = input.right != null
  const colors = input.curves.map((c) => resolveCurveColor(c.color, tc.axisLabel))
  const showLegend = input.curves.length + (hasRight ? 1 : 0) > 1

  const entries = input.curves.map((c, i) => ({ name: c.name, color: colors[i], dashed: c.dashed }))
  if (input.right) entries.push({ name: input.right.name, color: seriesColor(4), dashed: true })

  const yAxis: object[] = [
    {
      ...axis,
      type: 'value',
      scale: true,
      axisLabel: {
        ...axis.axisLabel,
        formatter: (value: number) =>
          formatTrendValue(value, input.unit, input.decimals, input.locale),
      },
    },
  ]
  if (hasRight) {
    yAxis.push({
      ...axis,
      type: 'value',
      scale: true,
      splitLine: { show: false },
      axisLabel: {
        ...axis.axisLabel,
        formatter: (value: number) => formatTrendValue(value, 'number', 0, input.locale),
      },
    })
  }

  const series: object[] = input.curves.map((c, i) => leftSeries(c, i, input, tc.axisLabel))
  if (input.right) series.push(rightSeries(input.right))

  return {
    backgroundColor: CHART_BG,
    grid: getGridBase({
      top: 16,
      bottom: showLegend ? 48 : 16,
      right: hasRight ? RIGHT_MARGIN_TWO_AXES : 16,
    }),
    tooltip: { ...getTooltipBase(tc), trigger: 'axis', formatter: tooltipFormatter(input) },
    ...(showLegend
      ? {
          legend: {
            ...getLegendBase(tc),
            left: 'center',
            ...(entries.some((e) => e.dashed) ? { itemWidth: LEGEND_ITEM_WIDTH_LINE } : {}),
            data: legendEntries(entries),
          },
        }
      : {}),
    xAxis: {
      ...axis,
      type: 'time',
      min: input.from,
      max: input.to,
      axisLabel: { ...axis.axisLabel, hideOverlap: true, formatter: xAxisLabelFormatter(input) },
    },
    yAxis,
    series,
  }
}
