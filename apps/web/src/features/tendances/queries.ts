/**
 * queries — la lecture TanStack Query de l'onglet Tendances.
 *
 * UN CHARGEMENT, UNE RÉPONSE : l'API rend d'un coup la matrice, les séries à tous les pas
 * sur 365 jours, le calendrier et les blocs par horizon. Changer d'horizon ou de pas ne
 * relit rien (le web découpe) ; seuls la vue, le type de partie et la locale — tous dans
 * la clé — refont la requête.
 *
 * LA RÉPONSE PRÉCÉDENTE RESTE AFFICHÉE pendant une relecture, mais au MÊME joueur et au MÊME
 * titre seulement : les options du filtre « Type de partie » viennent de la réponse, et la
 * page ne doit ni se démonter ni perdre son filtre quand on le change. Une réponse d'un
 * autre joueur ou d'un autre titre mentirait, elle n'est jamais gardée. Il en va de même d'une
 * AUTRE VUE, d'une autre composition ou d'une autre règle de composition : leurs lignes et leurs
 * types de partie sont ceux d'une autre population.
 *
 * LA VUE ESCOUADE SANS COÉQUIPIER NE PART PAS : l'API répond 400, la requête reste désactivée
 * et la page affiche l'invite à choisir une escouade.
 */
import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import type { TrendsPageResponse, TrendsQueryRequest } from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'

const STALE_TIME_MS = 5 * 60 * 1000

/** Index de la vue, de la composition et de l'option stricte dans la clé. */
const KEY_VIEW_INDEX = 3
const KEY_COMPOSITION_INDEX = 6
const KEY_EXACT_INDEX = 7

/** Identité d'une composition : les gamertags triés, joints — l'ordre de sélection n'y change rien. */
export function compositionKey(gamertags: readonly string[] | undefined): string {
  return [...(gamertags ?? [])].sort().join(',')
}

export function useTrendsPage(playerSlug: string, request: TrendsQueryRequest) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const view = request.view ?? 'solo'
  const gameType = request.game_type ?? ''
  const locale = request.locale ?? ''
  const squad = view === 'squad'
  const composition = squad ? compositionKey(request.selected_gamertags) : ''
  const exactComposition = squad && !!request.exact_composition
  return useQuery({
    queryKey: queryKeys.trends(
      playerSlug,
      titleSlug,
      view,
      gameType,
      locale,
      composition,
      exactComposition,
    ),
    queryFn: ({ signal }) =>
      api.post<TrendsPageResponse>(
        `/players/${playerSlug}/pages/trends`,
        request,
        undefined,
        { signal },
      ),
    enabled: !!playerSlug && (!squad || composition !== ''),
    staleTime: STALE_TIME_MS,
    placeholderData: (precedente, requete) =>
      requete &&
      requete.queryKey[1] === playerSlug &&
      requete.queryKey[2] === titleSlug &&
      requete.queryKey[KEY_VIEW_INDEX] === view &&
      requete.queryKey[KEY_COMPOSITION_INDEX] === composition &&
      requete.queryKey[KEY_EXACT_INDEX] === exactComposition
        ? precedente
        : undefined,
  })
}
