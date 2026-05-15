/**
 * createFilterStore — factory Zustand pour les stores de filtres contextuels.
 *
 * Le projet a deux contextes d'exploration disjoints :
 *  - Stats Solo (timeseries, history) : sessions solo (`is_squad=false`)
 *  - Escouade (squad pages) : sessions squad (`is_squad=true`)
 *
 * Avant ce split, un unique `useGlobalFilterStore` partageait son state entre
 * les deux contextes, ce qui polluait les filtres cross-page et faisait
 * auto-snap-er des sessions squad sur la page Stats Solo (et inversement).
 *
 * Cette factory permet d'instancier autant de stores indépendants que de
 * contextes, chacun avec sa propre clé localStorage et son propre
 * `lastKnownLatestSessionId`. Les composants shell (FilterOmnibar,
 * PeriodSessionRail) prennent le store en prop pour rester agnostiques.
 */

import { create, type UseBoundStore, type StoreApi } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'
import { log } from '@/features/filters/_logger'
import {
  computeNextSession,
  computeNextWindow,
  computePrevSession,
  computePrevWindow,
  getRailMode,
} from '@/features/filters/periodSessionNav'
import type {
  CascadeInput,
  FilterContextInput,
  FilterContextResolved,
  PeriodInput,
  SessionsInput,
} from '@/lib/api/types'
import { DEFAULT_GAP_MINUTES } from '@/stores/filterDefaults'

// ---------------------------------------------------------------------------
// Valeurs par défaut
// ---------------------------------------------------------------------------

const DEFAULT_PERIOD: PeriodInput = { start_date: null, end_date: null }
const DEFAULT_SESSIONS: SessionsInput = {
  picked_sessions: [],
  gap_minutes: DEFAULT_GAP_MINUTES,
}
const DEFAULT_CASCADE: CascadeInput = {
  experience_types: [],
  playlists: [],
  modes: [],
  maps: [],
}

export const DEFAULT_FILTER_CONTEXT: FilterContextInput = {
  filter_mode: 'period',
  period: DEFAULT_PERIOD,
  sessions: DEFAULT_SESSIONS,
  cascade: DEFAULT_CASCADE,
}

// ---------------------------------------------------------------------------
// Types du store
// ---------------------------------------------------------------------------

export interface FilterStoreState {
  /** Contexte courant envoyé au backend via POST /filters/resolve */
  filterContext: FilterContextInput

  /** Réponse résolue du backend (mise à jour par useFiltersResolve query) */
  resolvedContext: FilterContextResolved | null

  /** Hash stable du filterContext pour les query keys TanStack */
  filterContextHash: string

  /** Dernier session_id "latest" connu — sert à détecter l'arrivée de nouvelles
   *  sessions. Persisté en localStorage. */
  lastKnownLatestSessionId: string | null

  /** Indique que la session courante a été auto-sélectionnée suite à une nouvelle
   *  donnée détectée. Reset à false dès qu'un user interagit manuellement. */
  isAutoSnappingToLatest: boolean

  // --- Mutations ---
  setPeriod: (period: PeriodInput) => void
  setSessions: (sessions: SessionsInput) => void
  setCascade: (cascade: CascadeInput) => void
  /** Commit atomique d'un FilterContextInput complet (utilisé par le bouton Analyser). */
  setFilterContext: (ctx: FilterContextInput) => void
  setResolvedContext: (resolved: FilterContextResolved) => void
  resetFilters: () => void

  // --- Auto-snap on new data ---
  setLastKnownLatestSessionId: (id: string | null) => void
  setIsAutoSnappingToLatest: (snapping: boolean) => void
  /** Bascule le filtre sur la session passée en paramètre (filter_mode='sessions',
   *  sessions.picked_sessions=[id]). Si triggeredBySync=true, marque le snap auto. */
  autoSnapToLatestSession: (latestSessionId: string, triggeredBySync: boolean) => void

  // --- Rail de navigation période/session (no-op si pas applicable) ---
  goToPrevSession: () => void
  goToNextSession: () => void
  goToPrevPeriod: () => void
  goToNextPeriod: () => void
}

export type FilterStore = UseBoundStore<StoreApi<FilterStoreState>>

// ---------------------------------------------------------------------------
// Hash stable
// ---------------------------------------------------------------------------

// FNV-1a 32 bits sur le JSON. Sensible à toutes les diffs (incl. cascade).
function computeHash(ctx: FilterContextInput): string {
  const s = JSON.stringify(ctx) ?? ''
  let h = 0x811c9dc5
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i)
    h = Math.imul(h, 0x01000193) >>> 0
  }
  return h.toString(16).padStart(8, '0')
}

