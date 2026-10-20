/**
 * catalogLabel.ts — LIRE UN LIBELLÉ DU DOCUMENT DANS LA LANGUE DU LECTEUR.
 *
 * Depuis le schéma v2 (lot 3.2), l'artefact ne porte plus une chaîne par libellé mais
 * `{en, fr}` : il est construit UNE FOIS, hors ligne, et servi aux deux locales — y figer
 * une seule langue reviendrait à choisir la langue du lecteur au moment du décodage d'un
 * film. Ce fichier est le SEUL endroit du rejeu qui choisit entre les deux.
 *
 * REPLI EXPLICITE : si la langue demandée est vide, on rend l'autre plutôt que rien. Un
 * libellé manquant n'est pas une donnée absente — c'est une configuration incomplète, et
 * afficher un identifiant brut à sa place tromperait sur la cause.
 */
import type { ReplayLocale } from './i18n'

/** La forme d'un libellé bilingue du document (miroir de replay.Label côté Go). */
export interface CatalogLabel {
  en?: string
  fr?: string
  /**
   * Vignette du HUD du jeu (grenades, capacités), servie par le document. Absente = pas
   * de visuel : l'appelant garde le libellé — jamais la vignette d'un voisin. `tinted`
   * dit un masque à teindre (même contrat que les icônes d'arme).
   */
  img?: string
  tinted?: boolean
}

/**
 * catalogText rend le texte d'un libellé bilingue, ou undefined si le libellé est absent
 * des deux côtés — l'appelant décide alors quoi montrer (un identifiant, un numéro).
 */
export function catalogText(label: CatalogLabel | undefined, locale: ReplayLocale): string | undefined {
  if (!label) return undefined
  const wanted = locale === 'en' ? label.en : label.fr
  const other = locale === 'en' ? label.fr : label.en
  return wanted || other || undefined
}
