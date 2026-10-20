/**
 * translocationCuts.test.ts — UNE TRANSLOCATION COUPE LA PISTE : ni glissade, ni trait de traînée.
 *
 * Les pistes ci-dessous reprennent les valeurs MESURÉES de `1b2d9e08` (slot 535, image 1761 :
 * dernier échantillon au départ à 1761, premier à l'arrivée à 1762) et d'un film du parc où
 * l'événement tombe sur l'échantillon d'arrivée (`0ebffa7f`, image 4130). Les deux dispositions
 * existent au parc (30 et 92 événements sur 122) ; les deux doivent couper au bon endroit.
 */
import { describe, expect, it } from 'vitest'

import { altitudeAt, inGapAt, positionAt, trailAt } from './replayLogic'
import type { ReplayCutPoint } from './replayReadyTypes'
import { cutSampleIndex, markTranslocationCuts } from './translocationCuts'

/** Piste dont l'événement tombe sur la DERNIÈRE image au départ (cas `1b2d9e08`). */
const APRES_L_EVENEMENT: ReplayCutPoint[] = [
  { t: 1759, x: 2.79, y: 152.16, z: 80.94 },
  { t: 1760, x: 2.79, y: 152.17, z: 80.94 },
  { t: 1761, x: 2.79, y: 152.17, z: 80.94 },
  { t: 1762, x: 17.34, y: 135.5, z: 73.98 },
  { t: 1763, x: 17.34, y: 135.5, z: 73.98 },
]
const EV_1761 = { t: 1761, slot: 535, fx: 2.79, fy: 152.19, fz: 81.39, tx: 17.34, ty: 135.5, tz: 74.43 }

/** Piste dont l'événement tombe SUR l'échantillon d'arrivée (cas `0ebffa7f`). */
const SUR_L_EVENEMENT: ReplayCutPoint[] = [
  { t: 4128, x: 0.42, y: 25.51 },
  { t: 4129, x: 0.49, y: 25.42 },
  { t: 4130, x: 13.45, y: 17.71 },
  { t: 4131, x: 13.65, y: 17.62 },
]
const EV_4130 = { t: 4130, slot: 597, fx: 0.64, fy: 25.37, fz: 2.16, tx: 13.44, ty: 17.74, tz: 0.42 }

describe('cutSampleIndex — l’échantillon d’arrivée, vérifié par les deux positions', () => {
  it('événement sur la dernière image au départ : l’arrivée est l’échantillon suivant', () => {
    expect(cutSampleIndex(APRES_L_EVENEMENT, EV_1761)).toBe(3)
  })

  it('événement sur l’échantillon d’arrivée : c’est lui', () => {
    expect(cutSampleIndex(SUR_L_EVENEMENT, EV_4130)).toBe(2)
  })

  it('sans positions publiées : aucune coupure (rien n’est deviné)', () => {
    expect(cutSampleIndex(APRES_L_EVENEMENT, { t: 1761, slot: 535 })).toBe(-1)
  })

  it('positions qui ne correspondent à aucun couple de la piste : aucune coupure', () => {
    expect(cutSampleIndex(APRES_L_EVENEMENT, { ...EV_1761, tx: 90, ty: 90 })).toBe(-1)
  })

  it('événement après la fin de la piste : aucune coupure', () => {
    expect(cutSampleIndex(APRES_L_EVENEMENT, { ...EV_1761, t: 9999 })).toBe(-1)
  })
})

describe('markTranslocationCuts', () => {
  const track = { slot: 535, points: APRES_L_EVENEMENT }

  it('pose `cut` sur le seul échantillon d’arrivée, sans muter les points du contrat', () => {
    const [out] = markTranslocationCuts([track], [EV_1761])
    expect(out.points.map((p) => p.cut === true)).toEqual([false, false, false, true, false])
    expect(APRES_L_EVENEMENT.some((p) => p.cut)).toBe(false)
  })

  it('une piste sans coupure est rendue telle quelle (même référence)', () => {
    const autre = { slot: 1, points: SUR_L_EVENEMENT }
    const [a, b] = markTranslocationCuts([track, autre], [EV_1761])
    expect(b).toBe(autre)
    expect(a).not.toBe(track)
  })

  it('un slot peut porter plusieurs vies : la coupure va à celle qui la contient', () => {
    const vieAvant: { slot: number; points: ReplayCutPoint[] } = {
      slot: 535,
      points: [{ t: 10, x: 0, y: 0 }, { t: 20, x: 1, y: 1 }],
    }
    const [a, b] = markTranslocationCuts([vieAvant, track], [EV_1761])
    expect(a).toBe(vieAvant)
    expect(b.points[3].cut).toBe(true)
  })

  it('sans translocation, rien ne change', () => {
    expect(markTranslocationCuts([track], [])).toEqual([track])
  })
})

describe('la lecture : rien entre le départ et l’arrivée', () => {
  const [coupee] = markTranslocationCuts([{ slot: 535, points: APRES_L_EVENEMENT }], [EV_1761])
  const points = coupee.points

  it('à mi-image du saut, le pion est AU DÉPART — jamais entre les deux', () => {
    for (const t of [1761, 1761.25, 1761.5, 1761.75, 1761.99]) {
      expect(positionAt(points, t)).toEqual({ x: 2.79, y: 152.17 })
    }
  })

  it('à l’image d’arrivée, le pion est à l’arrivée', () => {
    expect(positionAt(points, 1762)).toEqual({ x: 17.34, y: 135.5 })
  })

  it('l’altitude est tenue de même', () => {
    expect(altitudeAt(points, 1761.5)).toBe(80.94)
    expect(altitudeAt(points, 1762)).toBe(73.98)
  })

  it('aucune traînée n’enjambe la coupure : elle repart de l’arrivée', () => {
    const trail = trailAt(points, 1763, 100)
    expect(trail).toEqual([{ x: 17.34, y: 135.5 }, { x: 17.34, y: 135.5 }])
    for (const p of trail) expect(Math.abs(p.x - 2.79)).toBeGreaterThan(10)
  })

  it('avant la coupure, la traînée s’arrête au départ', () => {
    const trail = trailAt(points, 1761.5, 100)
    expect(trail.every((p) => p.x === 2.79)).toBe(true)
  })

  it('le pion n’est PAS pâli : une coupure n’est pas une lacune de réplication', () => {
    expect(inGapAt(points, 1761.5)).toBe(false)
  })

  it('sans coupure posée, l’ancien défaut se reproduit (témoin)', () => {
    const p = positionAt(APRES_L_EVENEMENT, 1761.5)
    expect(p?.x).toBeGreaterThan(5)
  })
})
