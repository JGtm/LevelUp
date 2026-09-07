/**
 * cardGabarit.ts — LES COTES D'UNE FICHE JOUEUR, en un seul objet, et ses DEUX gabarits.
 *
 * POURQUOI UN OBJET (plan fiches compactes 2026-09-06, I1). Sept constantes de cotes vivaient
 * dans quatre fichiers (`WEAPON_CELL_W`, `AMMO_CELL_W`, `GRENADES_BOX_W`, `SCORE_CELL_W`,
 * `COUNT_CELL_W`, `GRENADE_ICON_PX`, `HUD_ICON_PX`, `WATERMARK_PX`…) et les hauteurs de
 * jauges étaient écrites dans la fiche. Le jour où une Grande équipe (12v12) demande une tuile
 * plus dense, chaque composant devrait apprendre un second jeu de valeurs — c'est exactement
 * la forme que le commit `10fe3228a` (2026-08-25) a supprimée : une prop `compact` drillée
 * avec sa branche `if (compact)` dans chaque rangée. Ici, les composants reçoivent DES NOMBRES
 * ET DES BOOLÉENS, jamais un mot de densité : ils rendent ce qu'on leur donne, et c'est
 * `cardDensity.ts` — et lui seul — qui choisit lequel des deux objets donner.
 *
 * `GABARIT_NORMAL` porte EXACTEMENT les constantes d'aujourd'hui : c'est ce qui garantit que
 * la fiche d'un match qui n'est pas une Grande équipe rend le même DOM, nœud pour nœud
 * (fixation `ui/__fixtures__/replayTeams.4v4.html`, prise avant ce module).
 * `GABARIT_COMPACT` porte les cotes de la maquette A2 (tuile 115 × 62, deux colonnes par camp).
 *
 * DEUX COTES RESTENT DES CLASSES, PAS DES NOMBRES APPLIQUÉS EN STYLE : la hauteur du corps
 * (`h-[35px]`) et la taille du nom (`text-[11.5px]`) sont des valeurs arbitraires Tailwind,
 * qui n'existent que si le littéral est écrit en clair dans une classe — une interpolation ne
 * produit AUCUNE règle, en silence (`rosterHeight.guard.test.ts` documente le piège). `bodyPx`
 * et `namePx` sont donc ici la VÉRITÉ que la tuile traduit en classe par une table fermée, et
 * le garde-rail `ui/cardGabarit.guard.test.ts` interdit qu'une cote revienne en constante dans
 * un composant.
 */

export interface CardGabarit {
  /** Nombre de cellules d'arme : deux (main + rangée) ou une (main seule, l'autre en infobulle). */
  weaponCells: 1 | 2
  /** Largeur d'UNE cellule d'arme, px. */
  weaponCellW: number
  /** Cellule de munitions de la main rendue (sinon : dans l'infobulle de l'arme). */
  showAmmo: boolean
  /** Largeur de la cellule de munitions, px — sans effet quand `showAmmo` est faux. */
  ammoCellW: number
  /** Boîte des trois stocks de grenades rendue (sinon : la grenade sélectionnée seule). */
  showGrenadeStock: boolean
  /** Largeur de la boîte de grenades, px — sans effet quand `showGrenadeStock` est faux. */
  grenadesBoxW: number
  /** Cellule du score personnel rendue (toujours, même vide — c'est elle qui aligne). */
  showScore: boolean
  /** Largeur minimale de la cellule de score, px — sans effet quand `showScore` est faux. */
  scoreCellW: number
  /** Largeur minimale d'UN compteur du triplet F/M/A, px. */
  countCellW: number
  /** Hauteur de la jauge de bouclier, px (au-dessus de la santé : l'ordre du jeu). */
  gaugeShieldPx: number
  /** Hauteur de la jauge de santé, px. */
  gaugeHealthPx: number
  /** Hauteur FIXE du corps de la fiche, px — la même vivant et mort (règle du 2026-08-24). */
  bodyPx: number
  /** Côté de la vignette d'un type de grenade, px. */
  iconGrenadePx: number
  /** Côté de la vignette de capacité, px. */
  iconAbilityPx: number
  /** Côté du filigrane de porteur d'objectif, px (critère : « reste un fond »). */
  watermarkPx: number
  /** Nombre d'éclairs de l'écran occultant (largeurs absolues sur une tuile étroite). */
  boltCount: number
  /** Taille du nom, px. */
  namePx: number
  /** Sièges d'un camp en grille à remplissage automatique (vrai) ou en colonne simple (faux). */
  seatGrid: boolean
}

/** La fiche d'aujourd'hui, valeur pour valeur — 235 px de large, corps de 35 px. */
export const GABARIT_NORMAL: CardGabarit = Object.freeze({
  weaponCells: 2,
  weaponCellW: 40,
  showAmmo: true,
  ammoCellW: 32,
  showGrenadeStock: true,
  grenadesBoxW: 56,
  showScore: true,
  scoreCellW: 30,
  countCellW: 15,
  gaugeShieldPx: 5,
  gaugeHealthPx: 3,
  bodyPx: 35,
  iconGrenadePx: 14,
  iconAbilityPx: 16,
  watermarkPx: 46,
  boltCount: 3,
  namePx: 11.5,
  seatGrid: false,
} as const)

/**
 * La tuile compacte de la maquette A2 (plan 2026-09-06, « Le gabarit tranché ») : 115 × 62,
 * trois lignes (nom 14 · jauges + triplet 12 · arme 48 + grenade 14 + capacité 16 sur 16),
 * corps fixe de 31 px. Ce qui quitte la tuile passe en infobulle avec sa valeur.
 * NON CONSOMMÉ PAR UN RENDU avant l'étape 2 du plan : ce module ne fait que le déclarer.
 */
export const GABARIT_COMPACT: CardGabarit = Object.freeze({
  weaponCells: 1,
  weaponCellW: 48,
  showAmmo: false,
  ammoCellW: 32,
  showGrenadeStock: false,
  grenadesBoxW: 56,
  showScore: false,
  scoreCellW: 30,
  countCellW: 10,
  gaugeShieldPx: 4,
  gaugeHealthPx: 2,
  bodyPx: 31,
  iconGrenadePx: 14,
  iconAbilityPx: 16,
  watermarkPx: 34,
  boltCount: 2,
  namePx: 11.5,
  seatGrid: true,
} as const)
