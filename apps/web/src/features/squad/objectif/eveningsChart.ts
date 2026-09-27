/**
 * eveningsChart.ts — « Rapport de force, soirée après soirée » (section « Objectif » de
 * Contributions, lot L3 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26) : trois courbes (une
 * soirée par point), ce soir à droite dans une colonne grisée, la médiane des soirées précédentes
 * en pointillé fin de la couleur de chaque courbe, le trait 50 %, et sous chaque soirée : la date,
 * la barre victoires / défaites, « x sur y » et le mélange de modes.
 *
 * Sorti de `objectifCharts.ts` (seuil de 500 lignes) à la revue L6.1 ; mêmes briques (axe en %,
 * courbe, colonne de ce soir). Toutes les couleurs arrivent résolues (jetons).
 */
import type { EChartsCoreOption } from 'echarts/core'

import { CHART_BG, getTooltipBase } from '@/components/charts/_utils'
import { withEndPoint } from '@/components/charts/endPoint'
import type { EChartsThemeColors } from '@/lib/echarts/themeColors'

import { OBJECTIVE_ROLES, type ObjectiveRole } from '../formes/model/objectives'
import type { EveningPoint } from './objectif.logic'
import { lineSeries, tip, tonightColumn, yAxisPct, type CustomApi, type ObjectifChartColors } from './objectifCharts'

export interface EveningsChartText {
  roles: Record<ObjectiveRole, string>
  pctFmt: (v: number, digits?: number) => string
  tonight: string
  dateOf: (iso: string) => string
  outOfFmt: (wins: number, matches: number) => string
  mixOf: (p: EveningPoint) => string
  pointTip: (role: string, evening: string, value: string, median: string | null) => string
  bandTip: (evening: string, wins: number, matches: number, mix: string) => string
  eveningOf: (date: string) => string
  medianTip: (role: string, value: string) => string
}

const EVENING_FOOT = 64

/** Le nom d'une soirée dans les infobulles : « ce soir », « Soirée du 28/07 ». */
type EveningNamer = (p: EveningPoint) => string

/**
 * La courbe d'un rôle : une soirée par point, ce soir (la dernière colonne) grossi ; la médiane
 * des soirées précédentes en pointillé fin de la couleur du rôle ; le trait 50 % sur la première.
 */
function roleEveningSeries(
  points: EveningPoint[],
  role: ObjectiveRole,
  medians: Record<ObjectiveRole, number | null>,
  first: boolean,
  ctx: { c: ObjectifChartColors; t: EveningsChartText; eveningName: EveningNamer },
) {
  const { c, t, eveningName } = ctx
  const color = c.roles[role]
  const med = medians[role]
  const data = points.map((p) => {
    const v = p.shares[role]
    if (v == null) return null
    return {
      value: v,
      symbol: 'circle',
      symbolSize: 6,
      itemStyle: { color, borderColor: c.theme.card, borderWidth: 1.5 },
      tip: t.pointTip(t.roles[role], eveningName(p), t.pctFmt(v, 1), med == null ? null : t.pctFmt(med, 1)),
    }
  })
  const extra: Record<string, unknown> = {
    symbol: 'circle',
    markLine: {
      symbol: 'none',
      label: { show: false },
      lineStyle: { color, type: [2, 3] as number[], width: 1 },
      data: [
        ...(med == null ? [] : [{ yAxis: med, tip: t.medianTip(t.roles[role], t.pctFmt(med, 1)) }]),
        ...(first ? [{ yAxis: 50, lineStyle: { color: c.parity, type: [4, 3], width: 1.5 }, silent: true }] : []),
      ],
    },
  }
  // Ce soir = la soirée affichée, jamais le dernier point non nul : un rôle sans part ce soir n'a
  // ni point grossi ni valeur au bout (ECharts l'écrirait sur une soirée passée — constat R15).
  const at = points.findIndex((p) => p.current)
  const base = lineSeries(t.roles[role], withEndPoint(data, c.theme.card, { at, size: 10, borderWidth: 1.5 }), color, (v) => t.pctFmt(v), extra)
  return { ...base, endLabel: { ...base.endLabel, show: at >= 0 && data[at] != null } }
}

