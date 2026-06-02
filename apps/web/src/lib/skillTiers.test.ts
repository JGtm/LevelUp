/**
 * Tests — skillTiers : invariants des grilles LUSR/CSR + sélection gridForRatingTypes.
 * Garde-fou contre une dérive de config (bornes non contiguës, sous-paliers manquants)
 * et contre une régression de la logique de choix de grille (CSR vs LUSR vs mixte).
 */
import { describe, it, expect } from 'vitest'
import { LUSR_TIER_GRID, CSR_TIER_GRID, gridForRatingTypes } from './skillTiers'

describe('grilles de paliers', () => {
  for (const [name, grid] of [['LUSR', LUSR_TIER_GRID], ['CSR', CSR_TIER_GRID]] as const) {
    it(`${name} : tiers triés, contigus, sous-paliers ≥ 1`, () => {
      const t = grid.tiers
      for (let i = 0; i < t.length; i++) {
        expect(t[i].max).toBeGreaterThan(t[i].min)
        expect(t[i].subTiers).toBeGreaterThanOrEqual(1)
        if (i + 1 < t.length) expect(t[i].max).toBe(t[i + 1].min) // contiguïté
      }
    })

    it(`${name} : palier sommet ouvert (Onyx, sans sous-palier)`, () => {
      const top = grid.tiers[grid.tiers.length - 1]
      expect(top.max).toBeGreaterThanOrEqual(9000)
      expect(top.subTiers).toBe(1)
      expect(top.en).toBe('Onyx')
    })
  }
})

describe('gridForRatingTypes', () => {
  it('tous CSR → grille CSR', () => {
    expect(gridForRatingTypes(['CSR', 'CSR'])).toBe(CSR_TIER_GRID)
  })
  it('insensible à la casse (csr)', () => {
    expect(gridForRatingTypes(['csr'])).toBe(CSR_TIER_GRID)
  })
  it('tous LUSR → grille LUSR', () => {
    expect(gridForRatingTypes(['LUSR'])).toBe(LUSR_TIER_GRID)
  })
  it('mixte LUSR+CSR → grille LUSR (non-régression, échelle legacy)', () => {
    expect(gridForRatingTypes(['LUSR', 'CSR'])).toBe(LUSR_TIER_GRID)
  })
  it('vide ou null/undefined → grille LUSR (défaut)', () => {
    expect(gridForRatingTypes([])).toBe(LUSR_TIER_GRID)
    expect(gridForRatingTypes([null, undefined])).toBe(LUSR_TIER_GRID)
  })
})
