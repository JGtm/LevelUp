/**
 * habitChart.ts — « Contrôle des ressources, par soirée » (bloc « Soirées précédentes » de l'onglet
 * Emprise ; plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, décision D5) : des barres GROUPÉES,
 * un groupe par soirée, une barre par ressource. Sorti de  (seuil de 500 lignes) ;
 * mêmes briques de graphe (axe en %, colonne de ce soir, infobulle) et même calcul du pied d'axe.
 * Toutes les couleurs arrivent résolues (jetons).
 */
import type { EChartsCoreOption } from 'echarts/core'

import { CHART_BG, getTooltipBase } from '@/components/charts/_utils'
import { footLabelStyle, layoutAxisFoot, lineHeightFor, type AxisFootSpec, type FootLabels } from '@/components/charts/axisFoot'
import { resolveToken } from '@/lib/accessibility'
import { getEChartsThemeColors, type EChartsThemeColors } from '@/lib/echarts/themeColors'

import { tonightColumn, yAxisPct } from '../objectif/objectifCharts'
import { tipHtml } from './empriseCharts'
import type { HabitPoint } from './habit.logic'
import { resolveResourceColor } from './resourceColors'

/** Corps des dates sous l'axe. */
const LABEL_FONT = 10.5
const LABEL_LINE = lineHeightFor(LABEL_FONT)

export interface HabitChartColors {
  resource: (resource: string) => string
  parity: string
  /** Le gris des soirées hors comparaison (encre atténuée du thème). */
  muted: string
  theme: EChartsThemeColors
}

export interface HabitChartText {
  resourceLabel: (resource: string) => string
  /** Part avec une décimale au plus (infobulles). */
  pctFmt: (v: number) => string
  /** Part entière (valeur au bout des courbes, graduation 100 %). */
  pctIntFmt: (v: number) => string
  /** « 28/07 ». */
  dateOf: (iso: string) => string
  tonight: string
  eveningOf: (date: string) => string
  pointTip: (resource: string, evening: string, value: string, median: string | null) => string
  medianTip: (resource: string, value: string) => string
  /** La raison d'une soirée hors comparaison, ses familles de mode jointes (« Bases, Drapeau »). */
  notComparableTip: (families: string) => string
}

/** Marges fixes ; à droite, seulement la place des valeurs de ce soir (plus de valeur au bout d'une courbe). */
const HABIT_LEFT = 36
const HABIT_RIGHT = 14
const HABIT_TOP = 12

/** Le pied : une ligne de dates sous l'axe, sans bande. */
const HABIT_DATE_LABELS: FootLabels = { lines: 1, lineHeight: LABEL_LINE, margin: 8 }
const HABIT_FOOT: AxisFootSpec = { above: HABIT_DATE_LABELS, band: null, below: null, pad: 8 }

/** Un groupe de barres : place entre deux soirées et entre deux barres d'une même soirée. */
const HABIT_BAR_MAX_WIDTH = 18
const HABIT_BAR_GAP = '8%'
const HABIT_CATEGORY_GAP = '26%'

/**
 * Une série de barres par ressource : une soirée par groupe, la part de la ressource dans la
 * soirée. Une soirée HORS comparaison (aucune des familles de mode de ce soir) garde sa barre,
 * GRISÉE et liserée de la couleur de la ressource, la raison au survol, hors médiane. La médiane
 * des soirées précédentes comparables est un pointillé fin de la couleur de la ressource, le
 * trait 50 % est posé sur la première série. La valeur est écrite sur la barre de CE SOIR
 * seulement (les autres se lisent à l'infobulle) : sans part ce soir, aucune valeur n'est écrite.
 */