// ---------------------------------------------------------------------------
// Factory
// ---------------------------------------------------------------------------

interface CreateFilterStoreOptions {
  /** Nom (clé) localStorage. Ex: 'levelup-solo-filter-v1'. */
  name: string
  /** Si true, encode/décode le contexte dans `?f=…` (share-link). */
  urlEnabled?: boolean
  /** Paramètre URL pour le share-link. Défaut : 'f'. */
  urlParam?: string
}

/**
 * Crée un store Zustand de filtres avec persistance localStorage.
 *
 * Chaque store instancié est totalement indépendant — pas de partage de state
 * entre stores. Pour deux contextes (solo + squad), instancier deux stores avec
 * des `name` distincts.
 */
export function createFilterStore(options: CreateFilterStoreOptions): FilterStore {
  const { name, urlEnabled = false, urlParam = 'f' } = options

  function encodeToUrl(ctx: FilterContextInput): void {
    if (!urlEnabled || typeof window === 'undefined') return
    try {
      const encoded = btoa(encodeURIComponent(JSON.stringify(ctx)))
      const url = new URL(window.location.href)
      url.searchParams.set(urlParam, encoded)
      window.history.replaceState(null, '', url.toString())
    } catch {
      // Silencieux si l'URL est invalide
    }
  }

  function decodeFromUrl(): FilterContextInput | null {
    if (!urlEnabled || typeof window === 'undefined') return null
    try {
      const url = new URL(window.location.href)
      const encoded = url.searchParams.get(urlParam)
      if (!encoded) return null
      const raw = JSON.parse(decodeURIComponent(atob(encoded)))
      if (raw.filter_mode !== 'period' && raw.filter_mode !== 'sessions') return null
      return raw as FilterContextInput
    } catch {
      return null
    }
  }

  return create<FilterStoreState>()(
    persist(
      (set, get) => ({
        filterContext: DEFAULT_FILTER_CONTEXT,
        resolvedContext: null,
        filterContextHash: computeHash(DEFAULT_FILTER_CONTEXT),
        lastKnownLatestSessionId: null,
        isAutoSnappingToLatest: false,

        setPeriod: (period) => {
          // Auto-derive filter_mode et exclusivité Période ↔ Session.
          const isPeriodSet = !!(period?.start_date || period?.end_date)
          const next: FilterContextInput = {
            ...get().filterContext,
            period,
            filter_mode: isPeriodSet ? 'period' : 'sessions',
            sessions: isPeriodSet ? DEFAULT_SESSIONS : get().filterContext.sessions,
          }
          encodeToUrl(next)
          set({
            filterContext: next,
            filterContextHash: computeHash(next),
            isAutoSnappingToLatest: false,
          })
        },

        setSessions: (sessions) => {
          const isSessionPicked = (sessions?.picked_sessions?.length ?? 0) > 0
          const next: FilterContextInput = {
            ...get().filterContext,
            sessions,
            filter_mode: isSessionPicked ? 'sessions' : 'period',
            period: isSessionPicked ? DEFAULT_PERIOD : get().filterContext.period,
          }
          encodeToUrl(next)
          set({
            filterContext: next,
            filterContextHash: computeHash(next),
            isAutoSnappingToLatest: false,
          })
        },

        setCascade: (cascade) => {
          const next = { ...get().filterContext, cascade }
          encodeToUrl(next)
          set({ filterContext: next, filterContextHash: computeHash(next) })
        },

        setFilterContext: (ctx) => {
          encodeToUrl(ctx)
          set({
            filterContext: ctx,
            filterContextHash: computeHash(ctx),
            isAutoSnappingToLatest: false,
          })
        },

        setResolvedContext: (resolved) => {
          set({ resolvedContext: resolved })
        },

        resetFilters: () => {
          const next = DEFAULT_FILTER_CONTEXT
          encodeToUrl(next)
          set({
            filterContext: next,
            resolvedContext: null,
            filterContextHash: computeHash(next),
            isAutoSnappingToLatest: false,
          })
        },

        setLastKnownLatestSessionId: (id) => set({ lastKnownLatestSessionId: id }),

        setIsAutoSnappingToLatest: (snapping) => set({ isAutoSnappingToLatest: snapping }),

        autoSnapToLatestSession: (latestSessionId, triggeredBySync) => {
          const current = get().filterContext
          const next: FilterContextInput = {
            ...current,
            filter_mode: 'sessions',
            sessions: {
              ...(current.sessions ?? DEFAULT_SESSIONS),
              picked_sessions: [latestSessionId],
            },
            period: DEFAULT_PERIOD,
          }
          encodeToUrl(next)
          set({
            filterContext: next,
            filterContextHash: computeHash(next),
            lastKnownLatestSessionId: latestSessionId,
            isAutoSnappingToLatest: triggeredBySync,
          })
          log.debug(
            `auto_snap:fired store=${name} session=${latestSessionId} trigger=${triggeredBySync ? 'sync' : 'manual'}`,
          )
        },

        goToPrevSession: () => {
          const { filterContext, resolvedContext } = get()
          const all = resolvedContext?.session_options?.all_sessions ?? []
          const mode = getRailMode(filterContext, all)
          if (mode.kind !== 'session') {
            log.debug(`rail_nav:noop store=${name} kind=goToPrevSession reason=mode-${mode.kind}`)
            return
          }
          const target = computePrevSession(mode.session.session_id, all)
          if (!target) {
            log.debug(`rail_nav:noop store=${name} kind=goToPrevSession reason=at-oldest from=${mode.session.session_id}`)
            return
          }
          log.debug(`rail_nav:fired store=${name} kind=goToPrevSession from=${mode.session.session_id} to=${target.session_id}`)
          get().setSessions({
            ...(filterContext.sessions ?? DEFAULT_SESSIONS),
            picked_sessions: [target.label],
          })
        },

        goToNextSession: () => {
          const { filterContext, resolvedContext } = get()
          const all = resolvedContext?.session_options?.all_sessions ?? []
          const mode = getRailMode(filterContext, all)
          if (mode.kind !== 'session') {
            log.debug(`rail_nav:noop store=${name} kind=goToNextSession reason=mode-${mode.kind}`)
            return
          }
          const target = computeNextSession(mode.session.session_id, all)
          if (!target) {
            log.debug(`rail_nav:noop store=${name} kind=goToNextSession reason=at-latest from=${mode.session.session_id}`)
            return
          }
          log.debug(`rail_nav:fired store=${name} kind=goToNextSession from=${mode.session.session_id} to=${target.session_id}`)
          get().setSessions({
            ...(filterContext.sessions ?? DEFAULT_SESSIONS),
            picked_sessions: [target.label],
          })
        },

        goToPrevPeriod: () => {
          const { filterContext, resolvedContext } = get()
          const all = resolvedContext?.session_options?.all_sessions ?? []
          const mode = getRailMode(filterContext, all)
          if (mode.kind !== 'period') {
            log.debug(`rail_nav:noop store=${name} kind=goToPrevPeriod reason=mode-${mode.kind}`)
            return
          }
          const target = computePrevWindow(mode.period)
          if (!target) {
            log.debug(`rail_nav:noop store=${name} kind=goToPrevPeriod reason=invalid-window`)
            return
          }
          log.debug(`rail_nav:fired store=${name} kind=goToPrevPeriod to=${target.start_date}..${target.end_date}`)
          get().setPeriod(target)
        },

        goToNextPeriod: () => {
          const { filterContext, resolvedContext } = get()
          const all = resolvedContext?.session_options?.all_sessions ?? []
          const mode = getRailMode(filterContext, all)
          if (mode.kind !== 'period') {
            log.debug(`rail_nav:noop store=${name} kind=goToNextPeriod reason=mode-${mode.kind}`)
            return
          }
          const target = computeNextWindow(mode.period)
          if (!target) {
            log.debug(`rail_nav:noop store=${name} kind=goToNextPeriod reason=at-today`)
            return
          }
          log.debug(`rail_nav:fired store=${name} kind=goToNextPeriod to=${target.start_date}..${target.end_date}`)
          get().setPeriod(target)
        },
      }),
      {
        name,
        storage: createJSONStorage(() => localStorage),
        partialize: (state) => ({
          filterContext: state.filterContext,
          filterContextHash: state.filterContextHash,
          lastKnownLatestSessionId: state.lastKnownLatestSessionId,
        }),
        onRehydrateStorage: () => (state) => {
          if (!state) return
          const fromUrl = decodeFromUrl()
          if (fromUrl) {
            state.filterContext = fromUrl
            state.filterContextHash = computeHash(fromUrl)
            log.debug(`hydrate:source=url store=${name}`)
          } else {
            log.debug(`hydrate:source=localStorage store=${name}`)
          }
        },
      },
    ),
  )
}

// Re-exports utiles aux consommateurs (FilterOmnibar, SquadLayout, etc.).
export { DEFAULT_GAP_MINUTES }
