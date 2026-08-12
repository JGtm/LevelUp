/**
 * equippedLogic.ts — QUELLE arme est en main, lue du film, et rien de deviné.
 *
 * D'OÙ VIENT L'INFORMATION. Le sélecteur d'emplacement dégainé (i42) voyage dans le bloc
 * d'inventaire des images-clés (`Inventory.D`) : 0 ou 1 = l'emplacement dégainé dans l'ordre
 * de `Loadout.W`, 2 = AUCUNE arme dégainée — une valeur à part entière, c'est l'état des
 * huit joueurs à la première image-clé, avant le départ. Absent = non lu.
 *
 * DEUX LECTURES, DEUX ÂGES, UNE HONNÊTETÉ. Le loadout et l'inventaire sont deux scans
 * distincts du même film : chacun est reporté séparément à sa dernière image-clé (~20 s).
 * Entre deux, un swap est invisible — l'âge et l'estompage le disent, jamais une continuité
 * jouée. Quand le sélecteur désigne un emplacement que le loadout lu ne porte pas (lectures
 * désappariées), la mise en valeur s'abstient : afficher une main douteuse serait afficher
 * une certitude qu'on n'a pas.
 *
 * Tout ce fichier est PUR : aucun React, aucun canvas, donc testable.
 */
import type { ReplayDocumentReady } from './replayNormalize'
import { inventoryAt, loadoutAt } from './rosterLogic'

/** Une arme de la rangée : son identifiant de famille, et si elle est EN MAIN. */
export interface EquippedWeapon {
  id: string
  inHand: boolean
}

/**
 * EquippedReading — les armes ordonnées pour l'affichage : l'arme EN MAIN d'abord (à
 * gauche, comme dans le jeu), la secondaire ensuite. L'ÂGE est celui de la lecture du
 * loadout, indissociable de la valeur (cf. LoadoutReading).
 */
export interface EquippedReading {
  weapons: EquippedWeapon[]
  /**
   * Indices d'EMPLACEMENT du record, dans l'ordre d'affichage de `weapons`. La ligne des
   * MUNITIONS suit ce même ordre — les deux lignes doivent rester alignées, la seconde se
   * lisant « dans l'ordre des armes au-dessus ».
   */
  order: number[]
  /** Emplacement dégainé (0 ou 1) quand le sélecteur le désigne, sinon null. */
  drawn: number | null
  /** Âge de la lecture du loadout, en frames. */
  age: number
  /** true = le sélecteur dit « rien de dégainé » (D=2) — une mesure, pas une lacune. */
  holstered: boolean
  /** true = le sélecteur n'a PAS été lu sur ce record — une lacune, pas une mesure. */
  drawnUnread: boolean
}

/**
 * equippedWeapons rend les armes d'un SLOT ordonnées par le sélecteur d'emplacement.
 *
 * LA RECHERCHE PORTE SUR LE SLOT, comme le loadout et l'inventaire : un slot est réattribué
 * à chaque réapparition, donc aucune lecture ne franchit une mort. Sans sélecteur lisible
 * (absent, ou désappariement des deux scans), l'ordre reste celui des emplacements et
 * AUCUNE arme n'est marquée en main.
 */
export function equippedWeapons(
  doc: ReplayDocumentReady,
  slot: number,
  frame: number,
): EquippedReading | null {
  const lo = loadoutAt(doc, slot, frame)
  if (!lo) return null
  const d = inventoryAt(doc, slot, frame)?.state.d
  const held = d === 0 || d === 1 ? lo.weapons[d] : undefined
  if (held === undefined) {
    return {
      weapons: lo.weapons.map((id) => ({ id, inHand: false })),
      order: lo.weapons.map((_, i) => i),
      drawn: null,
      age: lo.age,
      holstered: d === 2,
      drawnUnread: d === undefined,
    }
  }
  const rest = lo.weapons.map((_, i) => i).filter((i) => i !== d)
  return {
    weapons: [
      { id: held, inHand: true },
      ...rest.map((i) => ({ id: lo.weapons[i], inHand: false })),
    ],
    order: [d as number, ...rest],
    drawn: d as number,
    age: lo.age,
    holstered: false,
    drawnUnread: false,
  }
}

