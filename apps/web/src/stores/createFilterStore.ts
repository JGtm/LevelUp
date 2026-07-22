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
import { encodeFilterContextParam } from '@/features/filters/filterLink'
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
  SessionOption,
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

// Titre par défaut (convention partagée avec le client API : le header
// X-LevelUp-Title est omis pour ce titre). Un deep-link `?f=` legacy (généré
// avant le multi-titre, donc sans estampille de titre) est réputé appartenir à
// ce titre — c'était le seul titre existant à l'époque de la feature share-link.
const DEFAULT_TITLE_SLUG = 'halo_infinite'

function isValidFilterMode(mode: unknown): mode is 'period' | 'sessions' {
  return mode === 'period' || mode === 'sessions'
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

  /** Titre pour lequel le `filterContext` courant a été hydraté depuis un
   *  deep-link `?f=` (share-link). Non persisté, transitoire : posé par
   *  onRehydrateStorage au fresh-load, consommé une seule fois par
   *  reconcileActiveTitle une fois le titre actif résolu par le bootstrap.
   *  null = pas hydraté depuis l'URL (rien à valider). */
  urlHydratedTitleSlug: string | null

  // --- Mutations ---
  setPeriod: (period: PeriodInput) => void
  setSessions: (sessions: SessionsInput) => void
  setCascade: (cascade: CascadeInput) => void
  /** Commit atomique d'un FilterContextInput complet (utilisé par le bouton Analyser). */
  setFilterContext: (ctx: FilterContextInput) => void
  setResolvedContext: (resolved: FilterContextResolved) => void
  resetFilters: () => void
  /** Valide le filtre hydraté depuis un deep-link `?f=` contre le titre ACTIF
   *  (résolu par le bootstrap). Si le deep-link a été généré pour un AUTRE titre,
   *  reset le filtre (son état — sessions/cascade — n'a aucun sens hors de son
   *  titre). No-op si le filtre ne vient pas d'un deep-link. Consommation
   *  one-shot. Cf. appShellStore.hydrateFromBootstrap / switchTitle. */
  reconcileActiveTitle: (activeTitleSlug: string) => void

  // --- Auto-snap on new data ---
  setLastKnownLatestSessionId: (id: string | null) => void
  setIsAutoSnappingToLatest: (snapping: boolean) => void
  /** Bascule le filtre sur la session passée (filter_mode='sessions'). On écrit le
   *  LABEL dans picked_sessions (compatible solo ET escouade : le backend escouade
   *  ne matche que par label) et on mémorise latest.session_id comme clé de
   *  détection stable (le label porte un suffixe « (N) » volatil). triggeredBySync
   *  =true marque le snap auto. */
  autoSnapToLatestSession: (
    latest: Pick<SessionOption, 'session_id' | 'label'>,
    triggeredBySync: boolean,
  ) => void

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
  /** Titre actif courant, estampillé dans le deep-link `?f=` à l'encodage pour
   *  que le filtre ne se réapplique qu'au titre pour lequel il a été généré.
   *  Injecté (plutôt qu'importé) pour garder la factory découplée/testable.
   *  Défaut : titre par défaut (utile aux stores sans URL / aux tests). */
  getActiveTitleSlug?: () => string
}

/**
 * Crée un store Zustand de filtres avec persistance localStorage.
 *
 * Chaque store instancié est totalement indépendant — pas de partage de state
 * entre stores. Pour deux contextes (solo + squad), instancier deux stores avec
 * des `name` distincts.
 */
