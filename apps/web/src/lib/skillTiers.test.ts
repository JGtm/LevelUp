/**
 * Tests — skillTiers : invariants des grilles LUSR/CSR + sélection gridForRatingTypes.
 * Garde-fou contre une dérive de config (bornes non contiguës, sous-paliers manquants)
 * et contre une régression de la logique de choix de grille (CSR vs LUSR vs mixte).
 */
import { describe, it, expect } from 'vitest'
import {
  LUSR_TIER_GRID,
  CSR_TIER_GRID,
  gridForRatingTypes,
  subTierPosition,
  localizeTierName,
  localizeTierLabel,
} from './skillTiers'

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

describe('subTierPosition', () => {
  it('CSR : sous-paliers de 50 pts (Diamant)', () => {
    // Diamant CSR [1200,1500], 6 sous-paliers de 50. 1452 → sous-palier [1450,1500].
    const p = subTierPosition(CSR_TIER_GRID, 1452)
    expect(p).not.toBeNull()
    expect(p!.subTierMin).toBe(1450)
    expect(p!.subTierWidth).toBe(50)
    expect(p!.pct).toBeCloseTo(0.04, 5)
  })

  it('LUSR : largeur de sous-palier variable selon le tier (Platine = 100)', () => {
    // Platine LUSR [1600,1800], 2 sous-paliers de 100. 1770 → [1700,1800].
    const p = subTierPosition(LUSR_TIER_GRID, 1770)
    expect(p).not.toBeNull()
    expect(p!.subTierMin).toBe(1700)
    expect(p!.subTierWidth).toBe(100)
    expect(p!.pct).toBeCloseTo(0.7, 5)
  })

  it('LUSR : Or = sous-paliers de 33.3 pts (≠ 50)', () => {
    // Or LUSR [1400,1600], 6 sous-paliers ≈ 33.33. 1452 → [1433.3,1466.7].
    const p = subTierPosition(LUSR_TIER_GRID, 1452)
    expect(p).not.toBeNull()
    expect(p!.subTierWidth).toBeCloseTo(33.333, 2)
    expect(p!.subTierMin).toBeCloseTo(1433.333, 2)
  })

  it('palier ouvert (Onyx) → null', () => {
    expect(subTierPosition(CSR_TIER_GRID, 1600)).toBeNull()
    expect(subTierPosition(LUSR_TIER_GRID, 2100)).toBeNull()
  })

  it('hors grille (sous le plancher) → null', () => {
    expect(subTierPosition(LUSR_TIER_GRID, 500)).toBeNull()
  })
})

describe('localizeTierName', () => {
  it('EN → FR (Gold → Or, Platinum → Platine)', () => {
    expect(localizeTierName('Gold', 'fr')).toBe('Or')
    expect(localizeTierName('Platinum', 'fr')).toBe('Platine')
    expect(localizeTierName('Silver', 'fr')).toBe('Argent')
    expect(localizeTierName('Diamond', 'fr')).toBe('Diamant')
  })
  it('FR → EN (Or → Gold, Platine → Platinum)', () => {
    expect(localizeTierName('Or', 'en')).toBe('Gold')
    expect(localizeTierName('Platine', 'en')).toBe('Platinum')
  })
  it('invariants de locale (Bronze, Onyx, Champion)', () => {
    expect(localizeTierName('Bronze', 'en')).toBe('Bronze')
    expect(localizeTierName('Onyx', 'fr')).toBe('Onyx')
    expect(localizeTierName('Champion', 'fr')).toBe('Champion')
  })
  it('nom inconnu → renvoyé tel quel', () => {
    expect(localizeTierName('Placement', 'en')).toBe('Placement')
    expect(localizeTierName('', 'en')).toBe('')
  })
})

describe('localizeTierLabel', () => {
  it('libellé FR baké sous UI EN (« Or IV » → « Gold IV »)', () => {
    expect(localizeTierLabel('Or IV', 'en')).toBe('Gold IV')
    expect(localizeTierLabel('Platine II', 'en')).toBe('Platinum II')
    expect(localizeTierLabel('Diamant III', 'en')).toBe('Diamond III')
  })
  it('libellé H5 EN baké sous UI FR (« Platinum » → « Platine », arabe préservé)', () => {
    expect(localizeTierLabel('Platinum', 'fr')).toBe('Platine')
    expect(localizeTierLabel('Platinum 4', 'fr')).toBe('Platine 4')
    expect(localizeTierLabel('Gold 3', 'fr')).toBe('Or 3')
  })
  it('Onyx (invariant) + suffixe valeur préservé', () => {
    expect(localizeTierLabel('Onyx', 'en')).toBe('Onyx')
    expect(localizeTierLabel('Onyx 1500', 'fr')).toBe('Onyx 1500')
  })
  it('même locale : identité (« Or IV » sous UI FR reste « Or IV »)', () => {
    expect(localizeTierLabel('Or IV', 'fr')).toBe('Or IV')
    expect(localizeTierLabel('Gold IV', 'en')).toBe('Gold IV')
  })
  it('sentinelles / null / vide → inchangés', () => {
    expect(localizeTierLabel('Placement', 'en')).toBe('Placement')
    expect(localizeTierLabel('Placement (2 restants)', 'en')).toBe('Placement (2 restants)')
    expect(localizeTierLabel(null, 'en')).toBeNull()
    expect(localizeTierLabel(undefined, 'fr')).toBeUndefined()
    expect(localizeTierLabel('', 'en')).toBe('')
  })
})
