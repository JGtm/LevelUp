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
import { buildSubtitle } from '@/features/engagement/engagementSubtitle'
import { useMatchEngagement } from '@/features/engagement/queries'
import type { ApiError } from '@/lib/api/client'
import type { EngagementPointAPI, EngagementScoreResultAPI } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
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

  // Masquage de la série « Joueur attendu » indexé sur expected_basis (modèle
  // lobby-anchored v2) : cold_start => aucun historique exploitable => coef 1.0
  // par défaut => l'attendu ne veut rien dire, on le masque. (Avant : indexé sur
  // confidence, qui qualifie l'historique du percentile, pas l'attendu.)
  const hideAttendu = query.data?.expected_basis === 'cold_start'

  // Message d'indisponibilité, indexé sur la NATURE de l'échec (ne pas accuser à
  // tort le contenu du match) :
  //  - 503 engagement_unavailable → « migration en cours » (schéma manquant).
  //  - 5xx / erreur serveur transitoire (ex. B-swap RW→RO swap timeout côté
  //    DuckDB, ADR 0016) → message neutre « momentanément indisponible » : le
  //    match est valide, la lecture a juste échoué → NE PAS afficher « trop court
  //    ou peu d'action » (mensonger, cf. retour utilisateur match bc918a5a).
  //  - 422 engagement_insufficient / courbe vide (200) → « trop court ou peu
  //    d'action » (là c'est bien le contenu du match qui est en cause).
  const apiError = query.error as unknown as ApiError | null
  const errorCode = apiError?.code
  const isServerError = (apiError?.status ?? 0) >= 500
  const unavailableMessage =
    errorCode === 'engagement_unavailable'
      ? formatMessage(engagementManifest, 'engagement.error.unavailable', locale)
      : isServerError
        ? formatMessage(engagementManifest, 'engagement.error.temporary', locale)
        : (emptyMessage ?? formatMessage(engagementManifest, 'engagement.error.match_unavailable', locale))

  return (
    <EngagementCurve
      title={title ?? formatMessage(engagementManifest, 'engagement.match_view.section_title', locale)}
      subtitle={
        subtitle ??
        (isEmpty || query.isError ? unavailableMessage : buildSubtitle(query.data, locale))
      }
      points={points}
      granularity={granularity}
      hideAttendu={hideAttendu}
      seriesLabels={{
        team: formatMessage(engagementManifest, 'engagement.trace.team', locale),
        expected: formatMessage(engagementManifest, 'engagement.trace.expected', locale),
        player: formatMessage(engagementManifest, 'engagement.trace.player', locale),
        lobby: formatMessage(engagementManifest, 'engagement.trace.lobby', locale),
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
