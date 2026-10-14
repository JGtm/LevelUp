/**
 * carriedUsesLogic.ts — LES USAGES RESTANTS de l'équipement porté (schéma 95).
 *
 * LA SOURCE est `doc.carriedEquipment` : par objet porté (porteur, rang, prise `t0`, fin `t1`), son
 * énergie à chaque lecture du film et le nombre d'usages restants `n` que le SERVEUR calcule par la
 * règle du jeu (trunc((énergie + 1e-4) / coût), le coût lu dans les tags du jeu). Ce fichier n'en
 * calcule aucun : il retient la lecture la plus récente de l'objet que la vignette montre.
 *
 * DEUX JOINTURES : l'objet est celui du PORTEUR à cette image (prise passée, fin non atteinte) et de
 * MÊME RANG que la capacité affichée ; la lecture retenue est la plus récente de cet objet qui porte un
 * compte. Sans compte (famille dont le coût n'est pas établi, grappin et propulseur que le canal i56
 * compte déjà), rien n'est affirmé.
 *
 * PUR : aucun React, testé sans rien monter.
 */
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { AbilityChargeDisplay } from './abilityChargeLogic'

/**
 * carriedUsesAt — les usages restants de l'équipement de rang `rank` que porte `slot` à l'image `frame`,
 * ou null quand le document n'en établit aucun.
 */
export function carriedUsesAt(
  doc: ReplayDocumentReady,
  slot: number,
  frame: number,
  rank: number,
): AbilityChargeDisplay | null {
  let objet: ReplayDocumentReady['carriedEquipment'][number] | null = null
  for (const c of doc.carriedEquipment) {
    if (c.slot !== slot || c.r !== rank || c.t0 > frame) continue
    if (c.t1 !== undefined && c.t1 <= frame) continue
    if (!objet || c.t0 > objet.t0) objet = c
  }
  if (!objet) return null
  let lecture: ReplayDocumentReady['carriedEquipment'][number]['energy'][number] | null = null
  for (const e of objet.energy) {
    if (e.t > frame || e.n === undefined) continue
    if (!lecture || e.t >= lecture.t) lecture = e
  }
  if (!lecture || lecture.n === undefined) return null
  return { kind: 'count', charges: lecture.n, age: frame - lecture.t }
}
