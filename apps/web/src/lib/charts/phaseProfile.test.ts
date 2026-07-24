/**
 * Tests unitaires du helper pur `phaseProfile` :
 *  - `phaseShares` : normalisation phases[i]/Σ, manche vide (Σ=0) → null, phases
 *    absentes → null, projection sur PHASE_COUNT (pad 0).
 *  - `phaseProfile` : médiane des parts, enveloppe P25–P75 (cas « dents de scie »
 *    = enveloppe large), exclusion des manches vides, décompte nMatches pour le
 *    seuil « médiane seule » (< MIN_MATCHES_FOR_ENVELOPE).
 */
import { describe, it, expect } from 'vitest'

import {
  MIN_MATCHES_FOR_ENVELOPE,
  PHASE_COUNT,
  phaseProfile,
  phaseShares,
} from './phaseProfile'

describe('phaseShares', () => {
  it('normalise phases[i]/Σ sur un vecteur de longueur PHASE_COUNT', () => {
    const s = phaseShares([2, 2, 4, 0, 0, 0, 0, 0, 0, 0])
    expect(s).not.toBeNull()
    expect(s).toHaveLength(PHASE_COUNT)
    // Σ = 8 → parts 0.25, 0.25, 0.5, puis 0.
    expect(s?.[0]).toBeCloseTo(0.25)
    expect(s?.[2]).toBeCloseTo(0.5)
    expect(s?.[9]).toBe(0)
    // Somme des parts = 1.
    expect(s?.reduce((a, b) => a + b, 0)).toBeCloseTo(1)
  })

  it('manche sans frag (Σ = 0) → null (exclue)', () => {
    expect(phaseShares([0, 0, 0, 0, 0, 0, 0, 0, 0, 0])).toBeNull()
  })

  it('phases absentes (null/undefined/vide) → null', () => {
    expect(phaseShares(null)).toBeNull()
    expect(phaseShares(undefined)).toBeNull()
    expect(phaseShares([])).toBeNull()
  })

  it('phases plus courtes que PHASE_COUNT → complétées à 0', () => {
    const s = phaseShares([1, 1])
    expect(s).toHaveLength(PHASE_COUNT)
    expect(s?.[0]).toBeCloseTo(0.5)
    expect(s?.[1]).toBeCloseTo(0.5)
    expect(s?.[5]).toBe(0)
  })
})

describe('phaseProfile', () => {
  it('médiane des parts par phase sur les manches exploitables', () => {
    // 3 manches concentrées sur la phase 0 → médiane phase 0 = 1, reste 0.
    const res = phaseProfile([
      { phases: [1, 0, 0, 0, 0, 0, 0, 0, 0, 0] },
      { phases: [2, 0, 0, 0, 0, 0, 0, 0, 0, 0] },
      { phases: [5, 0, 0, 0, 0, 0, 0, 0, 0, 0] },
    ])
    expect(res.nMatches).toBe(3)
    expect(res.median[0]).toBeCloseTo(1)
    expect(res.median[1]).toBeCloseTo(0)
  })

  it('cas « dents de scie » : enveloppe P25–P75 large sur la phase irrégulière', () => {
    // Phase 0 alterne 0 % et 100 % de l'activité selon la manche → forte
    // dispersion : P25 bas, P75 haut, enveloppe large.
    const res = phaseProfile([
      { phases: [1, 0, 0, 0, 0, 0, 0, 0, 0, 0] }, // part phase0 = 1
      { phases: [0, 1, 0, 0, 0, 0, 0, 0, 0, 0] }, // part phase0 = 0
      { phases: [1, 0, 0, 0, 0, 0, 0, 0, 0, 0] }, // 1
      { phases: [0, 1, 0, 0, 0, 0, 0, 0, 0, 0] }, // 0
      { phases: [1, 0, 0, 0, 0, 0, 0, 0, 0, 0] }, // 1
      { phases: [0, 1, 0, 0, 0, 0, 0, 0, 0, 0] }, // 0
    ])
    expect(res.nMatches).toBe(6)
    const spread = res.p75[0] - res.p25[0]
    expect(spread).toBeGreaterThan(0.5)
    expect(res.p25[0]).toBeLessThan(res.p75[0])
  })

  it('manche vide (Σ = 0) exclue du décompte et des quantiles', () => {
    const res = phaseProfile([
      { phases: [1, 0, 0, 0, 0, 0, 0, 0, 0, 0] },
      { phases: [0, 0, 0, 0, 0, 0, 0, 0, 0, 0] }, // vide → ignorée
      { phases: [1, 0, 0, 0, 0, 0, 0, 0, 0, 0] },
    ])
    expect(res.nMatches).toBe(2)
    expect(res.median[0]).toBeCloseTo(1)
  })

  it('< MIN_MATCHES_FOR_ENVELOPE manches → nMatches signale la médiane seule', () => {
    const res = phaseProfile([
      { phases: [1, 1, 0, 0, 0, 0, 0, 0, 0, 0] },
      { phases: [1, 3, 0, 0, 0, 0, 0, 0, 0, 0] },
    ])
    expect(res.nMatches).toBe(2)
    expect(res.nMatches).toBeLessThan(MIN_MATCHES_FOR_ENVELOPE)
    // La médiane reste calculée (le builder omettra l'enveloppe).
    expect(res.median[0]).toBeGreaterThan(0)
  })

  it('aucune manche exploitable → vecteurs à 0, nMatches 0', () => {
    const res = phaseProfile([{ phases: null }, { phases: [0, 0, 0, 0, 0, 0, 0, 0, 0, 0] }])
    expect(res.nMatches).toBe(0)
    expect(res.median.every((v) => v === 0)).toBe(true)
    expect(res.p25.every((v) => v === 0)).toBe(true)
    expect(res.p75.every((v) => v === 0)).toBe(true)
  })
})
