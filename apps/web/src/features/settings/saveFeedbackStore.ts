/**
 * Retour d'enregistrement des Paramètres : un badge flottant en haut à droite de la page (« ✓ Enregistré »
 * 2 s, « ✗ Erreur lors de la sauvegarde » 4 s), le motif de la majorité des onglets. Un onglet qui
 * enregistre un réglage appelle `flashSaveStatus` ; `SaveStatusBadge` (monté une fois par `SettingsPage`)
 * l'affiche. Un nouvel enregistrement remplace le badge en cours et relance son délai.
 */
import { create } from 'zustand'

export type SaveStatus = 'saved' | 'error'

/** Durée d'affichage du badge, en ms : l'échec reste plus longtemps pour être lu. */
const SAVED_VISIBLE_MS = 2000
const ERROR_VISIBLE_MS = 4000

interface SaveFeedbackState {
  status: SaveStatus | null
  flash: (status: SaveStatus) => void
  clear: () => void
}

let hideTimer: ReturnType<typeof setTimeout> | null = null

export const useSaveFeedbackStore = create<SaveFeedbackState>((set) => ({
  status: null,
  flash: (status) => {
    if (hideTimer) clearTimeout(hideTimer)
    set({ status })
    hideTimer = setTimeout(
      () => set({ status: null }),
      status === 'saved' ? SAVED_VISIBLE_MS : ERROR_VISIBLE_MS,
    )
  },
  clear: () => {
    if (hideTimer) clearTimeout(hideTimer)
    hideTimer = null
    set({ status: null })
  },
}))

/** Montre le badge d'enregistrement (réglage enregistré, ou refusé). */
export function flashSaveStatus(status: SaveStatus): void {
  useSaveFeedbackStore.getState().flash(status)
}

