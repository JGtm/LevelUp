/**
 * Tests — knockbackSound (la poussée du répulseur) et translocationSound (la téléportation).
 *
 * LES DOCUMENTS SONT DES RELEVÉS RÉELS FIGÉS : les artefacts ne sont pas versionnés, la méthode du
 * dépôt est de mesurer hors ligne puis de figer (cf. `placementDroppedWitness.test.ts`). Les
 * poussées et les téléportations ci-dessous sont celles que la cuisson au schéma 94 publie sur
 * trois films, relevées le 2026-10-14 :
 *  - `0d9a9af9` (Doubles, Forest) : WolfzyLIVE (slot 513) pousse JGtm (519) aux images 715 et 811,
 *    Clearshields (532) pousse Sterio23 (535) à l'image 2557 ; aucun kill au répulseur.
 *  - `215e7022` (Argyle) : Elmo910 (560) pousse deux joueurs aux images 2957 et 2958 — un seul
 *    souffle —, et tue le premier : le kill au répulseur est programmé à 295 702 ms.
 *  - `1b2d9e08` (Dynasty) : trois téléportations, JGtm (535) image 1761, SpiffyDart (560) images
 *    3261 et 3419.
 */
import { describe, expect, it } from 'vitest'

import { knockbackSoundEvents, KNOCKBACK_SOUND_STEM } from './knockbackSound'
import { buildSoundTimeline } from './replaySound'
import { soundEvent } from './replaySoundVariants'
import { TRANSLOCATION_SOUND_STEM, translocationSoundEvents } from './translocationSound'
import { testReplayDoc } from '../test/testDoc'

const POUSSEES_0D9A9AF9 = [
  { t: 715, slot: 519, by: 513, vx: 2.95, vy: 4.8, vz: 2.06 },
  { t: 811, slot: 519, by: 513, vx: -5.45, vy: 1.45, vz: 2.05 },
  { t: 2557, slot: 535, by: 532, vx: 1.02, vy: 5.55, vz: 2.04 },
]
const POUSSEES_215E7022 = [
  { t: 2957, slot: 563, by: 560, vx: 2.34, vy: 5.12, vz: 2.06 },
  { t: 2958, slot: 564, by: 560, vx: 2.23, vy: 5.18, vz: 2.05 },
]
const TELEPORTATIONS_1B2D9E08 = [
  { t: 1761, slot: 535 },
  { t: 3261, slot: 560 },
  { t: 3419, slot: 560 },
]

describe('knockbackSoundEvents', () => {
  it('0d9a9af9 : un son par souffle publié, à sa frame — trois poussées, trois sons', () => {
    const doc = testReplayDoc({
      frameIntervalMs: 100,
      knockbacks: POUSSEES_0D9A9AF9,
    })
    const ev = knockbackSoundEvents(doc, [])
    expect(ev.map((e) => e.ms)).toEqual([71_500, 81_100, 255_700])
    expect(ev.every((e) => e.stem === KNOCKBACK_SOUND_STEM)).toBe(true)
  })

  it('215e7022 : deux joueurs poussés par le même souffle ne font qu’un son', () => {
    const doc = testReplayDoc({
      frameIntervalMs: 100,
      knockbacks: POUSSEES_215E7022,
    })
    expect(knockbackSoundEvents(doc, []).map((e) => e.ms)).toEqual([295_700])
  })

  it('215e7022 : le kill au répulseur du même instant sonne déjà — la poussée se tait', () => {
    const doc = testReplayDoc({
      frameIntervalMs: 100,
      knockbacks: POUSSEES_215E7022,
    })
    const kill = soundEvent(295_702, KNOCKBACK_SOUND_STEM)
    expect(knockbackSoundEvents(doc, [kill])).toEqual([])
  })

  it('la piste entière porte la poussée sous la catégorie Équipement', () => {
    const doc = testReplayDoc({
      frameIntervalMs: 100,
      knockbacks: POUSSEES_0D9A9AF9,
    })
    const on = {
      weapon: false,
      grenade: false,
      melee: false,
      equipment: true,
      objective: false,
    }
    expect(
      buildSoundTimeline(doc, [], on).filter((e) => e.stem === KNOCKBACK_SOUND_STEM),
    ).toHaveLength(3)
    expect(buildSoundTimeline(doc, [], { ...on, equipment: false })).toEqual([])
  })
})

describe('translocationSoundEvents', () => {
  it('1b2d9e08 : un son par téléportation publiée, à son départ', () => {
    const doc = testReplayDoc({
      frameIntervalMs: 100,
      translocations: TELEPORTATIONS_1B2D9E08,
    })
    const ev = translocationSoundEvents(doc)
    expect(ev.map((e) => e.ms)).toEqual([176_100, 326_100, 341_900])
    expect(ev.every((e) => e.stem === TRANSLOCATION_SOUND_STEM)).toBe(true)
  })
})
