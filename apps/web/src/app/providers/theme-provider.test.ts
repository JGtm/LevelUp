/**
 * theme-provider.test.ts — Vérifie le garde-fou défensif `pickPalette`.
 *
 * Le merge handler de Zustand persist (settingsDraftStore) ne valide pas
 * les valeurs persistées en localStorage. Une valeur `colorPalette` invalide
 * (ex. après rollback d'une palette) ne doit pas planter — `pickPalette`
 * doit retomber sur `defaultPalette` via le `default:` du switch.
 */
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { pickPalette } from './palette-picker'
import {
  defaultPalette,
  okabePalette,
  cividisPalette,
  tolBrightPalette,
} from '@/lib/accessibility'
import { log } from '@/lib/accessibility/_logger'
import type { ColorPalette } from '@/stores/settingsDraftStore'

beforeEach(() => {
  log._resetForTests()
})

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

  it('logue un warn (dédupliqué) quand la valeur est inconnue', () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    pickPalette('unknown' as ColorPalette)
    pickPalette('unknown' as ColorPalette) // 2e appel : déduplication
    pickPalette('autre' as ColorPalette)

    expect(warnSpy).toHaveBeenCalledTimes(2) // unknown logué 1×, autre 1×
    expect(warnSpy.mock.calls[0]![0]).toContain('palette inconnue "unknown"')
    expect(warnSpy.mock.calls[1]![0]).toContain('palette inconnue "autre"')

    warnSpy.mockRestore()
  })

  it('ne logue rien sur les valeurs valides', () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    pickPalette('default')
    pickPalette('okabe-ito')
    pickPalette('cividis')
    pickPalette('tol-bright')

    expect(warnSpy).not.toHaveBeenCalled()

    warnSpy.mockRestore()
  })
})
