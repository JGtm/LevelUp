/**
 * TimeseriesPerMinuteTrend — timeseries.14, « Stats par minute » : une colonne par match avec les
 * frags, les morts (sous l'axe zéro) et les assistances par minute, plus, quand les matchs portent
 * des dégâts, deux COURBES (dégâts infligés et subis par minute) sur un axe vertical secondaire à
 * droite, dans la même grille. Les deux axes partagent la ligne zéro.
 */
import { useMemo, type ReactNode } from 'react'
import type { ChartOption } from '@/components/charts/chartOption'

import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getGridBase,
  getLegendBase,
  getTooltipBase,
} from '@/components/charts/_utils'
import { axisTooltipFormatter } from '@/components/charts/axisTooltip'
import { tipNumber } from '@/components/charts/tipNumber'
import { resolveToken } from '@/lib/accessibility'
import { alignedZeroAxes } from '@/lib/charts/alignedZeroAxes'
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'
import type { TimeseriesMatchRow } from '@/lib/api/types'

import { ChartFromOption } from './ChartFromOption'
import { buildMatchCategories } from './matchLabels'
import { matchPace, paceHasDamage } from './timeseriesPerMinute.logic'

export interface PerMinuteTrendLabels {
  kills: string
  deaths: string
  assists: string
  damageDealt: string
  damageTaken: string
  /** Nom de l'axe de droite (dégâts par minute). */
  damageAxis: string
  /** Suffixe des valeurs de l'infobulle (« /min »). */
  suffix: string
}

/** Rang des courbes de dégâts dans le tableau de séries (après les trois colonnes). */
const DAMAGE_SERIES_FROM = 3

const round = (v: number | null, decimals: number) => (v == null ? null : Number(v.toFixed(decimals)))

/** Plus grande valeur d'une liste de cadences (les absentes comptent pour rien). */
const peak = (values: (number | null)[]) => Math.max(...values.map((v) => v ?? 0), 0)

/** Étiquettes de l'axe de droite : jamais de valeur négative (la partie sous le zéro est vide). */
const damageAxisLabel = (v: number) => (v < 0 ? '' : tipNumber(v, 0))

// eslint-disable-next-line react-refresh/only-export-components
export function buildPerMinuteTrendOption(
  rows: TimeseriesMatchRow[],
  labels: PerMinuteTrendLabels,
  withTaken: boolean,
): ChartOption | null {
  if (rows.length === 0) return null
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const paces = rows.map((r) => matchPace(r, withTaken))
  const hasDamage = paceHasDamage(rows, withTaken)

  const kills = paces.map((p) => round(p.kills, 2))
  // Les morts sont tracées sous l'axe zéro (négatives), comme sur Sessions et Escouade.
  const deaths = paces.map((p) => (p.deaths == null ? null : -Number(p.deaths.toFixed(2))))
  const assists = paces.map((p) => round(p.assists, 2))
  const dealt = paces.map((p) => round(p.damageDealt, 0))
  const taken = paces.map((p) => round(p.damageTaken, 0))

  // Zéro commun : l'axe de droite (dégâts, tous positifs) prend la part sous le zéro de l'axe de gauche.
  const aligned = hasDamage
    ? alignedZeroAxes(
        { neg: peak(paces.map((p) => p.deaths)), pos: Math.max(peak(paces.map((p) => p.kills)), peak(paces.map((p) => p.assists))) },
        { neg: 0, pos: Math.max(peak(dealt), peak(taken)) },
      )
    : null

  const colDealt = resolveToken('divergent-pos')
  const colTaken = resolveToken('divergent-neg')
  const bar = (name: string, data: (number | null)[], token: Parameters<typeof resolveToken>[0]) => ({
    type: 'bar' as const,
    name,
    data,
    itemStyle: { color: resolveToken(token), opacity: 0.85 },
    barMaxWidth: 14,
  })
  const line = (name: string, data: (number | null)[], color: string) => ({
    type: 'line' as const,
    name,
    data,
    yAxisIndex: 1,
    symbol: 'circle' as const,
    symbolSize: 5,
    connectNulls: false, // un match sans valeur laisse un trou, pas un zéro
    lineStyle: { color, width: 1.5 },
    itemStyle: { color },
  })

  return {
    backgroundColor: CHART_BG,
    grid: getGridBase({ top: 24, ...(hasDamage ? { right: 44 } : {}) }),
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      // Les morts sont tracées sous l'axe (négatives) : la bulle écrit leur valeur absolue ; les dégâts
      // sont des centaines, sans décimale.
      formatter: axisTooltipFormatter({
        value: (r) => `${tipNumber(Math.abs(r.value), r.seriesIndex >= DAMAGE_SERIES_FROM ? 0 : 2)}${labels.suffix}`,
      }),
    },
    legend: { ...getLegendBase(tc), bottom: 0 },
    xAxis: {
      ...axis,
      type: 'category',
      data: buildMatchCategories(rows),
      axisLine: { lineStyle: { color: tc.text, width: 1.5 } },
      axisLabel: { ...axis.axisLabel, interval: 0, fontSize: 9 },
    },
    yAxis: [
      {
        ...axis,
        type: 'value',
        ...(aligned ? { min: aligned.left[0], max: aligned.left[1] } : {}),
        axisLabel: { ...axis.axisLabel, formatter: (v: number) => tipNumber(Math.abs(v), 1) },
      },
      ...(aligned
        ? [
            {
              ...axis,
              type: 'value' as const,
              position: 'right' as const,
              min: aligned.right[0],
              max: aligned.right[1],
              name: labels.damageAxis,
              nameTextStyle: { color: tc.axisLabel, fontSize: 10, align: 'right' as const },
              splitLine: { show: false }, // la grille est celle de l'axe de gauche
              axisLabel: { ...axis.axisLabel, formatter: damageAxisLabel },
            },
          ]
        : []),
    ],
    series: [
      bar(labels.kills, kills, 'stat-kills'),
      bar(labels.deaths, deaths, 'stat-deaths'),
      bar(labels.assists, assists, 'stat-assists'),
      ...(hasDamage ? [line(labels.damageDealt, dealt, colDealt)] : []),
      ...(hasDamage && withTaken ? [line(labels.damageTaken, taken, colTaken)] : []),
    ],
  }
}

export interface TimeseriesPerMinuteTrendProps {
  rows: TimeseriesMatchRow[]
  height?: number
  title?: ReactNode
  emptyMessage?: string
  labels: PerMinuteTrendLabels
  /** Le titre fournit-il les dégâts subis ? Faux (Halo 5) : la courbe « subis » est retirée. */
  withTaken: boolean
}

export function TimeseriesPerMinuteTrend({
  rows,
  title,
  emptyMessage,
  height = 360,
  labels,
  withTaken,
}: TimeseriesPerMinuteTrendProps) {
  const themeVersion = useThemeVersion()
  // eslint-disable-next-line react-hooks/exhaustive-deps -- le thème change les couleurs résolues dans l'option
  const option = useMemo(() => buildPerMinuteTrendOption(rows, labels, withTaken), [rows, labels, withTaken, themeVersion])
  return <ChartFromOption title={title} option={option} height={height} emptyMessage={emptyMessage} />
}
