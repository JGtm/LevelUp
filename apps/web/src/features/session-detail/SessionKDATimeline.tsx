/**
 * SessionKDATimeline — courbes Kills / Deaths / Assists par match au fil de la session.
 *
 * 3 séries dans un TimeseriesLineChart, X = start_time (datetime).
 * Couleurs sémantiques : kills→outcome-win, deaths→outcome-loss, assists→outcome-draw.
 * Hauteur fixe 280px ; pas de marker outcome (les 3 traces sont colorées par stat).
 */
import { useMemo } from 'react'

import { TimeseriesLineChart, type ChartPoint2D } from '@/components/charts/TimeseriesLineChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import { resolveToken } from '@/lib/accessibility'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import type { SessionDetailMatchRow } from '@/lib/api/types'

import { sessionMatchAxisLabel } from './_shared'

interface Props {
  title: string
  matches: SessionDetailMatchRow[]
  height?: number
}

export function SessionKDATimeline({ title, matches, height = 280 }: Props) {
  const { data: fieldMappings } = useFieldMappings()
  const fields = fieldMappings?.fields

  const series: ChartSeries<ChartPoint2D>[] = useMemo(() => {
    if (matches.length === 0) return []
    const sorted = [...matches].sort((a, b) => a.start_time.localeCompare(b.start_time))
    const labelFor = (key: string): string => fields?.[key]?.label ?? key
    return [
      {
        key: 'kills',
        labelKey: labelFor('kills'),
        colorToken: 'outcome-win',
        datapoints: sorted.map((m, i) => ({ x: sessionMatchAxisLabel(i, m.map_name, m.pair_name), y: m.kills })),
      },
      {
        key: 'deaths',
        labelKey: labelFor('deaths'),
        colorToken: 'outcome-loss',
        datapoints: sorted.map((m, i) => ({ x: sessionMatchAxisLabel(i, m.map_name, m.pair_name), y: m.deaths })),
      },
      {
        key: 'assists',
        labelKey: labelFor('assists'),
        colorToken: 'outcome-draw',
        datapoints: sorted.map((m, i) => ({ x: sessionMatchAxisLabel(i, m.map_name, m.pair_name), y: m.assists })),
      },
    ]
  }, [matches, fields])

  // Override couleurs pour garantir la résolution des tokens (cf. TimeseriesLineChart prop
  // seriesColorResolver : prioritaire sur colorToken et fallback cycle).
  const colorByKey: Record<string, string> = {
    kills: resolveToken('outcome-win'),
    deaths: resolveToken('outcome-loss'),
    assists: resolveToken('outcome-draw'),
  }

  // Axe X catégoriel : 1 étiquette par match "#N\nCarte" (façon page Escouade),
  // intervalle clairsemé au-delà de 30 matchs pour éviter le chevauchement.
  const xLabelInterval = matches.length > 30 ? Math.floor(matches.length / 12) : 0

  return (
    <TimeseriesLineChart
      title={title}
      series={series}
      height={height}
      xAxisType="category"
      xAxisLabelInterval={xLabelInterval}
      outcomeMarkers={false}
      seriesNameResolver={(s) => s.labelKey ?? s.key}
      seriesColorResolver={(s) => colorByKey[s.key]}
    />
  )
}
