/**
 * Le calque des poses honore Seul et Masquer : une pose retirée ne se dessine pas, mais son capteur
 * continue de révéler (la marque est aussi au joueur révélé), et la marque a son propre filtre.
 */
import { describe, expect, it } from 'vitest'

import { draw, life, pose, SENSOR_ID, TIME } from '../test/placementFixtures'

const arcs = (ops: ReturnType<typeof draw>) => ops.filter((o) => o.op === 'arc')
const sensorPose = pose({ family: 'sensor', id: SENSOR_ID, t0: 50, x: 5, y: 5 })
const sides: Record<number, number | null> = { 3: 0, 7: 1 }
const campOfSlot = (slot: number) => sides[slot] ?? null

describe('drawEquipmentPlacementsLayer — Seul et Masquer', () => {
  it('une pose retirée ne se dessine pas, une autre se dessine', () => {
    const murs = [pose({ owner: 3, x: 2, y: 2 }), pose({ owner: 4, x: 8, y: 8 })]
    const sans = draw(murs).filter((o) => o.op === 'stroke').length
    const un = draw(murs, TIME, { shown: { placement: (p) => p.owner === 3, reveal: () => true } })
      .filter((o) => o.op === 'stroke').length
    expect(sans).toBeGreaterThan(0)
    expect(un).toBeGreaterThan(0)
    expect(un).toBeLessThan(sans)
    const aucune = draw(murs, TIME, { shown: { placement: () => false, reveal: () => true } })
    expect(aucune.filter((o) => o.op === 'stroke')).toHaveLength(0)
  })

  it('un capteur retiré continue de révéler : seule la marque sur la vie révélée se dessine', () => {
    const foe = life(7, 6, 5)
    const ops = draw([sensorPose], TIME, {
      lives: [foe], campOfSlot, shown: { placement: () => false, reveal: () => true },
    })
    // Zone et anneau du capteur absents ; halo et liseré de la marque présents.
    expect(arcs(ops)).toHaveLength(2)
  })

  it('une marque retirée ne se dessine pas, le capteur reste', () => {
    const foe = life(7, 6, 5)
    const ops = draw([sensorPose], TIME, {
      lives: [foe], campOfSlot, shown: { placement: () => true, reveal: () => false },
    })
    expect(arcs(ops)).toHaveLength(2) // zone + anneau, sans marque
  })

  it('le filtre de marque reçoit le poseur, son image de pose, la vie révélée et l’image courante', () => {
    const seen: unknown[] = []
    draw([sensorPose], TIME, {
      lives: [life(7, 6, 5)], campOfSlot,
      shown: { placement: () => true, reveal: (r, frame) => { seen.push([r.owner, r.ownerFrame, r.slot, frame]); return true } },
    })
    expect(seen).toEqual([[3, 50, 7, TIME.frame]])
  })
})
