import { describe, it, expect } from 'vitest'
import { favoriteWeaponSlots, signOf } from './ExplorerBriefing.logic'

// formatSignedFixed a migré vers `@/lib/formatters` (number.ts) — testé dans
// `lib/formatters/formatters.test.ts`.
//
// formatSignedPoints et isFullHistoryScope ont migré vers `@/lib/baseline` le
// 2026-09-06 (2e consommateur : le KPI d'échange de l'Escouade) — testés dans
// `lib/baseline.test.ts`.

describe('signOf', () => {
  it('retourne -1 / 0 / 1', () => {
    expect(signOf(2)).toBe(1)
    expect(signOf(-2)).toBe(-1)
    expect(signOf(0)).toBe(0)
    expect(signOf(null)).toBe(0)
  })
})

describe('favoriteWeaponSlots', () => {
  it('rend la place libre sous la cellule la plus haute, plafonnée à deux lignes', () => {
    expect(favoriteWeaponSlots({ dimensionLines: [6, 3, 2], rankedLines: 0 })).toBe(2)
    expect(favoriteWeaponSlots({ dimensionLines: [3], rankedLines: 0 })).toBe(1)
    expect(favoriteWeaponSlots({ dimensionLines: [2, 2], rankedLines: 0 })).toBe(0)
  })

  it('compte le Classement comme les dimensions, et rend zéro sans aucune cellule haute', () => {
    expect(favoriteWeaponSlots({ dimensionLines: [], rankedLines: 3 })).toBe(1)
    expect(favoriteWeaponSlots({ dimensionLines: [], rankedLines: 0 })).toBe(0)
  })

  it('retient le maximum quand dimensions et Classement coexistent', () => {
    expect(favoriteWeaponSlots({ dimensionLines: [6], rankedLines: 3 })).toBe(2)
  })
})
