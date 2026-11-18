/**
 * reliefPlan.logic — CE QUE LA COUPE PAR ÉTAGE FAIT DES CALQUES du plan de l'onglet Tactique, et ce que la légende en dit.
 *
 * Deux calques seulement portent une hauteur et suivent la coupe : les zones nommées de la lecture de placement et
 * celles de la lecture des réapparitions (`zonesInCut`, par `z_bottom` et `z_top`, conception §7). Tous les autres
 * restent ENTIERS — le contrat du serveur ne leur donne aucune hauteur —, et la légende du pied du plan le dit, calque
 * par calque, quand une coupe est choisie et que le calque est à l'écran.
 *
 * Pur : aucun React, aucune langue (les noms viennent de `TacticalText`).
 */
import type { TacticalText } from './i18n'

/** Les calques dessinés qui ne portent pas la hauteur, dans l'ordre de leur mention. */
export type CalqueSansHauteur = 'chaleur' | 'murs' | 'tueurs' | 'coequipiers'

/** Ce qui est à l'écran et ne porte pas la hauteur. */
export interface CalquesAffiches {
  /** La chaleur lissée de la lecture de placement est peinte. */
  chaleur: boolean
  /** Le plan des murs (calque, pas l'image plan) est tracé. */
  murs: boolean
  /** La couche des tueurs et victimes de la zone choisie est peinte. */
  tueurs: boolean
  /** La scène des coéquipiers (nuage, instantané, séquence, cercle du radar) est peinte. */
  coequipiers: boolean
}

/**
 * calquesSansHauteur : les calques à nommer dans la légende. Rien sans coupe (`cut === null`) : « Tout » montre
 * tous les étages, aucun calque n'est alors trompeur.
 */
export function calquesSansHauteur(cut: number | null, affiches: CalquesAffiches): CalqueSansHauteur[] {
  if (cut === null) return []
  const ordre: CalqueSansHauteur[] = ['chaleur', 'murs', 'tueurs', 'coequipiers']
  return ordre.filter((calque) => affiches[calque])
}

/** La mention de légende d'un calque entier : « {calque} : ce calque ne porte pas la hauteur, … ». */
export function mentionSansHauteur(t: TacticalText, calque: CalqueSansHauteur): string {
  const noms: Record<CalqueSansHauteur, string> = {
    chaleur: t.noHeightHeat,
    murs: t.noHeightWalls,
    tueurs: t.noHeightKillers,
    coequipiers: t.noHeightMates,
  }
  return t.legendNoHeight(noms[calque])
}
