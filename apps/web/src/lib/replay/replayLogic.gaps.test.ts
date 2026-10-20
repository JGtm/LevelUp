/**
 * replayLogic.gaps.test.ts — UNE LACUNE DE RÉPLICATION NE SE TRAVERSE PAS.
 *
 * RETOURS DU REJEU 2026-09-23 (lot L1.4, décision Q14). Le document publie `Point.g` : la durée,
 * en millisecondes, pendant laquelle le film n'a RIEN répliqué AVANT ce point — « la piste ne doit
 * donc pas être interpolée au travers (ni segment, ni position intermédiaire) »
 * (`film/replay/document_aim.go`, champ G). Le client ne le lisait nulle part : un pion glissait
 * 70 s vers un point aberrant et traversait la carte (974 lacunes au parc, 222 déplacent le pion de
 * plus de 10 m). Pendant la lacune, le pion est TENU à sa dernière position (et pâli au dessin).
 */
import { describe, expect, it } from 'vitest'

import { altitudeAt, inGapAt, positionAt, trailAt } from './replayLogic'

/** Une vie : 0 → 50 continu, puis une lacune de 5 s avant le point de 100, puis 110 continu. */
const PISTE = [
  { t: 0, x: 0, y: 0, z: 0 },
  { t: 50, x: 10, y: 0, z: 1 },
  { t: 100, x: 100, y: 0, z: 9, g: 5000 },
  { t: 110, x: 101, y: 0, z: 9 },
]

describe('positionAt / altitudeAt — tenus au travers d’une lacune', () => {
  it('pendant la lacune, la position est la DERNIÈRE avant elle, jamais une interpolation', () => {
    expect(positionAt(PISTE, 75)).toEqual({ x: 10, y: 0 })
    expect(positionAt(PISTE, 99.5)).toEqual({ x: 10, y: 0 })
  })

  it('au point qui clôt la lacune, le pion est à ce point', () => {
    expect(positionAt(PISTE, 100)).toEqual({ x: 100, y: 0 })
  })

  it('hors lacune, l’interpolation est inchangée', () => {
    expect(positionAt(PISTE, 25)).toEqual({ x: 5, y: 0 })
    expect(positionAt(PISTE, 105)).toEqual({ x: 100.5, y: 0 })
  })

  it('un g nul ou absent n’est pas une lacune', () => {
    const p = [{ t: 0, x: 0, y: 0 }, { t: 10, x: 10, y: 0, g: 0 }]
    expect(positionAt(p, 5)).toEqual({ x: 5, y: 0 })
  })

  it('l’altitude est tenue de même', () => {
    expect(altitudeAt(PISTE, 75)).toBe(1)
    expect(altitudeAt(PISTE, 100)).toBe(9)
    expect(altitudeAt(PISTE, 25)).toBe(0.5)
  })
})

describe('trailAt — aucun segment au travers d’une lacune', () => {
  it('après la lacune, la traînée repart du point qui la clôt', () => {
    expect(trailAt(PISTE, 110, 200)).toEqual([
      { x: 100, y: 0 },
      { x: 101, y: 0 },
    ])
  })

  it('pendant la lacune, la traînée s’arrête à la position tenue', () => {
    expect(trailAt(PISTE, 75, 200)).toEqual([
      { x: 0, y: 0 },
      { x: 10, y: 0 },
    ])
  })
})

describe('inGapAt — l’image tombe-t-elle dans une lacune ?', () => {
  it('vrai strictement entre le point qui précède la lacune et celui qui la clôt', () => {
    expect(inGapAt(PISTE, 75)).toBe(true)
    expect(inGapAt(PISTE, 50.5)).toBe(true)
  })

  it('faux sur un point, hors lacune, avant la vie et après son dernier point', () => {
    expect(inGapAt(PISTE, 50)).toBe(false)
    expect(inGapAt(PISTE, 100)).toBe(false)
    expect(inGapAt(PISTE, 25)).toBe(false)
    expect(inGapAt(PISTE, 105)).toBe(false)
    expect(inGapAt(PISTE, -1)).toBe(false)
    expect(inGapAt(PISTE, 200)).toBe(false)
    expect(inGapAt([], 10)).toBe(false)
  })
})
