import { describe, it, expect, vi } from 'vitest'

// Sans palette appliquée (boot app), resolveToken renvoie '' en test. On le mocke
// pour qu'il renvoie le NOM du token → on peut asserter quel token va à quel stop.
vi.mock('@/lib/accessibility', () => ({ resolveToken: (token: string) => token }))

import {
  ONE_LIFE_DAMAGE,
  ONE_LIFE_WINDOW_FACTOR,
  ONE_LIFE_ZONE_OPACITY,
  damagePerDeath,
  damagePerKill,
  defensiveDamageGradient,
  offensiveDamageGradient,
  oneLifeNeutralOffset,
  oneLifeWindowBounds,
  oneLifeZonesMarkArea,
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

describe('oneLifeWindowBounds — fenêtre FIXE autour du pivot', () => {
  it('demi-pivot … double pivot, quel que soit le barème du titre', () => {
    expect(oneLifeWindowBounds(225)).toEqual({ min: 112.5, max: 450 })
    expect(oneLifeWindowBounds(115)).toEqual({ min: 57.5, max: 230 })
    // En unité normalisée (pivot 100 %), c'est la fenêtre 50…200 % des cartes.
    expect(oneLifeWindowBounds(100)).toEqual({ min: 50, max: 200 })
  })
  it('défaut = barème Infinite', () => {
    expect(oneLifeWindowBounds()).toEqual(oneLifeWindowBounds(ONE_LIFE_DAMAGE))
    expect(ONE_LIFE_WINDOW_FACTOR).toBe(2)
  })
  it('fenêtre multiplicativement symétrique : le pivot est au centre géométrique', () => {
    const { min, max } = oneLifeWindowBounds(225)
    expect(min * max).toBeCloseTo(225 ** 2, 6)
  })
})

describe('oneLifeNeutralOffset — arrêt neutre du dégradé', () => {
  it('constant (2/3 pour un facteur 2) : la couleur ne dépend plus de la session', () => {
    expect(oneLifeNeutralOffset()).toBeCloseTo(2 / 3, 10)
  })
})

describe('offensiveDamageGradient (dégâts/frag — monotone, bas = vert)', () => {
  it('rouge en haut (gaspillage), neutre au pivot, vert en bas (efficace)', () => {
    const g = offensiveDamageGradient()
    expect(g.type).toBe('linear')
    expect(g.colorStops).toHaveLength(3)
    expect(g.colorStops[0]).toEqual({ offset: 0, color: 'divergent-neg' })
    expect(g.colorStops[1].color).toBe('divergent-neutral')
    expect(g.colorStops[1].offset).toBeCloseTo(2 / 3, 10)
    expect(g.colorStops[2]).toEqual({ offset: 1, color: 'divergent-pos' })
  })
  it('identique d\'une session à l\'autre (aucune donnée en entrée)', () => {
    expect(offensiveDamageGradient()).toEqual(offensiveDamageGradient())
  })
})

describe('defensiveDamageGradient (dégâts/mort — ascendant)', () => {
  it('vert en haut, neutre au pivot, rouge en bas', () => {
    const g = defensiveDamageGradient()
    expect(g.colorStops).toHaveLength(3)
    expect(g.colorStops[0]).toEqual({ offset: 0, color: 'divergent-pos' })
    expect(g.colorStops[1].color).toBe('divergent-neutral')
    expect(g.colorStops[2]).toEqual({ offset: 1, color: 'divergent-neg' })
  })
})

describe('oneLifeZonesMarkArea — zones de lecture partagées et ORIENTABLES', () => {
  it('above-is-good : vert du pivot au haut de la fenêtre, rouge du bas au pivot', () => {
    const { data } = oneLifeZonesMarkArea(225, 'above-is-good')
    expect(data).toHaveLength(2)
    expect(data[0][0]).toMatchObject({ yAxis: 225 })
    expect(data[0][0].itemStyle.color).toBe('divergent-pos')
    expect(data[0][1]).toEqual({ yAxis: 450 })
    expect(data[1][0]).toMatchObject({ yAxis: 112.5 })
    expect(data[1][0].itemStyle.color).toBe('divergent-neg')
    expect(data[1][1]).toEqual({ yAxis: 225 })
  })

  it('below-is-good : les couleurs S\'INVERSENT, les bornes NON', () => {
    const { data } = oneLifeZonesMarkArea(225, 'below-is-good')
    expect(data[0][0]).toMatchObject({ yAxis: 225 })
    expect(data[0][0].itemStyle.color).toBe('divergent-neg')
    expect(data[0][1]).toEqual({ yAxis: 450 })
    expect(data[1][0]).toMatchObject({ yAxis: 112.5 })
    expect(data[1][0].itemStyle.color).toBe('divergent-pos')
    expect(data[1][1]).toEqual({ yAxis: 225 })
  })

  it('les deux orientations sont bien opposées (aucun défaut ne les uniformise)', () => {
    const above = oneLifeZonesMarkArea(100, 'above-is-good').data
    const below = oneLifeZonesMarkArea(100, 'below-is-good').data
    expect(above[0][0].itemStyle.color).not.toBe(below[0][0].itemStyle.color)
    expect(above[1][0].itemStyle.color).not.toBe(below[1][0].itemStyle.color)
  })

  it('l\'opacité suit le TOKEN et non la position : le rouge reste le plus discret', () => {
    expect(ONE_LIFE_ZONE_OPACITY.neg).toBeLessThan(ONE_LIFE_ZONE_OPACITY.pos)
    for (const direction of ['above-is-good', 'below-is-good'] as const) {
      for (const [start] of oneLifeZonesMarkArea(100, direction).data) {
        const expected =
          start.itemStyle.color === 'divergent-pos'
            ? ONE_LIFE_ZONE_OPACITY.pos
            : ONE_LIFE_ZONE_OPACITY.neg
        expect(start.itemStyle.opacity).toBe(expected)
      }
    }
  })

  it('accepte des bornes explicites (fenêtre déjà calculée par l\'appelant)', () => {
    const { data } = oneLifeZonesMarkArea(100, 'above-is-good', { min: 50, max: 200 })
    expect(data[0][1].yAxis).toBe(200)
    expect(data[1][0].yAxis).toBe(50)
  })

  it('zones silencieuses : elles ne captent jamais le survol', () => {
    expect(oneLifeZonesMarkArea(225, 'above-is-good').silent).toBe(true)
  })
})
