/**
 * calloutShape — la règle unique « la zone a une forme » et l'énumération de ses contours.
 */
import { describe, expect, it } from 'vitest'

import { isRing, zoneHasShape, zoneRings } from './calloutShape'

const tri = (dx = 0) => [
  { x: dx, y: 0 },
  { x: dx + 1, y: 0 },
  { x: dx, y: 1 },
]
const segment = [
  { x: 0, y: 0 },
  { x: 1, y: 1 },
]

describe('isRing', () => {
  it('trois sommets au moins', () => {
    expect(isRing([])).toBe(false)
    expect(isRing(segment)).toBe(false)
    expect(isRing(tri())).toBe(true)
  })
})

describe('zoneHasShape', () => {
  it('un contour principal fait une forme', () => {
    expect(zoneHasShape({ polygon: tri(), parts: [] })).toBe(true)
  })

  it('une partie seule fait une forme (la règle du jeu)', () => {
    expect(zoneHasShape({ polygon: [], parts: [tri()] })).toBe(true)
  })

  it('ni contour ni partie exploitable : pas de forme', () => {
    expect(zoneHasShape({ polygon: [], parts: [] })).toBe(false)
    expect(zoneHasShape({ polygon: segment, parts: [segment] })).toBe(false)
  })
})

describe('zoneRings', () => {
  it('énumère contour, parties puis trous, sans les anneaux de moins de trois sommets', () => {
    const zone = { polygon: tri(0), parts: [tri(10), segment], holes: [tri(20)] }
    expect(zoneRings(zone)).toEqual([tri(0), tri(10), tri(20)])
  })

  it('une zone sans forme n’a aucun anneau, et des trous seuls en gardent un', () => {
    expect(zoneRings({ polygon: [], parts: [], holes: [] })).toEqual([])
    expect(zoneRings({ polygon: [], parts: [], holes: [tri()] })).toEqual([tri()])
  })
})
