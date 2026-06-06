import { describe, it, expect, vi } from 'vitest'

// Sans palette appliquée (boot app), resolveToken renvoie '' en test. On le mocke
// pour qu'il renvoie le NOM du token → on peut asserter quel token va à quel stop.
vi.mock('@/lib/accessibility', () => ({ resolveToken: (token: string) => token }))

import {
  ONE_LIFE_DAMAGE,
  damageAxisBounds,
  damagePerDeath,
  damagePerKill,
  defensiveDamageGradient,
  offensiveDamageGradient,
} from './oneLifeDamageGradient'

describe('damagePerKill / damagePerDeath', () => {
  it('dégâts/frag = ΣDD / frags, arrondi', () => {
    expect(damagePerKill(900, 4)).toBe(225)
    expect(damagePerKill(1000, 3)).toBe(333)
  })
  it('dégâts/mort = ΣDT / morts, arrondi', () => {
    expect(damagePerDeath(1000, 4)).toBe(250)
  })
  it('null si dénominateur nul ou donnée absente', () => {
    expect(damagePerKill(900, 0)).toBeNull()
    expect(damagePerKill(null, 4)).toBeNull()
    expect(damagePerDeath(undefined, 4)).toBeNull()
    expect(damagePerDeath(1000, 0)).toBeNull()
  })
})

describe('offensiveDamageGradient (dégâts/frag — divergent centré 225)', () => {
  it('valeurs à cheval sur 225 → vert (pos) au milieu, rouge (neg) aux extrémités', () => {
    const g = offensiveDamageGradient([150, 300])
    expect(g.type).toBe('linear')
    expect(g.colorStops).toHaveLength(3)
    expect(g.colorStops[0].color).toBe('divergent-neg')
    expect(g.colorStops[1].color).toBe('divergent-pos')
    expect(g.colorStops[2].color).toBe('divergent-neg')
    expect(g.colorStops[1].offset).toBeGreaterThan(0)
    expect(g.colorStops[1].offset).toBeLessThan(1)
  })
  it('toutes au-dessus de 225 → rouge en haut, vert en bas (le plus proche de 225)', () => {
    const g = offensiveDamageGradient([260, 400])
    expect(g.colorStops).toHaveLength(2)
    expect(g.colorStops[0]).toEqual({ offset: 0, color: 'divergent-neg' })
    expect(g.colorStops[1]).toEqual({ offset: 1, color: 'divergent-pos' })
  })
})

describe('defensiveDamageGradient (dégâts/mort — ascendant)', () => {
  it('valeurs à cheval sur 225 → vert haut, neutre au repère, rouge bas', () => {
    const g = defensiveDamageGradient([150, 320])
    expect(g.colorStops).toHaveLength(3)
    expect(g.colorStops[0].color).toBe('divergent-pos')
    expect(g.colorStops[1].color).toBe('divergent-neutral')
    expect(g.colorStops[2].color).toBe('divergent-neg')
    expect(g.colorStops[0].offset).toBe(0)
    expect(g.colorStops[2].offset).toBe(1)
  })
  it('vert (haut) ≠ rouge (bas)', () => {
    const g = defensiveDamageGradient([100, 500])
    expect(g.colorStops[0].color).toBe('divergent-pos')
    expect(g.colorStops[g.colorStops.length - 1].color).toBe('divergent-neg')
  })
})

describe('damageAxisBounds', () => {
  it('garantit la visibilité du repère 225', () => {
    const b = damageAxisBounds([260, 400])
    expect(b.min).toBeLessThanOrEqual(ONE_LIFE_DAMAGE)
    expect(b.max).toBeGreaterThanOrEqual(ONE_LIFE_DAMAGE)
  })
  it('ignore les null', () => {
    const b = damageAxisBounds([null, 300, null])
    expect(b.max).toBeGreaterThanOrEqual(300)
  })
})
