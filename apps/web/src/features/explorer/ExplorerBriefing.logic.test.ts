import { describe, it, expect } from 'vitest'
import {
  dimensionCardLines,
  favoriteWeaponSlots,
  INSUFFICIENT_CARD_LINES,
  signOf,
  SOCLE_GAP_PX,
  SOCLE_GRID_CLASSES,
  SOCLE_TILE_MIN_PX,
  socleGridClass,
} from './ExplorerBriefing.logic'

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
  it('en cellule propre : deux armes dès que la rangée offre deux lignes au-delà de deux', () => {
    const seul = (dimensionLines: number[], rankedLines = 0) =>
      favoriteWeaponSlots({ dimensionLines, rankedLines, stacked: false })
    expect(seul([6, 3, 2])).toBe(2)
    expect(seul([6])).toBe(2)
    expect(seul([3])).toBe(1)
    expect(seul([2, 2])).toBe(1)
    expect(seul([2])).toBe(1)
  })

  it('empilé : deux lignes de plus à payer, donc la seconde arme demande une rangée plus haute', () => {
    const empile = (dimensionLines: number[], rankedLines = 0) =>
      favoriteWeaponSlots({ dimensionLines, rankedLines, stacked: true })
    expect(empile([6])).toBe(2)
    expect(empile([5])).toBe(1)
    expect(empile([4])).toBe(1)
    expect(empile([3])).toBe(1)
  })

  it('ne rend JAMAIS zéro : le bloc est toujours là, au minimum une arme', () => {
    expect(favoriteWeaponSlots({ dimensionLines: [], rankedLines: 0, stacked: false })).toBe(1)
    expect(favoriteWeaponSlots({ dimensionLines: [], rankedLines: 0, stacked: true })).toBe(1)
    expect(favoriteWeaponSlots({ dimensionLines: [1], rankedLines: 0, stacked: true })).toBe(1)
  })

  it('compte le Classement comme les dimensions', () => {
    expect(favoriteWeaponSlots({ dimensionLines: [], rankedLines: 3, stacked: false })).toBe(1)
    expect(favoriteWeaponSlots({ dimensionLines: [], rankedLines: 4, stacked: false })).toBe(2)
    expect(favoriteWeaponSlots({ dimensionLines: [], rankedLines: 6, stacked: true })).toBe(2)
    expect(favoriteWeaponSlots({ dimensionLines: [], rankedLines: 3, stacked: true })).toBe(1)
  })

  it('retient le maximum quand dimensions et Classement coexistent', () => {
    expect(favoriteWeaponSlots({ dimensionLines: [6], rankedLines: 3, stacked: false })).toBe(2)
    expect(favoriteWeaponSlots({ dimensionLines: [3], rankedLines: 6, stacked: true })).toBe(2)
  })

  it('une carte de dimension en état vide pèse INSUFFICIENT_CARD_LINES lignes, pas zéro', () => {
    expect(INSUFFICIENT_CARD_LINES).toBe(2)
    expect(dimensionCardLines({ entries: [], insufficient: true })).toBe(INSUFFICIENT_CARD_LINES)
    expect(dimensionCardLines({ entries: [], insufficient: false })).toBe(0)
    expect(dimensionCardLines({ entries: [{}, {}, {}] })).toBe(3)
    expect(dimensionCardLines({ entries: null })).toBe(0)
    // Une rangée faite d'une seule carte en état vide n'offre pas de quoi porter une 2e arme.
    const lines = [{ entries: [], insufficient: true }].map(dimensionCardLines)
    expect(favoriteWeaponSlots({ dimensionLines: lines, rankedLines: 0, stacked: false })).toBe(1)
  })
})

describe('socleGridClass', () => {
  // Recalcule la règle de colonnes (indépendamment de la table) : { n, ceil(n/2), 2 } bornées à 2 au moins, sans
  // doublon (n si n < 2), de la plus petite à la plus grande. Aucun palier intermédiaire.
  const candidates = (n: number): number[] =>
    n < 2
      ? [n]
      : [...new Set([n, Math.max(2, Math.ceil(n / 2)), 2])].sort((a, b) => a - b)
  const fits = (c: number) => c * SOCLE_TILE_MIN_PX + (c - 1) * SOCLE_GAP_PX

  const expected = (n: number): string => {
    const [base, ...rest] = candidates(n)
    return [`grid-cols-${base}`, ...rest.map((c) => `@min-[${fits(c)}px]:grid-cols-${c}`)].join(' ')
  }

  it('la règle vaut 150 px par tuile et 8 px de gouttière', () => {
    expect(SOCLE_TILE_MIN_PX).toBe(150)
    expect(SOCLE_GAP_PX).toBe(8)
  })

  it.each([1, 2, 3, 4, 5, 6, 7, 8])('la table de %i tuile(s) correspond à la règle recalculée', (n) => {
    expect(SOCLE_GRID_CLASSES[n]).toBe(expected(n))
    expect(socleGridClass(n)).toBe(expected(n))
  })

  it('les configurations attendues : 8 -> 2/4/8, 7 -> 2/4/7, 6 -> 2/3/6, 5 -> 2/3/5, 4 -> 2/4, 3 -> 2/3', () => {
    expect(candidates(8)).toEqual([2, 4, 8])
    expect(candidates(7)).toEqual([2, 4, 7])
    expect(candidates(6)).toEqual([2, 3, 6])
    expect(candidates(5)).toEqual([2, 3, 5])
    expect(candidates(4)).toEqual([2, 4])
    expect(candidates(3)).toEqual([2, 3])
  })

  it('borne le nombre de tuiles à 1..8', () => {
    expect(socleGridClass(0)).toBe(SOCLE_GRID_CLASSES[1])
    expect(socleGridClass(12)).toBe(SOCLE_GRID_CLASSES[8])
    expect(socleGridClass(Number.NaN)).toBe(SOCLE_GRID_CLASSES[1])
  })
})
