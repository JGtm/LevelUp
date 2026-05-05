/**
 * useNavigateToMatch — hook unique pour ouvrir la page d'un match en
 * propageant un contexte de navigation chaînée (Phase 2a).
 *
 * Usage type :
 *   const navigateToMatch = useNavigateToMatch(playerSlug)
 *   navigateToMatch(row.match_id, {
 *     source: 'history',
 *     matchIds: rowsOnPage.map(r => r.match_id),
 *     filtersLabel: t.activeFiltersSummary,
 *   })
 *
 * Sans `ctx`, comportement identique à un `navigate({ to, params })` simple.
 *
 * Le contexte est :
 *   - poussé dans le router state (instantané, scope onglet)
 *   - sauvegardé dans sessionStorage (survit F5 / nav arrière)
 *   - (Phase 2b) sera également sérialisé en query params
 */
import { useCallback } from 'react'
import { useNavigate } from '@tanstack/react-router'

import { persistNavContext, type MatchNavContext } from './navContext'

export function useNavigateToMatch(playerSlug: string) {
  const navigate = useNavigate()

  return useCallback(
    (matchId: string, ctx?: MatchNavContext) => {
      if (!matchId) return
      if (ctx?.matchIds?.length) {
        persistNavContext(matchId, ctx)
      }
      void navigate({
        to: '/players/$playerSlug/matches/$matchId',
        params: { playerSlug, matchId },
        // Le router state est typé `HistoryState` côté TanStack Router.
        // On y range notre propre clé `matchNavContext`. La lecture côté
        // useMatchNeighbors fait le cast inverse.
        state: ctx
          ? ((prev: Record<string, unknown>) => ({
              ...prev,
              matchNavContext: ctx,
            })) as never
          : undefined,
      })
    },
    [navigate, playerSlug],
  )
}
