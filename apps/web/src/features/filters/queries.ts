/**
 * queries.ts — Hook TanStack Query pour POST /filters/resolve.
 *
 * Le globalFilterStore définit un slot `resolvedContext` (sessions, options
 * cascade) que FilterOmnibar / SessionNavBar / SquadLayout consomment. Sans ce hook,
 * resolvedContext reste null, donc :
 *  - le sélecteur de session affiche "Aucune session disponible"
 *  - les filtres cascade (Playlists, Modes, Cartes, Types) sont absents
 *  - le default-to-latest sur Squad ne se déclenche jamais
 *
 * Ce hook fait le pont : à chaque changement de filterContext, on POST
 * l'endpoint et on dispatche la réponse via setResolvedContext.
 */
import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import type { FilterContextInput, FilterContextResolved } from '@/lib/api/types'

/**
 * Résout le filterContext courant côté backend (sessions disponibles + options
 * cascade) et synchronise le résultat dans le globalFilterStore.
 *
 * À monter une fois par page joueur (typiquement dans le layout `$playerSlug`).
 * Le hook re-fetch automatiquement quand `filterContextHash` change.
 */
export function useFiltersResolve(playerSlug: string) {
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const filterContextHash = useGlobalFilterStore((s) => s.filterContextHash)
  const setResolvedContext = useGlobalFilterStore((s) => s.setResolvedContext)

  const query = useQuery<FilterContextResolved>({
    queryKey: queryKeys.filtersResolve(playerSlug, filterContextHash),
    queryFn: () =>
      api.post<FilterContextResolved>(
        `/players/${playerSlug}/filters/resolve`,
        filterContext satisfies FilterContextInput,
      ),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })

  // Synchronise la réponse dans le store + track le latest session_id pour
  // permettre la détection "nouvelles sessions arrivées" (auto-snap dans
  // PlayerLayout sur fin de sync).
  useEffect(() => {
    if (!query.data) return
    setResolvedContext(query.data)
    const latestId = query.data.session_options?.all_sessions?.[0]?.session_id
    if (latestId) {
      useGlobalFilterStore.getState().setLastKnownLatestSessionId(latestId)
    }
  }, [query.data, setResolvedContext])

  return query
}
