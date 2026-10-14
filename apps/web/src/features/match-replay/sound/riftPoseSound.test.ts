/**
 * Tests — riftPoseSound (la pose de la faille du translocateur) et la faille dessinée dès sa pose.
 *
 * LES DOCUMENTS SONT DES RELEVÉS RÉELS FIGÉS (même méthode que `knockbackSound.test.ts`) : les poses
 * de faille, les téléportations, les changements d'équipement et les fenêtres de piste ci-dessous sont
 * ceux que la cuisson au schéma 95 publie sur `1b2d9e08` (Dynasty), relevés le 2026-10-14. JGtm
 * (slot 535) pose à l'image 1550 — le contrôle de trajectoire la place à 164,06 s, soit cette image —
 * et se téléporte à 1761 ; SpiffyDart (560) pose à 3216 puis échange deux fois ; les poses de 531 et
 * 574 ne sont suivies d'aucun échange.
 */
import { describe, expect, it } from 'vitest'

import { riftStations } from '../model/riftStations'
import { testReplayDoc } from '../test/testDoc'
import { RIFT_POSE_HIT_OFFSET_MS, RIFT_POSE_SOUND_STEM, riftPoseSoundEvents } from './riftPoseSound'

const POSES_1B2D9E08 = [
  { t: 1256, a: 1240, slot: 531 },
  { t: 1550, a: 1534, slot: 535, x: 17.34, y: 135.5, z: 74.43 },
  { t: 3216, a: 3200, slot: 560, x: 17.35, y: 123.11, z: 74.42 },
  { t: 3943, a: 3927, slot: 574 },
]
const TELEPORTATIONS_1B2D9E08 = [
  { t: 1761, slot: 535, fx: 2.79, fy: 152.19, fz: 81.39, tx: 17.34, ty: 135.5, tz: 74.43 },
  { t: 3261, slot: 560, fx: 18.44, fy: 120.25, fz: 74.34, tx: 17.35, ty: 123.11, tz: 74.42 },
  { t: 3419, slot: 560, fx: 11.31, fy: 116.66, fz: 76.01, tx: 18.42, ty: 120.25, tz: 74.36 },
]
const CHANGEMENTS_1B2D9E08 = [
  { t: 1520, slot: 535, kind: 'taken' as const, r: 11, from: 4 },
  { t: 1851, slot: 535, kind: 'spent' as const, r: -1, from: 11 },
  { t: 3191, slot: 560, kind: 'taken' as const, r: 11, from: 4 },
]
const PISTES_1B2D9E08 = [
  { slot: 535, team: 0, xuid: '', points: [{ t: 1164, x: 0, y: 0 }, { t: 2246, x: 0, y: 0 }] },
  { slot: 560, team: 1, xuid: '', points: [{ t: 2845, x: 0, y: 0 }, { t: 3432, x: 0, y: 0 }] },
]

function doc1b2d9e08() {
  return testReplayDoc({
    frameCount: 6000,
    frameIntervalMs: 100,
    riftPoses: POSES_1B2D9E08,
    translocations: TELEPORTATIONS_1B2D9E08,
    equipmentChanges: CHANGEMENTS_1B2D9E08,
    tracks: PISTES_1B2D9E08,
    abilityLabels: { '11': { en: 'Quantum Translocator', fr: 'translocateur quantique', family: 'translocator_beacon' } },
  })
}

describe('riftPoseSoundEvents — le COUP de pose tombe à la pose', () => {
  it('1b2d9e08 : un son par pose, parti 1,045 s avant elle (après son activation)', () => {
    const ev = riftPoseSoundEvents(doc1b2d9e08())
    expect(ev.map((e) => e.ms)).toEqual([
      125_600 - RIFT_POSE_HIT_OFFSET_MS,
      155_000 - RIFT_POSE_HIT_OFFSET_MS,
      321_600 - RIFT_POSE_HIT_OFFSET_MS,
      394_300 - RIFT_POSE_HIT_OFFSET_MS,
    ])
    expect(ev.every((e) => e.stem === RIFT_POSE_SOUND_STEM)).toBe(true)
    for (const [i, p] of POSES_1B2D9E08.entries()) expect(ev[i].ms).toBeGreaterThanOrEqual(p.a * 100)
  })
})

describe('riftStations — la faille apparaît à sa POSE', () => {
  it('1b2d9e08 : de la pose au premier échange, à l arrivée de cet échange ; puis le va-et-vient', () => {
    expect(riftStations(doc1b2d9e08())).toEqual([
      { slot: 535, t0: 1550, t1: 1760, x: 17.34, y: 135.5 },
      { slot: 535, t0: 1761, t1: 1851, x: 2.79, y: 152.19 },
      { slot: 560, t0: 3216, t1: 3260, x: 17.35, y: 123.11 },
      { slot: 560, t0: 3261, t1: 3418, x: 18.44, y: 120.25 },
      { slot: 560, t0: 3419, t1: 3432, x: 11.31, y: 116.66 },
    ])
  })
})
