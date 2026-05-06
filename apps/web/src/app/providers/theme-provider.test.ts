/**
 * theme-provider.test.ts — Vérifie le garde-fou défensif `pickPalette`.
 *
 * Le merge handler de Zustand persist (settingsDraftStore) ne valide pas
 * les valeurs persistées en localStorage. Une valeur `colorPalette` invalide
 * (ex. après rollback d'une palette) ne doit pas planter — `pickPalette`
 * doit retomber sur `defaultPalette` via le `default:` du switch.
 */
import { describe, it, expect } from 'vitest'
import { pickPalette } from './palette-picker'
import {
  defaultPalette,
  okabePalette,
  cividisPalette,
  tolBrightPalette,
} from '@/lib/accessibility'
import type { ColorPalette } from '@/stores/settingsDraftStore'

describe('pickPalette', () => {
  it('retourne defaultPalette pour "default"', () => {
    expect(pickPalette('default')).toBe(defaultPalette)
  })

  it('retourne okabePalette pour "okabe-ito"', () => {
    expect(pickPalette('okabe-ito')).toBe(okabePalette)
  })

  it('retourne cividisPalette pour "cividis"', () => {
    expect(pickPalette('cividis')).toBe(cividisPalette)
  })

  it('retourne tolBrightPalette pour "tol-bright"', () => {
    expect(pickPalette('tol-bright')).toBe(tolBrightPalette)
  })

  it('retombe sur defaultPalette pour une valeur inconnue (garde-fou rollback)', () => {
    expect(pickPalette('unknown' as ColorPalette)).toBe(defaultPalette)
    expect(pickPalette('' as ColorPalette)).toBe(defaultPalette)
  })
})
