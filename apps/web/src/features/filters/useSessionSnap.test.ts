/**
 * Tests unitaires — useSessionSnap.
 *
 * Couvre la politique partagée /stats et /squad :
 *   - Nouvelle session du kind détectée → reset TOTAL (cascade + période +
 *     sessions wipées) puis snap, inconditionnel.
 *   - Pas de nouvelle session :
 *     - Période user-set → no-op.
 *     - Sélection courante valide pour le kind → no-op.
 *     - Fallback (jamais hydraté ou autre kind) → snap en préservant la
 *       cascade.
 */
import { describe, it, expect, beforeEach } from 'vitest'
import { renderHook } from '@testing-library/react'

import { useGlobalFilterStore, DEFAULT_GAP_MINUTES } from '@/stores/globalFilterStore'
import type { CascadeInput, SessionOption } from '@/lib/api/types'

import { useSessionSnap } from './useSessionSnap'

const SOLO_LATEST: SessionOption = {
  session_id: 'solo-latest',
  label: '07/05 22:00',
  match_count: 5,
  match_count_filtered: 5,
  is_squad: false,
}
const SOLO_OLD: SessionOption = {
  session_id: 'solo-old',
  label: '06/05 19:00',
  match_count: 4,
  match_count_filtered: 4,
  is_squad: false,
}
const SQUAD_LATEST: SessionOption = {
  session_id: 'squad-latest',
  label: '07/05 21:00',
  match_count: 6,
  match_count_filtered: 6,
  is_squad: true,
}

const NON_DEFAULT_CASCADE: CascadeInput = {
  experience_types: ['pvp'],
  playlists: ['ranked'],
  modes: [],
  maps: [],
}

function resetStore() {
  useGlobalFilterStore.getState().resetFilters()
  useGlobalFilterStore.getState().setLastKnownLatestSessionId(null)
}

describe('useSessionSnap', () => {
  beforeEach(() => {
    resetStore()
  })

  it('sessions vide → no-op (ne snap pas, ne touche pas le store)', () => {
    const before = useGlobalFilterStore.getState().filterContext
    renderHook(() => useSessionSnap({ sessions: [] }))
    const after = useGlobalFilterStore.getState().filterContext
    expect(after).toEqual(before)
  })

  it('jamais hydraté + pas de filtre → snap fallback sur latest, cascade DEFAULT préservée', () => {
    renderHook(() => useSessionSnap({ sessions: [SOLO_LATEST, SOLO_OLD] }))
    const state = useGlobalFilterStore.getState()
    expect(state.filterContext.filter_mode).toBe('sessions')
    expect(state.filterContext.sessions!.picked_sessions).toEqual([SOLO_LATEST.label])
    expect(state.lastKnownLatestSessionId).toBe(SOLO_LATEST.session_id)
  })

  it('jamais hydraté + cascade user posée → snap fallback préserve la cascade', () => {
    useGlobalFilterStore.getState().setCascade(NON_DEFAULT_CASCADE)
    renderHook(() => useSessionSnap({ sessions: [SOLO_LATEST] }))
    const state = useGlobalFilterStore.getState()
    expect(state.filterContext.sessions!.picked_sessions).toEqual([SOLO_LATEST.label])
    expect(state.filterContext.cascade!.playlists).toEqual(['ranked'])
    expect(state.filterContext.cascade!.experience_types).toEqual(['pvp'])
  })

  it('nouvelle session détectée → reset TOTAL : cascade + période + sessions wipées', () => {
    // Setup : tracker hydraté sur ancienne session, cascade et période posées
    useGlobalFilterStore.getState().setLastKnownLatestSessionId(SOLO_OLD.session_id)
    useGlobalFilterStore.getState().setCascade(NON_DEFAULT_CASCADE)
    useGlobalFilterStore.getState().setPeriod({ start_date: '2026-04-01', end_date: '2026-04-30' })

    renderHook(() => useSessionSnap({ sessions: [SOLO_LATEST, SOLO_OLD] }))

    const state = useGlobalFilterStore.getState()
    expect(state.filterContext.sessions!.picked_sessions).toEqual([SOLO_LATEST.label])
    expect(state.filterContext.cascade!.playlists).toEqual([]) // wipé
    expect(state.filterContext.cascade!.experience_types).toEqual([]) // wipé
    expect(state.filterContext.period!.start_date).toBeNull() // wipé
    expect(state.filterContext.period!.end_date).toBeNull() // wipé
    expect(state.lastKnownLatestSessionId).toBe(SOLO_LATEST.session_id)
  })

  it('pas de nouvelle session + période custom posée → no-op (cascade + période préservées)', () => {
    useGlobalFilterStore.getState().setLastKnownLatestSessionId(SOLO_LATEST.session_id)
    useGlobalFilterStore.getState().setCascade(NON_DEFAULT_CASCADE)
    useGlobalFilterStore.getState().setPeriod({ start_date: '2026-04-01', end_date: '2026-04-30' })

    const beforeCtx = useGlobalFilterStore.getState().filterContext
    renderHook(() => useSessionSnap({ sessions: [SOLO_LATEST, SOLO_OLD] }))
    const afterCtx = useGlobalFilterStore.getState().filterContext

    expect(afterCtx).toEqual(beforeCtx) // strictly no change
  })

  it('pas de nouvelle session + sélection courante valide → no-op (cascade préservée)', () => {
    useGlobalFilterStore.getState().setLastKnownLatestSessionId(SOLO_LATEST.session_id)
    useGlobalFilterStore.getState().setCascade(NON_DEFAULT_CASCADE)
    useGlobalFilterStore.getState().setSessions({
      picked_sessions: [SOLO_OLD.label],
      gap_minutes: DEFAULT_GAP_MINUTES,
    })

    const beforeCtx = useGlobalFilterStore.getState().filterContext
    renderHook(() => useSessionSnap({ sessions: [SOLO_LATEST, SOLO_OLD] }))
    const afterCtx = useGlobalFilterStore.getState().filterContext

    expect(afterCtx.sessions!.picked_sessions).toEqual([SOLO_OLD.label])
    expect(afterCtx.cascade!.playlists).toEqual(['ranked'])
    expect(afterCtx).toEqual(beforeCtx)
  })

  it('sélection courante invalide (hors liste) → snap fallback préserve cascade', () => {
    useGlobalFilterStore.getState().setLastKnownLatestSessionId(SOLO_LATEST.session_id)
    useGlobalFilterStore.getState().setCascade(NON_DEFAULT_CASCADE)
    // Sélection sur une session absente de la liste passée au hook.
    useGlobalFilterStore.getState().setSessions({
      picked_sessions: [SQUAD_LATEST.label],
      gap_minutes: DEFAULT_GAP_MINUTES,
    })

    renderHook(() => useSessionSnap({ sessions: [SOLO_LATEST, SOLO_OLD] }))

    const state = useGlobalFilterStore.getState()
    expect(state.filterContext.sessions!.picked_sessions).toEqual([SOLO_LATEST.label])
    expect(state.filterContext.cascade!.playlists).toEqual(['ranked'])
  })
})
