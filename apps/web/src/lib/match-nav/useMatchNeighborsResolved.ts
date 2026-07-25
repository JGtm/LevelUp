/**
 * useMatchNeighborsResolved — résolution cascade de la navigation prev/next.
 *
 * Ordre :
 *   1. Router state (instantané, scope onglet courant)
 *   2. sessionStorage (survit F5 / nav arrière, TTL 24h — Phase 3)
 *   3. URL query params (survit Ctrl+Click / lien partagé) — Phase 2b
 *   4. Fallback API Q25 (chronologie globale du joueur)
 *
 * Quand un contexte avec `matchIds` est résolu en local (1 ou 2), aucun
 * appel API n'est fait — la latence est nulle.
 *
 * Phase 2b : si l'URL contient des query params filterSpec mais ni state ni
 * sessionStorage ne portent de matchIds → on tape l'API avec ces filtres.
 * Le serveur calcule prev/next dans le scope filtré via Q25NeighborMatchesTemplate.
 */
import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useRouterState } from '@tanstack/react-router'

import { api } from '@/lib/api/client'
import type { MatchNeighbors } from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import {
  filterSpecToQueryString,
  parseFilterSpecFromSearch,
  readNavContext,
  resolveNeighborsFromContext,
  type ContextDescriptor,
  type MatchFilterSpec,
  type MatchNavContext,
} from './navContext'

/**
 * useMatchNeighborsAPI — fallback API Q25 (chronologie globale ou filtrée).
 *
 * Quand `spec` non null et non vide, l'URL inclut les filtres → le backend
 * appelle Q25NeighborMatchesTemplate (Phase 2b). Sinon Q25 global.
 */
function useMatchNeighborsAPI(
  playerSlug: string,
  matchId: string,
  spec: MatchFilterSpec | null,
) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.matchNeighbors(
      playerSlug,
      titleSlug,
      matchId,
      spec as Record<string, unknown> | null,
    ),
    queryFn: () => {
      const qs = filterSpecToQueryString(spec)
      const url = qs
        ? `/players/${playerSlug}/matches/${matchId}/neighbors?${qs}`
        : `/players/${playerSlug}/matches/${matchId}/neighbors`
      return api.get<MatchNeighbors>(url)
    },
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
  /** Label humain du contexte d'origine (legacy `filtersLabel`, si state/session). */
  contextLabel?: string
  /** Descriptor typé du contexte d'origine — Phase 2c. Préféré à `contextLabel`. */
  contextDescriptor?: ContextDescriptor
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

  // 1bis. URL query params — pour Phase 2b. Lecture via useRouterState pour
  // qu'on bénéficie du re-render TanStack Router quand l'URL change.
  const urlSearch = useRouterState({
    select: (s) => s.location.search as Record<string, unknown>,
  })

  // 2. sessionStorage — fallback si state vide (post-F5)
  // Memoize sur matchId : la lecture est synchronome mais le résultat ne
  // change qu'avec le matchId (TTL purgé à la lecture).
  const storedCtx = useMemo<MatchNavContext | null>(
    () => (stateCtx ? null : readNavContext(matchId)),
    [stateCtx, matchId],
  )

  const localCtx = stateCtx ?? storedCtx ?? null

  // 3. Spec depuis URL query params si pas de ctx local avec matchIds
  const urlSpec = useMemo<MatchFilterSpec | null>(
    () => (localCtx ? null : parseFilterSpecFromSearch(urlSearch)),
    [localCtx, urlSearch],
  )

  // 4. Fallback API — toujours appelé (règle des hooks). Le spec URL est
  // passé pour que le backend filtre côté serveur. Si localCtx résoud, la
  // requête tourne mais son résultat n'est pas utilisé (cache pour ouverture
  // future).
  const apiQuery = useMatchNeighborsAPI(playerSlug, matchId, urlSpec)

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
        contextDescriptor: localCtx.contextDescriptor,
        navContext: localCtx,
      }
    }
    // matchId hors liste du contexte → fallback API. Observabilité (Phase 3) :
    // ce cas est anormal (le contexte devrait contenir le match courant) et
    // était auparavant totalement silencieux. On le signale en dev pour le
    // diagnostic, sans bloquer (le fallback API reste correct).
    if (import.meta.env.DEV) {
      console.warn(
        `[match-nav] matchId "${matchId}" absent du contexte (${localCtx.matchIds.length} matchs, source=${localCtx.source}) → fallback API Q25`,
      )
    }
  }

  return {
    data: apiQuery.data,
    isPending: apiQuery.isPending,
    source: 'api',
    // Si on a tapé l'API avec un spec URL, le data inclut applied_filters —
    // on l'expose via navContext synthétique pour que l'UI affiche un label.
    navContext: urlSpec
      ? { source: 'history', matchIds: [], filterSpec: urlSpec }
      : undefined,
  }
}
