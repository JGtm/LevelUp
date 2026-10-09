import { describe, expect, it } from 'vitest'

import type { ReplayDocumentReady, ReplayFireBurstReady } from '../../../lib/replay/replayNormalize'
import { burstShotFrames, fireBurstShots } from './fireBursts'

/** Une rafale de la gâchette principale, bornes lues au tick (pas du document : 100 ms). */
function rafale(over: Partial<ReplayFireBurstReady>): ReplayFireBurstReady {
  return { t0: 0, t1: 10, slot: 10, w: '0x0001543500000000', rate: 7.5, holes: [], b0: 'pressed',
    b1: 'released', ...over } as ReplayFireBurstReady
}

describe('burstShotFrames — les coups à la cadence du tag', () => {
  it('le premier coup à la pose de la gâchette, les suivants à la cadence (Ghost, 7,5/s)', () => {
    const t = burstShotFrames(rafale({}), 100)
    expect(t).toHaveLength(8)
    expect(t[0]).toBe(0)
    expect(t[1]).toBeCloseTo(4 / 3, 6)
  })

  it('une arme qui MONTE en cadence commence lentement (LAAG : 5 -> 18/s en 1,6 s)', () => {
    const t = burstShotFrames(rafale({ rate: 18, rate0: 5, ramp: 1.6 }), 100)
    expect(t[1] - t[0]).toBeCloseTo(2, 6) // 5 coups par seconde = 200 ms
    expect(t.length).toBeGreaterThan(7)
    expect(t.length).toBeLessThan(15)
  })

  it('un passage MUET ne porte aucun coup', () => {
    const t = burstShotFrames(rafale({ holes: [{ t0: 1, t1: 9 }] }), 100)
    expect(t.every((x) => x < 1 || x >= 9)).toBe(true)
    expect(t).toContain(0)
  })

  it('sans cadence ou sans pas de temps, aucun coup n est inventé', () => {
    expect(burstShotFrames(rafale({ rate: 0 }), 100)).toEqual([])
    expect(burstShotFrames(rafale({}), 0)).toEqual([])
  })
})

describe('fireBurstShots — les coups posés comme des tirs', () => {
  const vie = { slot: 11, xuid: '42', team: 0, points: [{ t: 0, x: 0, y: 0 }, { t: 100, x: 10, y: 0 }] }
  const doc = {
    frameIntervalMs: 100,
    tracks: [vie],
    vehicles: [],
    bursts: [rafale({ slot: 11, w: '0xA0955E9E', rate: 60, t0: 50, t1: 52 })],
  } as unknown as ReplayDocumentReady

  it('un faisceau à pied (60/s) : UN coup par image, posé sur le tireur, arme de la rafale', () => {
    const tirs = fireBurstShots(doc)
    expect(tirs.map((s) => s.t)).toEqual([50, 51, 52])
    expect(tirs[0]).toMatchObject({ slot: 11, x: 5, y: 0, w: '0xA0955E9E' })
    expect(tirs[0].v).toBeUndefined()
  })

  it('un tireur que rien ne nomme ne pose aucun coup', () => {
    const anonyme = { ...doc, tracks: [{ ...vie, xuid: '' }] } as unknown as ReplayDocumentReady
    expect(fireBurstShots(anonyme)).toEqual([])
  })

  it('sans rafale, rien', () => {
    expect(fireBurstShots({ ...doc, bursts: [] } as unknown as ReplayDocumentReady)).toEqual([])
  })
})
