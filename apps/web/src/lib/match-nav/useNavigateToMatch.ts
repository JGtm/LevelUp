/**
 * useNavigateToMatch — hook unique pour ouvrir la page d'un match en
 * propageant un contexte de navigation chaînée.
 *
 * Phase 2a : router state + sessionStorage.
 * Phase 2b : ajout sérialisation `filterSpec` en URL query params pour
 *            survivre Ctrl+Click / lien partagé / nouvel onglet.
 *
 * Usage type :
 *   const navigateToMatch = useNavigateToMatch(playerSlug)
 *   navigateToMatch(row.match_id, {
 *     source: 'history',
 *     matchIds: rowsOnPage.map(r => r.match_id),
 *     filtersLabel: t.activeFiltersSummary,
 *     filterSpec: { playlist_names: ['Ranked Arena'], date_from: '...' },
 *   })
 *
 * Sans `ctx`, comportement identique à un `navigate({ to, params })` simple.
 */
import { useCallback } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { useTitleSlug } from '@/lib/title-routing'

import {
  filterSpecToQueryString,
  persistNavContext,
  type MatchNavContext,
} from './navContext'

/**
 * Convertit un MatchFilterSpec en object `search` compatible TanStack Router.
 * Pas de filterSpec ou spec vide → undefined (URL non polluée).
 */
function searchFromFilterSpec(ctx?: MatchNavContext): Record<string, string> | undefined {
  if (!ctx?.filterSpec) return undefined
  const qs = filterSpecToQueryString(ctx.filterSpec)
  if (!qs) return undefined
  return Object.fromEntries(new URLSearchParams(qs))
}

export function useNavigateToMatch(playerSlug: string) {
  const navigate = useNavigate()
  const titleSlug = useTitleSlug()

  return useCallback(
    (matchId: string, ctx?: MatchNavContext) => {
      if (!matchId) return
      if (ctx?.matchIds?.length) {
        persistNavContext(matchId, ctx)
      }
      void navigate({
        to: '/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId',
        params: { titleSlug, playerSlug, matchId },
        // Phase 2b : sérialisation filterSpec en query params pour que la
        // navigation contextuelle survive Ctrl+Click / lien partagé. Si pas
        // de filterSpec, search reste undefined (URL non polluée).
        search: searchFromFilterSpec(ctx) as never,
        // Router state : matchNavContext entier (matchIds + filtersLabel +
        // filterSpec). Plus rapide que le parsing URL au mount.
        state: ctx
          ? ((prev: Record<string, unknown>) => ({
              ...prev,
              matchNavContext: ctx,
            })) as never
          : undefined,
      })
    },
    [navigate, titleSlug, playerSlug],
  )
}
