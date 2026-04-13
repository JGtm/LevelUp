/**
 * Store global des filtres — source de vérité unique pour FilterContextInput.
 *
 * Synchronisation URL ↔ store :
 * - Les filtres sont encodés dans l'URL (query param `f`) pour permettre le
 *   partage de lien et la navigation arrière/avant.
 * - Au chargement de la page, le store est hydraté depuis l'URL.
 * - Chaque mutation du store met à jour l'URL silencieusement.
 *
 * Cycle de vie :
 *   URL → hydrateFromUrl() → store → useFiltersResolve() → resolved opts
 *         ↑
 *   mutation (setFilterMode, togglePlaylist...) → updateUrl()
 *
 * Invariant : `gap_minutes` est toujours 120 (hérité du code Streamlit).
 */

import { create } from 'zustand'
import type {
  CascadeInput,
  FilterContextInput,
  FilterContextResolved,
  PeriodInput,
  SessionsInput,
} from '@/lib/api/types'

// ---------------------------------------------------------------------------
// Valeurs par défaut
// ---------------------------------------------------------------------------

export const DEFAULT_GAP_MINUTES = 120

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

  // --- Mutations ---
  setFilterMode: (mode: 'period' | 'sessions') => void
  setPeriod: (period: PeriodInput) => void
  setSessions: (sessions: SessionsInput) => void
  setCascade: (cascade: CascadeInput) => void
  setResolvedContext: (resolved: FilterContextResolved) => void
  resetFilters: () => void

  /** Met à jour le store depuis l'URL (appelé au montage du composant). */
  hydrateFromUrl: () => void
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
    // Validation minimale
    if (raw.filter_mode !== 'period' && raw.filter_mode !== 'sessions') return null
    return raw as FilterContextInput
  } catch {
    return null
  }
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useGlobalFilterStore = create<GlobalFilterState>((set, get) => ({
  filterContext: DEFAULT_FILTER_CONTEXT,
  resolvedContext: null,
  filterContextHash: computeHash(DEFAULT_FILTER_CONTEXT),

  setFilterMode: (mode) => {
    const next = { ...get().filterContext, filter_mode: mode }
    encodeToUrl(next)
    set({ filterContext: next, filterContextHash: computeHash(next) })
  },

  setPeriod: (period) => {
    const next = { ...get().filterContext, period }
    encodeToUrl(next)
    set({ filterContext: next, filterContextHash: computeHash(next) })
  },

  setSessions: (sessions) => {
    const next = { ...get().filterContext, sessions }
    encodeToUrl(next)
    set({ filterContext: next, filterContextHash: computeHash(next) })
  },

  setCascade: (cascade) => {
    const next = { ...get().filterContext, cascade }
    encodeToUrl(next)
    set({ filterContext: next, filterContextHash: computeHash(next) })
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
    })
  },

  hydrateFromUrl: () => {
    const fromUrl = decodeFromUrl()
    if (!fromUrl) return
    set({
      filterContext: fromUrl,
      filterContextHash: computeHash(fromUrl),
    })
  },
}))