/** Sous chaque soirée : la barre victoires / défaites. */
function eveningsBandSeries(points: EveningPoint[], c: ObjectifChartColors, t: EveningsChartText, eveningName: EveningNamer) {
  return {
    type: 'custom' as const,
    xAxisIndex: 1,
    yAxisIndex: 1,
    data: points.map((p, i) => ({ value: [i, 0], tip: t.bandTip(eveningName(p), p.wins, p.matches, t.mixOf(p)) })),
    renderItem: (_params: unknown, api: CustomApi) => {
      const i = api.value(0) as number
      const p = points[i]
      const [cx, cy] = api.coord([i, 0])
      const bw = (api.size([1, 0]) as number[])[0] - 6
      const wv = p.matches > 0 ? (p.wins / p.matches) * bw : 0
      const x0 = cx - bw / 2
      const h = 10
      const children: unknown[] = []
      if (p.wins > 0) children.push({ type: 'rect', shape: { x: x0, y: cy - h / 2, width: wv, height: h, r: 2 }, style: { fill: c.win } })
      if (p.wins < p.matches) {
        children.push({ type: 'rect', shape: { x: x0 + wv, y: cy - h / 2, width: bw - wv, height: h, r: 2 }, style: { fill: c.loss } })
      }
      return { type: 'group', children }
    },
  }
}

/** Les deux abscisses : la date (« ce soir » en gras), puis « x sur y » et le mélange de modes. */
function eveningsXAxes(points: EveningPoint[], categories: string[], tc: EChartsThemeColors, t: EveningsChartText) {
  return [
    {
      gridIndex: 0,
      type: 'category',
      data: categories,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: {
        interval: 0,
        margin: 4,
        formatter: (_v: string, i: number) =>
          points[i]?.current ? `{cur|${t.tonight}}` : `{d|${t.dateOf(points[i]?.startTime ?? '')}}`,
        rich: {
          d: { color: tc.axisLabel, fontSize: 10.5 },
          cur: { color: tc.text, fontSize: 10.5, fontWeight: 600 },
        },
      },
    },
    {
      gridIndex: 1,
      type: 'category',
      data: categories,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: {
        interval: 0,
        margin: 6,
        lineHeight: 13,
        formatter: (_v: string, i: number) =>
          `{n|${t.outOfFmt(points[i]?.wins ?? 0, points[i]?.matches ?? 0)}}\n{m|${points[i] ? t.mixOf(points[i]) : ''}}`,
        rich: {
          n: { color: tc.text, fontSize: 9.5 },
          m: { color: tc.axisLabel, fontSize: 9.5 },
        },
      },
    },
  ]
}

export function buildEveningsOption(
  points: EveningPoint[],
  medians: Record<ObjectiveRole, number | null>,
  c: ObjectifChartColors,
  t: EveningsChartText,
): EChartsCoreOption {
  const tc = c.theme
  const categories = points.map((_, i) => String(i))
  const eveningName = (p: EveningPoint) => (p.current ? t.tonight : t.eveningOf(t.dateOf(p.startTime)))
  const series: unknown[] = [
    ...OBJECTIVE_ROLES.map((role, ri) => roleEveningSeries(points, role, medians, ri === 0, { c, t, eveningName })),
    tonightColumn(points.length, tc.splitAreaB),
    eveningsBandSeries(points, c, t, eveningName),
  ]

  return {
    backgroundColor: CHART_BG,
    animation: false,
    grid: [
      { left: 40, right: 52, top: 12, bottom: EVENING_FOOT },
      { left: 40, right: 52, bottom: EVENING_FOOT - 8 - 10, height: 10 },
    ],
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'item',
      formatter: (p: { data?: { tip?: string } }) => (p.data?.tip ? tip(p.data.tip) : ''),
    },
    xAxis: eveningsXAxes(points, categories, tc, t),
    yAxis: [yAxisPct((v) => t.pctFmt(v), tc), { gridIndex: 1, type: 'value', min: -1, max: 1, show: false }],
    series,
    legend: { show: false },
    aria: { enabled: true },
  }
}
