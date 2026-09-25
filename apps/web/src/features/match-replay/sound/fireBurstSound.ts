/**
 * fireBurstSound.ts — LE SON D'UNE RAFALE DE TIR CONTINU (schéma 71, lot M4b des retours du rejeu).
 *
 * LA DÉCISION DE L'UTILISATEUR (Q5, validée) : un son PROLONGÉ pendant le tir, du début à la fin de
 * la rafale — pas une boucle qui relance l'attaque —, et le COUP en queue au lâcher. Deux formes,
 * selon ce que l'arme a de livré :
 *
 *  - UNE BOUCLE LIVRÉE (LMG du Wasp : `vehicleWeapons[w].loop`, reconnue par `VEHICLE_SHOT_LOOPS`) :
 *    elle est TENUE sur chaque passage lu de la rafale (les passages muets la coupent), et le coup
 *    de queue sonne au lâcher LU (`b1 = released`) — un lâcher perdu dans un trou n'a pas d'instant ;
 *  - SANS BOUCLE (Ghost, Banshee, Chopper, LMG, LAAG) : le coup se rejoue À LA CADENCE DE L'ARME,
 *    chaque coup COUPÉ au suivant — c'est le conteneur du jeu lui-même, en mode « cadence de
 *    déclenchement » (AkTransitionMode 5, manifeste V3 : Ghost 0,130 s, Banshee 0,125 s, Chopper
 *    0,250 s, LMG 0,077 s, la période de tir de chaque arme). Une voix par rafale, jamais un mur ;
 *    le dernier coup sonne entier ;
 *  - UN FAISCEAU (au-delà de `CADENCE_FAISCEAU`, le Rayon de Sentinelle à 60 coups par seconde : un
 *    coup par tick du jeu) : ses coups ne se distinguent plus, le son de l'arme est TENU comme une
 *    boucle.
 *
 * Une rafale dont l'arme n'a pas de son (silence décidé au registre, arme absente) se tait. Les
 * instants sont ceux de `model/fireBursts.ts` — les mêmes que les éclairs.
 */
import { frameToMs } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady, ReplayFireBurstReady } from '../../../lib/replay/replayNormalize'
import { burstShotFrames } from '../model/fireBursts'
import { vehicleWeaponOf } from '../model/vehicleWeaponRegistry'
import { soundEvent, type ReplaySoundEvent } from './replaySoundVariants'
import { VEHICLE_SHOT_LOOPS } from './vehicleShotSound'

/**
 * CADENCE_FAISCEAU — au-delà (coups par seconde), le tir est un FAISCEAU : 60/s est un coup par
 * tick de simulation (Rayon de Sentinelle, lu dans son tag), et aucune arme à projectiles lue au
 * lot M4b ne dépasse 18/s (LAAG). Le seuil sépare les deux familles mesurées ; une arme future qui
 * tomberait entre les deux se déciderait à l'oreille.
 */
export const CADENCE_FAISCEAU = 30

/** Les boucles que le client sait tenir (manifeste `VEHICLE_SHOT_LOOPS`). */
const BOUCLES_CONNUES = new Set(Object.values(VEHICLE_SHOT_LOOPS))

/**
 * fireBurstSoundEvents — les sons des rafales du document. `stemOf` résout le son d'une arme
 * (`shotSoundStem` de la piste : registre des armes de joueur, puis des armes de véhicule).
 */
export function fireBurstSoundEvents(
  doc: ReplayDocumentReady,
  stemOf: (w: string | undefined) => string | undefined,
): ReplaySoundEvent[] {
  const out: ReplaySoundEvent[] = []
  for (const b of doc.bursts) {
    const coup = stemOf(b.w)
    if (!coup) continue
    const boucle = boucleDe(doc, b, coup)
    if (boucle) out.push(...sonsTenus(doc, b, boucle, boucle === coup ? undefined : coup))
    else out.push(...coupsALaCadence(doc, b, coup))
  }
  return out
}

/** boucleDe — le son TENU d'une rafale : la boucle livrée de l'arme, ou son coup si c'est un faisceau. */
function boucleDe(doc: ReplayDocumentReady, b: ReplayFireBurstReady, coup: string): string | undefined {
  const livree = vehicleWeaponOf(doc, b.w)?.loop
  if (livree && BOUCLES_CONNUES.has(livree)) return livree
  return b.rate >= CADENCE_FAISCEAU ? coup : undefined
}

/**
 * sonsTenus — la boucle tenue sur chaque passage LU de la rafale, et le coup de queue au lâcher lu
 * (absent pour un faisceau, dont la boucle EST le coup).
 */
function sonsTenus(
  doc: ReplayDocumentReady,
  b: ReplayFireBurstReady,
  boucle: string,
  queue: string | undefined,
): ReplaySoundEvent[] {
  const out: ReplaySoundEvent[] = []
  let debut = b.t0
  for (const h of [...b.holes].sort((x, y) => x.t0 - y.t0)) {
    if (h.t0 > debut) out.push(tenu(doc, boucle, debut, h.t0))
    debut = Math.max(debut, h.t1)
  }
  // Une rafale d'UNE image tient tout de même une image : c'est la plus courte gâchette lue.
  if (b.t1 > debut || (b.holes.length === 0 && b.t1 === b.t0)) {
    out.push(tenu(doc, boucle, debut, Math.max(b.t1, debut + 1)))
  }
  if (queue && b.b1 === 'released') out.push(soundEvent(frameToMs(b.t1, doc), queue))
  return out
}

/** tenu — un événement de son tenu entre deux images. */
function tenu(doc: ReplayDocumentReady, stem: string, de: number, a: number): ReplaySoundEvent {
  const ms = frameToMs(de, doc)
  return { ...soundEvent(ms, stem), holdMs: frameToMs(a, doc) - ms }
}

/** coupsALaCadence — un coup par coup simulé, coupé au suivant ; le dernier sonne entier. */
function coupsALaCadence(doc: ReplayDocumentReady, b: ReplayFireBurstReady, coup: string): ReplaySoundEvent[] {
  const instants = burstShotFrames(b, frameToMs(1, doc)).map((t) => frameToMs(t, doc))
  return instants.map((ms, i) => {
    const suivant = instants[i + 1]
    const e = soundEvent(ms, coup)
    return suivant === undefined ? e : { ...e, cutMs: suivant - ms }
  })
}
