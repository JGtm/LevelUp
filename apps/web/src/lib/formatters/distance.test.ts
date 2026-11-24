import { describe, expect, it } from 'vitest'

import {
  METERS_PER_GAME_UNIT,
  formatGameDistance,
  gameUnitsToMeters,
  metersToGameUnits,
} from './distance'

describe('conversion unités du jeu / mètres', () => {
  it('une unité du jeu vaut 10 pieds, soit 3,048 m', () => {
    expect(METERS_PER_GAME_UNIT).toBe(3.048)
    expect(gameUnitsToMeters(1)).toBeCloseTo(3.048, 10)
    expect(gameUnitsToMeters(7)).toBeCloseTo(21.336, 10)
  })

  it('les deux sens se répondent', () => {
    expect(metersToGameUnits(gameUnitsToMeters(12.5))).toBeCloseTo(12.5, 10)
    expect(metersToGameUnits(18)).toBeCloseTo(5.905511811, 6)
  })
})

describe('formatGameDistance — des unités du jeu, écrites en mètres lisibles', () => {
  it('au mètre dès 10 m, au dixième en dessous', () => {
    expect(formatGameDistance(7, 'fr')).toBe('21')
    expect(formatGameDistance(18, 'fr')).toBe('55')
    expect(formatGameDistance(1.5, 'fr')).toBe('4,6')
    expect(formatGameDistance(1.5, 'en')).toBe('4.6')
  })

  it('jamais de décimale parasite (pas de « 54,864 »)', () => {
    expect(formatGameDistance(18, 'en')).toBe('55')
    expect(formatGameDistance(0.5, 'fr')).toBe('1,5')
    expect(formatGameDistance(2, 'fr')).toBe('6,1')
  })

  it('un zéro reste « 0 », une absence reste un tiret', () => {
    expect(formatGameDistance(0, 'fr')).toBe('0')
    expect(formatGameDistance(null, 'fr')).toBe('—')
    expect(formatGameDistance(undefined, 'en')).toBe('—')
  })

  it('une distance négative (coordonnée) garde son signe moins typographique', () => {
    expect(formatGameDistance(-4, 'fr')).toBe('−12')
  })
})
