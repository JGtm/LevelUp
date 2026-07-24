import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import { SPARTAN_DEFAULT_COLORS } from '@/lib/accessibility/palettes/spartan'
import type { RecolorColors } from './recolor'

/**
 * Apparence Spartan choisie par le joueur — persistée en localStorage, **par titre ET
 * par joueur affiché**.
 *
 * v1 LOCAL (pas de backend). Le store détient l'apparence ENREGISTRÉE par couple
 * (titre, joueur) ; la modale tient un brouillon local et écrit ici au clic
 * « Enregistrer » (Annuler = pas d'écriture). Title-agnostic : chaque titre déclarant
 * la capability `spartan_customizer` garde sa propre apparence (ids emblème/bannière
 * spécifiques au titre). Dimension JOUEUR (V72-14) : la composition est propre au joueur
 * affiché (Home = joueur de la page ; Explorer = cible du bandeau), sinon un unique
 * réglage fuitait sur tous les joueurs. Follow-up backend cross-device : cf.
 * .ai/PLAN_H5_SPARTAN_CUSTOMIZER.md.
 */
export interface SpartanAppearance {
  /** id d'emblème (clé de spartan-catalog.json), null = aucun choix. */
  emblemId: string | null
  /** id de bannière (nameplate) — INDÉPENDANT de l'emblème ; null = aucun choix. */
  nameplateId: string | null
  /** Couleurs (primaire/secondaire/tertiaire) de l'EMBLÈME — indépendantes de la bannière. */
  emblemColors: RecolorColors
  /** Couleurs (primaire/secondaire/tertiaire) de la BANNIÈRE — indépendantes de l'emblème. */
  nameplateColors: RecolorColors
}

export const DEFAULT_SPARTAN_APPEARANCE: SpartanAppearance = {
  // Défaut neutre #160 → la bannière d'identité n'est jamais vide pour un nouveau
  // joueur. Convention : un titre activant `spartan_customizer` doit fournir un asset
  // d'id 160 (sinon la modale resélectionne le 1er id du catalogue). Cf.
  // .ai/PLAN_H5_SPARTAN_CUSTOMIZER.md.
  emblemId: '160',
  nameplateId: '160',
  // Emblème et bannière partent des mêmes couleurs par défaut (jamais de bandeau vide),
  // mais restent réglables séparément ensuite.
  emblemColors: { ...SPARTAN_DEFAULT_COLORS },
  nameplateColors: { ...SPARTAN_DEFAULT_COLORS },
}

/** Clé composite d'une apparence : `${titleSlug}::${playerSlug}`. */
export function appearanceKey(titleSlug: string, playerSlug: string): string {
  return `${titleSlug}::${playerSlug}`
}

interface SpartanAppearanceState {
  /** Apparence enregistrée par clé composite `${titleSlug}::${playerSlug}`. */
  byKey: Record<string, SpartanAppearance>
  setAppearance: (titleSlug: string, playerSlug: string, appearance: SpartanAppearance) => void
}

/**
 * Migration vers la v3 (re-clé par joueur + couleurs scindées emblème/bannière).
 *
 * Les données v1 (Halo-5-only, global) et v2 (`byTitle`, partagé entre tous les joueurs
 * d'un titre) portaient exactement le bug V72-14 : un fallback les rattachant à une clé
 * partagée recréerait la fuite. On RÉINITIALISE donc proprement — l'utilisateur re-règle
 * sa composition une fois, sur des données saines (par joueur, couleurs séparées).
 */
export function migrateSpartanAppearance(): SpartanAppearanceState {
  return { byKey: {} } as unknown as SpartanAppearanceState
}

export const useSpartanAppearanceStore = create<SpartanAppearanceState>()(
  persist(
    (set) => ({
      byKey: {},
      setAppearance: (titleSlug, playerSlug, appearance) =>
        set((s) => ({
          byKey: { ...s.byKey, [appearanceKey(titleSlug, playerSlug)]: appearance },
        })),
    }),
    {
      name: 'levelup-spartan-appearance',
      version: 3,
      migrate: migrateSpartanAppearance,
    },
  ),
)

/**
 * Apparence enregistrée pour un couple (titre, joueur), ou défaut si rien choisi.
 * Hook sélecteur. `playerSlug` = joueur AFFICHÉ (Home : joueur de la page ; Explorer :
 * la cible du bandeau, jamais le propriétaire).
 */
export function useSpartanAppearance(titleSlug: string, playerSlug: string): SpartanAppearance {
  return useSpartanAppearanceStore(
    (s) => s.byKey[appearanceKey(titleSlug, playerSlug)] ?? DEFAULT_SPARTAN_APPEARANCE,
  )
}
