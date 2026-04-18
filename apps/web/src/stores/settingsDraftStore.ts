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

interface LocalUiPrefs {
  /** Afficher les hints contextuels dans l'UI */
  showHints: boolean
  /** Dernier slug joueur sélectionné par titre (restaure le contexte au rechargement) */
  lastPlayerSlugByTitle: Record<string, string | null>
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
  /** @deprecated Utiliser setLastPlayerSlugForTitle */
  setLastPlayerSlug: (slug: string | null) => void
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
  lastPlayerSlugByTitle: {},
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

      setLastPlayerSlug: (slug) =>
        set((state) => ({
          localUiPrefs: {
            ...state.localUiPrefs,
            lastPlayerSlugByTitle: {
              ...state.localUiPrefs.lastPlayerSlugByTitle,
              halo_infinite: slug,
            },
          },
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
    },
  ),
)
