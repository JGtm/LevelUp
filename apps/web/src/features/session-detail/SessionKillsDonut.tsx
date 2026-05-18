/**
 * SessionKillsDonut — donut Kills / Deaths / Assists agrégés sur la session.
 *
 * Slices colorés via `sliceColors` (outcome-win/loss/draw) — pas de hex direct.
 * Libellés de slices résolus via `useFieldMappings.fields[*]` (i18n TOML).
 */
import { useMemo } from 'react'

import { DonutChart, type ChartPointDonut } from '@/components/charts/DonutChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { SemanticToken } from '@/lib/accessibility'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import type { SessionDetailMatchRow } from '@/lib/api/types'

interface Props {
  title: string
  matches: SessionDetailMatchRow[]
  height?: number
}

export function SessionKillsDonut({ title, matches, height = 260 }: Props) {
  const { data: fieldMappings } = useFieldMappings()
  const fields = fieldMappings?.fields

  const labels = useMemo(
    () => ({
      kills: fields?.kills?.label ?? 'kills',
      deaths: fields?.deaths?.label ?? 'deaths',
      assists: fields?.assists?.label ?? 'assists',
    }),
    [fields],
  )

  const { series, total } = useMemo(() => {
    let kills = 0
    let deaths = 0
    let assists = 0
    for (const m of matches) {
      kills += m.kills
      deaths += m.deaths
      assists += m.assists
    }
    const s: ChartSeries<ChartPointDonut>[] = [
      {
        key: 'kda-breakdown',
        datapoints: [
          { name: labels.kills, value: kills },
          { name: labels.deaths, value: deaths },
          { name: labels.assists, value: assists },
        ].filter((d) => d.value > 0),
      },
    ]
    return { series: s, total: kills + deaths + assists }
  }, [matches, labels])

  // Couleurs sémantiques mappées par nom de slice (résolu via tokens).
  const sliceColors: Record<string, SemanticToken> = {
    [labels.kills]: 'outcome-win',
    [labels.deaths]: 'outcome-loss',
    [labels.assists]: 'outcome-draw',
  }

  if (total === 0) return null

  return (
    <DonutChart title={title} series={series} sliceColors={sliceColors} height={height} />
  )
}
