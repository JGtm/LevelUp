/**
 * Tests — vehicleKilledLabel : la parenthèse « tués avec le véhicule » d'une ligne de destruction.
 *
 * Ce qu'ils protègent : aucun tué nommé = aucune parenthèse ; UN tué = son nom ; deux ou davantage =
 * leur nombre, au bon pluriel de la langue (`Intl.PluralRules`), sans possessif ni siège.
 */
import { describe, expect, it } from 'vitest'

import { REPLAY_TEXT } from '../i18n/i18n'
import { vehicleKilledLabel } from './vehicleKilledLabel'

describe('vehicleKilledLabel', () => {
  it('aucun tué nommé : aucune parenthèse', () => {
    expect(vehicleKilledLabel([], 'fr')).toBe('')
    expect(vehicleKilledLabel([], 'en')).toBe('')
  })

  it('un seul tué : son nom', () => {
    expect(vehicleKilledLabel(['Cobra01'], 'fr')).toBe('(Cobra01 tué avec le véhicule)')
    expect(vehicleKilledLabel(['Cobra01'], 'en')).toBe('(Cobra01 killed with the vehicle)')
  })

  it('deux tués ou plus : leur nombre, au pluriel de la langue', () => {
    expect(vehicleKilledLabel(['A', 'B'], 'fr')).toBe('(2 tués avec le véhicule)')
    expect(vehicleKilledLabel(['A', 'B', 'C'], 'fr')).toBe('(3 tués avec le véhicule)')
    expect(vehicleKilledLabel(['A', 'B', 'C'], 'en')).toBe('(3 killed with the vehicle)')
  })

  it('les deux formes du catalogue portent le nombre, et aucune ne dit « à bord » ni un possessif', () => {
    for (const locale of ['fr', 'en'] as const) {
      const t = REPLAY_TEXT[locale]
      for (const form of [t.feedVehicleKilledOne, t.feedVehicleKilledOther]) {
        expect(form).toContain('{n}')
        expect(form).not.toMatch(/à bord|aboard|\b(son|sa|ses|leur|his|her|its|their)\b/i)
      }
    }
  })

  it('le singulier français se dit « tué », le pluriel « tués » : une forme par catégorie', () => {
    expect(REPLAY_TEXT.fr.feedVehicleKilledOne).toBe('({n} tué avec le véhicule)')
    expect(REPLAY_TEXT.fr.feedVehicleKilledOther).toBe('({n} tués avec le véhicule)')
  })
})
