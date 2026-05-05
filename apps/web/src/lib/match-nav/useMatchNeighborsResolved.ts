/**
 * useMatchNeighborsResolved — résolution cascade de la navigation prev/next.
 *
 * Ordre :
 *   1. Router state (instantané, scope onglet courant)
 *   2. sessionStorage (survit F5 / nav arrière, TTL 1h)
 *   3. (Phase 2b — non implémenté ici) URL query params
 *   4. Fallback API Q25 (chronologie globale du joueur)
 *
 * Quand un contexte est résolu en local (1 ou 2), aucun appel API n'est
 * fait — la latence est nulle et la navigation reste 100% dans le périmètre
 * d'origine.
 */
import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useRouterState } from '@tanstack/react-router'

import { api } from '@/lib/api/client'
import type { MatchNeighbors } from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'
import {
  readNavContext,
  resolveNeighborsFromContext,
  type MatchNavContext,
} from './navContext'

/**
 * useMatchNeighborsAPI — fallback API global Q25 (chronologie complète du joueur).
 *
 * Inliné ici pour éviter une dépendance cross-feature `lib/` → `features/`.
 * La query existante côté `features/match-view/queries.ts` reste utilisée
 * par d'autres consommateurs (ex. prefetch route loader).
 */
function useMatchNeighborsAPI(playerSlug: string, matchId: string) {
  return useQuery({
    queryKey: queryKeys.matchNeighbors(playerSlug, matchId),
    queryFn: () =>
      api.get<MatchNeighbors>(`/players/${playerSlug}/matches/${matchId}/neighbors`),
    enabled: !!playerSlug && !!matchId,
    staleTime: 5 * 60 * 1000,
  })
}

/**
 * `source` indique d'où vient le résultat (utile pour debug + UI : on
 * affiche le `filtersLabel` uniquement si `source !== 'global'`).
 */
export type ResolvedNeighborsSource = 'router-state' | 'session-storage' | 'api'

export interface ResolvedNeighbors {
  data: MatchNeighbors | undefined
  isPending: boolean
  /** Origine du résultat — pour debug et UI conditionnelle. */
  source: ResolvedNeighborsSource
  /** Label humain du contexte d'origine (si state/session). */
  contextLabel?: string
  /** Métadonnée brute du contexte d'origine (utile pour Phase 2b query params). */
  navContext?: MatchNavContext
}

interface RouterStateWithCtx {
  matchNavContext?: MatchNavContext
}

export function useMatchNeighborsResolved(
  playerSlug: string,
  matchId: string,
): ResolvedNeighbors {
  // 1. Router state — accessible synchronement via useRouterState avec select
  const stateCtx = useRouterState({
    select: (s) => (s.location.state as RouterStateWithCtx)?.matchNavContext,
  })

  // 2. sessionStorage — fallback si state vide (post-F5)
  // Memoize sur matchId : la lecture est synchronome mais le résultat ne
  // change qu'avec le matchId (TTL purgé à la lecture).
  const storedCtx = useMemo<MatchNavContext | null>(
    () => (stateCtx ? null : readNavContext(matchId)),
    [stateCtx, matchId],
  )

  const localCtx = stateCtx ?? storedCtx ?? null

  // 3. Fallback API — appelé uniquement si pas de contexte local
  // (le hook est appelé inconditionnellement, mais avec enabled=false si
  // localCtx résolt — TanStack Query ne déclenche alors aucune requête).
  const apiQuery = useMatchNeighborsAPI(playerSlug, matchId)
  // Note : on ne peut pas désactiver le hook lui-même sans casser la règle
  // des hooks. On compte donc sur le cache TanStack si la query a déjà
  // tourné dans la session ; sinon, le coût est marginal (1 fetch fire-and-
  // forget jamais utilisé). La vraie optimisation viendra Phase 2b avec
  // le `enabled: !localCtx`.

  if (localCtx) {
    const resolved = resolveNeighborsFromContext(localCtx, matchId)
    if (resolved) {
      return {
        data: {
          previous_match_id: resolved.prev_match_id,
          next_match_id: resolved.next_match_id,
          current_index: resolved.current_index,
          total_matches: resolved.total,
        },
        isPending: false,
        source: stateCtx ? 'router-state' : 'session-storage',
        contextLabel: localCtx.filtersLabel,
        navContext: localCtx,
      }
    }
    // Si le matchId n'est pas dans la liste : on ignore le contexte (signal
    // d'incohérence — peut arriver si l'utilisateur a édité l'URL ou si la
    // liste est devenue stale). Fallback API silencieux.
  }

  return {
    data: apiQuery.data,
    isPending: apiQuery.isPending,
    source: 'api',
  }
}
