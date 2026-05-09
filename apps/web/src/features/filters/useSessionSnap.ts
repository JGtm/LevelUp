/**
 * useSessionSnap — hook partagé /stats et /squad pour le snap automatique
 * sur la dernière session du kind ciblé.
 *
 * Politique unifiée (cf. PersonalStatsLayout / SquadLayout) :
 *  - **Nouvelle session du kind détectée** (latest !== tracker store) →
 *    reset TOTAL : cascade + période + sessions wipées, snap sur la nouvelle.
 *    Inconditionnel — même une période user-set est effacée. Justification :
 *    consultation typique = post-jeu, vue d'ensemble fraîche.
 *  - **Pas de nouvelle session** → respect intégral des filtres user.
 *    - Période custom posée → no-op (cascade + période + session préservées).
 *    - Sélection courante valide pour ce kind → no-op (cascade préservée).
 *    - Fallback (jamais hydraté ou sélection d'un autre kind) → snap en
 *      **préservant la cascade** (seule la session change, période vidée par
 *      exclusivité).
 *
 * Le hook utilise `applySessionLabels`-like logic via `setFilterContext`
 * direct (LABEL côté `picked_sessions` pour matcher SessionMultiSelect).
 * Il ne touche pas le state local `pickedSessionLabels` du layout — le
 * layout sync via son propre useEffect post-mount sur
 * `filterContext.sessions.picked_sessions`.
 *
 * Logging : événements `session_snap:*` via `features/filters/_logger.ts`.
 */
import { useEffect } from 'react'

import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import {
  DEFAULT_CASCADE,
  DEFAULT_PERIOD,
  DEFAULT_SESSIONS,
} from '@/components/shell/FilterOmnibar'
import type { SessionOption } from '@/lib/api/types'

import { log } from './_logger'

export interface UseSessionSnapParams {
  /**
   * Liste des sessions du kind ciblé, triée DESC (plus récente en tête).
   * Le caller pré-filtre par `is_squad === false` (solo) ou `=== true` (squad).
   */
  sessions: SessionOption[]
}

export function useSessionSnap({ sessions }: UseSessionSnapParams): void {
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const setFilterContext = useGlobalFilterStore((s) => s.setFilterContext)
  const lastKnown = useGlobalFilterStore((s) => s.lastKnownLatestSessionId)
  const setLastKnown = useGlobalFilterStore((s) => s.setLastKnownLatestSessionId)

  useEffect(() => {
    const latest = sessions[0]
    if (!latest) return

    const isFresh = lastKnown !== null && latest.session_id !== lastKnown

    // Nouvelle session du kind → reset TOTAL inconditionnel.
    if (isFresh) {
      setFilterContext({
        filter_mode: 'sessions',
        cascade: DEFAULT_CASCADE,
        period: DEFAULT_PERIOD,
        sessions: {
          ...DEFAULT_SESSIONS,
          picked_sessions: [latest.label],
        },
      })
      setLastKnown(latest.session_id)
      log.debug(`session_snap:fresh session=${latest.session_id}`)
      return
    }

    // Mode "respect des filtres user".
    const hasPeriod = !!(filterContext.period?.start_date || filterContext.period?.end_date)
    if (hasPeriod) return

    const currentPicked = filterContext.sessions?.picked_sessions ?? []
    const isCurrent =
      currentPicked.length > 0
      && sessions.some(
        (s) => currentPicked.includes(s.label) || currentPicked.includes(s.session_id),
      )
    if (isCurrent) return

    // Fallback : jamais hydraté OU sélection d'un autre kind → snap en
    // préservant la cascade.
    setFilterContext({
      ...filterContext,
      filter_mode: 'sessions',
      period: DEFAULT_PERIOD,
      sessions: {
        ...DEFAULT_SESSIONS,
        picked_sessions: [latest.label],
      },
    })
    setLastKnown(latest.session_id)
    log.debug(`session_snap:fallback session=${latest.session_id}`)
  }, [sessions, filterContext, setFilterContext, lastKnown, setLastKnown])
}
