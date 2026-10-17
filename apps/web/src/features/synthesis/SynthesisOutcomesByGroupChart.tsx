/**
 * SynthesisOutcomesByGroupChart — synthesis.01 (par carte) et synthesis.02 (par mode).
 * Stacked bar vertical avec labels count à l'intérieur de chaque segment.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'
import { barLabel } from '@/components/charts/barLabel'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
} from '@/components/charts/_utils'
import { axisTooltipFormatter, stackTopFirstOrder } from '@/components/charts/axisTooltip'
import { tipNumberAuto } from '@/components/charts/tipNumber'
import { resolveToken } from '@/lib/accessibility'
import { useFieldMappings, useOutcomeLabel } from '@/lib/i18n/fieldMappings'

interface GroupEntry {
  name: string
  wins: number
  losses: number
  ties: number
  unfinished: number
  win_rate?: number
}

interface Props {
  entries: GroupEntry[]
  title?: string
  height?: number
}

interface OutcomeLabels { win: string; loss: string; tie: string; dnf: string }

// eslint-disable-next-line react-refresh/only-export-components
export function buildOutcomesOption(entries: GroupEntry[], labels: OutcomeLabels, yAxisLabel: string): EChartsCoreOption {
  if (entries.length === 0) return { backgroundColor: CHART_BG }
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  const categories = entries.map((e) => e.name)

  const labelAll = barLabel({ formatter: (p: { value: number }) => tipNumberAuto(p.value) })
  const labelIfNonzero = barLabel({ formatter: (p: { value: number }) => (p.value > 0 ? tipNumberAuto(p.value) : '') })
  const winColor = resolveToken('outcome-win')
  const lossColor = resolveToken('outcome-loss')
  const tieColor = resolveToken('outcome-draw')
  const dnfColor = resolveToken('outcome-dnf')

  const series = [
    {
      name: labels.win,
      type: 'bar',
      stack: 'total',
      data: entries.map((e) => ({ value: e.wins, win_rate: e.win_rate })),
      itemStyle: { color: winColor, opacity: 0.85 },
      label: labelAll,
      barCategoryGap: '15%',
    },
    {
      name: labels.loss,
      type: 'bar',
      stack: 'total',
      data: entries.map((e) => ({ value: e.losses })),
      itemStyle: { color: lossColor, opacity: 0.75 },
      label: labelAll,
      barCategoryGap: '15%',
    },
    {
      name: labels.tie,
      type: 'bar',
      stack: 'total',
      data: entries.map((e) => ({ value: e.ties })),
      itemStyle: { color: tieColor, opacity: 0.7 },
      label: labelIfNonzero,
      barCategoryGap: '15%',
    },
    {
      name: labels.dnf,
      type: 'bar',
      stack: 'total',
      data: entries.map((e) => ({ value: e.unfinished })),
      itemStyle: { color: dnfColor, opacity: 0.6 },
      label: labelIfNonzero,
      barCategoryGap: '15%',
    },
  ]

  return {
    backgroundColor: CHART_BG,
    grid: { left: 40, right: 20, top: 30, bottom: 100, containLabel: false },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      // Une ligne par issue présente : un segment à zéro n'est ni dessiné ni étiqueté.
      formatter: axisTooltipFormatter({
        value: (r) => tipNumberAuto(r.value),
        order: stackTopFirstOrder(series),
        skip: (r) => r.value === 0,
      }),
    },
    legend: { ...getLegendBase(tc), orient: 'horizontal', bottom: 8, left: 'center', right: undefined, top: undefined },
    xAxis: {
      ...axis,
      type: 'category',
      data: categories,
      splitLine: { show: true, lineStyle: { color: tc.splitLine } },
      axisLabel: { ...axis.axisLabel, rotate: 45 },
    },
    yAxis: {
      ...axis,
      type: 'value',
      name: yAxisLabel,
      splitLine: { show: true, lineStyle: { color: tc.splitLine } },
    },
    series,
  }
}

type Pt = { name: string }

export function SynthesisOutcomesByGroupChart({ entries, title, height }: Props) {
  const { data: fieldMappings } = useFieldMappings()
  const winLabel  = useOutcomeLabel('win')
  const lossLabel = useOutcomeLabel('loss')
  const tieLabel  = useOutcomeLabel('tie')
  const dnfLabel  = useOutcomeLabel('dnf')
  const yAxisLabel = fieldMappings?.fields['match_count']?.label ?? 'Matchs'

  const series: ChartSeries<Pt>[] = entries.length > 0
    ? [{ key: 'outcomes', datapoints: entries.map((e) => ({ name: e.name })) }]
    : []

  // Clé de recalcul stable : on ne rebuild l'option que si le contenu change.
  const entriesKey = JSON.stringify(entries)
  const buildOption = useCallback(
    () => buildOutcomesOption(entries, { win: winLabel, loss: lossLabel, tie: tieLabel, dnf: dnfLabel }, yAxisLabel),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [entriesKey, winLabel, lossLabel, tieLabel, dnfLabel, yAxisLabel],
  )

  return (
    <ChartCard
      title={title}
      series={series}
      buildOption={buildOption as (s: ChartSeries<Pt>[]) => EChartsCoreOption}
      height={height ?? 520}
    />
  )
}
