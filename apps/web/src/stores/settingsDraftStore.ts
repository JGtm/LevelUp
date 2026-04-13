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
  /** Dernier slug joueur sélectionné (restaure le contexte au rechargement) */
  lastPlayerSlug: string | null
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
  setLastPlayerSlug: (slug: string | null) => void
}

// ---------------------------------------------------------------------------
// Valeurs par défaut
// ---------------------------------------------------------------------------

const DEFAULT_LOCAL_UI_PREFS: LocalUiPrefs = {
  showHints: true,
  lastPlayerSlug: null,
}

// ---------------------------------------------------------------------------
// Zustand store — persisté en localStorage pour les prefs locales
// ---------------------------------------------------------------------------

export const useSettingsDraftStore = create<SettingsDraftState>()(
  persist(
    (set) => ({
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
          localUiPrefs: { ...state.localUiPrefs, lastPlayerSlug: slug },
        })),
    }),
    {
      name: 'levelup-ui-prefs',
      // Persister uniquement les préférences locales — pas les dirty fields
      partialize: (state) => ({
        localUiPrefs: state.localUiPrefs,
        lastSavedAt: state.lastSavedAt,
      }),
    },
  ),
)
