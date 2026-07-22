/**
 * EngagementTimeseriesSection — Mock 11 dans Timeseries intensity tab.
 *
 * 1 point = 1 match OU 1 agrégat (session/week/month) selon la densité du
 * scope filtré (binning adaptatif côté backend). 3 traces : joueur (saturé),
 * attendu (pointillé), équipe (gris fin).
 */
import { useMemo } from 'react'

import { EngagementCurve, type EngagementPoint } from '@/components/charts/EngagementCurve'
import { useEngagementTimeseries } from '@/features/engagement/queries'
import { truncateMap } from '@/lib/charts/matchLabels'
import type { EngagementGranularity, FilterContextInput } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { engagementManifest, type EngagementManifestKey } from '@/lib/i18n/generated/engagement'
import { useAppShellStore } from '@/stores/appShellStore'

export interface EngagementTimeseriesSectionProps {
  playerSlug: string
  /** Scope filtré (page Timeseries injecte le soloFilterContext). */
  filters: FilterContextInput
  /** Hash dérivé du filterStore — invalide la query au moindre changement. */
  filterHash: string
  limit?: number
  title?: string
  /** Surcharge optionnelle — par défaut, sous-titre dérivé de la granularité. */
  subtitle?: string
}

const GRANULARITY_KEYS: Record<EngagementGranularity, EngagementManifestKey> = {
  match: 'engagement.granularity.match',
  session: 'engagement.granularity.session',
  week: 'engagement.granularity.week',
  month: 'engagement.granularity.month',
}

export function EngagementTimeseriesSection(props: EngagementTimeseriesSectionProps) {
  const { playerSlug, filters, filterHash, limit = 30, title, subtitle } = props
  const locale = useAppShellStore((s) => s.locale)
  const query = useEngagementTimeseries(playerSlug, filters, filterHash, limit)

  const data = query.data
  const pointsAPI = useMemo(() => data?.points ?? [], [data?.points])

  const points: EngagementPoint[] = useMemo(() => {
    return pointsAPI.map((m, i) => ({
      x: i,
      paceTeam: m.pace_team,
      paceAttendu: m.pace_attendu,
      paceJoueur: m.pace_joueur,
      paceLobby: m.pace_lobby,
    }))
  }, [pointsAPI])

  // Plus de `return null` sur erreur / dataset vide : on laisse EngagementCurve
  // (ChartCard) rendre son état error / empty dans le même bloc titré.

  // Étiquettes X : pour "match" on garde `#N\nMap`. Pour les agrégats on
  // affiche le label brut (session_label / "2026-S18" / "2026-05").
  const rawGranularity = data?.granularity
  const granularity: EngagementGranularity =
    rawGranularity === 'session' ||
    rawGranularity === 'week' ||
    rawGranularity === 'month'
      ? rawGranularity
      : 'match'
  const xLabels = pointsAPI.map((m, i) => {
    if (granularity === 'match') {
      return m.map_name ? `#${i + 1}\n${truncateMap(m.map_name)}` : `#${i + 1}`
    }
    return m.label
  })
  const xFormatter = (i: number) => xLabels[i] ?? `#${i + 1}`

  const total = data?.total_matches ?? 0
  const computedSubtitle = (() => {
    if (subtitle !== undefined) return subtitle
    if (!data) return undefined
    const granLabel = formatMessage(engagementManifest, GRANULARITY_KEYS[granularity], locale)
    const truncNote = data.truncated_to_recent
      ? formatMessage(engagementManifest, 'engagement.timeseries.subtitle_truncated', locale, {
          recent: data.truncated_to_recent,
          total,
        })
      : total > 0
        ? formatMessage(engagementManifest, 'engagement.timeseries.subtitle_count', locale, { total })
        : ''
    return `${granLabel}${truncNote}`
  })()

  return (
    <EngagementCurve
      title={title ?? formatMessage(engagementManifest, 'engagement.match_view.section_title', locale)}
      subtitle={computedSubtitle}
      points={points}
      granularity="session"
      state={query.isLoading ? 'loading' : query.isError ? 'error' : 'ready'}
      errorMessage={formatMessage(engagementManifest, 'engagement.timeseries.error', locale)}
      xFormatter={xFormatter}
    />
  )
}
