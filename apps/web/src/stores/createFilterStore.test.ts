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

  it('autoSnapToLatestSession ne mute qu’un seul store (écrit le label, mémorise l’id)', () => {
    const storeA = createFilterStore({ name: nextName() })
    const storeB = createFilterStore({ name: nextName() })

    storeA.getState().autoSnapToLatestSession({ session_id: 'latest-A-id', label: 'L-A (5)' }, true)

    expect(storeA.getState().filterContext.filter_mode).toBe('sessions')
    // picked_sessions porte le LABEL (pas le session_id) — cf. backend escouade.
    expect(storeA.getState().filterContext.sessions?.picked_sessions).toEqual(['L-A (5)'])
    // lastKnownLatestSessionId reste le session_id (clé de détection stable).
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

  it('une mutation n’écrit JAMAIS dans l’URL, même pour un store urlEnabled', () => {
    // Le share-link n'est plus poussé automatiquement (fix pollution ?f=).
    const useStore = createFilterStore({ name: nextName(), urlEnabled: true, urlParam: 'f' })
    const before = window.location.search
    useStore.getState().setSessions({ picked_sessions: ['s1'], gap_minutes: 120 })
    useStore.getState().setPeriod({ start_date: '2025-01-01', end_date: '2025-01-31' })
    useStore.getState().setCascade({ experience_types: ['PVE'], playlists: [], modes: [], maps: [] })
    useStore.getState().resetFilters()
    expect(window.location.search).toBe(before)
  })

  it('buildShareUrl retourne null pour un store sans share-link (urlEnabled=false)', () => {
    const useStore = createFilterStore({ name: nextName(), urlEnabled: false })
    useStore.getState().setCascade({ experience_types: ['PVE'], playlists: [], modes: [], maps: [] })
    expect(useStore.getState().buildShareUrl()).toBeNull()
  })
})

// ---------------------------------------------------------------------------
// Garde deep-link ?f= par titre — fix fuite inter-titres au fresh-load.
// Le reset au switch de titre (applyActiveTitle) ne couvre PAS le fresh-load /
// bookmark : là, seul le titre estampillé dans ?f= + le titre résolu au
// bootstrap permettent de rejeter un filtre généré pour un AUTRE titre.
// ---------------------------------------------------------------------------

