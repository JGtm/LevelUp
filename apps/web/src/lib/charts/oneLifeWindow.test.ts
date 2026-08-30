import { describe, it, expect, vi } from 'vitest'

// Sans palette appliquée (boot app), resolveToken renvoie '' en test. On le mocke
// pour qu'il renvoie le NOM du token → on peut asserter quel token va à quelle zone.
vi.mock('@/lib/accessibility', () => ({ resolveToken: (token: string) => token }))

import {
  ONE_LIFE_DAMAGE,
  ONE_LIFE_RATE_BOUNDS,
  ONE_LIFE_RATE_PCT,
  ONE_LIFE_WINDOW_FACTOR,
  ONE_LIFE_ZONE_OPACITY,
  damagePerDeath,
  oneLifeDefensiveRatePct,
  oneLifeOffensiveRatePct,
  oneLifeWindowBounds,
  oneLifeWindowBoundsForData,
  oneLifeZonesMarkArea,
} from './oneLifeWindow'

describe('damagePerDeath', () => {
  it('dégâts/mort = ΣDT / morts, arrondi', () => {
    expect(damagePerDeath(1000, 4)).toBe(250)
    expect(damagePerDeath(900, 4)).toBe(225)
  })
  it('null si dénominateur nul ou donnée absente', () => {
    expect(damagePerDeath(undefined, 4)).toBeNull()
    expect(damagePerDeath(1000, 0)).toBeNull()
    expect(damagePerDeath(null, 4)).toBeNull()
  })
})

