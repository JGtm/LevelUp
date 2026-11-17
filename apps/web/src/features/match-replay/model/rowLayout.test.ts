/**
 * Tests — rowLayout : la rangée de fiches tient (D-12). Les largeurs demandées sont écrites dans
 * les noms des tests : 2 × n × 115 + 270 + 2 × n × 6 + 2 × 24 (4 contre 4 = 1286, 5 contre 5 = 1528).
 */
import { describe, expect, it } from 'vitest'

import { largeurRequiseRangee, rowFits } from './rowLayout'

const DEUX_CAMPS = 2
const LARGEUR_1440 = 1440
const LARGEUR_1280 = 1280
const LARGEUR_1600 = 1600
const LARGEUR_4000 = 4000

describe('largeurRequiseRangee', () => {
  it('4 places par camp demandent 1286 px', () => {
    expect(largeurRequiseRangee(4)).toBe(1286)
  })

  it('5 places par camp demandent 1528 px', () => {
    expect(largeurRequiseRangee(5)).toBe(1528)
  })
})

describe('rowFits', () => {
  it('4 contre 4 tient à 1440 px (demande 1286 px)', () => {
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [4, 4], largeurCorps: LARGEUR_1440 })).toBe(true)
  })

  it('4 contre 4 ne tient PAS à 1280 px (demande 1286 px)', () => {
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [4, 4], largeurCorps: LARGEUR_1280 })).toBe(false)
  })

  it('5 contre 5 ne tient pas à 1440 px (demande 1528 px)', () => {
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [5, 5], largeurCorps: LARGEUR_1440 })).toBe(false)
  })

  it('5 contre 5 tient à 1600 px (demande 1528 px)', () => {
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [5, 5], largeurCorps: LARGEUR_1600 })).toBe(true)
  })

  it('3 contre 3 tient à 1280 px', () => {
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [3, 3], largeurCorps: LARGEUR_1280 })).toBe(true)
  })

  it('Grande équipe (12 par camp) ne tient jamais, même à 4000 px', () => {
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [12, 12], largeurCorps: LARGEUR_4000 })).toBe(false)
  })

  it('chacun pour soi (plus de deux camps) ne tient jamais, même à 4000 px', () => {
    expect(rowFits({ camps: 8, placesParCamp: [1, 1, 1, 1, 1, 1, 1, 1], largeurCorps: LARGEUR_4000 })).toBe(false)
  })

  it('chacun pour soi (un seul camp) ne tient jamais, même à 4000 px', () => {
    expect(rowFits({ camps: 1, placesParCamp: [8], largeurCorps: LARGEUR_4000 })).toBe(false)
  })

  it('cas limite : une largeur exactement égale à la demande (4 contre 4, 1286 px) tient', () => {
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [4, 4], largeurCorps: 1286 })).toBe(true)
  })

  it('cas limite : un pixel de moins que la demande (4 contre 4, 1285 px) ne tient pas', () => {
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [4, 4], largeurCorps: 1285 })).toBe(false)
  })

  it('camps de tailles inégales : la rangée compte le plus grand (4 et 2 demandent 1286 px)', () => {
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [4, 2], largeurCorps: 1286 })).toBe(true)
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [4, 2], largeurCorps: 1285 })).toBe(false)
  })

  it('une liste de places qui ne compte pas deux camps ne tient pas', () => {
    expect(rowFits({ camps: DEUX_CAMPS, placesParCamp: [3], largeurCorps: LARGEUR_4000 })).toBe(false)
  })
})
