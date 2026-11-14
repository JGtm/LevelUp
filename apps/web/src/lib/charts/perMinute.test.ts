import { describe, expect, it } from 'vitest'
import { perMinute, pooledPerMinute } from './perMinute'

interface Row {
  v: number | null
  s: number | null
}

const count = (r: Row) => r.v
const seconds = (r: Row) => r.s

describe('perMinute', () => {
  it('divise par les minutes jouées', () => {
    expect(perMinute(30, 600)).toBe(3)
  })
  it('durée inconnue, nulle ou négative : pas de cadence', () => {
    expect(perMinute(30, null)).toBeNull()
    expect(perMinute(30, undefined)).toBeNull()
    expect(perMinute(30, 0)).toBeNull()
    expect(perMinute(30, -5)).toBeNull()
  })
})

describe('pooledPerMinute', () => {
  it('somme des comptes / somme des minutes des seuls matchs complets', () => {
    const rows: Row[] = [
      { v: 3000, s: 600 },
      { v: 900, s: 180 },
      { v: null, s: 600 }, // compte inconnu : hors du numérateur ET du dénominateur
      { v: 5000, s: null }, // durée inconnue : idem
    ]
    expect(pooledPerMinute(rows, count, seconds)).toBeCloseTo(3900 / 13, 10)
  })
  it('pondère par les minutes, pas par les matchs', () => {
    const rows: Row[] = [
      { v: 10, s: 60 },
      { v: 100, s: 600 },
      { v: 0, s: 600 },
    ]
    expect(pooledPerMinute(rows, count, seconds)).toBeCloseTo(110 / 21, 10)
  })
  it('aucun match retenu : absent, pas zéro', () => {
    expect(pooledPerMinute([{ v: null, s: 600 }], count, seconds)).toBeNull()
    expect(pooledPerMinute<Row>([], count, seconds)).toBeNull()
  })
})