describe('oneLifeWindowBounds — fenêtre FIXE autour du pivot', () => {
  it('demi-pivot … double pivot, quel que soit le barème du titre', () => {
    expect(oneLifeWindowBounds(225)).toEqual({ min: 112.5, max: 450 })
    expect(oneLifeWindowBounds(115)).toEqual({ min: 57.5, max: 230 })
    // Unité de TOUTES les surfaces (pivot 100 %) : la fenêtre 50…200 %.
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
  it('la fenêtre en % est constante et partagée : 50…200 autour de 100', () => {
    expect(ONE_LIFE_RATE_PCT).toBe(100)
    expect(ONE_LIFE_RATE_BOUNDS).toEqual({ min: 50, max: 200 })
  })
})

describe('oneLifeWindowBoundsForData — élargit sans jamais rétrécir (DEC-5)', () => {
  it('aucune valeur → la fenêtre de base, inchangée', () => {
    expect(oneLifeWindowBoundsForData([])).toEqual(ONE_LIFE_RATE_BOUNDS)
  })

  it('toutes les valeurs DANS la fenêtre → la fenêtre de base, inchangée', () => {
    expect(oneLifeWindowBoundsForData([50, 108, 199.9])).toEqual(ONE_LIFE_RATE_BOUNDS)
  })

  it('une valeur PILE sur une borne n\'est pas un dépassement', () => {
    expect(oneLifeWindowBoundsForData([200])).toEqual(ONE_LIFE_RATE_BOUNDS)
    expect(oneLifeWindowBoundsForData([50])).toEqual(ONE_LIFE_RATE_BOUNDS)
  })

  it('dépassement HAUT : le plafond s\'élargit à la dizaine supérieure, le plancher ne bouge pas', () => {
    expect(oneLifeWindowBoundsForData([237])).toEqual({ min: 50, max: 240 })
  })

  it('dépassement BAS : le plancher s\'élargit à la dizaine inférieure, le plafond ne bouge pas', () => {
    expect(oneLifeWindowBoundsForData([42])).toEqual({ min: 40, max: 200 })
  })

  it('dépassement des DEUX côtés simultanément (une session très inégale)', () => {
    expect(oneLifeWindowBoundsForData([5.6, 300])).toEqual({ min: 0, max: 300 })
  })

  it('plusieurs dépassements du même côté : retient le PLUS large', () => {
    expect(oneLifeWindowBoundsForData([210, 237, 205])).toEqual({ min: 50, max: 240 })
  })

  it('valeurs null/undefined/non-finies ignorées (jamais NaN ni Infinity en borne)', () => {
    expect(oneLifeWindowBoundsForData([null, undefined, NaN, Infinity, -Infinity, 108])).toEqual(
      ONE_LIFE_RATE_BOUNDS,
    )
  })

  it('accepte une base explicite (ex. fenêtre en dégâts bruts, pivot 225)', () => {
    const base = oneLifeWindowBounds(225) // { min: 112.5, max: 450 }
    expect(oneLifeWindowBoundsForData([500], base)).toEqual({ min: 112.5, max: 500 })
    expect(oneLifeWindowBoundsForData([50], base)).toEqual({ min: 50, max: 450 })
  })
})

describe('conversion brut → taux « une vie » (%)', () => {
  it('offensif : hp / dégâts par frag effectif × 100 — une vie par frag = 100 %', () => {
    expect(oneLifeOffensiveRatePct(225, 225)).toBe(100)
    // 1800 dégâts pour 9 frags effectifs = 200 / frag → une demi-vie de trop.
    expect(oneLifeOffensiveRatePct(200, 225)).toBeCloseTo(112.5, 10)
    // Barème d'un autre titre : le pivot reste 100 %.
    expect(oneLifeOffensiveRatePct(115, 115)).toBe(100)
  })
  it('défensif : dégâts par mort / hp × 100 — une vie encaissée = 100 %', () => {
    expect(oneLifeDefensiveRatePct(225, 225)).toBe(100)
    expect(oneLifeDefensiveRatePct(450, 225)).toBe(200)
    expect(oneLifeDefensiveRatePct(115, 115)).toBe(100)
  })
  it('la polarité s\'inverse à la conversion : MOINS de dégâts par frag = PLUS de taux', () => {
    const efficace = oneLifeOffensiveRatePct(150, 225) as number
    const gaspilleur = oneLifeOffensiveRatePct(450, 225) as number
    expect(efficace).toBeGreaterThan(100)
    expect(gaspilleur).toBeLessThan(100)
  })
  it('valeur absente / non calculable → null (jamais 0 % ni Infinity)', () => {
    expect(oneLifeOffensiveRatePct(null)).toBeNull()
    expect(oneLifeOffensiveRatePct(undefined)).toBeNull()
    expect(oneLifeOffensiveRatePct(0)).toBeNull()
    expect(oneLifeDefensiveRatePct(null)).toBeNull()
    expect(oneLifeDefensiveRatePct(300, 0)).toBeNull()
  })
  it('défaut = barème Infinite pour les deux conversions', () => {
    expect(oneLifeOffensiveRatePct(225)).toBe(oneLifeOffensiveRatePct(225, ONE_LIFE_DAMAGE))
    expect(oneLifeDefensiveRatePct(225)).toBe(oneLifeDefensiveRatePct(225, ONE_LIFE_DAMAGE))
  })
})

describe('oneLifeZonesMarkArea — zones de lecture partagées, polarité UNIQUE', () => {
  it('vert du pivot au haut de la fenêtre, rouge du bas au pivot', () => {
    const { data } = oneLifeZonesMarkArea(225)
    expect(data).toHaveLength(2)
    expect(data[0][0]).toMatchObject({ yAxis: 225 })
    expect(data[0][0].itemStyle.color).toBe('divergent-pos')
    expect(data[0][1]).toEqual({ yAxis: 450 })
    expect(data[1][0]).toMatchObject({ yAxis: 112.5 })
    expect(data[1][0].itemStyle.color).toBe('divergent-neg')
    expect(data[1][1]).toEqual({ yAxis: 225 })
  })

  it('aucune orientation à choisir : le pivot 100 % se peint comme tout autre pivot', () => {
    // Garde-rail anti-retour du paramètre d'orientation : toutes les surfaces
    // tracent un TAUX qui monte quand le joueur va mieux. Une grandeur de
    // polarité inverse se CONVERTIT (oneLifeOffensiveRatePct), elle ne retourne
    // pas les zones.
    const { data } = oneLifeZonesMarkArea(ONE_LIFE_RATE_PCT, ONE_LIFE_RATE_BOUNDS)
    expect(data[0][0].itemStyle.color).toBe('divergent-pos')
    expect(data[1][0].itemStyle.color).toBe('divergent-neg')
    expect(oneLifeZonesMarkArea.length).toBe(1) // pivot seul obligatoire : pas de `direction`
  })

  it('l\'opacité suit le TOKEN et non la position : le rouge reste le plus discret', () => {
    expect(ONE_LIFE_ZONE_OPACITY.neg).toBeLessThan(ONE_LIFE_ZONE_OPACITY.pos)
    for (const [start] of oneLifeZonesMarkArea(100).data) {
      const expected =
        start.itemStyle.color === 'divergent-pos'
          ? ONE_LIFE_ZONE_OPACITY.pos
          : ONE_LIFE_ZONE_OPACITY.neg
      expect(start.itemStyle.opacity).toBe(expected)
    }
  })

  it('accepte des bornes explicites (fenêtre déjà calculée par l\'appelant)', () => {
    const { data } = oneLifeZonesMarkArea(100, { min: 50, max: 200 })
    expect(data[0][1].yAxis).toBe(200)
    expect(data[1][0].yAxis).toBe(50)
  })

  it('zones silencieuses : elles ne captent jamais le survol', () => {
    expect(oneLifeZonesMarkArea(225).silent).toBe(true)
  })
})