/**
 * drawnSwapAt — l'ÉCHANGE D'ARME en cours : l'âge, en frames, de la dernière BASCULE du
 * sélecteur d'emplacement (i42) pour ce slot, ou null hors fenêtre.
 *
 * LA BASCULE EST DATÉE À L'IMAGE-CLÉ — le moment où le changement devient LISIBLE, pas le
 * moment du geste : vingt secondes séparent deux images-clés et le film ne dit pas l'instant
 * exact. L'animation marque donc une lecture, pas un événement. Sans mouvement, une
 * permutation de vignettes est indiscernable d'un changement d'arme — c'est sa raison d'être.
 */
export function drawnSwapAt(
  doc: ReplayDocumentReady,
  slot: number,
  frame: number,
  windowFrames: number,
): number | null {
  let prev: number | null = null
  let swapT: number | null = null
  for (const inv of doc.inventory) {
    if (inv.slot !== slot || inv.t > frame) continue
    const d = inv.d
    if (d !== 0 && d !== 1) continue
    if (prev !== null && d !== prev) swapT = inv.t
    prev = d
  }
  if (swapT === null) return null
  const age = frame - swapT
  return age >= 0 && age <= windowFrames ? age : null
}

/**
 * SwapReading — un CHANGEMENT de dotation entre les deux dernières lectures d'un slot.
 *
 * C'est un ÉTAT DE RÉFÉRENCE à ~20 s de granularité, jamais un suivi continu : le film ne
 * porte aucun événement de ramassage, seul le diff entre deux images-clés le laisse voir.
 * Un aller-retour complet entre deux lectures reste invisible — et on ne l'invente pas.
 */
export interface SwapReading {
  /** Identifiants ENTRÉS dans la dotation depuis la lecture précédente. */
  picked: string[]
  /** Identifiants SORTIS de la dotation depuis la lecture précédente. */
  dropped: string[]
  /** Âge de la lecture courante, en frames : la borne récente de l'intervalle du swap. */
  age: number
}

/**
 * loadoutSwapAt compare les DEUX dernières lectures de loadout d'un SLOT au plus tard à
 * `frame`, et rend ce qui a changé — null sans changement, ou sans deux lectures à comparer.
 *
 * La comparaison est un diff de MULTIENSEMBLE : porter deux fois la même arme puis n'en
 * garder qu'une est un changement, et l'ordre des emplacements n'en est pas un.
 */
export function loadoutSwapAt(
  doc: ReplayDocumentReady,
  slot: number,
  frame: number,
): SwapReading | null {
  let cur: { t: number; w: string[] } | null = null
  let prev: { t: number; w: string[] } | null = null
  for (const l of doc.loadouts) {
    if (l.slot !== slot || l.t > frame) continue
    if (!cur || l.t > cur.t) {
      prev = cur
      cur = l
    } else if (!prev || l.t > prev.t) {
      prev = l
    }
  }
  if (!cur || !prev) return null
  const picked = multisetDiff(cur.w, prev.w)
  const dropped = multisetDiff(prev.w, cur.w)
  if (picked.length === 0 && dropped.length === 0) return null
  return { picked, dropped, age: frame - cur.t }
}

/** multisetDiff rend les éléments de `a` restants après avoir retiré ceux de `b`, occurrence par occurrence. */
function multisetDiff(a: string[], b: string[]): string[] {
  const counts = new Map<string, number>()
  for (const id of b) counts.set(id, (counts.get(id) ?? 0) + 1)
  return a.filter((id) => {
    const c = counts.get(id) ?? 0
    if (c > 0) {
      counts.set(id, c - 1)
      return false
    }
    return true
  })
}
