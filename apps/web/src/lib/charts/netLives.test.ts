/**
 * netLives.test.ts — « Balance des dégâts » d'un match, en vies du titre.
 *
 * (dégâts infligés − dégâts subis) / PV-pour-tuer. null si un terme manque /
 * non-fini ou barème invalide. Division par le barème du titre (225 Infinite,
 * 115 Halo 5).
 */
import { describe, it, expect } from 'vitest'

import { netLives } from './netLives'

describe('netLives', () => {
  it('vies nettes = (dégâts infligés − dégâts subis) / PV-pour-tuer', () => {
    // 900 infligés, 450 subis, 225 PV → (900-450)/225 = 2 vies.
    expect(netLives(900, 450, 225)).toBe(2)
    // Négatif quand on subit plus qu'on inflige.
    expect(netLives(450, 900, 225)).toBe(-2)
    expect(netLives(225, 0, 225)).toBe(1)
  })

  it('divise par le barème du titre (H5 = 115)', () => {
    expect(netLives(230, 0, 115)).toBe(2)
  })

  it('null si un terme manque ou est non-fini', () => {
    expect(netLives(null, 450, 225)).toBeNull()
    expect(netLives(900, undefined, 225)).toBeNull()
    expect(netLives(Infinity, 450, 225)).toBeNull()
    expect(netLives(900, NaN, 225)).toBeNull()
  })

  it('null si le barème du titre est invalide (0, négatif, non-fini)', () => {
    expect(netLives(900, 450, 0)).toBeNull()
    expect(netLives(900, 450, -225)).toBeNull()
    expect(netLives(900, 450, Number.NaN)).toBeNull()
  })

  it('zéro net (autant infligé que subi) = 0 vie, pas null', () => {
    expect(netLives(500, 500, 225)).toBe(0)
  })
})