describe('createFilterStore — garde deep-link ?f= par titre', () => {
  let counter = 0
  const nextName = () => `levelup-deeplink-filter-${++counter}`
  const originalHref = window.location.href

  // Contexte de session (label purement temporel → title-agnostic, donc fuiterait
  // sans estampille de titre) utilisé comme deep-link « d'un autre titre ».
  const SESSION_CTX = {
    filter_mode: 'sessions' as const,
    period: { start_date: null, end_date: null },
    sessions: { picked_sessions: ['03/07/2026 18:32–18:57 (3)'], gap_minutes: 120 },
    cascade: { experience_types: [], playlists: [], modes: [], maps: [] },
  }

  function setUrl(param: string, payload: unknown) {
    const encoded = btoa(encodeURIComponent(JSON.stringify(payload)))
    window.history.replaceState(null, '', `/players/p/home?${param}=${encoded}`)
  }

  beforeEach(() => {
    localStorage.clear()
    window.history.replaceState(null, '', '/')
  })
  afterEach(() => {
    localStorage.clear()
    window.history.replaceState(null, '', originalHref)
  })

  it('hydrate le filtre depuis un deep-link enveloppe v2 et mémorise le titre estampillé', () => {
    setUrl('f', { t: 'halo_5', c: SESSION_CTX })
    const store = createFilterStore({
      name: nextName(), urlEnabled: true, urlParam: 'f', getActiveTitleSlug: () => 'halo_5',
    })
    expect(store.getState().filterContext.sessions?.picked_sessions).toEqual(['03/07/2026 18:32–18:57 (3)'])
    expect(store.getState().urlHydratedTitleSlug).toBe('halo_5')
  })

  it('décode un deep-link legacy (payload = ctx brut, sans titre) comme Halo Infinite', () => {
    setUrl('f', SESSION_CTX)
    const store = createFilterStore({ name: nextName(), urlEnabled: true, urlParam: 'f' })
    expect(store.getState().filterContext.filter_mode).toBe('sessions')
    expect(store.getState().urlHydratedTitleSlug).toBe('halo_infinite')
  })

  it('reconcileActiveTitle RESET le filtre si le titre du deep-link ≠ titre actif', () => {
    setUrl('f', { t: 'halo_5', c: SESSION_CTX })
    const store = createFilterStore({
      name: nextName(), urlEnabled: true, urlParam: 'f', getActiveTitleSlug: () => 'halo_infinite',
    })
    expect(store.getState().filterContext.filter_mode).toBe('sessions') // hydraté (avant reconcile)
    store.getState().reconcileActiveTitle('halo_infinite') // actif = infinite, deep-link = h5 → mismatch
    expect(store.getState().filterContext).toEqual(DEFAULT_FILTER_CONTEXT)
    expect(store.getState().urlHydratedTitleSlug).toBeNull() // consommé one-shot
  })

  it('reconcileActiveTitle CONSERVE le filtre si titre du deep-link == titre actif (non-régression share-link)', () => {
    setUrl('f', { t: 'halo_5', c: SESSION_CTX })
    const store = createFilterStore({
      name: nextName(), urlEnabled: true, urlParam: 'f', getActiveTitleSlug: () => 'halo_5',
    })
    store.getState().reconcileActiveTitle('halo_5')
    expect(store.getState().filterContext.sessions?.picked_sessions).toEqual(['03/07/2026 18:32–18:57 (3)'])
    expect(store.getState().urlHydratedTitleSlug).toBeNull()
  })

  it('deep-link legacy CONSERVÉ sur Halo Infinite (non-régression des shares existants)', () => {
    setUrl('f', SESSION_CTX)
    const store = createFilterStore({ name: nextName(), urlEnabled: true, urlParam: 'f' })
    store.getState().reconcileActiveTitle('halo_infinite')
    expect(store.getState().filterContext.filter_mode).toBe('sessions')
  })

  it('deep-link legacy RESET sur un autre titre (share Infinite-only chargé sur Halo 5)', () => {
    setUrl('f', SESSION_CTX)
    const store = createFilterStore({
      name: nextName(), urlEnabled: true, urlParam: 'f', getActiveTitleSlug: () => 'halo_5',
    })
    store.getState().reconcileActiveTitle('halo_5')
    expect(store.getState().filterContext).toEqual(DEFAULT_FILTER_CONTEXT)
  })

  it('reconcileActiveTitle no-op si le filtre ne vient pas d’un deep-link (localStorage seul)', () => {
    // Pas de ?f= → hydraté depuis localStorage/défaut → urlHydratedTitleSlug null.
    const store = createFilterStore({
      name: nextName(), urlEnabled: true, urlParam: 'f', getActiveTitleSlug: () => 'halo_5',
    })
    store.getState().setSessions({ picked_sessions: ['manuel'], gap_minutes: 120 })
    store.getState().reconcileActiveTitle('halo_infinite') // titre différent mais AUCUN deep-link
    expect(store.getState().filterContext.sessions?.picked_sessions).toEqual(['manuel'])
  })

  it('buildShareUrl estampille le titre actif dans ?f= (enveloppe v2) sans toucher l’URL', () => {
    const store = createFilterStore({
      name: nextName(), urlEnabled: true, urlParam: 'f', getActiveTitleSlug: () => 'halo_5',
    })
    store.getState().setSessions({ picked_sessions: ['s'], gap_minutes: 120 })
    // La mutation ne pousse rien dans l'URL...
    expect(new URL(window.location.href).searchParams.has('f')).toBe(false)
    // ...c'est buildShareUrl qui produit le lien à la demande.
    const shareUrl = store.getState().buildShareUrl()!
    const f = new URL(shareUrl).searchParams.get('f')!
    const payload = JSON.parse(decodeURIComponent(atob(f)))
    expect(payload.t).toBe('halo_5')
    expect(payload.c.sessions.picked_sessions).toEqual(['s'])
  })

  it('roundtrip buildShareUrl→decode préserve le contexte ET le titre', () => {
    const store = createFilterStore({
      name: nextName(), urlEnabled: true, urlParam: 'f', getActiveTitleSlug: () => 'halo_5',
    })
    store.getState().setCascade({ experience_types: ['PVE'], playlists: [], modes: [], maps: [] })
    // Simuler un partage : coller le lien généré dans le navigateur, puis fresh-load.
    const shareUrl = store.getState().buildShareUrl()!
    window.history.replaceState(null, '', shareUrl)
    const store2 = createFilterStore({
      name: nextName(), urlEnabled: true, urlParam: 'f', getActiveTitleSlug: () => 'halo_5',
    })
    expect(store2.getState().filterContext.cascade?.experience_types).toEqual(['PVE'])
    expect(store2.getState().urlHydratedTitleSlug).toBe('halo_5')
  })

  it('retire le param ?f= de l’URL après une hydratation réussie (one-shot)', () => {
    setUrl('f', { t: 'halo_5', c: SESSION_CTX })
    const store = createFilterStore({
      name: nextName(), urlEnabled: true, urlParam: 'f', getActiveTitleSlug: () => 'halo_5',
    })
    // Hydraté depuis l'URL...
    expect(store.getState().filterContext.sessions?.picked_sessions).toEqual(['03/07/2026 18:32–18:57 (3)'])
    // ...et l'URL a été nettoyée après consommation.
    expect(new URL(window.location.href).searchParams.has('f')).toBe(false)
  })

  it('laisse le param ?f= corrompu dans l’URL et retombe sur localStorage/défaut', () => {
    // ?f= non décodable (base64 invalide) : on NE strip PAS (décodage échoué).
    window.history.replaceState(null, '', '/players/p/home?f=@@@invalid@@@')
    const store = createFilterStore({
      name: nextName(), urlEnabled: true, urlParam: 'f', getActiveTitleSlug: () => 'halo_5',
    })
    expect(store.getState().filterContext).toEqual(DEFAULT_FILTER_CONTEXT)
    expect(store.getState().urlHydratedTitleSlug).toBeNull()
    // Param laissé intact (pas de strip sur décodage échoué).
    expect(new URL(window.location.href).searchParams.has('f')).toBe(true)
  })
})
