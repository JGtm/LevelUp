import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import { SPARTAN_DEFAULT_COLORS } from '@/lib/accessibility/palettes/spartan'

/**
 * Apparence Spartan choisie par le joueur — persistée en localStorage, **par titre**.
 *
 * v1 LOCAL (pas de backend). Le store détient l'apparence ENREGISTRÉE par slug de
 * titre ; la modale tient un brouillon local et écrit ici au clic « Enregistrer »
 * (Annuler = pas d'écriture). Title-agnostic : chaque titre déclarant la capability
 * `spartan_customizer` garde sa propre apparence (les ids emblème/bannière sont
 * spécifiques au titre). Follow-up backend cross-device : cf. .ai/PLAN_H5_SPARTAN_CUSTOMIZER.md.
 */
export interface SpartanAppearance {
  /** id d'emblème (clé de spartan-catalog.json), null = aucun choix. */
  emblemId: string | null
  /** id de bannière (nameplate) — INDÉPENDANT de l'emblème ; null = aucun choix. */
  nameplateId: string | null
  /** Couleur zone primaire (hex). */
  primary: string
  /** Couleur zone secondaire (hex). */
  secondary: string
  /** Couleur zone tertiaire (hex). */
  tertiary: string
}

export const DEFAULT_SPARTAN_APPEARANCE: SpartanAppearance = {
  // Défaut neutre #160 → la bannière d'identité n'est jamais vide pour un nouveau
  // joueur. Convention : un titre activant `spartan_customizer` doit fournir un asset
  // d'id 160 (sinon la modale resélectionne le 1er id du catalogue). Cf.
  // .ai/PLAN_H5_SPARTAN_CUSTOMIZER.md.
  emblemId: '160',
  nameplateId: '160',
  ...SPARTAN_DEFAULT_COLORS,
}

interface SpartanAppearanceState {
  /** Apparence enregistrée par slug de titre. */
  byTitle: Record<string, SpartanAppearance>
  setAppearance: (titleSlug: string, appearance: SpartanAppearance) => void
}

export const useSpartanAppearanceStore = create<SpartanAppearanceState>()(
  persist(
    (set) => ({
      byTitle: {},
      setAppearance: (titleSlug, appearance) =>
        set((s) => ({ byTitle: { ...s.byTitle, [titleSlug]: appearance } })),
    }),
    {
      name: 'levelup-spartan-appearance',
      version: 2,
      // v1 (Halo-5-only) stockait `{ appearance }` global → migrer vers `{ byTitle }`
      // en rattachant l'apparence existante au slug halo_5 (seul titre de l'époque).
      migrate: (persisted, version) => {
        if (
          version < 2 &&
          persisted &&
          typeof persisted === 'object' &&
          'appearance' in persisted
        ) {
          const legacy = (persisted as { appearance: SpartanAppearance }).appearance
          return { byTitle: { halo_5: legacy } } as unknown as SpartanAppearanceState
        }
        return persisted as SpartanAppearanceState
      },
    },
  ),
)

/** Apparence enregistrée pour un titre (ou défaut si rien choisi). Hook sélecteur. */
export function useSpartanAppearance(titleSlug: string): SpartanAppearance {
  return useSpartanAppearanceStore((s) => s.byTitle[titleSlug] ?? DEFAULT_SPARTAN_APPEARANCE)
}