function habitSeries(
  points: HabitPoint[],
  resource: string,
  median: number | null,
  first: boolean,
  c: HabitChartColors,
  t: HabitChartText,
) {
  const color = c.resource(resource)
  const label = t.resourceLabel(resource)
  const med = median == null ? null : t.pctFmt(median)
  const eveningName = (p: HabitPoint) =>
    p.current ? t.tonight.charAt(0).toUpperCase() + t.tonight.slice(1) : t.eveningOf(t.dateOf(p.startTime))
  const data = points.map((p) => {
    const v = p.shares[resource]
    if (v == null) return null
    const tipBase = t.pointTip(label, eveningName(p), t.pctFmt(v), p.comparable ? med : null)
    return {
      value: v,
      itemStyle: p.comparable ? { color } : { color: c.muted, borderColor: color, borderWidth: 1.5 },
      tip: p.comparable ? tipBase : `${tipBase}\n${t.notComparableTip(p.families.join(', '))}`,
      ...(p.current
        ? {
            label: {
              show: true,
              position: 'top' as const,
              color,
              fontSize: 10.5,
              fontWeight: 700,
              formatter: () => t.pctIntFmt(v),
            },
          }
        : {}),
    }
  })
  return {
    name: label,
    type: 'bar' as const,
    xAxisIndex: 0,
    yAxisIndex: 0,
    z: 3,
    data,
    barMaxWidth: HABIT_BAR_MAX_WIDTH,
    barMinHeight: 2,
    barGap: HABIT_BAR_GAP,
    barCategoryGap: HABIT_CATEGORY_GAP,
    itemStyle: { color, borderRadius: [2, 2, 0, 0] },
    label: { show: false },
    labelLayout: { moveOverlap: 'shiftX' as const },
    markLine: {
      symbol: 'none',
      label: { show: false },
      data: [
        ...(median == null
          ? []
          : [{ yAxis: median, tip: t.medianTip(label, med ?? ''), lineStyle: { color, type: [2, 3], width: 1, opacity: 0.9 } }]),
        ...(first ? [{ yAxis: 50, silent: true, lineStyle: { color: c.parity, type: [4, 3], width: 1.5 } }] : []),
      ],
    },
  }
}

/**
 * buildHabitOption — « Contrôle des ressources, par soirée » : des barres GROUPÉES, un groupe par
 * soirée (une barre par ressource, couleurs `resource-*`), ce soir à droite dans une colonne
 * grisée avec ses valeurs écrites, une soirée hors comparaison en barres grises, la médiane des
 * soirées précédentes comparables en pointillé fin de la couleur de chaque ressource, le trait
 * 50 % ; sous l'axe, la date de chaque soirée, « ce soir » en gras.
 */
export function buildHabitOption(
  resources: string[],
  points: HabitPoint[],
  medians: Record<string, number | null>,
  c: HabitChartColors,
  t: HabitChartText,
): EChartsCoreOption {
  const tc = c.theme
  const foot = layoutAxisFoot(HABIT_FOOT)
  const categories = points.map((_, i) => String(i))
  return {
    backgroundColor: CHART_BG,
    animation: false,
    grid: { left: HABIT_LEFT, right: HABIT_RIGHT, top: HABIT_TOP, bottom: foot.gridBottom },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'item',
      formatter: (p: { data?: { tip?: string } }) => (p.data?.tip ? tipHtml(p.data.tip) : ''),
    },
    xAxis: {
      type: 'category',
      data: categories,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: {
        ...footLabelStyle(HABIT_DATE_LABELS),
        formatter: (_v: string, i: number) =>
          points[i]?.current ? `{cur|${t.tonight}}` : `{d|${t.dateOf(points[i]?.startTime ?? '')}}`,
        rich: {
          d: { color: tc.axisLabel, fontSize: LABEL_FONT },
          cur: { color: tc.text, fontSize: LABEL_FONT, fontWeight: 700 },
        },
      },
    },
    yAxis: yAxisPct(t.pctIntFmt, tc),
    series: [
      tonightColumn(points.length, tc.splitAreaB),
      ...resources.map((r, i) => habitSeries(points, r, medians[r] ?? null, i === 0, c, t)),
    ],
    legend: { show: false },
    aria: { enabled: true },
  }
}

/** Les couleurs du graphe d'habitude, résolues au rendu (thème, palette). */
export function resolveHabitColors(): HabitChartColors {
  const theme = getEChartsThemeColors()
  return { resource: resolveResourceColor, parity: resolveToken('warning'), muted: theme.axisLabel, theme }
}
