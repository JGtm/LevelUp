/**
 * SessionDamageComposite — barre composite Dégâts infligés / subis, une par match.
 *
 * Barres horizontales empilées (Y = matchs #N + carte, ordre chronologique) : segment
 * "infligés" (divergent-pos) + segment "subis" (divergent-neutral). La longueur totale =
 * implication aux dégâts du match, le partage = ratio infligés/subis. Mêmes couleurs que
 * le composite Rendement/Résistance (OffDefComposite).
 */
import { useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, getAxisBase, getEChartsThemeColors, getTooltipBase } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import type { SessionDetailMatchRow } from '@/lib/api/types'

import { sessionMatchAxisLabel } from './_shared'

interface DamagePoint {
  label: string
  dealt: number
  taken: number
}

interface DamageOpts {
  dealtLabel: string
  takenLabel: string
}

const fmtInt = (n: number) => Math.round(n).toLocaleString('fr-FR')

// eslint-disable-next-line react-refresh/only-export-components
export function buildSessionDamageOption(
  series: ChartSeries<DamagePoint>[],
  opts: DamageOpts,
): EChartsCoreOption {
  const points = series[0]?.datapoints ?? []
  if (points.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const dealtColor = resolveToken('divergent-pos')
  const takenColor = resolveToken('divergent-neutral')
  const labels = points.map((p) => p.label)

  const segLabel = {
    show: true,
    position: 'inside' as const,
    color: tc.text,
    fontSize: 10,
    formatter: (p: { value: number }) => (p.value > 0 ? fmtInt(p.value) : ''),
  }

  return {
    backgroundColor: CHART_BG,
    grid: { top: 12, bottom: 28, left: 8, right: 16, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => {
        const arr = Array.isArray(params) ? params : []
        if (arr.length === 0) return ''
        const idx = (arr[0] as { dataIndex: number }).dataIndex
        const p = points[idx]
        if (!p) return ''
        return [
          `<strong>${p.label.replace('\n', ' · ')}</strong>`,
          `${opts.dealtLabel}: <b>${fmtInt(p.dealt)}</b>`,
          `${opts.takenLabel}: <b>${fmtInt(p.taken)}</b>`,
        ].join('<br/>')
      },
    },
    legend: {
      data: [opts.dealtLabel, opts.takenLabel],
      textStyle: { color: tc.axisLabel },
      bottom: 0,
      itemWidth: 10,
      itemHeight: 10,
    },
    xAxis: { ...axis, type: 'value' },
    yAxis: { ...axis, type: 'category', inverse: true, data: labels, axisTick: { show: false } },
    series: [
      {
        name: opts.dealtLabel,
        type: 'bar',
        stack: 'dmg',
        data: points.map((p) => p.dealt),
        itemStyle: { color: dealtColor },
        barMaxWidth: 18,
        label: segLabel,
      },
      {
        name: opts.takenLabel,
        type: 'bar',
        stack: 'dmg',
        data: points.map((p) => p.taken),
        itemStyle: { color: takenColor },
        barMaxWidth: 18,
        label: segLabel,
      },
    ],
  }
}

interface Props {
  title: string
  matches: SessionDetailMatchRow[]
  height?: number
}

export function SessionDamageComposite({ title, matches, height }: Props) {
  const { data: fieldMappings } = useFieldMappings()
  const fields = fieldMappings?.fields

  const series = useMemo<ChartSeries<DamagePoint>[]>(() => {
    const sorted = [...matches]
      .filter((m) => m.damage_dealt != null || m.damage_taken != null)
      .sort((a, b) => a.start_time.localeCompare(b.start_time))
    if (sorted.length === 0) return []
    return [
      {
        key: 'damage',
        datapoints: sorted.map((m, i) => ({
          label: sessionMatchAxisLabel(i, m.map_name, m.pair_name),
          dealt: m.damage_dealt ?? 0,
          taken: m.damage_taken ?? 0,
        })),
      },
    ]
  }, [matches])

  const rows = series[0]?.datapoints.length ?? 0
  const computedHeight = height ?? Math.min(560, Math.max(220, rows * 30 + 60))

  return (
    <ChartCard
      title={title}
      series={series}
      height={computedHeight}
      buildOption={(s) =>
        buildSessionDamageOption(s, {
          dealtLabel: fields?.damage_dealt?.label ?? 'damage_dealt',
          takenLabel: fields?.damage_taken?.label ?? 'damage_taken',
        })
      }
    />
  )
}
