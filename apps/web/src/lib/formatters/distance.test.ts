import { describe, expect, it } from 'vitest'

import { METERS_PER_GAME_UNIT, gameUnitsToMeters, metersToGameUnits } from './distance'

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
