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
  useGlobalFilterStore.getState().setLastKnownLatestSoloSessionId(null)
  useGlobalFilterStore.getState().setLastKnownLatestSquadSessionId(null)
}

describe('useSessionSnap', () => {
  beforeEach(() => {
    resetStore()
  })

  it('sessions vide → no-op (ne snap pas, ne touche pas le store)', () => {
    const before = useGlobalFilterStore.getState().filterContext
    renderHook(() => useSessionSnap({ sessions: [], kind: 'solo' }))
    const after = useGlobalFilterStore.getState().filterContext
    expect(after).toEqual(before)
  })

  it('jamais hydraté + pas de filtre → snap fallback sur latest solo, cascade DEFAULT préservée', () => {
    renderHook(() => useSessionSnap({ sessions: [SOLO_LATEST, SOLO_OLD], kind: 'solo' }))
    const state = useGlobalFilterStore.getState()
    expect(state.filterContext.filter_mode).toBe('sessions')
    expect(state.filterContext.sessions!.picked_sessions).toEqual([SOLO_LATEST.label])
    expect(state.lastKnownLatestSoloSessionId).toBe(SOLO_LATEST.session_id)
  })

  it('jamais hydraté + cascade user posée → snap fallback préserve la cascade', () => {
    useGlobalFilterStore.getState().setCascade(NON_DEFAULT_CASCADE)
    renderHook(() => useSessionSnap({ sessions: [SOLO_LATEST], kind: 'solo' }))
    const state = useGlobalFilterStore.getState()
    expect(state.filterContext.sessions!.picked_sessions).toEqual([SOLO_LATEST.label])
    expect(state.filterContext.cascade!.playlists).toEqual(['ranked'])
    expect(state.filterContext.cascade!.experience_types).toEqual(['pvp'])
  })

  it('nouvelle session solo détectée → reset TOTAL : cascade + période + sessions wipées', () => {
    // Setup : tracker hydraté sur ancienne session, cascade et période posées
    useGlobalFilterStore.getState().setLastKnownLatestSoloSessionId(SOLO_OLD.session_id)
    useGlobalFilterStore.getState().setCascade(NON_DEFAULT_CASCADE)
    useGlobalFilterStore.getState().setPeriod({ start_date: '2026-04-01', end_date: '2026-04-30' })

    renderHook(() => useSessionSnap({ sessions: [SOLO_LATEST, SOLO_OLD], kind: 'solo' }))

    const state = useGlobalFilterStore.getState()
    expect(state.filterContext.sessions!.picked_sessions).toEqual([SOLO_LATEST.label])
    expect(state.filterContext.cascade!.playlists).toEqual([]) // wipé
    expect(state.filterContext.cascade!.experience_types).toEqual([]) // wipé
    expect(state.filterContext.period!.start_date).toBeNull() // wipé
    expect(state.filterContext.period!.end_date).toBeNull() // wipé
    expect(state.lastKnownLatestSoloSessionId).toBe(SOLO_LATEST.session_id)
  })

  it('pas de nouvelle session + période custom posée → no-op (cascade + période préservées)', () => {
    useGlobalFilterStore.getState().setLastKnownLatestSoloSessionId(SOLO_LATEST.session_id)
    useGlobalFilterStore.getState().setCascade(NON_DEFAULT_CASCADE)
    useGlobalFilterStore.getState().setPeriod({ start_date: '2026-04-01', end_date: '2026-04-30' })

    const beforeCtx = useGlobalFilterStore.getState().filterContext
    renderHook(() => useSessionSnap({ sessions: [SOLO_LATEST, SOLO_OLD], kind: 'solo' }))
    const afterCtx = useGlobalFilterStore.getState().filterContext

    expect(afterCtx).toEqual(beforeCtx) // strictly no change
  })

  it('pas de nouvelle session + sélection solo courante valide → no-op (cascade préservée)', () => {
    useGlobalFilterStore.getState().setLastKnownLatestSoloSessionId(SOLO_LATEST.session_id)
    useGlobalFilterStore.getState().setCascade(NON_DEFAULT_CASCADE)
    useGlobalFilterStore.getState().setSessions({
      picked_sessions: [SOLO_OLD.label],
      gap_minutes: DEFAULT_GAP_MINUTES,
    })

    const beforeCtx = useGlobalFilterStore.getState().filterContext
    renderHook(() => useSessionSnap({ sessions: [SOLO_LATEST, SOLO_OLD], kind: 'solo' }))
    const afterCtx = useGlobalFilterStore.getState().filterContext

    // Sélection solo "old" valide → on respecte. Cascade préservée.
    expect(afterCtx.sessions!.picked_sessions).toEqual([SOLO_OLD.label])
    expect(afterCtx.cascade!.playlists).toEqual(['ranked'])
    expect(afterCtx).toEqual(beforeCtx)
  })

  it('sélection courante d\'un autre kind (squad sur /stats) → snap fallback préserve cascade', () => {
    useGlobalFilterStore.getState().setLastKnownLatestSoloSessionId(SOLO_LATEST.session_id)
    useGlobalFilterStore.getState().setCascade(NON_DEFAULT_CASCADE)
    // Sélection sur la session squad — pas dans la liste solo passée au hook.
    useGlobalFilterStore.getState().setSessions({
      picked_sessions: [SQUAD_LATEST.label],
      gap_minutes: DEFAULT_GAP_MINUTES,
    })

    renderHook(() => useSessionSnap({ sessions: [SOLO_LATEST, SOLO_OLD], kind: 'solo' }))

    const state = useGlobalFilterStore.getState()
    expect(state.filterContext.sessions!.picked_sessions).toEqual([SOLO_LATEST.label])
    // Fallback préserve la cascade (pas de nouvelle session, juste fix de cohérence)
    expect(state.filterContext.cascade!.playlists).toEqual(['ranked'])
  })

  it('kind=squad : snap met à jour le tracker squad uniquement', () => {
    useGlobalFilterStore.getState().setLastKnownLatestSoloSessionId('solo-X')

    renderHook(() => useSessionSnap({ sessions: [SQUAD_LATEST], kind: 'squad' }))

    const state = useGlobalFilterStore.getState()
    expect(state.filterContext.sessions!.picked_sessions).toEqual([SQUAD_LATEST.label])
    expect(state.lastKnownLatestSquadSessionId).toBe(SQUAD_LATEST.session_id)
    // Tracker solo intact (n'a pas été touché par un snap squad)
    expect(state.lastKnownLatestSoloSessionId).toBe('solo-X')
  })

  it('nouvelle session squad détectée → reset TOTAL pour le kind squad', () => {
    useGlobalFilterStore.getState().setLastKnownLatestSquadSessionId('squad-prev')
    useGlobalFilterStore.getState().setCascade(NON_DEFAULT_CASCADE)
    useGlobalFilterStore.getState().setPeriod({ start_date: '2026-04-01', end_date: '2026-04-30' })

    renderHook(() => useSessionSnap({ sessions: [SQUAD_LATEST], kind: 'squad' }))

    const state = useGlobalFilterStore.getState()
    expect(state.filterContext.sessions!.picked_sessions).toEqual([SQUAD_LATEST.label])
    expect(state.filterContext.cascade!.playlists).toEqual([])
    expect(state.filterContext.period!.start_date).toBeNull()
    expect(state.lastKnownLatestSquadSessionId).toBe(SQUAD_LATEST.session_id)
  })
})
