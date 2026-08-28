/**
 * skullPresence.ts — LA RÈGLE DE PRÉSENCE DU CRÂNE d'Oddball (schéma 23), pure.
 *
 * POURQUOI CE MODULE. Le crâne libre (`objectiveObjects`) et le crâne porté (`skullCarries`)
 * sont DEUX couches disjointes : la première publie des positions réellement émises, la
 * seconde des intervalles de portage. Entre les deux, il reste des TROUS DE REPOS — le crâne
 * pose sur son socle et n'émet qu'un INSTANT UNIQUE (une vie où t0 == t1, un seul point). Le
 * primitif `objectiveObjectAt` ne réplique que dans [t0, t1] : une vie-socle clignote alors une
 * image puis disparaît, et le crâne devient invisible au repos (un tiers du film mesuré).
 *
 * Ce module compose les deux couches en une PRÉSENCE UNIQUE par image, sans jamais dessiner à
 * deux endroits (présence unique + précédence du portage). Il ne connaît ni React, ni le DOM,
 * ni la couleur, ni le moindre libellé : l'encre et le tracé sont résolus par l'appelant.
 *
 * LA RÈGLE EST LE TEST DU PROCHAIN ÉVÉNEMENT, et c'est la SEULE correcte (une formule naïve
 * « tenir jusqu'à la prochaine prise » garerait le crâne au point de chute-dans-le-vide pendant
 * le cooldown de respawn). Un repos n'est tenu que si une PRISE le corrobore : le porteur arrive
 * SUR la position tenue. Un repos suivi d'une VIE (respawn au socle, lâcher qui roule) rend
 * `absent` — pas de fantôme à la mauvaise place.
 *
 * DÉGRADATION : avec `carries: []` (artefacts antérieurs au schéma 23), aucune prise ne suit
 * jamais un repos → le maintien ne se déclenche pas → la présence retombe EXACTEMENT sur le
 * comportement historique (vie active seule, muet ailleurs).
 */
import { objectiveObjectAt } from './objectiveObjectsLayer'

import type { XY } from './replayLogic'
import type { ReplayObjectiveObjectReady, ReplaySkullCarry } from './replayNormalize'

/** SkullPresence — ce que le crâne montre à une image donnée, en une seule réponse. */
export type SkullPresence =
  /** Quelqu'un le porte : le calque libre se tait, `skullCarrierLayer` le dessine. */
  | { state: 'carried' }
  /** Il est au sol à `at` ; `rolling` s'il bouge encore (un point suit dans sa vie). */
  | { state: 'free'; at: XY; rolling: boolean }
  /** Il n'a pas de position honnête à cette image (pré-émission, respawn, retombée). */
  | { state: 'absent' }

/**
 * skullPresenceAt applique le test du prochain événement à l'image servie.
 *
 * 1. Une CARRY couvre F → `carried` (précédence sûre : lives et carries sont disjoints).
 * 2. Sinon une VIE ACTIVE couvre F → `free` au dernier point émis (= `objectiveObjectAt`).
 * 3. Sinon (TROU DE REPOS) : `lastRest` = la vie de plus grand `t1 <= F`. Aucune → `absent`.
 *    Sinon on regarde le PROCHAIN début strictement > F (toutes vies ET carries) :
 *      - une PRISE (carry.t0) → `free` TENU au dernier repos de `lastRest`, `rolling:false` ;
 *      - une VIE (life.t0), ou aucun → `absent`.
 *    Égalité vie/carry au même début : la VIE l'emporte (choix sûr, évite le fantôme).
 */
export function skullPresenceAt(
  lives: readonly ReplayObjectiveObjectReady[],
  carries: readonly ReplaySkullCarry[],
  frame: number,
): SkullPresence {
  for (const carry of carries) {
    if (carry.t0 <= frame && frame <= carry.t1) return { state: 'carried' }
  }
  for (const life of lives) {
    const now = objectiveObjectAt(life, frame)
    if (now) return { state: 'free', at: now.at, rolling: now.rolling }
  }
  let lastRest: ReplayObjectiveObjectReady | null = null
  for (const life of lives) {
    if (life.t1 <= frame && life.pts.length > 0 && (lastRest === null || life.t1 > lastRest.t1)) {
      lastRest = life
    }
  }
  if (lastRest === null) return { state: 'absent' }
  // Prochain début > F : les vies posent le seuil, une carry ne l'emporte que STRICTEMENT
  // plus tôt (une carry ex æquo avec une vie ne prend pas le pas — la vie gagne l'égalité).
  let nextStart = Infinity
  let nextIsCarry = false
  for (const life of lives) {
    if (life.t0 > frame && life.t0 < nextStart) {
      nextStart = life.t0
      nextIsCarry = false
    }
  }
  for (const carry of carries) {
    if (carry.t0 > frame && carry.t0 < nextStart) {
      nextStart = carry.t0
      nextIsCarry = true
    }
  }
  if (!nextIsCarry) return { state: 'absent' }
  const p = lastRest.pts[lastRest.pts.length - 1]
  return { state: 'free', at: { x: p.x, y: p.y }, rolling: false }
}
