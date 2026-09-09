/**
 * Tests du packing des blocs de catalogue (Médailles / Citations).
 *
 * Ce qui est vérifié : la largeur suit le COMPTAGE, l'ordre d'origine ne bouge jamais,
 * et une rangée ne dépasse jamais la grille.
 */
import { describe, it, expect } from 'vitest'

import {
  BLOCK_GRID_COLUMNS,
  blockSpan,
  packBlockRows,
  rowGridTemplate,
} from './blockRowPacking'

interface Cat {
  key: string
  n: number
}

const pack = (cats: Cat[]) => packBlockRows(cats, (c) => c.n)

describe('blockSpan', () => {
  it('donne le tiers de largeur jusqu’à 3 vignettes', () => {
    expect(blockSpan(0)).toBe(2)
    expect(blockSpan(1)).toBe(2)
    expect(blockSpan(3)).toBe(2)
  })

  it('donne la demi-largeur de 4 à 8 vignettes', () => {
    expect(blockSpan(4)).toBe(3)
    expect(blockSpan(8)).toBe(3)
  })

  it('donne la pleine largeur à partir de 9 vignettes', () => {
    expect(blockSpan(9)).toBe(BLOCK_GRID_COLUMNS)
    expect(blockSpan(40)).toBe(BLOCK_GRID_COLUMNS)
  })
})

describe('packBlockRows', () => {
  it('met trois petits blocs sur la même rangée', () => {
    const rows = pack([
      { key: 'ctf', n: 3 },
      { key: 'koth', n: 2 },
      { key: 'oddball', n: 3 },
    ])
    expect(rows).toHaveLength(1)
    expect(rows[0].blocks.map((b) => b.key)).toEqual(['ctf', 'koth', 'oddball'])
    expect(rows[0].spans).toEqual([2, 2, 2])
    expect(rows[0].rowRemainder).toBe(0)
  })

  it('isole un gros bloc sur sa propre rangée', () => {
    const rows = pack([
      { key: 'combat', n: 30 },
      { key: 'ctf', n: 2 },
    ])
    expect(rows.map((r) => r.blocks.map((b) => b.key))).toEqual([['combat'], ['ctf']])
    expect(rows[0].rowRemainder).toBe(0)
    expect(rows[1].rowRemainder).toBe(4)
  })

  it('associe un demi et un tiers sans jamais dépasser la grille', () => {
    const rows = pack([
      { key: 'a', n: 6 },
      { key: 'b', n: 5 },
      { key: 'c', n: 1 },
    ])
    // a(3) + b(3) = 6 → rangée pleine ; c passe à la suivante.
    expect(rows.map((r) => r.spans)).toEqual([[3, 3], [2]])
    for (const row of rows) {
      const total = row.spans.reduce((s, v) => s + v, 0)
      expect(total).toBeLessThanOrEqual(BLOCK_GRID_COLUMNS)
    }
  })

  it('préserve l’ordre d’entrée, jamais de réordonnancement pour mieux remplir', () => {
    // Un glouton qui réordonnerait mettrait « c » avec « a » ; l'ordre de tri prime.
    const rows = pack([
      { key: 'a', n: 2 },
      { key: 'b', n: 20 },
      { key: 'c', n: 2 },
    ])
    expect(rows.flatMap((r) => r.blocks.map((b) => b.key))).toEqual(['a', 'b', 'c'])
  })

  it('rend une liste vide sans rangée', () => {
    expect(pack([])).toEqual([])
  })
})

describe('rowGridTemplate', () => {
  it('donne une piste par bloc quand la rangée est pleine', () => {
    const rows = pack([
      { key: 'a', n: 2 },
      { key: 'b', n: 2 },
      { key: 'c', n: 2 },
    ])
    expect(rowGridTemplate(rows[0])).toBe('2fr 2fr 2fr')
  })

  it('ajoute une piste fantôme pour les colonnes libres', () => {
    const rows = pack([{ key: 'a', n: 2 }])
    expect(rowGridTemplate(rows[0])).toBe('2fr 4fr')
  })
})
