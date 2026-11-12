/**
 * Tests — vehicleAboardLabel : la parenthèse « à bord » d'une ligne de destruction.
 *
 * Ce qu'ils protègent : aucun joueur connu = aucune parenthèse ; UN joueur connu = son nom ; deux ou
 * davantage = leur nombre, au bon pluriel de la langue (`Intl.PluralRules`), sans possessif.
 */
import { describe, expect, it } from 'vitest'

import { REPLAY_TEXT } from '../i18n/i18n'
import { vehicleAboardLabel } from './vehicleAboardLabel'

describe('vehicleAboardLabel', () => {
  it('personne de connu à bord : aucune parenthèse', () => {
    expect(vehicleAboardLabel([], 'fr')).toBe('')
    expect(vehicleAboardLabel([], 'en')).toBe('')
  })

  it('un seul joueur connu : son nom', () => {
    expect(vehicleAboardLabel(['Cobra01'], 'fr')).toBe('(Cobra01 à bord)')
    expect(vehicleAboardLabel(['Cobra01'], 'en')).toBe('(Cobra01 aboard)')
  })

  it('deux joueurs connus ou plus : leur nombre', () => {
    expect(vehicleAboardLabel(['A', 'B'], 'fr')).toBe('(2 à bord)')
    expect(vehicleAboardLabel(['A', 'B', 'C'], 'en')).toBe('(3 aboard)')
  })

  it('le nombre suit la catégorie de pluriel de la langue, par les deux formes du catalogue', () => {
    for (const locale of ['fr', 'en'] as const) {
      const t = REPLAY_TEXT[locale]
      expect(t.feedVehicleAboardOne).toContain('{n}')
      expect(t.feedVehicleAboardOther).toContain('{n}')
    }
  })
})
