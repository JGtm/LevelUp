/**
 * SquadEngagementSection — adapter pour Mock 15 v2 sur Squad page.
 *
 * Charge via /players/{slug}/pages/squad/v2/engagement, prepare le payload
 * pour SquadEngagementView.
 */
import { useMemo } from 'react'

import {
  SquadEngagementView,
  type SquadEngagementSession as ViewSession,
  type SquadPlayerEngagement as ViewPlayer,
} from '@/features/squad/v2/SquadEngagementView'
import { useSquadEngagementSession, type SquadTeammateEntry } from '@/features/engagement/queries'
import type { SemanticToken } from '@/lib/accessibility'
import { formatMessage } from '@/lib/i18n/format'
import { engagementManifest } from '@/lib/i18n/generated/engagement'
import { useAppShellStore } from '@/stores/appShellStore'

export interface SquadEngagementSectionProps {
  playerSlug: string
  matchIds?: string[]
  teammates?: SquadTeammateEntry[]
  /** Couleurs hex par gamertag — utilise en priorité sur les tokens internes. */
  colorByPlayer?: Record<string, string>
}

const COLOR_TOKENS: SemanticToken[] = [
  'chart-series-3',
  'chart-series-4',
  'chart-series-5',
  'chart-series-6',
]

export function SquadEngagementSection(props: SquadEngagementSectionProps) {
  const { playerSlug, matchIds = [], teammates = [], colorByPlayer } = props
  const locale = useAppShellStore((s) => s.locale)
  const query = useSquadEngagementSession(playerSlug, matchIds, teammates)

  // colorByPlayer est recréé à chaque rendu parent (nouvel objet) : on le suit par
  // contenu pour rafraîchir la vue quand une couleur change réellement, sans
  // recalculer à chaque rendu.
  const colorKey = JSON.stringify(colorByPlayer ?? {})
  const session: ViewSession = useMemo(() => {
    if (!query.data) {
      return { labels: [], mapNames: [], lobbyPerPlayer: [], teamExpected: [], teamObserved: [], players: [] }
    }
    const players: ViewPlayer[] = query.data.players.map((p, i) => ({
      xuid: p.xuid,
      gamertag: p.gamertag,
      paceObserved: p.pace_observed,
      colorToken: COLOR_TOKENS[i % COLOR_TOKENS.length],
      colorHex: colorByPlayer?.[p.gamertag],
    }))
    return {
      labels: query.data.labels,
      mapNames: query.data.map_names ?? [],
      lobbyPerPlayer: query.data.lobby_per_player,
      teamExpected: query.data.team_expected,
      teamObserved: query.data.team_observed,
      players,
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- colorByPlayer suivi via colorKey (2026-07-22)
  }, [query.data, colorKey])

  // Pas de `return null` sur erreur / session vide : SquadEngagementView
  // (ChartCard) rend son état error / empty dans le même bloc titré.
  return (
    <SquadEngagementView
      session={session}
      state={query.isLoading ? 'loading' : query.isError ? 'error' : 'ready'}
      seriesLabels={{
        lobby: formatMessage(engagementManifest, 'engagement.squad.trace.lobby', locale),
        expected: formatMessage(engagementManifest, 'engagement.squad.trace.expected', locale),
        observed: formatMessage(engagementManifest, 'engagement.squad.trace.observed', locale),
      }}
    />
  )
}
