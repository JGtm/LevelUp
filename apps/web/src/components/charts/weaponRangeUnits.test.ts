import { describe, expect, it } from 'vitest'

import type { WeaponRangeRow } from '@/lib/api/types'
import { gameUnitsToMeters } from '@/lib/formatters/distance'

import { weaponRangeLines } from './weaponRangeChart'
import { roleAxis, roleRangeLines } from './weaponRangeRoles'

/**
 * Le contrat sert les distances de portée en UNITÉS DU JEU (champs `median`, `p10`, `p90`, `min_m`,
 * `max_m`) ; les graphes, tableaux et infobulles les écrivent en mètres. La conversion se fait UNE
 * fois, à la projection des lignes : tout ce qui est en aval lit des mètres.
 */
const COTE = { measured: 12, p10: 2, median: 4, p90: 10, min_m: 1, max_m: 20, above_pct: 0, below_pct: 0, level_pct: 100 }
const ARME = { weapon_key: 'hinf_br75', label: 'BR75', kills: COTE } as unknown as WeaponRangeRow

describe('portée des engagements : unités du jeu → mètres', () => {
  it('weaponRangeLines convertit les cinq distances d’un côté, et laisse les parts intactes', () => {
    const [ligne] = weaponRangeLines([ARME])
    expect(ligne.top).toMatchObject({ measured: 12, level_pct: 100 })
    expect(ligne.top?.median).toBeCloseTo(gameUnitsToMeters(4), 10)
    expect(ligne.top?.p10).toBeCloseTo(gameUnitsToMeters(2), 10)
    expect(ligne.top?.p90).toBeCloseTo(gameUnitsToMeters(10), 10)
    expect(ligne.top?.min_m).toBeCloseTo(gameUnitsToMeters(1), 10)
    expect(ligne.top?.max_m).toBeCloseTo(gameUnitsToMeters(20), 10)
    expect(ligne.bottom).toBeNull()
  })

  it('roleRangeLines convertit de même (Comparer, Explorateur)', () => {
    const bloc = { weapons: [ARME] } as unknown as Parameters<typeof roleAxis>[0][number]
    const axe = roleAxis([bloc], (k) => k)
    const [ligne] = roleRangeLines(axe, bloc, null, 'kills')
    expect(ligne.top?.median).toBeCloseTo(12.192, 10)
    expect(ligne.bottom).toBeNull()
  })
})
