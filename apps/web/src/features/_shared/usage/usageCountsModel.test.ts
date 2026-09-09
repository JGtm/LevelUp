/**
 * usageCountsModel.test.ts — la variante COMPTES du bloc « servi ou gâché » (décision P9,
 * PLAN_EQUIPEMENT_GACHIS_2026-09-09, étape E5.8) : axe en objets pris, aucun pourcentage
 * dans les barres, aucun trait de parité, lignes triées du plus pris au moins pris.
 */
import { describe, expect, it } from 'vitest'

import { buildCountsGrid } from './usageCountsModel'
import { USAGE_TEXT } from './usageI18n'

const t = USAGE_TEXT.fr

describe('buildCountsGrid — P9 : axe en comptes, aucun trait de parité', () => {
  it('trie les lignes du plus pris au moins pris, même si l entrée ne l est pas', () => {
    const grid = buildCountsGrid(
      [
        { key: 'a', label: 'A', taken: 10 },
        { key: 'b', label: 'B', taken: 90 },
        { key: 'c', label: 'C', taken: 40 },
      ],
      { t, locale: 'fr', unit: 'equipment' },
    )
    expect(grid.rows.map((r) => r.key)).toEqual(['b', 'c', 'a'])
  })

  it('parityPct est TOUJOURS null (P9 : pas de trait de parité sur un axe de comptes)', () => {
    const grid = buildCountsGrid([{ key: 'a', label: 'A', taken: 10 }], {
      t,
      locale: 'fr',
      unit: 'equipment',
    })
    expect(grid.rows[0].gauge.parityPct).toBeNull()
  })

  it('valuePct est la longueur RELATIVE A L AXE (pas une part d equipe)', () => {
    const grid = buildCountsGrid(
      [
        { key: 'a', label: 'A', taken: 100 },
        { key: 'b', label: 'B', taken: 50 },
      ],
      { t, locale: 'fr', unit: 'equipment' },
    )
    const a = grid.rows.find((r) => r.key === 'a')!
    const b = grid.rows.find((r) => r.key === 'b')!
    expect(a.gauge.valuePct).toBeGreaterThan(0)
    expect(b.gauge.valuePct).toBeCloseTo(a.gauge.valuePct! / 2, 5)
  })

  it('une ligne SANS outcomes rend un aplat simple (segments undefined), jamais un zero invente', () => {
    const grid = buildCountsGrid([{ key: 'a', label: 'A', taken: 12 }], {
      t,
      locale: 'fr',
      unit: 'weapon',
    })
    expect(grid.rows[0].gauge.segments).toBeUndefined()
    expect(grid.rows[0].gauge.teammatesRatePct).toBeNull()
    expect(grid.rows[0].gauge.opponentsRatePct).toBeNull()
    expect(grid.rows[0].gauge.valueText).toBe('12 prises')
  })

  it('une ligne AVEC outcomes empile utilise -> lache -> garde et porte les deux reperes', () => {
    const grid = buildCountsGrid(
      [
        {
          key: 'wall',
          label: 'Mur',
          taken: 20,
          outcomes: {
            used: 12,
            dropped: 5,
            kept: 3,
            teammates_used_rate_pct: 55,
            opponents_used_rate_pct: 40,
          },
        },
      ],
      { t, locale: 'fr', unit: 'equipment' },
    )
    const row = grid.rows[0]
    expect(row.gauge.segments?.map((s) => s.key)).toEqual(['used', 'dropped', 'kept'])
    expect(row.gauge.teammatesRatePct).toBe(55)
    expect(row.gauge.opponentsRatePct).toBe(40)
    expect(row.gauge.valueText).toBe('20 pris')
  })

  it('axisMaxText porte l unite complete (equipement) et respecte le formatage FR', () => {
    const grid = buildCountsGrid([{ key: 'a', label: 'A', taken: 138 }], {
      t,
      locale: 'fr',
      unit: 'equipment',
    })
    expect(grid.axisMaxText).toMatch(/objets pris$/)
  })

  it('axisMaxText en armes speciales utilise « prises »', () => {
    const grid = buildCountsGrid([{ key: 'a', label: 'A', taken: 41 }], {
      t,
      locale: 'fr',
      unit: 'weapon',
    })
    expect(grid.axisMaxText).toMatch(/prises$/)
  })

  it('aucune ligne : grille vide, pas de crash', () => {
    const grid = buildCountsGrid([], { t, locale: 'fr', unit: 'equipment' })
    expect(grid.rows).toEqual([])
  })

  it('sort:false conserve l ordre d entree (Go trie deja families[])', () => {
    const grid = buildCountsGrid(
      [
        { key: 'a', label: 'A', taken: 10 },
        { key: 'b', label: 'B', taken: 90 },
      ],
      { t, locale: 'fr', unit: 'equipment', sort: false },
    )
    expect(grid.rows.map((r) => r.key)).toEqual(['a', 'b'])
  })
})
