/**
 * matchRefreshPolicy — politique de rafraîchissement des lectures d'un match (vue complète et
 * contexte du rejeu), écrite une fois.
 *
 * Deux états transitoires, une même réponse :
 *  - une lecture a ÉCHOUÉ côté serveur : la réponse est périmée dès réception et un nouvel
 *    essai part toutes les 30 s (le message « Nouvel essai automatique » de la vue est vrai) ;
 *  - l'analyse du film est EN COURS : périmée dès réception, rechargée toutes les 60 s.
 * Hors de ces deux états : 10 min de fraîcheur, aucun rechargement.
 */
import { ANALYSIS_PENDING_REFETCH_MS } from './analysisPending.logic'

/** Période du nouvel essai automatique tant qu'une lecture du match est en échec. */
export const MATCH_SOURCE_RETRY_MS = 30_000

/** Fraîcheur d'une lecture de match complète et stable. */
export const MATCH_DATA_STALE_MS = 10 * 60 * 1000

/** Ce que la politique lit d'une réponse : chaque appelant le dérive de sa propre forme. */
export interface MatchRefreshState {
  sourceUnavailable: boolean
  analysisPending: boolean
}

/** `refetchInterval` : 30 s sur lecture en échec, sinon 60 s pendant l'analyse, sinon arrêt. */
export function matchRefetchInterval({ sourceUnavailable, analysisPending }: MatchRefreshState): number | false {
  if (sourceUnavailable) return MATCH_SOURCE_RETRY_MS
  return analysisPending ? ANALYSIS_PENDING_REFETCH_MS : false
}

/** `staleTime` : nulle dans les deux états transitoires, 10 min sinon. */
export function matchStaleTime({ sourceUnavailable, analysisPending }: MatchRefreshState): number {
  return sourceUnavailable || analysisPending ? 0 : MATCH_DATA_STALE_MS
}
