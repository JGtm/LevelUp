/**
 * equipmentFx.ts — L'ÉTAT ACTIF D'UN ÉQUIPEMENT sur la fiche joueur, lu dans les épisodes.
 *
 * LA SOURCE est le document (schéma 7) : `equipmentEpisodes`, des épisodes datés PAR VIE
 * (slot) sur l'axe des frames — les deux seules familles dont l'état est MESURÉ côté film
 * (camouflage : i28 queue[1], interrupteur binaire ; surbouclier : i5 non clampé, règle
 * q > 64). La DURÉE de l'effet EST l'épisode : actif exactement quand la frame est dans
 * [t0, t1], aucune rémanence inventée.
 *
 * L'EFFET S'ATTACHE À LA FICHE PAR LA VIE (slot -> fiche), la même chaîne que le flash de
 * mort : `playerStateAt` donne la vie courante, son slot désigne les épisodes. Un épisode
 * se ferme à la mort au plus tard (le document le garantit), donc une fiche morte n'a
 * jamais d'effet d'équipement.
 *
 * Pas de React, pas de canvas : logique pure, testée (equipmentFx.test.ts).
 */
import type { ReplayDocumentReady } from './replayNormalize'

/**
 * Les deux familles publiées par le document. Identifiants STABLES du contrat (`fam`) —
 * une famille inconnue ne reçoit AUCUN effet, jamais celui d'une voisine.
 */
export const EQUIP_FAMILY_CAMO = 'camo'
export const EQUIP_FAMILY_OVERSHIELD = 'overshield'

/** Ce que la fiche doit dire à une frame : chaque famille active, indépendamment. */
export interface ActiveEquipment {
  camo: boolean
  overshield: boolean
}

/** Aucune allocation par frame : la valeur « rien d'actif » est partagée. */
const NONE: ActiveEquipment = Object.freeze({ camo: false, overshield: false })

/**
 * activeEquipmentAt dit quelles familles sont actives pour un SLOT à une frame.
 *
 * La recherche porte sur le slot, jamais sur le joueur — un slot est une vie, et c'est ce
 * qui rend la lecture sûre : un épisode ne peut pas survivre à son porteur. Les épisodes
 * sont peu nombreux (dizaines par film) : le balayage linéaire par frame reste négligeable
 * devant le rendu.
 */
export function activeEquipmentAt(
  doc: ReplayDocumentReady,
  slot: number,
  frame: number,
): ActiveEquipment {
  let camo = false
  let overshield = false
  for (const e of doc.equipmentEpisodes) {
    if (e.slot !== slot || frame < e.t0 || frame > e.t1) continue
    if (e.fam === EQUIP_FAMILY_CAMO) camo = true
    else if (e.fam === EQUIP_FAMILY_OVERSHIELD) overshield = true
    if (camo && overshield) break
  }
  if (!camo && !overshield) return NONE
  return { camo, overshield }
}
