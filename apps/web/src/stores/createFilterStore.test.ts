/**
 * Tests unitaires — createFilterStore factory.
 *
 * Vérifie le contrat clé de la factory : deux stores instanciés sont isolés
 * (clés localStorage distinctes, mutations de l'un n'affectent pas l'autre).
 * C'est l'invariant qui permet la séparation solo/squad sans pollution.
 */
import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { createFilterStore, DEFAULT_FILTER_CONTEXT } from './createFilterStore'

describe('createFilterStore', () => {
  // Compteur pour générer des noms uniques par test (évite la collision avec
  // les stores réels solo/squad de l'app et entre tests).
  let storeCounter = 0
  const nextName = () => `levelup-test-filter-${++storeCounter}`

  beforeEach(() => {
    localStorage.clear()
  })

  afterEach(() => {
    localStorage.clear()
  })

  it('initialise un nouveau store avec DEFAULT_FILTER_CONTEXT', () => {
    const useStore = createFilterStore({ name: nextName() })
    const state = useStore.getState()
    expect(state.filterContext).toEqual(DEFAULT_FILTER_CONTEXT)
    expect(state.lastKnownLatestSessionId).toBeNull()
    expect(state.isAutoSnappingToLatest).toBe(false)
  })

  it('persiste dans la clé localStorage fournie', () => {
    const name = nextName()
    const useStore = createFilterStore({ name })
    useStore.getState().setCascade({
      experience_types: ['PVP classé'],
      playlists: [],
      modes: [],
      maps: [],
    })
    const stored = localStorage.getItem(name)
    expect(stored).not.toBeNull()
    const parsed = JSON.parse(stored!)
    expect(parsed.state.filterContext.cascade.experience_types).toContain('PVP classé')
  })

  it('deux stores instanciés sont isolés (clés et state distincts)', () => {
    const nameA = nextName()
    const nameB = nextName()
    const storeA = createFilterStore({ name: nameA })
    const storeB = createFilterStore({ name: nameB })

    // Mutation isolée sur A
    storeA.getState().setCascade({
      experience_types: ['PVP classé'],
      playlists: [],
      modes: [],
      maps: [],
    })

    // B reste à DEFAULT
    expect(storeB.getState().filterContext.cascade?.experience_types ?? []).toEqual([])

    // A et B persistent dans des clés distinctes
    expect(localStorage.getItem(nameA)).not.toBeNull()
    // B n'est pas encore persisté (Zustand persist écrit uniquement après mutation)
    storeB.getState().setLastKnownLatestSessionId('session-from-B')
    expect(localStorage.getItem(nameB)).not.toBeNull()

    // Les deux états sont indépendants
    const stateA = JSON.parse(localStorage.getItem(nameA)!).state
    const stateB = JSON.parse(localStorage.getItem(nameB)!).state
    expect(stateA.filterContext.cascade.experience_types).toEqual(['PVP classé'])
    expect(stateA.lastKnownLatestSessionId).toBeNull()
    expect(stateB.filterContext.cascade.experience_types).toEqual([])
    expect(stateB.lastKnownLatestSessionId).toBe('session-from-B')
  })

  it('autoSnapToLatestSession ne mute qu’un seul store', () => {
    const storeA = createFilterStore({ name: nextName() })
    const storeB = createFilterStore({ name: nextName() })

    storeA.getState().autoSnapToLatestSession('latest-A-id', true)

    expect(storeA.getState().filterContext.filter_mode).toBe('sessions')
    expect(storeA.getState().filterContext.sessions?.picked_sessions).toEqual(['latest-A-id'])
    expect(storeA.getState().lastKnownLatestSessionId).toBe('latest-A-id')
    expect(storeA.getState().isAutoSnappingToLatest).toBe(true)

    // B n'a pas bougé
    expect(storeB.getState().filterContext.filter_mode).toBe('period')
    expect(storeB.getState().filterContext.sessions?.picked_sessions ?? []).toEqual([])
    expect(storeB.getState().lastKnownLatestSessionId).toBeNull()
    expect(storeB.getState().isAutoSnappingToLatest).toBe(false)
  })

  it('resetFilters ne mute qu’un seul store', () => {
    const storeA = createFilterStore({ name: nextName() })
    const storeB = createFilterStore({ name: nextName() })

    storeA.getState().setCascade({
      experience_types: ['PVP classé'],
      playlists: [],
      modes: [],
      maps: [],
    })
    storeB.getState().setCascade({
      experience_types: ['PVE'],
      playlists: [],
      modes: [],
      maps: [],
    })

    storeA.getState().resetFilters()

    expect(storeA.getState().filterContext.cascade?.experience_types ?? []).toEqual([])
    // B inchangé
    expect(storeB.getState().filterContext.cascade?.experience_types).toEqual(['PVE'])
  })

  it('urlEnabled=false n’écrit pas dans la querystring', () => {
    const name = nextName()
    const useStore = createFilterStore({ name, urlEnabled: false })

    const before = window.location.search
    useStore.getState().setCascade({
      experience_types: ['PVP classé'],
      playlists: [],
      modes: [],
      maps: [],
    })
    expect(window.location.search).toBe(before)
  })
})
