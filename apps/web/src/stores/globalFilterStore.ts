/**
 * Store global des filtres — source de vérité unique pour FilterContextInput.
 *
 * Trois sources d'hydratation, par ordre de priorité :
 *   1. URL `?f=…` (share-link explicite)              — gagne toujours
 *   2. localStorage (persistence cross-session)       — fallback
 *   3. DEFAULT_FILTER_CONTEXT                          — premier lancement
 *
 * Sur fin de sync, si une nouvelle session est détectée (latest ≠
 * lastKnownLatestSessionId), `autoSnapToLatestSession` bascule le filtre
 * sur la dernière session — sauf si l'utilisateur a déjà fait un choix
 * différent (auto-reset de `isAutoSnappingToLatest`).
 *
 * `filter_mode` est auto-dérivé : pas de toggle exposé. Une session pickée
 * implique mode='sessions' + period vidée. Une période posée implique
 * mode='period' + sessions vidées. La cascade reste orthogonale.
 *
 * Invariant : `gap_minutes` est toujours 120 (hérité du code Streamlit).
 */

import { create } from 'zustand'
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

// Re-export pour compat des consommateurs existants (tests, hooks).
export { DEFAULT_GAP_MINUTES }

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

interface GlobalFilterState {
  /** Contexte courant envoyé au backend via POST /filters/resolve */
  filterContext: FilterContextInput

  /** Réponse résolue du backend (mise à jour par useFiltersResolve query) */
  resolvedContext: FilterContextResolved | null

  /** Hash stable du filterContext pour les query keys TanStack */
  filterContextHash: string

  /** Dernier session_id "latest" connu — sert à détecter l'arrivée de nouvelles
   *  sessions (lors d'une fin de sync ou d'un resolve). Persisté en localStorage. */
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

// ---------------------------------------------------------------------------
// Hash stable
// ---------------------------------------------------------------------------

function computeHash(ctx: FilterContextInput): string {
  try {
    return btoa(JSON.stringify(ctx)).slice(0, 32)
  } catch {
    return 'default'
  }
}

// ---------------------------------------------------------------------------
// URL encoding/decoding
// ---------------------------------------------------------------------------

const URL_PARAM = 'f'

function encodeToUrl(ctx: FilterContextInput): void {
  if (typeof window === 'undefined') return
  try {
    const encoded = btoa(encodeURIComponent(JSON.stringify(ctx)))
    const url = new URL(window.location.href)
    url.searchParams.set(URL_PARAM, encoded)
    window.history.replaceState(null, '', url.toString())
  } catch {
    // Silencieux si l'URL est invalide
  }
}

function decodeFromUrl(): FilterContextInput | null {
  if (typeof window === 'undefined') return null
  try {
    const url = new URL(window.location.href)
    const encoded = url.searchParams.get(URL_PARAM)
    if (!encoded) return null
    const raw = JSON.parse(decodeURIComponent(atob(encoded)))
    if (raw.filter_mode !== 'period' && raw.filter_mode !== 'sessions') return null
    return raw as FilterContextInput
  } catch {
    return null
  }
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useGlobalFilterStore = create<GlobalFilterState>()(
  persist(
    (set, get) => ({
      filterContext: DEFAULT_FILTER_CONTEXT,
      resolvedContext: null,
      filterContextHash: computeHash(DEFAULT_FILTER_CONTEXT),
      lastKnownLatestSessionId: null,
      isAutoSnappingToLatest: false,

      setPeriod: (period) => {
        // Auto-derive filter_mode et exclusivité Période ↔ Session.
        // Si une période est posée → mode='period' + sessions vidées.
        // Sinon → mode='sessions' (laisse les sessions intactes).
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
        // Auto-derive filter_mode + exclusivité.
        // Une session pickée → mode='sessions' + period vidée.
        // Sinon (toutes les sessions) → mode='period'.
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
          period: DEFAULT_PERIOD, // exclusivité avec Période
        }
        encodeToUrl(next)
        set({
          filterContext: next,
          filterContextHash: computeHash(next),
          lastKnownLatestSessionId: latestSessionId,
          isAutoSnappingToLatest: triggeredBySync,
        })
        log.debug(
          `auto_snap:fired session=${latestSessionId} trigger=${triggeredBySync ? 'sync' : 'manual'}`,
        )
      },

      // ---------------------------------------------------------------------
      // Navigation rail (period/session prev/next) — no-op si pas applicable.
      // Délègue le calcul aux helpers purs `periodSessionNav.ts` et réutilise
      // setSessions / setPeriod existants : le hash change, react-query refetch.
      // ---------------------------------------------------------------------

      goToPrevSession: () => {
        const { filterContext, resolvedContext } = get()
        const all = resolvedContext?.session_options?.all_sessions ?? []
        const mode = getRailMode(filterContext, all)
        if (mode.kind !== 'session') return
        const target = computePrevSession(mode.session.session_id, all)
        if (!target) return
        get().setSessions({
          ...(filterContext.sessions ?? DEFAULT_SESSIONS),
          picked_sessions: [target.session_id],
        })
      },

      goToNextSession: () => {
        const { filterContext, resolvedContext } = get()
        const all = resolvedContext?.session_options?.all_sessions ?? []
        const mode = getRailMode(filterContext, all)
        if (mode.kind !== 'session') return
        const target = computeNextSession(mode.session.session_id, all)
        if (!target) return
        get().setSessions({
          ...(filterContext.sessions ?? DEFAULT_SESSIONS),
          picked_sessions: [target.session_id],
        })
      },

      goToPrevPeriod: () => {
        const { filterContext, resolvedContext } = get()
        const all = resolvedContext?.session_options?.all_sessions ?? []
        const mode = getRailMode(filterContext, all)
        if (mode.kind !== 'period') return
        const target = computePrevWindow(mode.period)
        if (!target) return
        get().setPeriod(target)
      },

      goToNextPeriod: () => {
        const { filterContext, resolvedContext } = get()
        const all = resolvedContext?.session_options?.all_sessions ?? []
        const mode = getRailMode(filterContext, all)
        if (mode.kind !== 'period') return
        const target = computeNextWindow(mode.period)
        if (!target) return
        get().setPeriod(target)
      },
    }),
    {
      // Persistence localStorage : seulement filterContext (les choix utilisateur)
      // et lastKnownLatestSessionId (pour la détection cross-session de
      // nouvelles sessions). resolvedContext est toujours re-fetch côté backend
      // donc inutile à persister, et isAutoSnappingToLatest est éphémère.
      name: 'levelup-filter-store-v1',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        filterContext: state.filterContext,
        filterContextHash: state.filterContextHash,
        lastKnownLatestSessionId: state.lastKnownLatestSessionId,
      }),
      // Priorité d'hydratation : URL `?f=…` > localStorage > défauts.
      // Si l'URL contient un contexte explicite, il l'emporte sur la valeur
      // localStorage (explicit > implicit, share-link friendly).
      onRehydrateStorage: () => (state) => {
        if (!state) return
        const fromUrl = decodeFromUrl()
        if (fromUrl) {
          state.filterContext = fromUrl
          state.filterContextHash = computeHash(fromUrl)
          log.debug('hydrate:source=url')
        } else {
          log.debug('hydrate:source=localStorage')
        }
      },
    },
  ),
)
