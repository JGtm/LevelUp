/**
 * replayMatchContext — ce que la page de rejeu lit de la base pour un match, et RIEN D'AUTRE.
 *
 * Quatre morceaux, chacun du TYPE EXACT de son homologue dans la vue complète du match :
 * l'en-tête (horloge, fenêtre, score final, libellés), les lignes du tableau de score
 * (identités, camps, K/D/A, entrées et sorties en cours de match), les événements du match
 * (kills ET médailles : la frise de chaque joueur, ses médailles, le fil des éliminations) et
 * les médias. Servis par GET .../matches/{id}/replay/context, construits côté API par les
 * mêmes builders que la vue complète. Le modèle du rejeu et ses composants sont typés sur ce
 * contrat : un lecteur qui voudrait un champ de la vue complète hors de ces morceaux ne compile pas.
 */
import type {
  MatchHighlightEvent,
  MatchMediaTab,
  MatchScoreboardRow,
  MatchViewHeader,
} from '@/lib/api/types'
import { matchViewHasAnalysisPending } from '@/lib/analysis-pending/analysisPending.logic'

export interface ReplayMatchContext {
  header: MatchViewHeader
  scoreboard: MatchScoreboardRow[]
  highlight_events: MatchHighlightEvent[]
  media: MatchMediaTab
}

/** Le contexte tel que l'API le transporte : une tranche vide côté Go peut arriver `null`. */
export type ReplayMatchContextWire = Pick<ReplayMatchContext, 'header'> & {
  scoreboard: MatchScoreboardRow[] | null
  highlight_events: MatchHighlightEvent[] | null
  media: MatchMediaTab | null
}

/**
 * Comble les tableaux absents par des vides, à l'entrée : les lecteurs du contexte n'ont pas à
 * se protéger d'un `null` de transport.
 */
export function normalizeReplayMatchContext(raw: ReplayMatchContextWire): ReplayMatchContext {
  return {
    header: raw.header,
    scoreboard: raw.scoreboard ?? [],
    highlight_events: raw.highlight_events ?? [],
    media: raw.media ?? { media_items: [] },
  }
}

const REPLAY_CONTEXT_STALE_MS = 10 * 60 * 1000

/**
 * Fraîcheur du contexte en cache : nulle tant que l'analyse du film est en cours (il change d'ici
 * quelques instants), 10 min sinon — la même règle que la vue complète du match.
 */
export function replayContextStaleTime(data: ReplayMatchContext | undefined): number {
  return matchViewHasAnalysisPending(data) ? 0 : REPLAY_CONTEXT_STALE_MS
}
