import { describe, expect, it } from 'vitest'

import type { ReplayDocumentReady, ReplayFireBurstReady } from '../../../lib/replay/replayNormalize'
import { soundEnvelopeOf, SOUND_HOLD_RELEASE_S } from './replayAudio'
import { CADENCE_FAISCEAU, fireBurstSoundEvents } from './fireBurstSound'

const GHOST = '0x0001543500000000'
const WASP_LMG = '0xD3C407ED00000000'
const RAYON = '0xA0955E9E'

function rafale(over: Partial<ReplayFireBurstReady>): ReplayFireBurstReady {
  return { t0: 10, t1: 20, slot: 5, w: GHOST, rate: 7.5, holes: [], b0: 'pressed', b1: 'released',
    ...over } as ReplayFireBurstReady
}

const doc = (bursts: ReplayFireBurstReady[]) =>
  ({
    frameIntervalMs: 100,
    bursts,
    vehicleWeapons: {
      [GHOST]: { vehicle: 'ghost', en: 'a', fr: 'b', fire: 'continuous', fx: 'plasma', tint: 'plasma_hot',
        sound: 'vehicle_shot_ghost_1' },
      [WASP_LMG]: { vehicle: 'wasp', en: 'a', fr: 'b', fire: 'continuous', fx: 'ballistic', tint: 'kinetic',
        sound: 'vehicle_shot_wasp_lmg_1', loop: 'vehicle_shot_wasp_lmg_loop' },
    },
  }) as unknown as ReplayDocumentReady

const stems: Record<string, string> = {
  [GHOST]: 'vehicle_shot_ghost_1',
  [WASP_LMG]: 'vehicle_shot_wasp_lmg_1',
  [RAYON]: 'hinf_sentinel_beam',
}
const stemOf = (w: string | undefined) => (w ? stems[w] : undefined)

describe('fireBurstSoundEvents — le son prolongé d une rafale', () => {
  it('sans boucle livrée : le coup se rejoue à la cadence, chaque coup coupé au suivant, le dernier entier', () => {
    const ev = fireBurstSoundEvents(doc([rafale({})]), stemOf)
    expect(ev).toHaveLength(8)
    expect(ev[0]).toMatchObject({ ms: 1000, stem: 'vehicle_shot_ghost_1' })
    expect(ev[0].cutMs).toBeCloseTo(1000 / 7.5, 6)
    expect(ev[ev.length - 1].cutMs).toBeUndefined()
    expect(ev.every((e) => e.holdMs === undefined)).toBe(true)
  })

  it('une boucle livrée (LMG du Wasp) est TENUE sur chaque passage lu, puis le coup de queue au lâcher', () => {
    const ev = fireBurstSoundEvents(doc([rafale({ w: WASP_LMG, rate: 10, holes: [{ t0: 14, t1: 16 }] })]), stemOf)
    expect(ev.map((e) => [e.ms, e.stem, e.holdMs])).toEqual([
      [1000, 'vehicle_shot_wasp_lmg_loop', 400],
      [1600, 'vehicle_shot_wasp_lmg_loop', 400],
      [2000, 'vehicle_shot_wasp_lmg_1', undefined],
    ])
  })

  it('un lâcher perdu dans un trou n a pas de coup de queue, et le trou final ne tient rien', () => {
    const ev = fireBurstSoundEvents(
      doc([rafale({ w: WASP_LMG, rate: 10, holes: [{ t0: 15, t1: 20 }], b1: 'hole' })]),
      stemOf,
    )
    expect(ev.map((e) => [e.ms, e.stem, e.holdMs])).toEqual([[1000, 'vehicle_shot_wasp_lmg_loop', 500]])
  })

  it('un FAISCEAU (Rayon de Sentinelle, 60/s) tient le son de l arme, sans coup de queue', () => {
    expect(60).toBeGreaterThanOrEqual(CADENCE_FAISCEAU)
    const ev = fireBurstSoundEvents(doc([rafale({ w: RAYON, rate: 60 })]), stemOf)
    expect(ev.map((e) => [e.stem, e.holdMs])).toEqual([['hinf_sentinel_beam', 1000]])
  })

  it('une arme sans son se tait', () => {
    expect(fireBurstSoundEvents(doc([rafale({ w: '0x001B33E800000000' })]), stemOf)).toEqual([])
  })
})

describe('soundEnvelopeOf — la forme d un son de rafale', () => {
  it('tenu : boucle jusqu au lâcher, puis relâchement bref', () => {
    expect(soundEnvelopeOf(8, { holdS: 2 })).toEqual({
      fadeStartS: 2,
      stopS: 2 + SOUND_HOLD_RELEASE_S,
      loop: true,
    })
  })

  it('coupé : arrêt au coup suivant, fondu borné à la moitié', () => {
    const e = soundEnvelopeOf(1.2, { cutS: 0.1 })
    expect(e.stopS).toBeCloseTo(0.1, 9)
    expect(e.fadeStartS).toBeCloseTo(0.05, 9)
    expect(e.loop).toBe(false)
  })

  it('sans forme : l enveloppe ordinaire', () => {
    expect(soundEnvelopeOf(0.5).loop).toBe(false)
    expect(soundEnvelopeOf(0.5).stopS).toBe(0.5)
  })
})
