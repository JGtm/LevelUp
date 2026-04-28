/**
 * EngagementMatchSection — section Match View tab "team" / Timeseries intensity tab.
 *
 * Charge le score + courbe via /matches/{id}/engagement et adapte les points
 * API en EngagementPoint pour le composant EngagementCurve.
 *
 * Mode 'intra' : Match View intra-match (smooth, X = mm:ss).
 * Mode 'session' : Timeseries / Session (linear avec markers, X = label match).
 */
import { useMemo } from 'react'

import { EngagementCurve, type EngagementPoint } from '@/components/charts/EngagementCurve'
import { useMatchEngagement } from '@/features/engagement/queries'
import type { EngagementPointAPI, EngagementScoreResultAPI } from '@/lib/api/types'

export interface EngagementMatchSectionProps {
  playerSlug: string
  matchId: string
  granularity?: 'intra' | 'session'
  title?: string
  subtitle?: string
}

export function EngagementMatchSection(props: EngagementMatchSectionProps) {
  const { playerSlug, matchId, granularity = 'intra', title, subtitle } = props
  const query = useMatchEngagement(playerSlug, matchId)

  const points: EngagementPoint[] = useMemo(
    () => mapApiPointsToCurve(query.data),
    [query.data],
  )

  // Etats : 503 / 422 -> ne rien afficher (la section est masquee silencieusement).
  if (query.isError) {
    return null
  }
  // Score absent (insufficient_history ou no curve) -> on n'affiche rien.
  if (query.data && (!query.data.EngagementCurve || query.data.EngagementCurve.length === 0)) {
    return null
  }

  return (
    <EngagementCurve
      title={title ?? 'Engagement'}
      subtitle={subtitle ?? buildSubtitle(query.data)}
      points={points}
      granularity={granularity}
      state={query.isLoading ? 'loading' : query.isError ? 'error' : 'ready'}
      xFormatter={granularity === 'intra' ? fmtMillisToTimeStamp : undefined}
    />
  )
}

// ---------------------------------------------------------------------------
// Adapters
// ---------------------------------------------------------------------------

function mapApiPointsToCurve(data: EngagementScoreResultAPI | undefined): EngagementPoint[] {
  if (!data || !data.EngagementCurve) return []
  return data.EngagementCurve.map(toEngagementPoint)
}

function toEngagementPoint(p: EngagementPointAPI): EngagementPoint {
  return {
    x: p.TimeMS,
    paceTeam: p.PaceTeam,
    paceAttendu: p.PaceAttendu,
    paceJoueur: p.PaceJoueur,
    paceLobby: p.PaceLobby,
    isPassiveDeath: p.IsPassiveDeath,
    postDeath: p.PostDeathFlag,
  }
}

function fmtMillisToTimeStamp(ms: number): string {
  const totalSec = Math.floor(ms / 1000)
  const m = Math.floor(totalSec / 60)
  const s = totalSec % 60
  return `${m}:${s.toString().padStart(2, '0')}`
}

function buildSubtitle(data: EngagementScoreResultAPI | undefined): string | undefined {
  if (!data) return undefined
  if (data.Confidence === 'insufficient_history') {
    return 'Historique insuffisant — minimum 10 matchs requis'
  }
  if (data.EngagementScore === null) return undefined
  const p = Math.round(data.EngagementScore)
  if (p > 60) return `Au-dessus de votre habitude (P${p})`
  if (p < 40) return `Sous votre habitude (P${p})`
  return `Engagement normal (P${p})`
}
