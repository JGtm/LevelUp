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
  it('en cellule propre : la place libre au-dessus de deux lignes, plafonnée à deux', () => {
    const seul = (dimensionLines: number[], rankedLines = 0) =>
      favoriteWeaponSlots({ dimensionLines, rankedLines, stacked: false })
    expect(seul([6, 3, 2])).toBe(2)
    expect(seul([6])).toBe(2)
    expect(seul([3])).toBe(1)
    expect(seul([2, 2])).toBe(0)
    expect(seul([2])).toBe(0)
  })

  it('empilé : deux lignes de plus à payer, donc la même place demande une rangée plus haute', () => {
    const empile = (dimensionLines: number[], rankedLines = 0) =>
      favoriteWeaponSlots({ dimensionLines, rankedLines, stacked: true })
    expect(empile([6])).toBe(2)
    expect(empile([5])).toBe(1)
    expect(empile([4])).toBe(0)
    expect(empile([3])).toBe(0)
  })

  it('compte le Classement comme les dimensions, et rend zéro sans aucune cellule haute', () => {
    expect(favoriteWeaponSlots({ dimensionLines: [], rankedLines: 3, stacked: false })).toBe(1)
    expect(favoriteWeaponSlots({ dimensionLines: [], rankedLines: 0, stacked: false })).toBe(0)
    expect(favoriteWeaponSlots({ dimensionLines: [], rankedLines: 6, stacked: true })).toBe(2)
    expect(favoriteWeaponSlots({ dimensionLines: [], rankedLines: 3, stacked: true })).toBe(0)
  })

  it('retient le maximum quand dimensions et Classement coexistent', () => {
    expect(favoriteWeaponSlots({ dimensionLines: [6], rankedLines: 3, stacked: false })).toBe(2)
    expect(favoriteWeaponSlots({ dimensionLines: [3], rankedLines: 6, stacked: true })).toBe(2)
  })
})
