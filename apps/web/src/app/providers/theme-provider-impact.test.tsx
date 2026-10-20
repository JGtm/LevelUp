/**
 * theme-provider-impact.test.tsx — le basculement de palette et de thème écrit sur :root les
 * valeurs des rôles d'impact propres à la palette choisie et au thème (`paletteForTheme`).
 */
import { act, render } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'

import { tokenVar } from '@/lib/accessibility'
import { _resetActivePalette } from '@/lib/accessibility/applyPalette'
import {
  IMPACT_BLUE_VERMILLION,
  IMPACT_GREEN_RED,
  IMPACT_TOL_BLUE_RED,
  type ImpactRoleRamps,
  type ImpactRoleToken,
} from '@/lib/accessibility/palettes/_impactRoleColors'
import { type ColorPalette, useSettingsDraftStore } from '@/stores/settingsDraftStore'

import { ThemeProvider } from './theme-provider'

const EXPECTED: Record<ColorPalette, ImpactRoleRamps> = {
  default: IMPACT_GREEN_RED,
  'okabe-ito': IMPACT_BLUE_VERMILLION,
  cividis: IMPACT_BLUE_VERMILLION,
  'tol-bright': IMPACT_TOL_BLUE_RED,
}

const initial = useSettingsDraftStore.getState().localUiPrefs

afterEach(() => {
  useSettingsDraftStore.setState({ localUiPrefs: initial })
  _resetActivePalette()
})

function applied(token: ImpactRoleToken): string {
  return document.documentElement.style.getPropertyValue(tokenVar(token))
}

function expectRamp(ramp: Record<ImpactRoleToken, string>): void {
  for (const [token, hex] of Object.entries(ramp) as [ImpactRoleToken, string][]) {
    expect(applied(token), token).toBe(hex)
  }
}

describe('ThemeProvider — rôles d’impact', () => {
  it('chaque palette, en clair puis en sombre, pose ses propres valeurs', () => {
    useSettingsDraftStore.setState({ localUiPrefs: { ...initial, theme: 'light', colorPalette: 'default' } })
    render(<ThemeProvider><span /></ThemeProvider>)
    const { setColorPalette, setTheme } = useSettingsDraftStore.getState()

    for (const [key, ramps] of Object.entries(EXPECTED) as [ColorPalette, ImpactRoleRamps][]) {
      act(() => { setTheme('light'); setColorPalette(key) })
      expectRamp(ramps.light)
      act(() => setTheme('dark'))
      expectRamp(ramps.dark)
    }
  })
})
