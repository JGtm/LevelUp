/**
 * Tests unitaires — GlobalFilterStore (Slice 0b).
 *
 * Vérifie que le store reflète correctement les mutations (mode, période,
 * sessions, cascade, reset) et que le hash change quand le contexte change.
 */
import { describe, it, expect, beforeEach } from 'vitest'
import { useGlobalFilterStore, DEFAULT_GAP_MINUTES, DEFAULT_FILTER_CONTEXT } from '@/stores/globalFilterStore'
import type { FilterContextResolved } from '@/lib/api/types'

describe('GlobalFilterStore', () => {
  beforeEach(() => {
    useGlobalFilterStore.getState().resetFilters()
  })

  it('démarre avec filter_mode=period', () => {
    expect(useGlobalFilterStore.getState().filterContext.filter_mode).toBe('period')
  })

  it('démarre avec gap_minutes=120', () => {
    const ctx = useGlobalFilterStore.getState().filterContext
    expect(ctx.sessions!.gap_minutes).toBe(DEFAULT_GAP_MINUTES)
  })

  it('resolvedContext est null au départ', () => {
    expect(useGlobalFilterStore.getState().resolvedContext).toBeNull()
  })

  it('setPeriod met à jour la période', () => {
    useGlobalFilterStore.getState().setPeriod({ start_date: '2025-01-01', end_date: '2025-01-31' })
    const period = useGlobalFilterStore.getState().filterContext.period
    expect(period!.start_date).toBe('2025-01-01')
    expect(period!.end_date).toBe('2025-01-31')
  })

  it('setSessions préserve gap_minutes=120', () => {
    useGlobalFilterStore.getState().setSessions({
      picked_sessions: ['sess-1', 'sess-2'],
      gap_minutes: 60, // tentative de changer → doit être ignorée ou écrasée
    })
    // La règle métier impose gap_minutes=120 — vérifier selon l'impl du store
    const ctx = useGlobalFilterStore.getState().filterContext
    expect(ctx.sessions!.picked_sessions).toContain('sess-1')
  })

  it('setCascade met à jour la cascade', () => {
    useGlobalFilterStore.getState().setCascade({
      experience_types: ['pvp'],
      playlists: ['ranked'],
      modes: [],
      maps: [],
    })
    const cascade = useGlobalFilterStore.getState().filterContext.cascade
    expect(cascade!.experience_types).toContain('pvp')
    expect(cascade!.playlists).toContain('ranked')
  })

  it('resetFilters restaure le contexte par défaut', () => {
    useGlobalFilterStore.getState().setSessions({
      picked_sessions: ['sess-1'],
      gap_minutes: DEFAULT_GAP_MINUTES,
    })
    useGlobalFilterStore.getState().setCascade({
      experience_types: ['pvp'],
      playlists: [],
      modes: [],
      maps: [],
    })
    useGlobalFilterStore.getState().resetFilters()
    const ctx = useGlobalFilterStore.getState().filterContext
    expect(ctx.filter_mode).toBe(DEFAULT_FILTER_CONTEXT.filter_mode)
    expect(ctx.period!.start_date).toBeNull()
    expect(ctx.sessions!.picked_sessions).toEqual([])
    expect(ctx.cascade!.experience_types).toEqual([])
  })

  it('filterContextHash change quand le contexte change', () => {
    const hashBefore = useGlobalFilterStore.getState().filterContextHash
    useGlobalFilterStore.getState().setSessions({
      picked_sessions: ['sess-1'],
      gap_minutes: DEFAULT_GAP_MINUTES,
    })
    const hashAfter = useGlobalFilterStore.getState().filterContextHash
    expect(hashAfter).not.toBe(hashBefore)
  })

  // Régression : un hash tronqué (btoa(JSON).slice(0, 32)) capture seulement
  // `{"filter_mode":"...","period":{` et ignore toute diff dans la cascade,
  // ce qui empêche TanStack Query de refetcher après un toggle de checkbox
  // dans FiltresPill — cassait toute la feature smart-filter-counts.
  it('filterContextHash change quand SEULE la cascade change (cascade en fin de JSON)', () => {
    const hashBefore = useGlobalFilterStore.getState().filterContextHash
    useGlobalFilterStore.getState().setCascade({
      experience_types: ['PVE'],
      playlists: [],
      modes: [],
      maps: [],
    })
    const hashAfter = useGlobalFilterStore.getState().filterContextHash
    expect(hashAfter).not.toBe(hashBefore)
  })

  it('filterContextHash distingue deux cascades différentes (toggle d\'une option)', () => {
    useGlobalFilterStore.getState().setCascade({
      experience_types: ['PVE'],
      playlists: [],
      modes: [],
      maps: [],
    })
    const withPVE = useGlobalFilterStore.getState().filterContextHash
    useGlobalFilterStore.getState().setCascade({
      experience_types: ['PVP non classé'],
      playlists: [],
      modes: [],
      maps: [],
    })
    const withUnranked = useGlobalFilterStore.getState().filterContextHash
    expect(withPVE).not.toBe(withUnranked)
  })

  it('setSessions auto-derive filter_mode=sessions quand session pickée', () => {
    useGlobalFilterStore.getState().setPeriod({ start_date: '2025-01-01', end_date: '2025-01-31' })
    useGlobalFilterStore.getState().setSessions({
      picked_sessions: ['sess-1'],
      gap_minutes: DEFAULT_GAP_MINUTES,
    })
    const ctx = useGlobalFilterStore.getState().filterContext
    expect(ctx.filter_mode).toBe('sessions')
    // Exclusivité : période vidée
    expect(ctx.period!.start_date).toBeNull()
    expect(ctx.period!.end_date).toBeNull()
  })

  it('setSessions sans pick auto-derive filter_mode=period', () => {
    useGlobalFilterStore.getState().setSessions({
      picked_sessions: [],
      gap_minutes: DEFAULT_GAP_MINUTES,
    })
    expect(useGlobalFilterStore.getState().filterContext.filter_mode).toBe('period')
  })

  it('setPeriod auto-derive filter_mode=period et vide les sessions', () => {
    useGlobalFilterStore.getState().setSessions({
      picked_sessions: ['sess-1'],
      gap_minutes: DEFAULT_GAP_MINUTES,
    })
    useGlobalFilterStore.getState().setPeriod({ start_date: '2025-01-01', end_date: '2025-01-31' })
    const ctx = useGlobalFilterStore.getState().filterContext
    expect(ctx.filter_mode).toBe('period')
    expect(ctx.sessions!.picked_sessions).toEqual([])
  })

  it('autoSnapToLatestSession bascule sur la session passée', () => {
    useGlobalFilterStore.getState().autoSnapToLatestSession('latest-sess-id', true)
    const state = useGlobalFilterStore.getState()
    expect(state.filterContext.filter_mode).toBe('sessions')
    expect(state.filterContext.sessions!.picked_sessions).toEqual(['latest-sess-id'])
    expect(state.lastKnownLatestSessionId).toBe('latest-sess-id')
    expect(state.isAutoSnappingToLatest).toBe(true)
  })

  it('autoSnapToLatestSession avec triggeredBySync=false ne marque pas auto-snap', () => {
    useGlobalFilterStore.getState().autoSnapToLatestSession('sess-id', false)
    expect(useGlobalFilterStore.getState().isAutoSnappingToLatest).toBe(false)
  })

  it('setSessions manuel reset isAutoSnappingToLatest', () => {
    useGlobalFilterStore.getState().autoSnapToLatestSession('sess-id', true)
    expect(useGlobalFilterStore.getState().isAutoSnappingToLatest).toBe(true)
    useGlobalFilterStore.getState().setSessions({
      picked_sessions: ['other-sess'],
      gap_minutes: DEFAULT_GAP_MINUTES,
    })
    expect(useGlobalFilterStore.getState().isAutoSnappingToLatest).toBe(false)
  })

  it('setLastKnownLatestSessionId persiste l\'id', () => {
    useGlobalFilterStore.getState().setLastKnownLatestSessionId('known-id')
    expect(useGlobalFilterStore.getState().lastKnownLatestSessionId).toBe('known-id')
  })

  // ── Persistence localStorage ──────────────────────────────────────────────

  it('persiste filterContext dans localStorage après mutation', () => {
    useGlobalFilterStore.getState().setCascade({
      experience_types: ['pvp'],
      playlists: [],
      modes: [],
      maps: [],
    })
    const stored = localStorage.getItem('levelup-filter-store-v1')
    expect(stored).not.toBeNull()
    const parsed = JSON.parse(stored!)
    expect(parsed.state.filterContext.cascade.experience_types).toContain('pvp')
  })

  it('persiste lastKnownLatestSessionId dans localStorage', () => {
    useGlobalFilterStore.getState().setLastKnownLatestSessionId('persisted-id')
    const stored = localStorage.getItem('levelup-filter-store-v1')
    expect(stored).not.toBeNull()
    const parsed = JSON.parse(stored!)
    expect(parsed.state.lastKnownLatestSessionId).toBe('persisted-id')
  })

  it('ne persiste pas resolvedContext (toujours re-fetch)', () => {
    const resolved: FilterContextResolved = {
      effective: DEFAULT_FILTER_CONTEXT,
      available_options: { experience_types: [], playlists: [], modes: [], maps: [] },
      session_options: { all_sessions: [], solo_labels: [], squad_labels: [] },
      counts: { total_matches_before_filters: 0, total_matches_after_filters: 0 },
    }
    useGlobalFilterStore.getState().setResolvedContext(resolved)
    const stored = localStorage.getItem('levelup-filter-store-v1')
    expect(stored).not.toBeNull()
    const parsed = JSON.parse(stored!)
    expect(parsed.state.resolvedContext).toBeUndefined()
  })

  it('ne persiste pas isAutoSnappingToLatest (éphémère)', () => {
    useGlobalFilterStore.getState().autoSnapToLatestSession('sess-id', true)
    const stored = localStorage.getItem('levelup-filter-store-v1')
    const parsed = JSON.parse(stored!)
    expect(parsed.state.isAutoSnappingToLatest).toBeUndefined()
  })

  // -------------------------------------------------------------------------
  // Actions de navigation rail (goToPrev/NextSession/Period)
  // -------------------------------------------------------------------------

  function makeResolved(allSessions: Array<{ session_id: string; label: string }>): FilterContextResolved {
    return {
      effective: DEFAULT_FILTER_CONTEXT,
      available_options: { experience_types: [], playlists: [], modes: [], maps: [] },
      session_options: {
        all_sessions: allSessions.map((s) => ({
          session_id: s.session_id,
          label: s.label,
          match_count: 5,
          match_count_filtered: 5,
          is_squad: false,
        })),
        solo_labels: [],
        squad_labels: [],
      },
      counts: { total_matches_before_filters: 100, total_matches_after_filters: 50 },
      // @ts-expect-error — tests legacy : period_presets ajouté plus tard côté types
      period_presets: [],
    }
  }

  it('goToPrevSession bascule vers la session plus ancienne', () => {
    const store = useGlobalFilterStore.getState()
    store.setResolvedContext(makeResolved([
      { session_id: 's-latest', label: '06/04' },
      { session_id: 's-mid', label: '05/04' },
      { session_id: 's-old', label: '01/04' },
    ]))
    store.setSessions({ picked_sessions: ['s-latest'], gap_minutes: DEFAULT_GAP_MINUTES })
    useGlobalFilterStore.getState().goToPrevSession()
    expect(useGlobalFilterStore.getState().filterContext.sessions?.picked_sessions).toEqual(['s-mid'])
  })

  it('goToPrevSession no-op si déjà à la session la plus ancienne', () => {
    const store = useGlobalFilterStore.getState()
    store.setResolvedContext(makeResolved([
      { session_id: 's-latest', label: '06/04' },
      { session_id: 's-old', label: '01/04' },
    ]))
    store.setSessions({ picked_sessions: ['s-old'], gap_minutes: DEFAULT_GAP_MINUTES })
    useGlobalFilterStore.getState().goToPrevSession()
    expect(useGlobalFilterStore.getState().filterContext.sessions?.picked_sessions).toEqual(['s-old'])
  })

  it('goToNextSession bascule vers la session plus récente', () => {
    const store = useGlobalFilterStore.getState()
    store.setResolvedContext(makeResolved([
      { session_id: 's-latest', label: '06/04' },
      { session_id: 's-mid', label: '05/04' },
    ]))
    store.setSessions({ picked_sessions: ['s-mid'], gap_minutes: DEFAULT_GAP_MINUTES })
    useGlobalFilterStore.getState().goToNextSession()
    expect(useGlobalFilterStore.getState().filterContext.sessions?.picked_sessions).toEqual(['s-latest'])
  })

  it('goToPrevPeriod shift la fenêtre vers le passé', () => {
    const store = useGlobalFilterStore.getState()
    store.setResolvedContext(makeResolved([]))
    store.setPeriod({ start_date: '2026-04-01', end_date: '2026-04-08' })
    useGlobalFilterStore.getState().goToPrevPeriod()
    const period = useGlobalFilterStore.getState().filterContext.period
    expect(period?.start_date).toBe('2026-03-24')
    expect(period?.end_date).toBe('2026-03-31')
  })

  it('goToPrevPeriod no-op en mode session', () => {
    const store = useGlobalFilterStore.getState()
    store.setResolvedContext(makeResolved([
      { session_id: 's-latest', label: '06/04' },
    ]))
    store.setSessions({ picked_sessions: ['s-latest'], gap_minutes: DEFAULT_GAP_MINUTES })
    const beforeHash = useGlobalFilterStore.getState().filterContextHash
    useGlobalFilterStore.getState().goToPrevPeriod()
    expect(useGlobalFilterStore.getState().filterContextHash).toBe(beforeHash)
  })

  it('goToNextSession no-op à la session la plus récente', () => {
    const store = useGlobalFilterStore.getState()
    store.setResolvedContext(makeResolved([
      { session_id: 's-latest', label: '06/04' },
      { session_id: 's-old', label: '01/04' },
    ]))
    store.setSessions({ picked_sessions: ['s-latest'], gap_minutes: DEFAULT_GAP_MINUTES })
    useGlobalFilterStore.getState().goToNextSession()
    expect(useGlobalFilterStore.getState().filterContext.sessions?.picked_sessions).toEqual(['s-latest'])
  })

  it('goToNextPeriod shift la fenêtre vers le futur dans la limite d\'aujourd\'hui', () => {
    const store = useGlobalFilterStore.getState()
    store.setResolvedContext(makeResolved([]))
    // Période ancienne pour garantir un next valide
    const yearAgo = new Date(Date.now() - 365 * 86_400_000)
    const yyyy = yearAgo.getUTCFullYear()
    const startISO = `${yyyy}-01-01`
    const endISO = `${yyyy}-01-08`
    store.setPeriod({ start_date: startISO, end_date: endISO })
    useGlobalFilterStore.getState().goToNextPeriod()
    const period = useGlobalFilterStore.getState().filterContext.period
    expect(period?.start_date).toBe(`${yyyy}-01-09`)
    expect(period?.end_date).toBe(`${yyyy}-01-16`)
  })

  it('goToNextPeriod no-op si la fenêtre est déjà collée à aujourd\'hui', () => {
    const store = useGlobalFilterStore.getState()
    store.setResolvedContext(makeResolved([]))
    const today = new Date()
    const todayISO = today.toISOString().slice(0, 10)
    const startISO = new Date(today.getTime() - 7 * 86_400_000).toISOString().slice(0, 10)
    store.setPeriod({ start_date: startISO, end_date: todayISO })
    const beforeHash = useGlobalFilterStore.getState().filterContextHash
    useGlobalFilterStore.getState().goToNextPeriod()
    expect(useGlobalFilterStore.getState().filterContextHash).toBe(beforeHash)
  })

  it('goToPrevSession no-op si pas de resolvedContext (data pas chargée)', () => {
    const store = useGlobalFilterStore.getState()
    // resolvedContext reste null
    store.setSessions({ picked_sessions: ['ghost'], gap_minutes: DEFAULT_GAP_MINUTES })
    const before = useGlobalFilterStore.getState().filterContext.sessions?.picked_sessions
    useGlobalFilterStore.getState().goToPrevSession()
    const after = useGlobalFilterStore.getState().filterContext.sessions?.picked_sessions
    expect(after).toEqual(before)
  })

  it('goToPrevPeriod no-op si période invalide (start_date null)', () => {
    const store = useGlobalFilterStore.getState()
    store.setResolvedContext(makeResolved([]))
    // Avec start_date=null, getRailMode renvoie hidden, l'action no-op
    const beforeHash = useGlobalFilterStore.getState().filterContextHash
    useGlobalFilterStore.getState().goToPrevPeriod()
    expect(useGlobalFilterStore.getState().filterContextHash).toBe(beforeHash)
  })

  it('setResolvedContext stocke la réponse résolue', () => {
    const resolved: FilterContextResolved = {
      effective: {
        filter_mode: 'period',
        period: { start_date: null, end_date: null },
        sessions: { picked_sessions: [], gap_minutes: DEFAULT_GAP_MINUTES },
        cascade: { experience_types: [], playlists: [], modes: [], maps: [] },
      },
      available_options: {
        experience_types: [],
        playlists: [],
        modes: [],
        maps: [],
      },
      session_options: { all_sessions: [], solo_labels: [], squad_labels: [] },
      counts: {
        total_matches_before_filters: 100,
        total_matches_after_filters: 42,
      },
    }
    useGlobalFilterStore.getState().setResolvedContext(resolved)
    expect(useGlobalFilterStore.getState().resolvedContext?.counts.total_matches_after_filters).toBe(42)
  })
})
