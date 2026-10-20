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
import {
  matchRefetchInterval,
  matchStaleTime,
  type MatchRefreshState,
} from '@/lib/analysis-pending/matchRefreshPolicy'

export interface ReplayMatchContext {
  header: MatchViewHeader
  scoreboard: MatchScoreboardRow[]
  highlight_events: MatchHighlightEvent[]
  media: MatchMediaTab
  /**
   * Une lecture a échoué côté serveur pendant la construction : le contexte est transitoire
   * (nouvel essai toutes les 30 s, jamais conservé en cache) — lu par la politique de
   * rafraîchissement seule, aucun composant du rejeu ne l'affiche.
   */
  source_unavailable: boolean
}

/** Le contexte tel que l'API le transporte : une tranche vide côté Go peut arriver `null`. */
export type ReplayMatchContextWire = Pick<ReplayMatchContext, 'header' | 'source_unavailable'> & {
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
    source_unavailable: raw.source_unavailable === true,
  }
}

/** L'état transitoire du contexte, lu par la politique partagée avec la vue complète. */
function refreshStateOf(data: ReplayMatchContext | undefined): MatchRefreshState {
  return {
    sourceUnavailable: data?.source_unavailable === true,
    analysisPending: matchViewHasAnalysisPending(data),
  }
}

/**
 * Fraîcheur du contexte en cache : nulle tant qu'une lecture est en échec ou que l'analyse du
 * film est en cours, 10 min sinon — la règle de la vue complète du match (`matchRefreshPolicy`).
 */
export function replayContextStaleTime(data: ReplayMatchContext | undefined): number {
  return matchStaleTime(refreshStateOf(data))
}

/** Rechargement du contexte : 30 s sur lecture en échec, 60 s pendant l'analyse, arrêt sinon. */
export function replayContextRefetchInterval(data: ReplayMatchContext | undefined): number | false {
  return matchRefetchInterval(refreshStateOf(data))
}
