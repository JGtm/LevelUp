import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import { SPARTAN_DEFAULT_COLORS } from '@/lib/accessibility/palettes/spartan'

/**
 * Apparence Spartan choisie par le joueur (Halo 5) — persistée en localStorage.
 *
 * v1 LOCAL (pas de backend) : précédent `settingsDraftStore.localUiPrefs`
 * (couleurs d'équipe). Le store détient l'apparence ENREGISTRÉE ; la modale tient
 * un brouillon local et écrit ici au clic « Enregistrer » (Annuler = pas d'écriture).
 * Follow-up backend cross-device : cf. .ai/PLAN_H5_SPARTAN_CUSTOMIZER.md.
 */
export interface SpartanAppearance {
  /** id d'emblème (clé de spartan-catalog.json), null = aucun choix. */
  emblemId: string | null
  /** Couleur zone primaire (hex). */
  primary: string
  /** Couleur zone secondaire (hex). */
  secondary: string
  /** Couleur zone tertiaire (hex). */
  tertiary: string
}

export const DEFAULT_SPARTAN_APPEARANCE: SpartanAppearance = {
  // Emblème par défaut (#160, insigne neutre) → la bannière d'identité n'est jamais
  // vide pour un nouveau joueur. Cf. .ai/PLAN_H5_SPARTAN_CUSTOMIZER.md.
  emblemId: '160',
  ...SPARTAN_DEFAULT_COLORS,
}

interface SpartanAppearanceState {
  appearance: SpartanAppearance
  setAppearance: (a: SpartanAppearance) => void
  reset: () => void
}

export const useSpartanAppearanceStore = create<SpartanAppearanceState>()(
  persist(
    (set) => ({
      appearance: DEFAULT_SPARTAN_APPEARANCE,
      setAppearance: (a) => set({ appearance: a }),
      reset: () => set({ appearance: DEFAULT_SPARTAN_APPEARANCE }),
    }),
    { name: 'levelup-spartan-appearance' },
  ),
)
