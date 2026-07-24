/**
 * Store du brouillon des paramètres utilisateur (Settings Draft).
 *
 * Gère les modifications non persistées des settings avant sauvegarde,
 * ainsi que les préférences UI locales (non synchronisées avec le backend).
 *
 * Cycle de vie des dirty fields :
 *   useQuery(['settings']) → hydration initiale depuis le backend
 *   → setDirtyField() pour chaque modification (PATCH optimiste)
 *   → useMutation(PATCH /settings) → on success : clearDirtyFields() + setLastSavedAt()
 *                                  → on error : conserver dirty fields (retry possible)
 *
 * Préférences locales (localUiPrefs) :
 *   Persistées en localStorage uniquement — jamais envoyées au backend.
 *   Exemple : hints visibles, dernier slug joueur sélectionné.
 */

import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { UpdateSettingsRequest } from '@/lib/api/types'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type UiTheme = 'dark' | 'light'
export type ColorPalette = 'default' | 'okabe-ito' | 'cividis' | 'tol-bright'

interface LocalUiPrefs {
  /** Afficher les hints contextuels dans l'UI */
  showHints: boolean
  /** Thème visuel local de l'application */
  theme: UiTheme
  /** Palette de couleurs (accessibilité daltoniens) */
  colorPalette: ColorPalette
  /** Dernier slug joueur sélectionné par titre (restaure le contexte au rechargement) */
  lastPlayerSlugByTitle: Record<string, string | null>
  /** Id couleur outline équipe alliée (HaloOutlineColor.id ou null = défaut palette) */
  allyTeamColor: string | null
  /** Id couleur outline équipe ennemie (HaloOutlineColor.id ou null = défaut palette) */
  enemyTeamColor: string | null
  /** Afficher la colonne « Ouvrir sur Halo Waypoint » sur les tableaux de matchs
   *  (I19). Préférence purement locale — jamais envoyée au backend. Masquée de
   *  toute façon si le titre courant ne déclare pas la capability
   *  `waypoint_match_url` (cf. useCapability). */
  showWaypointColumn: boolean
}

interface SettingsDraftState {
  /** Champs modifiés non encore persistés */
  dirtyFields: UpdateSettingsRequest

  /** Horodatage ISO-8601 de la dernière persistance réussie */
  lastSavedAt: string | null

  /** Préférences locales persistées en localStorage */
  localUiPrefs: LocalUiPrefs

  // --- Actions settings ---
  setDirtyField: <K extends keyof UpdateSettingsRequest>(
    key: K,
    value: UpdateSettingsRequest[K],
  ) => void
  setDirtyFields: (fields: UpdateSettingsRequest) => void
  clearDirtyFields: () => void
  setLastSavedAt: (at: string) => void

  // --- Actions prefs locales ---
  setShowHints: (value: boolean) => void
  setTheme: (theme: UiTheme) => void
  toggleTheme: () => void
  setColorPalette: (palette: ColorPalette) => void
  setAllyTeamColor: (id: string | null) => void
  setEnemyTeamColor: (id: string | null) => void
  setShowWaypointColumn: (value: boolean) => void
  /** Définit le dernier slug joueur pour un titre donné */
  setLastPlayerSlugForTitle: (titleSlug: string, slug: string | null) => void
  /** Récupère le dernier slug joueur pour un titre donné */
  getLastPlayerSlugForTitle: (titleSlug: string) => string | null
}

// ---------------------------------------------------------------------------
// Valeurs par défaut
// ---------------------------------------------------------------------------

const DEFAULT_LOCAL_UI_PREFS: LocalUiPrefs = {
  showHints: true,
  theme: 'dark',
  colorPalette: 'default',
  lastPlayerSlugByTitle: {},
  allyTeamColor: null,
  enemyTeamColor: null,
  showWaypointColumn: true,
}

// ---------------------------------------------------------------------------
// Zustand store — persisté en localStorage pour les prefs locales
// ---------------------------------------------------------------------------

export const useSettingsDraftStore = create<SettingsDraftState>()(
  persist(
    (set, get) => ({
      dirtyFields: {},
      lastSavedAt: null,
      localUiPrefs: DEFAULT_LOCAL_UI_PREFS,

      setDirtyField: (key, value) =>
        set((state) => ({
          dirtyFields: { ...state.dirtyFields, [key]: value },
        })),

      setDirtyFields: (fields) =>
        set((state) => ({
          dirtyFields: { ...state.dirtyFields, ...fields },
        })),

      clearDirtyFields: () => set({ dirtyFields: {} }),

      setLastSavedAt: (at) => set({ lastSavedAt: at }),

      setShowHints: (value) =>
        set((state) => ({
          localUiPrefs: { ...state.localUiPrefs, showHints: value },
        })),

      setTheme: (theme) =>
        set((state) => ({
          localUiPrefs: { ...state.localUiPrefs, theme },
        })),

      toggleTheme: () =>
        set((state) => ({
          localUiPrefs: {
            ...state.localUiPrefs,
            theme: state.localUiPrefs.theme === 'dark' ? 'light' : 'dark',
          },
        })),

      setColorPalette: (palette) =>
        set((state) => ({
          localUiPrefs: { ...state.localUiPrefs, colorPalette: palette },
        })),

      setAllyTeamColor: (id) =>
        set((state) => ({
          localUiPrefs: { ...state.localUiPrefs, allyTeamColor: id },
        })),

      setEnemyTeamColor: (id) =>
        set((state) => ({
          localUiPrefs: { ...state.localUiPrefs, enemyTeamColor: id },
        })),

      setShowWaypointColumn: (value) =>
        set((state) => ({
          localUiPrefs: { ...state.localUiPrefs, showWaypointColumn: value },
        })),

      setLastPlayerSlugForTitle: (titleSlug, slug) =>
        set((state) => ({
          localUiPrefs: {
            ...state.localUiPrefs,
            lastPlayerSlugByTitle: {
              ...state.localUiPrefs.lastPlayerSlugByTitle,
              [titleSlug]: slug,
            },
          },
        })),

      getLastPlayerSlugForTitle: (titleSlug: string): string | null => {
        const state = get()
        return state.localUiPrefs.lastPlayerSlugByTitle[titleSlug] ?? null
      },
    }),
    {
      name: 'levelup-ui-prefs',
      // Persister uniquement les préférences locales — pas les dirty fields
      partialize: (state): Pick<SettingsDraftState, 'localUiPrefs' | 'lastSavedAt'> => ({
        localUiPrefs: state.localUiPrefs,
        lastSavedAt: state.lastSavedAt,
      }),
      merge: (persistedState, currentState) => {
        const persisted = persistedState as Partial<
          Pick<SettingsDraftState, 'localUiPrefs' | 'lastSavedAt'>
        >
        return {
          ...currentState,
          ...persisted,
          localUiPrefs: {
            ...currentState.localUiPrefs,
            ...(persisted.localUiPrefs ?? {}),
            lastPlayerSlugByTitle: {
              ...currentState.localUiPrefs.lastPlayerSlugByTitle,
              ...(persisted.localUiPrefs?.lastPlayerSlugByTitle ?? {}),
            },
          },
        }
      },
    },
  ),
)
