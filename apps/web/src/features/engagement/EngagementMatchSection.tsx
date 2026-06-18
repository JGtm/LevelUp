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
import { formatMessage, type ManifestLocale } from '@/lib/i18n/format'
import { engagementManifest } from '@/lib/i18n/generated/engagement'
import { useAppShellStore } from '@/stores/appShellStore'

export interface EngagementMatchSectionProps {
  playerSlug: string
  matchId: string
  granularity?: 'intra' | 'session'
  title?: string
  subtitle?: string
  /**
   * Comportement quand la donnée est absente / insuffisante (503, 422, courbe
   * vide). Par défaut `'hide'` : la section disparaît (legacy onglet Combat).
   * `'placeholder'` : rend une carte vide avec un message — utilisé dans
   * l'onglet Équipe pour que la section reste visible et explicite.
   */
  emptyBehavior?: 'hide' | 'placeholder'
  /** Texte affiché en empty state quand emptyBehavior === 'placeholder'. */
  emptyMessage?: string
}

export function EngagementMatchSection(props: EngagementMatchSectionProps) {
  const {
    playerSlug,
    matchId,
    granularity = 'intra',
    title,
    subtitle,
    emptyBehavior = 'hide',
    emptyMessage,
  } = props
  const locale = useAppShellStore((s) => s.locale)
  const query = useMatchEngagement(playerSlug, matchId)

  const points: EngagementPoint[] = useMemo(
    () => mapApiPointsToCurve(query.data),
    [query.data],
  )

  const isEmpty =
    query.data && (!query.data.engagement_curve || query.data.engagement_curve.length === 0)

  if (emptyBehavior === 'hide') {
    if (query.isError) return null
    if (isEmpty) return null
  }

  // Quand confidence === 'insufficient_history', coef = 1.0 (cold-start) =>
  // pace_attendu = pace_team => courbes superposées => masquer Attendu.
  const hideAttendu = query.data?.confidence === 'insufficient_history'

  return (
    <EngagementCurve
      title={title ?? formatMessage(engagementManifest, 'engagement.match_view.section_title', locale)}
      subtitle={
        subtitle ??
        (isEmpty || query.isError
          ? (emptyMessage ?? formatMessage(engagementManifest, 'engagement.error.unavailable', locale))
          : buildSubtitle(query.data, locale))
      }
      points={points}
      granularity={granularity}
      hideAttendu={hideAttendu}
      seriesLabels={{
        team: formatMessage(engagementManifest, 'engagement.trace.team', locale),
        expected: formatMessage(engagementManifest, 'engagement.trace.expected', locale),
        player: formatMessage(engagementManifest, 'engagement.trace.player', locale),
      }}
      state={
        query.isLoading
          ? 'loading'
          : (query.isError || isEmpty)
            ? 'empty'
            : 'ready'
      }
      xFormatter={granularity === 'intra' ? fmtMillisToTimeStamp : undefined}
    />
  )
}

// ---------------------------------------------------------------------------
// Adapters
// ---------------------------------------------------------------------------

function mapApiPointsToCurve(data: EngagementScoreResultAPI | undefined): EngagementPoint[] {
  if (!data || !data.engagement_curve) return []
  return data.engagement_curve.map(toEngagementPoint)
}

function toEngagementPoint(p: EngagementPointAPI): EngagementPoint {
  return {
    x: p.time_ms,
    paceTeam: p.pace_team,
    paceAttendu: p.pace_attendu,
    paceJoueur: p.pace_joueur,
    paceLobby: p.pace_lobby,
    isPassiveDeath: p.is_passive_death,
    postDeath: p.post_death_flag,
  }
}

function fmtMillisToTimeStamp(ms: number): string {
  const totalSec = Math.floor(ms / 1000)
  const m = Math.floor(totalSec / 60)
  const s = totalSec % 60
  return `${m}:${s.toString().padStart(2, '0')}`
}

function buildSubtitle(
  data: EngagementScoreResultAPI | undefined,
  locale: ManifestLocale,
): string | undefined {
  if (!data) return undefined
  if (data.confidence === 'insufficient_history') {
    return formatMessage(engagementManifest, 'engagement.narrative.insufficient', locale)
  }
  if (data.engagement_score == null) return undefined
  const percentile = Math.round(data.engagement_score)
  if (percentile > 60) {
    return formatMessage(engagementManifest, 'engagement.narrative.above', locale, { percentile })
  }
  if (percentile < 40) {
    return formatMessage(engagementManifest, 'engagement.narrative.below', locale, { percentile })
  }
  return formatMessage(engagementManifest, 'engagement.narrative.normal', locale, { percentile })
}
