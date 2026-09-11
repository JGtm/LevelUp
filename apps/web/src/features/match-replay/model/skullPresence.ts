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
 * SUR la position tenue. Un repos suivi d'une VIE (respawn au socle, lâcher qui roule) NE tient
 * pas la position précédente — pas de fantôme à la mauvaise place.
 *
 * MAIS LE CRÂNE N'EST JAMAIS NULLE PART : quand il n'est ni porté, ni au sol en jeu, il est
 * RENTRÉ SUR SON SOCLE (son point de réapparition). Là où la règle disait `absent` (avant sa
 * première émission, pendant un cooldown de respawn hors-zone), on le pose sur son socle si on
 * sait où il est — et on le sait : `skullSocle` le lit comme le MODE des vies-instant (le crâne
 * y réapparaît en boucle, une chute dans le vide n'y émet qu'un instant isolé). Sans socle
 * identifiable (artefact trop court), on retombe sur `absent`.
 *
 * LA PRÉCÉDENCE DU PORTAGE EST CONDITIONNÉE À UNE POSITION DU PORTEUR (audit du 2026-09-10,
 * causes C9/C10 ; lot 6.7 phase B2). Le portage est un INTERVALLE, la trajectoire du bipède une
 * suite d'échantillons qui peut s'interrompre : sur 694 images du parc Oddball (7,0 %, rapport
 * 6.2 §2.1) le portage est vrai mais le porteur n'a PAS de position. Rendre `carried` là-dessus
 * faisait taire ce module pendant que `skullCarrierLayer` ne dessinait rien faute de position —
 * le crâne DISPARAISSAIT. La règle commune aux trois glyphes portés (`carriedGlyphPlace.ts`) le
 * rend désormais LIBRE à sa dernière position connue : celle du porteur, à défaut le dernier
 * repos, à défaut le socle. Un seul glyphe à l'écran, jamais zéro, jamais deux.
 *
 * DÉGRADATION : avec `carries: []` (artefacts antérieurs au schéma 23), aucune prise ne suit
 * jamais un repos → le maintien ne se déclenche pas → la présence retombe EXACTEMENT sur le
 * comportement historique (vie active seule, muet ailleurs).
 */
import { objectiveObjectAt } from '../layers/objectiveObjectsLayer'

import { carriedGlyphPlaceAt, type CarrierPosAt } from './carriedGlyphPlace'
import type { XY } from '../../../lib/replay/replayLogic'
import type { ReplayObjectiveObjectReady, ReplaySkullCarry } from '../../../lib/replay/replayNormalize'
import { covers } from './replaySpans'

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
 * 1. Une CARRY couvre F → `carried`, MAIS SEULEMENT SI SON PORTEUR A UNE POSITION À CETTE IMAGE
 *    (cf. l'en-tête, précédence conditionnée). Sinon le crâne est LIBRE à sa dernière position
 *    connue — celle du porteur, à défaut le dernier repos, à défaut le socle.
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
  socle: XY | null = null,
  posOf: CarrierPosAt | null = null,
): SkullPresence {
  for (const carry of carries) {
    if (!covers(carry, frame)) continue
    // SANS LECTEUR DE POSITION, la précédence du portage reste entière : c'est la dégradation
    // des appelants qui ne relisent pas les trajectoires (et le comportement d'avant ce lot).
    if (!posOf) return { state: 'carried' }
    const place = carriedGlyphPlaceAt(posOf, carry, frame, lastRestPointOf(lives, frame) ?? socle)
    if (place.state === 'carried') return { state: 'carried' }
    // Le crâne LIBRE au dernier point connu : il ne ROULE pas — cette position est un repos
    // constaté, pas un échantillon de trajectoire en cours.
    return place.state === 'free' ? { state: 'free', at: place.at, rolling: false } : { state: 'absent' }
  }
  for (const life of lives) {
    const now = objectiveObjectAt(life, frame)
    if (now) return { state: 'free', at: now.at, rolling: now.rolling }
  }
  const lastRest = lastRestOf(lives, frame)
  if (lastRest === null) return restOnSocle(socle)
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
  if (!nextIsCarry) return restOnSocle(socle)
  const p = lastRest.pts[lastRest.pts.length - 1]
  return { state: 'free', at: { x: p.x, y: p.y }, rolling: false }
}

/**
 * lastRestOf — la vie de plus grand `t1 <= frame` qui ait émis au moins un point, ou `null`.
 * C'est le DERNIER ENDROIT OÙ LE CRÂNE A ÉTÉ VU au sol avant l'image servie.
 */
function lastRestOf(
  lives: readonly ReplayObjectiveObjectReady[],
  frame: number,
): ReplayObjectiveObjectReady | null {
  let lastRest: ReplayObjectiveObjectReady | null = null
  for (const life of lives) {
    if (life.t1 <= frame && life.pts.length > 0 && (lastRest === null || life.t1 > lastRest.t1)) {
      lastRest = life
    }
  }
  return lastRest
}

/** Le dernier point émis du dernier repos, ou `null` — le repli de position de l'objet lui-même. */
function lastRestPointOf(
  lives: readonly ReplayObjectiveObjectReady[],
  frame: number,
): XY | null {
  const rest = lastRestOf(lives, frame)
  if (rest === null) return null
  const p = rest.pts[rest.pts.length - 1]
  return { x: p.x, y: p.y }
}

/**
 * restOnSocle — le crâne au REPOS est sur son SOCLE. Quand la présence est autrement `absent`
 * (avant sa première émission, pendant un cooldown de respawn hors-zone), le crâne n'a pas
 * disparu : il est rentré sur son socle. On l'y pose SI le socle est connu ; sinon, faute de
 * position honnête, `absent` (comportement historique, artefact sans socle identifiable).
 */
function restOnSocle(socle: XY | null): SkullPresence {
  return socle ? { state: 'free', at: socle, rolling: false } : { state: 'absent' }
}

/**
 * skullSocle rend la position du SOCLE du crâne, ou `null`. Le socle est là où le crâne
 * RÉAPPARAÎT : ses vies-instant (t0 == t1, un seul point) s'y répètent, tandis qu'une chute
 * dans le vide n'émet son instant qu'UNE fois, ailleurs. Le socle est donc le MODE des positions
 * de vies-instant, au mètre. On exige une RÉCURRENCE (≥ 2) : un instant unique isolé est plus
 * probablement une chute qu'un socle. Se calcule UNE fois par document (l'appelant mémoïse).
 */
export function skullSocle(lives: readonly ReplayObjectiveObjectReady[]): XY | null {
  const tally = new Map<string, { at: XY; n: number }>()
  for (const life of lives) {
    if (life.t0 !== life.t1 || life.pts.length === 0) continue
    const p = life.pts[0]
    const key = `${Math.round(p.x)},${Math.round(p.y)}`
    const seen = tally.get(key)
    if (seen) seen.n += 1
    else tally.set(key, { at: { x: p.x, y: p.y }, n: 1 })
  }
  let best: { at: XY; n: number } | null = null
  for (const e of tally.values()) {
    if (best === null || e.n > best.n) best = e
  }
  return best && best.n >= 2 ? best.at : null
}
