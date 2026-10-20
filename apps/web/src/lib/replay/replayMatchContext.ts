/**
 * replayMatchContext — ce que la page de rejeu lit de la base pour un match, et RIEN D'AUTRE.
 *
 * Quatre morceaux, chacun du TYPE EXACT de son homologue dans la vue complète du match :
 * l'en-tête (horloge, fenêtre, score final, libellés), les lignes du tableau de score
 * (identités, camps, K/D/A, entrées et sorties en cours de match), les événements du match
 * (kills ET médailles : la frise de chaque joueur, ses médailles, le fil des éliminations) et
 * les médias. Le modèle du rejeu et ses composants sont typés sur ce contrat : un lecteur qui
 * voudrait un champ de la vue complète hors de ces morceaux ne compile pas.
 */
import type {
  MatchHighlightEvent,
  MatchMediaTab,
  MatchScoreboardRow,
  MatchViewHeader,
  MatchViewResponse,
} from '@/lib/api/types'

export interface ReplayMatchContext {
  header: MatchViewHeader
  scoreboard: MatchScoreboardRow[]
  highlight_events: MatchHighlightEvent[]
  media: MatchMediaTab
}

/**
 * Projette la vue complète du match sur les quatre morceaux que le rejeu lit. Les tableaux
 * absents deviennent vides, jamais `undefined` : les lecteurs du contexte n'ont pas à se
 * protéger d'une vue partielle.
 */
export function replayContextFromMatchView(
  matchView: MatchViewResponse | null | undefined,
): ReplayMatchContext | undefined {
  if (!matchView) return undefined
  return {
    header: matchView.header,
    scoreboard: matchView.team_tab?.scoreboard ?? [],
    highlight_events: matchView.combat_tab?.highlight_events ?? [],
    media: matchView.media_tab ?? { media_items: [] },
  }
}
