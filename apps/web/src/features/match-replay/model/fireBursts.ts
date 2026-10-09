/**
 * fireBursts.ts — LES COUPS D'UNE RAFALE DE TIR CONTINU, POSÉS COMME THEATER LES POSE (schéma 71,
 * lot M4b des retours du rejeu).
 *
 * LE FILM N'ÉCRIT PAS L'INSTANT DE CHAQUE COUP d'une arme à tir continu (Ghost, canons de la
 * Banshee, Chopper, LMG, LAAG, tourelles, Rayon de Sentinelle) : il écrit la gâchette TENUE, au
 * tick, dans la vue de contrôle. Le jeu simule les coups à la cadence du TAG de l'arme ; le document
 * publie la rafale (`doc.bursts` : bornes, arme, véhicule porteur, cadence, passages muets) et ce
 * module pose les coups — le PREMIER à la pose de la gâchette, les suivants à la cadence, montée
 * en cadence comprise (LAAG : de 5 à 18 coups par seconde en 1,6 s). C'est le calcul du serveur
 * (`replay/fire_bursts.go`, `instantsDesCoups`), et le compte `coverage.continuousFire.shots` en
 * est le contrôle.
 *
 * LES PASSAGES MUETS (`holes`) ne portent AUCUN coup : la lecture ne les a pas atteints, et
 * rien n'autorise à y supposer la gâchette tenue (décision de l'utilisateur du 2026-09-24).
 *
 * UN COUP DEVIENT UN TIR DU DOCUMENT (`fireBurstShots`) : même forme que `doc.shots` — frame,
 * slot du tireur, position, arme, véhicule — pour que l'éclair de bouche et le « ! » du tireur le
 * dessinent comme n'importe quel tir. Un coup par FRAME au plus et par rafale : au-delà (Rayon de
 * Sentinelle, 60 coups par seconde), l'éclair se relance à chaque image, ce qui EST un faisceau.
 *
 * Pur : aucun React, aucun canvas.
 */
import { frameToMs } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady, ReplayFireBurstReady } from '../../../lib/replay/replayNormalize'
import { buildCarrierPosAt } from './carrierPosition'
import { buildLivesBySlot, lifeOfSlotAt } from './livesPosition'

/**
 * EPSILON_FRAME — la tolérance de la borne de fin : les coups s'accumulent en flottants, et un coup
 * qui tombe EXACTEMENT sur la dernière image (60/s : 6 coups par image) ne doit pas s'en perdre
 * par l'arrondi. La même que le serveur (`replay/fire_bursts.go`, `epsilonFrame`).
 */
const EPSILON_FRAME = 1e-6

/** Un tir simulé d'une rafale, dans la forme d'un tir du document. */
export interface FireBurstShot {
  t: number
  slot: number
  x: number
  y: number
  w?: string
  v?: number
}

/**
 * burstShotFrames — les instants des coups d'une rafale, en frames FRACTIONNAIRES, hors de ses
 * passages muets. `frameIntervalMs` est le pas du document.
 */
export function burstShotFrames(b: ReplayFireBurstReady, frameIntervalMs: number): number[] {
  if (!(b.rate > 0) || !(frameIntervalMs > 0)) return []
  const parFrame = frameIntervalMs / 1000
  const out: number[] = []
  for (let t = b.t0; t <= b.t1 + EPSILON_FRAME; ) {
    if (!inHole(b, t)) out.push(t)
    const ecoule = (t - b.t0) * parFrame
    const monte = (b.rate0 ?? 0) > 0 && (b.ramp ?? 0) > 0 && ecoule < (b.ramp ?? 0)
    const cadence = monte
      ? (b.rate0 ?? 0) + ((b.rate - (b.rate0 ?? 0)) * ecoule) / (b.ramp ?? 1)
      : b.rate
    t += 1 / (cadence * parFrame)
  }
  return out
}

/** inHole dit si un instant tombe dans un passage muet de la rafale. */
function inHole(b: ReplayFireBurstReady, t: number): boolean {
  return b.holes.some((h) => t >= h.t0 && t < h.t1)
}

/**
 * fireBurstShots — les coups simulés de toutes les rafales, un par frame et par rafale au plus,
 * posés sur le VÉHICULE porteur (sa position à cette image) ou sur la VIE du tireur à pied. Un
 * coup sans position (véhicule sans échantillon ni naissance, vie hors de sa fenêtre) ne se pose
 * pas : il ne se dessinerait nulle part.
 */
export function fireBurstShots(doc: ReplayDocumentReady): FireBurstShot[] {
  if (doc.bursts.length === 0) return []
  // LA POSITION EST CELLE DU PORTEUR (`carrierPosition.ts`, le chemin unique) : embarqué, le
  // véhicule qu il monte — à l interpolation même du sprite ; à pied, son bipède.
  const posOf = buildCarrierPosAt(doc)
  const bySlot = buildLivesBySlot(doc.tracks)
  const out: FireBurstShot[] = []
  for (const b of doc.bursts) {
    const xuid = xuidDuTireur(doc, bySlot, b)
    if (!xuid) continue
    let derniere = Number.NEGATIVE_INFINITY
    for (const t of burstShotFrames(b, frameToMs(1, doc))) {
      const frame = Math.floor(t + EPSILON_FRAME)
      if (frame === derniere) continue
      derniere = frame
      const pos = posOf(xuid, frame)
      if (!pos) continue
      out.push({ t: frame, slot: b.slot, x: pos.x, y: pos.y, w: b.w, v: b.v })
    }
  }
  return out.sort((a, b) => a.t - b.t)
}

/**
 * xuidDuTireur — le joueur de la rafale : la vie de son slot à la pose de la gâchette, sinon
 * l épisode d occupation de ce slot sur le véhicule porteur (un tireur embarqué ne réplique plus
 * sa position, sa vie peut s être close avant). Sans nom, la rafale ne se pose pas.
 */
function xuidDuTireur(
  doc: ReplayDocumentReady,
  bySlot: ReturnType<typeof buildLivesBySlot>,
  b: ReplayFireBurstReady,
): string | undefined {
  const life = lifeOfSlotAt(bySlot, b.slot, b.t0)
  if (life?.xuid) return life.xuid
  for (const v of doc.vehicles) {
    if (v.slot !== b.v) continue
    const ride = v.rides.find((r) => r.slot === b.slot && r.xuid)
    if (ride?.xuid) return ride.xuid
  }
  return undefined
}
