/**
 * medalEvent.ts — LES TYPES D'UNE MÉDAILLE DU FIL, lue dans le document de rejeu (`matchMedals`). Ils vivent ici
 * pour que le fil (`killFeedLogic`, qui les réexporte) ne porte que sa logique.
 */

/** La découpe d'une feuille de sprites : où lire la médaille, et sa taille native, en pixels de la feuille. */
export interface MedalSprite {
  sheet: string
  left: number
  top: number
  width: number
  height: number
}

/** Une médaille du fil, lue dans le document de rejeu (cf. `matchMedals`). */
export interface MedalEvent {
  /** Instant sur l'horloge gameplay (`medals[].ms` du document). */
  tMs: number
  xuid: string
  gamertag: string
  /** Nom ANGLAIS mesuré (film). Toujours présent — c'est la clé du référentiel. */
  name: string
  /** Libellé dans la langue de la page. Vide = non résolu : le nom brut s'affiche en toutes lettres. */
  label: string
  description: string
  imageUrl: string
  /**
   * La découpe d'une feuille de sprites (titre à planches d'images), que le document sert quand la médaille
   * n'a pas d'image propre ; `null` ou absente sinon. Elle se dessine par `MedalIcon`, comme sur la fiche.
   */
  sprite?: MedalSprite | null
}
