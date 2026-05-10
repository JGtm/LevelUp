/**
 * EngagementTimeseriesSection — Mock 11 dans Timeseries intensity tab.
 *
 * 1 point = 1 match. 3 traces (joueur, attendu, équipe).
 * Charge via /players/{slug}/engagement/timeseries.
 */
import { useMemo } from 'react'

import { EngagementCurve, type EngagementPoint } from '@/components/charts/EngagementCurve'
import { useEngagementTimeseries } from '@/features/engagement/queries'
import { truncateMap } from '@/lib/charts/matchLabels'

export interface EngagementTimeseriesSectionProps {
  playerSlug: string
  limit?: number
  title?: string
  subtitle?: string
}

export function EngagementTimeseriesSection(props: EngagementTimeseriesSectionProps) {
  const { playerSlug, limit = 30, title, subtitle } = props
  const query = useEngagementTimeseries(playerSlug, limit)

  const points: EngagementPoint[] = useMemo(() => {
    if (!query.data) return []
    return query.data.map((m, i) => ({
      x: i,
      paceTeam: m.pace_team,
      paceAttendu: m.pace_attendu,
      paceJoueur: m.pace_joueur,
      paceLobby: m.pace_lobby,
    }))
  }, [query.data])

  if (query.isError) return null
  if (query.data && query.data.length === 0) return null

  // Étiquettes X au format `#N\nMap` (aligné sur les autres charts timeseries
  // — cf. matchLabels.ts). Fallback `#N` si pas de map_name.
  const xLabels =
    query.data?.map((m, i) => {
      const map = m.map_name
      return map ? `#${i + 1}\n${truncateMap(map)}` : `#${i + 1}`
    }) ?? []
  const xFormatter = (i: number) => xLabels[i] ?? `#${i + 1}`

  return (
    <EngagementCurve
      title={title ?? 'Engagement'}
      subtitle={subtitle ?? "Joueur vs équipe vs attendu sur les derniers matchs"}
      points={points}
      granularity="session"
      state={query.isLoading ? 'loading' : query.isError ? 'error' : 'ready'}
      xFormatter={xFormatter}
    />
  )
}
