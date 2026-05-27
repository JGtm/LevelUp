/**
 * SessionCompareKillsDonut — répartition K/D/A côte à côte (Session A vs B).
 * Deux donuts ECharts, un par session, avec labels résolus via fieldMappings TOML.
 */
import { useMemo } from 'react'

import { DonutChart, type ChartPointDonut } from '@/components/charts/DonutChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { SemanticToken } from '@/lib/accessibility'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import type { SessionCompareEntry } from '@/lib/api/types'

export interface SessionCompareKillsDonutProps {
  sessionA: SessionCompareEntry | null
  sessionB: SessionCompareEntry | null
  labels: {
    title: string
    sessionA: string
    sessionB: string
    empty: string
  }
  height?: number
}

function buildSlices(
  entry: SessionCompareEntry,
  fieldLabels: { kills: string; deaths: string; assists: string },
): ChartPointDonut[] {
  let kills = 0
  let deaths = 0
  let assists = 0
  for (const m of entry.matches ?? []) {
    kills += m.kills
    deaths += m.deaths
    assists += m.assists
  }
  return [
    { name: fieldLabels.kills, value: kills },
    { name: fieldLabels.deaths, value: deaths },
    { name: fieldLabels.assists, value: assists },
  ].filter((d) => d.value > 0)
}

export function SessionCompareKillsDonut({
  sessionA,
  sessionB,
  labels,
  height = 220,
}: SessionCompareKillsDonutProps) {
  const { data: fieldMappings } = useFieldMappings()
  const fields = fieldMappings?.fields

  const fieldLabels = useMemo(
    () => ({
      kills: fields?.kills?.label ?? 'kills',
      deaths: fields?.deaths?.label ?? 'deaths',
      assists: fields?.assists?.label ?? 'assists',
    }),
    [fields],
  )

  const sliceColors: Record<string, SemanticToken> = {
    [fieldLabels.kills]: 'outcome-win',
    [fieldLabels.deaths]: 'outcome-loss',
    [fieldLabels.assists]: 'outcome-draw',
  }

  const seriesA: ChartSeries<ChartPointDonut>[] = useMemo(() => {
    if (!sessionA?.matches?.length) return []
    const slices = buildSlices(sessionA, fieldLabels)
    if (slices.length === 0) return []
    return [{ key: 'kda-a', datapoints: slices }]
  }, [sessionA, fieldLabels])

  const seriesB: ChartSeries<ChartPointDonut>[] = useMemo(() => {
    if (!sessionB?.matches?.length) return []
    const slices = buildSlices(sessionB, fieldLabels)
    if (slices.length === 0) return []
    return [{ key: 'kda-b', datapoints: slices }]
  }, [sessionB, fieldLabels])

  const hasData = seriesA.length > 0 || seriesB.length > 0
  if (!hasData) {
    return <p className="text-sm text-muted-foreground italic text-center py-4">{labels.empty}</p>
  }

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <div className="flex flex-col items-center">
        <p className="mb-1 text-xs font-semibold text-compare-a">{labels.sessionA}</p>
        <DonutChart
          title={labels.title}
          series={seriesA}
          sliceColors={sliceColors}
          height={height}
        />
      </div>
      <div className="flex flex-col items-center">
        <p className="mb-1 text-xs font-semibold text-compare-b">{labels.sessionB}</p>
        <DonutChart
          title={labels.title}
          series={seriesB}
          sliceColors={sliceColors}
          height={height}
        />
      </div>
    </div>
  )
}
