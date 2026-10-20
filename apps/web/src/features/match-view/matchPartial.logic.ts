/**
 * Logique du bandeau de la vue match (E4, décisions D-8 / D-9 / D-10).
 *
 * Deux situations que la vue doit distinguer :
 *  - une source est VIDE (`<source>_empty`) : la lecture a réussi, il n'y a rien à
 *    afficher — bandeau « match partiel » avec le détail des sections absentes ;
 *  - une source est EN PANNE (`<source>_unavailable`) : la lecture a échoué, la vue est
 *    transitoire — bandeau « données momentanément indisponibles », qui PRIME sur le
 *    précédent quand les deux coexistent.
 */
import type { MatchViewResponse } from '@/lib/api/types'
import { matchViewHasAnalysisPending } from '@/lib/analysis-pending/analysisPending.logic'
import {
  matchRefetchInterval,
  matchStaleTime,
  type MatchRefreshState,
} from '@/lib/analysis-pending/matchRefreshPolicy'

/**
 * Raisons « lecture en échec » émises par l'API (miroir de `domain.IsUnavailableReason`) :
 * une par source portant un `_empty`, plus le code générique `section_unavailable` (un autre
 * chargeur a échoué ; la source est dans le journal serveur). Ensemble FERMÉ, pas un test de
 * suffixe.
 */
export const UNAVAILABLE_REASONS: readonly string[] = [
  'scoreboard_unavailable',
  'events_unavailable',
  'player_stats_unavailable',
  'medals_unavailable',
  'section_unavailable',
]

export function isUnavailableReason(code: string): boolean {
  return UNAVAILABLE_REASONS.includes(code)
}

export type MatchPartialBanner =
  | { kind: 'unavailable' }
  | { kind: 'partial'; reasons: string[] }

/** Bandeau à afficher, ou null quand la vue est complète. */
export function selectMatchPartialBanner(
  data: Pick<MatchViewResponse, 'is_partial' | 'partial_reasons'>,
): MatchPartialBanner | null {
  const reasons = data.partial_reasons ?? []
  if (!data.is_partial || reasons.length === 0) return null
  if (reasons.some(isUnavailableReason)) return { kind: 'unavailable' }
  return { kind: 'partial', reasons }
}

type MatchViewRefreshInput =
  | (Pick<MatchViewResponse, 'is_partial' | 'partial_reasons'> & Partial<Pick<MatchViewResponse, 'header'>>)
  | undefined

/** L'état transitoire de la vue, lu par la politique partagée (`matchRefreshPolicy`). */
function refreshStateOf(data: MatchViewRefreshInput): MatchRefreshState {
  return {
    sourceUnavailable: !!data && selectMatchPartialBanner(data)?.kind === 'unavailable',
    analysisPending: matchViewHasAnalysisPending(data),
  }
}

/**
 * `refetchInterval` de la vue match : 30 s tant qu'une source est en panne (le message
 * « Nouvel essai automatique » est vrai), sinon 60 s tant que l'analyse du film n'est pas
 * finie, sinon plus de rechargement.
 */
export function matchViewPollInterval(data: MatchViewRefreshInput): number | false {
  return matchRefetchInterval(refreshStateOf(data))
}

/**
 * Durée de fraîcheur d'une vue en cache (D-9). Une vue construite avec une source en
 * panne est périmée dès sa réception : le cache de requêtes ne la rejoue pas, la
 * prochaine lecture (retour sur la page, focus) interroge de nouveau l'API — la panne ne
 * survit pas au retour du chargeur. Même règle tant que l'analyse du film est en cours
 * (`header.analysis_pending`) : la vue change d'ici quelques instants, elle ne reste pas
 * figée par la fraîcheur de 10 min. Une vue complète ou seulement partielle (source vide)
 * garde la fraîcheur normale.
 */
export function matchViewStaleTime(data: MatchViewRefreshInput): number {
  return matchStaleTime(refreshStateOf(data))
}
