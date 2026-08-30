import { describe, expect, it } from 'vitest'

import type { ReplayDocumentReady } from './replayNormalize'
import {
  EQUIPMENT_PAD_SPAWN_SOUND_STEM,
  PAD_SPAWN_MAX_PAR_MATCH,
  PAD_SPAWN_SOUND_STEM,
  padSpawnSoundEvents,
} from './padSpawnSound'

/**
 * padSpawnSound.test.ts — l'apparition d'une arme sur son socle.
 *
 * La règle est courte parce que la source l'est : le calque PUBLIE les instants, il n'y a rien à
 * déduire. Ce qui se teste, ce sont donc les bords — un socle sans apparition, un artefact sans
 * socle, l'ordre, et le plafond de sûreté.
 */

/** Un document minimal : le pas de grille et les socles suffisent. */
function doc(weaponPads: unknown[]): ReplayDocumentReady {
  return { frameIntervalMs: 100, weaponPads } as unknown as ReplayDocumentReady
}

describe('apparition sur socle — le calque date, on joue', () => {
  it('un son par apparition, à sa frame', () => {
    const d = doc([{ spawns: [0, 30] }])
    expect(padSpawnSoundEvents(d)).toEqual([
      { ms: 0, stem: PAD_SPAWN_SOUND_STEM },
      { ms: 3000, stem: PAD_SPAWN_SOUND_STEM },
    ])
  })

  it('les socles sont fusionnés et TRIÉS : la piste est une horloge, pas une liste de socles', () => {
    const d = doc([{ spawns: [50, 10] }, { spawns: [30] }])
    expect(padSpawnSoundEvents(d).map((e) => e.ms)).toEqual([1000, 3000, 5000])
  })

  it('aucun camp : le son sonne sans ligne « moi » au tableau de score', () => {
    // La signature ne prend PAS de camp allié, et c'est le test qui le fige : un socle
    // n'appartient à personne, contrairement aux sons d'état de zone.
    expect(padSpawnSoundEvents(doc([{ spawns: [10] }]))).toHaveLength(1)
  })

  it('MUET sans socle, et sans apparition', () => {
    expect(padSpawnSoundEvents(doc([]))).toEqual([])
    expect(padSpawnSoundEvents(doc([{ spawns: [] }]))).toEqual([])
    expect(padSpawnSoundEvents(doc([{ spawns: null }]))).toEqual([])
  })

  it('le plafond de sûreté coupe, et il coupe les DERNIÈRES', () => {
    const beaucoup = Array.from({ length: PAD_SPAWN_MAX_PAR_MATCH + 10 }, (_, i) => i)
    const evs = padSpawnSoundEvents(doc([{ spawns: beaucoup }]))
    expect(evs).toHaveLength(PAD_SPAWN_MAX_PAR_MATCH)
    expect(evs[evs.length - 1].ms).toBe((PAD_SPAWN_MAX_PAR_MATCH - 1) * 100)
  })
})

describe('apparition sur socle — arme et équipement, chacun son son', () => {
  it('un socle d équipement sonne le son d équipement, un socle d arme celui d arme', () => {
    const d = doc([
      { weapon: '0x1234abcd', spawns: [10] },
      { weapon: 'powerup_camo', spawns: [20] },
      { weapon: 'powerup_overshield', spawns: [30] },
    ])
    expect(padSpawnSoundEvents(d)).toEqual([
      { ms: 1000, stem: PAD_SPAWN_SOUND_STEM },
      { ms: 2000, stem: EQUIPMENT_PAD_SPAWN_SOUND_STEM },
      { ms: 3000, stem: EQUIPMENT_PAD_SPAWN_SOUND_STEM },
    ])
  })
})
