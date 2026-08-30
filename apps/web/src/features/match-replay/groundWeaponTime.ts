/**
 * groundWeaponTime.ts — QUAND une arme au sol se voit, et AVEC QUELLE FRANCHISE.
 *
 * LA DONNÉE (schéma 26, `groundWeapons`) porte trois bornes et non deux, et c'est tout le sujet
 * de ce fichier : `t0` l'apparition, `t1` la dernière PREUVE de présence, `t1max` la première
 * preuve d'ABSENCE. Sur une fin `seen` — l'objet cesse d'être recensé par les images-clés — la
 * disparition est un INTERVALLE, jamais un instant : le film ne dit pas où, entre les deux, elle
 * a eu lieu.
 *
 * D'OÙ LES DEUX RÉGIMES D'AFFICHAGE, et ils disent exactement ce que la mesure dit :
 *   - PLEIN de `t0` à `t1` — tant qu'une preuve tient, l'objet est là ;
 *   - ESTOMPÉ de `t1` à `t1max` — plus rien ne le prouve, rien ne le réfute encore. L'opacité
 *     descend jusqu'au plancher, et c'est cette DESCENTE qui se lit comme une réserve : une
 *     coupe franche à `t1` affirmerait une disparition que personne n'a vue, et tenir l'objet
 *     plein jusqu'à `t1max` affirmerait une présence que plus rien n'atteste ;
 *   - RIEN après `t1max` — la borne est une preuve d'absence, on ne la franchit pas.
 *
 * SUR `pickup` ET `open`, LE SECOND RÉGIME N'EXISTE PAS : le document y publie `t1max == t1`,
 * donc la fenêtre d'estompage est vide et l'objet passe du plein au néant. C'est voulu — un
 * ramassage est daté à la milliseconde, il n'a aucune réserve à porter ; une fin `open` court
 * jusqu'à la dernière image du document, il n'y a rien après elle.
 *
 * Tout ce fichier est PUR : aucun React, aucun canvas, donc testable — même partage que
 * `weaponPadTime.ts` face à `weaponPadsLayer.ts`.
 */
import type { ReplayGroundWeapon } from '@/lib/api/types'

/**
 * Opacité d'une arme au sol dont la présence est PROUVÉE.
 *
 * SOUS 1, ET C'EST UN CHOIX DE LECTURE : une arme abandonnée est un objet du décor, pas le
 * sujet. Elle doit se voir sans concurrencer les marqueurs de joueurs ni les socles, qui sont
 * peints par-dessus (cf. l'ordre des calques dans `ReplayCanvas`).
 */
export const GROUND_WEAPON_ALPHA_FULL = 0.8

/**
 * Opacité au bout de l'estompage — le PLANCHER, atteint à `t1max`.
 *
 * NON NULLE JUSQU'À LA DERNIÈRE IMAGE de l'intervalle : à zéro, l'objet s'éteindrait avant
 * `t1max` pour l'œil, et la borne haute mesurée ne se lirait plus. Elle vaut un quart du plein :
 * assez pour distinguer « il est peut-être encore là » de « il y est », assez peu pour qu'on ne
 * la confonde pas avec une présence établie.
 */
export const GROUND_WEAPON_ALPHA_FADED = 0.2

/** Ce que le rendu a besoin de savoir d'une arme au sol à une image donnée. */
export interface GroundWeaponPresence {
  /** Opacité de la vignette, dans [GROUND_WEAPON_ALPHA_FADED, GROUND_WEAPON_ALPHA_FULL]. */
  alpha: number
  /**
   * Vrai dans l'intervalle ]t1, t1max] : la disparition n'est pas datée, l'objet s'efface.
   *
   * PUBLIÉ SÉPARÉMENT DE L'OPACITÉ parce que ce n'est pas la même information : l'opacité est
   * un réglage d'écran, ceci est un état de la MESURE — un appelant qui voudrait le dire
   * autrement (un libellé, un pointillé) n'a pas à le redéduire d'un nombre flottant.
   */
  vanishing: boolean
}

/**
 * groundWeaponPresenceAt rend la présence d'UNE arme au sol à une image, ou `null` quand elle
 * ne doit pas être dessinée (pas encore apparue, ou passée sa première preuve d'absence).
 *
 * LA DESCENTE EST LINÉAIRE, et ce n'est pas un modèle : le film ne dit RIEN de la façon dont
 * l'objet a disparu dans l'intervalle. Une courbe savante suggérerait une connaissance qu'on
 * n'a pas ; une rampe droite se lit comme ce qu'elle est — « quelque part là-dedans ».
 */
export function groundWeaponPresenceAt(
  item: ReplayGroundWeapon,
  frame: number,
): GroundWeaponPresence | null {
  if (frame < item.t0) return null
  if (frame <= item.t1) return { alpha: GROUND_WEAPON_ALPHA_FULL, vanishing: false }
  // BORNE HAUTE STRICTE : `t1max` est une preuve d'ABSENCE, la franchir serait affirmer une
  // présence que le film réfute. `t1max <= t1` (fins `pickup` et `open`) tombe ici même.
  if (frame > item.t1max) return null
  const span = item.t1max - item.t1
  const done = span > 0 ? (frame - item.t1) / span : 1
  return {
    alpha: GROUND_WEAPON_ALPHA_FULL + (GROUND_WEAPON_ALPHA_FADED - GROUND_WEAPON_ALPHA_FULL) * done,
    vanishing: true,
  }
}

/**
 * groundWeaponsAt rend les armes VISIBLES à une image, chacune avec sa présence.
 *
 * UN BALAYAGE COMPLET PAR IMAGE, et c'est assumé : le calque en compte quelques dizaines par
 * match (les armes de socle, elles, restent à `weaponPads`), là où les trajectoires en portent
 * des milliers de points. Un index par frame coûterait plus à tenir qu'il ne fait gagner.
 */
export function groundWeaponsAt(
  items: readonly ReplayGroundWeapon[],
  frame: number,
): { item: ReplayGroundWeapon; presence: GroundWeaponPresence }[] {
  const out: { item: ReplayGroundWeapon; presence: GroundWeaponPresence }[] = []
  for (const item of items) {
    const presence = groundWeaponPresenceAt(item, frame)
    if (presence) out.push({ item, presence })
  }
  return out
}
