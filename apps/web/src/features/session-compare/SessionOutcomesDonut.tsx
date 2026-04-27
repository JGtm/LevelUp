/**
 * SessionOutcomesDonut — paire de donuts ECharts comparant la répartition
 * des outcomes de 2 sessions (A vs B).
 *
 * Phase 3 P3.D : remplace l'ancien `<PlotlyChart figure={outcomes_chart} />`
 * server-side. Construit côté client à partir des `wins` / `losses` /
 * `total_matches` portés par `SessionCompareEntry`.
 */
import { useMemo } from 'react'

import { DonutChart } from '@/components/charts/DonutChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { ChartPointDonut } from '@/components/charts/DonutChart'
import type { SessionCompareEntry } from '@/lib/api/types'

export interface SessionOutcomesDonutProps {
  sessionA: SessionCompareEntry | null
  sessionB: SessionCompareEntry | null
  /** Libellés résolus i18n. */
  labels: {
    sessionA: string
    sessionB: string
    wins: string
    losses: string
    other: string
  }
}

function buildSlices(
  entry: SessionCompareEntry,
  labels: { wins: string; losses: string; other: string },
): ChartPointDonut[] {
  const wins = entry.wins
  const losses = entry.losses
  const other = Math.max(0, entry.total_matches - wins - losses)
  const slices: ChartPointDonut[] = []
  if (wins > 0) slices.push({ name: labels.wins, value: wins })
  if (losses > 0) slices.push({ name: labels.losses, value: losses })
  if (other > 0) slices.push({ name: labels.other, value: other })
  return slices
}

export function SessionOutcomesDonut({
  sessionA,
  sessionB,
  labels,
}: SessionOutcomesDonutProps) {
  const sliceColors = useMemo(
    () =>
      ({
        [labels.wins]: 'outcome-win',
        [labels.losses]: 'outcome-loss',
        [labels.other]: 'divergent-neutral',
      }) as Record<string, 'outcome-win' | 'outcome-loss' | 'divergent-neutral'>,
    [labels.wins, labels.losses, labels.other],
  )

  const seriesA: ChartSeries<ChartPointDonut>[] = useMemo(() => {
    if (!sessionA) return []
    const slices = buildSlices(sessionA, labels)
    if (slices.length === 0) return []
    return [
      {
        key: 'session.compare.donut.a',
        meta: { gamertag: labels.sessionA },
        datapoints: slices,
      },
    ]
  }, [sessionA, labels])

  const seriesB: ChartSeries<ChartPointDonut>[] = useMemo(() => {
    if (!sessionB) return []
    const slices = buildSlices(sessionB, labels)
    if (slices.length === 0) return []
    return [
      {
        key: 'session.compare.donut.b',
        meta: { gamertag: labels.sessionB },
        datapoints: slices,
      },
    ]
  }, [sessionB, labels])

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <div className="flex flex-col items-center">
        <p className="mb-1 text-xs font-semibold text-compare-a">{labels.sessionA}</p>
        <DonutChart series={seriesA} sliceColors={sliceColors} height={220} />
      </div>
      <div className="flex flex-col items-center">
        <p className="mb-1 text-xs font-semibold text-compare-b">{labels.sessionB}</p>
        <DonutChart series={seriesB} sliceColors={sliceColors} height={220} />
      </div>
    </div>
  )
}
