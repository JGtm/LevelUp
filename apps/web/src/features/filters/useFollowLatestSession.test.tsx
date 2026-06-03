/**
 * Tests — useFollowLatestSession.
 *
 * Vérifie l'atterrissage "follow-latest" piloté par l'état : snap sur la dernière
 * session tant que rien n'est épinglé, garde anti-boucle (déjà sur la dernière),
 * respect d'une sélection manuelle (rétention), re-snap à l'arrivée d'une nouvelle
 * session, isolation par scope solo/squad, et écriture du LABEL (pas du session_id).
 */
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { createFilterStore, type FilterStore } from '@/stores/createFilterStore'
import { useFollowLatestSession } from './queries'
import type { FilterContextResolved, SessionOption } from '@/lib/api/types'

let counter = 0
function makeStore(): FilterStore {
  return createFilterStore({ name: `levelup-test-follow-${++counter}`, urlEnabled: false })
}

function session(p: Pick<SessionOption, 'session_id' | 'label' | 'is_squad'>): SessionOption {
  return { session_id: p.session_id, label: p.label, match_count: 5, match_count_filtered: 5, is_squad: p.is_squad }
}

function resolved(all: SessionOption[]): FilterContextResolved {
  return {
    effective: {
      filter_mode: 'period',
      period: { start_date: null, end_date: null },
      sessions: { picked_sessions: [], gap_minutes: 120 },
      cascade: { experience_types: [], playlists: [], modes: [], maps: [] },
    },
    available_options: { experience_types: [], playlists: [], modes: [], maps: [] },
    session_options: { all_sessions: all, solo_labels: [], squad_labels: [] },
    counts: { total_matches_before_filters: 100, total_matches_after_filters: 50 },
    period_presets: [],
  }
}

describe('useFollowLatestSession', () => {
  beforeEach(() => localStorage.clear())

  it('snap initial (état vierge) → pose le LABEL de la dernière session solo + marque auto', () => {
    const store = makeStore()
    store.getState().setResolvedContext(
      resolved([
        session({ session_id: 's2', label: '06/04 (5)', is_squad: false }),
        session({ session_id: 's1', label: '05/04 (3)', is_squad: false }),
      ]),
    )
    renderHook(() => useFollowLatestSession('madina', store, 'solo'))
    const state = store.getState()
    expect(state.filterContext.filter_mode).toBe('sessions')
    expect(state.filterContext.sessions?.picked_sessions).toEqual(['06/04 (5)'])
    expect(state.lastKnownLatestSessionId).toBe('s2')
    expect(state.isAutoSnappingToLatest).toBe(true)
  })

  it('ne re-snappe pas si déjà sur la dernière (garde anti-boucle)', () => {
    const store = makeStore()
    store.getState().setResolvedContext(resolved([session({ session_id: 's2', label: '06/04 (5)', is_squad: false })]))
    // Mime l'état post-snap sans appeler autoSnap (qu'on espionne juste après).
    store.getState().setSessions({ picked_sessions: ['06/04 (5)'], gap_minutes: 120 })
    store.getState().setIsAutoSnappingToLatest(true)
    store.getState().setLastKnownLatestSessionId('s2')
    const hashBefore = store.getState().filterContextHash
    const spy = vi.spyOn(store.getState(), 'autoSnapToLatestSession')

    renderHook(() => useFollowLatestSession('madina', store, 'solo'))

    expect(spy).not.toHaveBeenCalled()
    expect(store.getState().filterContextHash).toBe(hashBefore)
    expect(store.getState().filterContext.sessions?.picked_sessions).toEqual(['06/04 (5)'])
  })

  it('respecte une sélection manuelle épinglée → pas de snap (rétention)', () => {
    const store = makeStore()
    store.getState().setResolvedContext(
      resolved([
        session({ session_id: 's2', label: '06/04 (5)', is_squad: false }),
        session({ session_id: 's1', label: '05/04 (3)', is_squad: false }),
      ]),
    )
    // L'utilisateur épingle une session ancienne → setSessions repasse isAutoSnapping=false.
    store.getState().setSessions({ picked_sessions: ['05/04 (3)'], gap_minutes: 120 })

    renderHook(() => useFollowLatestSession('madina', store, 'solo'))

    expect(store.getState().filterContext.sessions?.picked_sessions).toEqual(['05/04 (3)'])
    expect(store.getState().isAutoSnappingToLatest).toBe(false)
  })

  it('re-snappe quand une nouvelle session plus récente arrive', () => {
    const store = makeStore()
    store.getState().setResolvedContext(resolved([session({ session_id: 's1', label: '05/04 (3)', is_squad: false })]))
    renderHook(() => useFollowLatestSession('madina', store, 'solo'))
    expect(store.getState().filterContext.sessions?.picked_sessions).toEqual(['05/04 (3)'])

    act(() => {
      store.getState().setResolvedContext(
        resolved([
          session({ session_id: 's2', label: '06/04 (1)', is_squad: false }),
          session({ session_id: 's1', label: '05/04 (3)', is_squad: false }),
        ]),
      )
    })

    expect(store.getState().filterContext.sessions?.picked_sessions).toEqual(['06/04 (1)'])
    expect(store.getState().lastKnownLatestSessionId).toBe('s2')
  })

  it('scope squad → snappe sur la dernière session squad (LABEL)', () => {
    const store = makeStore()
    store.getState().setResolvedContext(
      resolved([
        session({ session_id: 's-solo', label: 'SOLO (5)', is_squad: false }),
        session({ session_id: 'sq2', label: 'SQUAD (4)', is_squad: true }),
        session({ session_id: 'sq1', label: 'SQUAD-old (2)', is_squad: true }),
      ]),
    )
    renderHook(() => useFollowLatestSession('madina', store, 'squad'))
    expect(store.getState().filterContext.sessions?.picked_sessions).toEqual(['SQUAD (4)'])
    expect(store.getState().lastKnownLatestSessionId).toBe('sq2')
  })

  it('no-op si pas de resolvedContext', () => {
    const store = makeStore()
    renderHook(() => useFollowLatestSession('madina', store, 'solo'))
    expect(store.getState().filterContext.sessions?.picked_sessions ?? []).toEqual([])
    expect(store.getState().isAutoSnappingToLatest).toBe(false)
  })

  it('no-op si aucune session du scope demandé', () => {
    const store = makeStore()
    store.getState().setResolvedContext(resolved([session({ session_id: 'sq', label: 'SQUAD (2)', is_squad: true })]))
    renderHook(() => useFollowLatestSession('madina', store, 'solo')) // aucune session solo
    expect(store.getState().filterContext.sessions?.picked_sessions ?? []).toEqual([])
  })

  it('tolère un session_id legacy dans picked_sessions (pas de re-snap, resync lastKnown)', () => {
    const store = makeStore()
    store.getState().setResolvedContext(resolved([session({ session_id: 's2', label: '06/04 (5)', is_squad: false })]))
    // État hérité : picked contient le session_id (pas le label), suivi actif.
    store.getState().setSessions({ picked_sessions: ['s2'], gap_minutes: 120 })
    store.getState().setIsAutoSnappingToLatest(true)
    const spy = vi.spyOn(store.getState(), 'autoSnapToLatestSession')

    renderHook(() => useFollowLatestSession('madina', store, 'solo'))

    expect(spy).not.toHaveBeenCalled()
    expect(store.getState().lastKnownLatestSessionId).toBe('s2')
  })
})
