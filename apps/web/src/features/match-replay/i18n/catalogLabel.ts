/**
 * catalogLabel.ts — LIRE UN LIBELLÉ DU DOCUMENT DANS LA LANGUE DU LECTEUR.
 *
 * Depuis le schéma v2 (lot 3.2), l'artefact ne porte plus une chaîne par libellé mais une
 * propriété par langue (`{ en, fr, … }`, ADR 0040) : il est construit UNE FOIS, hors ligne, et
 * servi à toutes les langues — y figer une seule langue reviendrait à choisir la langue du
 * lecteur au moment du décodage d'un film. Le choix de la langue se fait dans `pickText`
 * (`lib/i18n/pickText`), le seul endroit qui connaisse les noms de propriété de langue.
 *
 * REPLI : la langue du lecteur, sinon l'anglais (D3, noms de jeu), sinon rien. Un libellé
 * manquant n'est pas une donnée absente — c'est une configuration incomplète, et afficher un
 * identifiant brut à sa place tromperait sur la cause.
 */
import { type LocaleTexts, pickText } from '@/lib/i18n/pickText'

import type { ReplayLocale } from './i18n'

/** La forme d'un libellé du document (miroir de replay.Label côté Go) : un texte par langue, plus son visuel. */
export interface CatalogLabel extends LocaleTexts {
  /**
   * Vignette du HUD du jeu (grenades, capacités), servie par le document. Absente = pas
   * de visuel : l'appelant garde le libellé — jamais la vignette d'un voisin. `tinted`
   * dit un masque à teindre (même contrat que les icônes d'arme).
   */
  img?: string
  tinted?: boolean
}

/**
 * catalogText rend le texte d'un libellé du document, ou undefined si le libellé est absent
 * dans la langue demandée et dans celle de repli — l'appelant décide alors quoi montrer (un
 * identifiant, un numéro).
 */
export function catalogText(label: CatalogLabel | undefined, locale: ReplayLocale): string | undefined {
  return pickText(label, locale) || undefined
}