export function createFilterStore(options: CreateFilterStoreOptions): FilterStore {
  const { name, urlEnabled = false, urlParam = 'f', getActiveTitleSlug } = options

  const activeTitle = (): string => getActiveTitleSlug?.() ?? DEFAULT_TITLE_SLUG

  function encodeToUrl(ctx: FilterContextInput): void {
    if (!urlEnabled || typeof window === 'undefined') return
    try {
      // Enveloppe v2 : { t: titre actif, c: FilterContextInput }. Estampiller le
      // titre permet, au fresh-load, de rejeter un deep-link généré pour un AUTRE
      // titre (cf. decodeFromUrl / reconcileActiveTitle) — les labels de session
      // sont purement temporels, donc title-agnostic, et fuiteraient sinon.
      // Format encodé mutualisé avec buildSoloFilterLink (features/filters/filterLink).
      const encoded = encodeFilterContextParam(activeTitle(), ctx)
      const url = new URL(window.location.href)
      url.searchParams.set(urlParam, encoded)
      window.history.replaceState(null, '', url.toString())
    } catch {
      // Silencieux si l'URL est invalide
    }
  }

  function decodeFromUrl(): { context: FilterContextInput; titleSlug: string } | null {
    if (!urlEnabled || typeof window === 'undefined') return null
    try {
      const url = new URL(window.location.href)
      const encoded = url.searchParams.get(urlParam)
      if (!encoded) return null
      const raw: unknown = JSON.parse(decodeURIComponent(atob(encoded)))
      if (typeof raw !== 'object' || raw === null) return null
      const obj = raw as Record<string, unknown>
      // Enveloppe v2 : { t: titleSlug, c: FilterContextInput }.
      const inner = obj.c as Record<string, unknown> | undefined
      if (inner && isValidFilterMode(inner.filter_mode)) {
        const titleSlug = typeof obj.t === 'string' && obj.t ? obj.t : DEFAULT_TITLE_SLUG
        return { context: inner as unknown as FilterContextInput, titleSlug }
      }
      // Legacy (pré-multi-titre) : le payload EST le FilterContextInput brut,
      // sans estampille de titre → titre implicite = défaut (Halo Infinite).
      if (isValidFilterMode(obj.filter_mode)) {
        return { context: raw as FilterContextInput, titleSlug: DEFAULT_TITLE_SLUG }
      }
      return null
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
        urlHydratedTitleSlug: null,

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

        reconcileActiveTitle: (activeTitleSlug) => {
          const urlTitle = get().urlHydratedTitleSlug
          if (urlTitle == null) return // pas hydraté depuis un deep-link → rien à valider
          // Consommation one-shot : quel que soit le verdict, on ne re-valide pas
          // (une éventuelle mutation utilisateur ultérieure ré-estampille l'URL).
          set({ urlHydratedTitleSlug: null })
          if (urlTitle === activeTitleSlug) return // deep-link valide pour le titre actif → conservé
          // Deep-link d'un AUTRE titre : reset propre (réécrit aussi l'URL/localStorage
          // au titre actif via resetFilters → encodeToUrl).
          log.debug(`reconcile:reset store=${name} urlTitle=${urlTitle} active=${activeTitleSlug}`)
          get().resetFilters()
        },

        setLastKnownLatestSessionId: (id) => set({ lastKnownLatestSessionId: id }),

        setIsAutoSnappingToLatest: (snapping) => set({ isAutoSnappingToLatest: snapping }),

        autoSnapToLatestSession: (latest, triggeredBySync) => {
          const current = get().filterContext
          const next: FilterContextInput = {
            ...current,
            filter_mode: 'sessions',
            sessions: {
              ...(current.sessions ?? DEFAULT_SESSIONS),
              // LABEL, pas session_id : filterSynthesisByPickedSessions (escouade)
              // matche uniquement par SessionLabel ; applySessionFilter (solo)
              // accepte les deux. Le rail goToPrev/NextSession écrit aussi le label.
              picked_sessions: [latest.label],
            },
            period: DEFAULT_PERIOD,
          }
          encodeToUrl(next)
          set({
            filterContext: next,
            filterContextHash: computeHash(next),
            // session_id stable comme clé de détection « nouvelle session arrivée ».
            lastKnownLatestSessionId: latest.session_id,
            isAutoSnappingToLatest: triggeredBySync,
          })
          log.debug(
            `auto_snap:fired store=${name} session=${latest.session_id} label=${latest.label} trigger=${triggeredBySync ? 'sync' : 'manual'}`,
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
          // Persisté pour reprendre le suivi « dernière session » au reload :
          // sans ça, isAutoSnappingToLatest repasse à false et useFollowLatestSession
          // figerait sur l'ancienne session au lieu de re-snapper sur la plus récente.
          isAutoSnappingToLatest: state.isAutoSnappingToLatest,
        }),
        onRehydrateStorage: () => (state) => {
          if (!state) return
          const fromUrl = decodeFromUrl()
          if (fromUrl) {
            state.filterContext = fromUrl.context
            state.filterContextHash = computeHash(fromUrl.context)
            // Le titre actif RÉEL n'est pas encore connu ici (bootstrap async) :
            // on mémorise le titre du deep-link, validé plus tard par
            // reconcileActiveTitle (appelé par hydrateFromBootstrap).
            state.urlHydratedTitleSlug = fromUrl.titleSlug
            log.debug(`hydrate:source=url store=${name} urlTitle=${fromUrl.titleSlug}`)
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
