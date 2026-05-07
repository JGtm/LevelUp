/**
 * Tests unitaires — AccessibilityTab.
 */
import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { AccessibilityTab } from './AccessibilityTab'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'

const T_STUB = {
  accessibilityTitle: 'Accessibilité visuelle',
  accessibilityDescription: 'Choisissez une palette',
  paletteLabel: 'Palette de couleurs',
  paletteDefault: 'Standard (défaut)',
  paletteDefaultDesc: 'Palette originale.',
  paletteOkabeIto: 'Okabe-Ito',
  paletteOkabeItoDesc: 'Palette daltonisme.',
  paletteCividis: 'Cividis',
  paletteCividisDesc: 'Palette séquentielle CVD.',
  paletteTolBright: 'Tol Bright',
  paletteTolBrightDesc: 'Palette catégorielle Tol.',
  previewLabel: 'Aperçu',
  teamColorsTitle: 'Couleurs de jeu',
  teamColorsDescription: 'Couleurs outline Halo.',
  allyColorLabel: 'Alliés',
  enemyColorLabel: 'Ennemis',
  teamColorDefault: 'Défaut',
} as Parameters<typeof AccessibilityTab>[0]['t']

const LOCALE = 'fr' as const

beforeEach(() => {
  useSettingsDraftStore.setState((s) => ({
    ...s,
    localUiPrefs: { ...s.localUiPrefs, colorPalette: 'default' },
  }))
})

describe('AccessibilityTab', () => {
  it('affiche les quatre options de palette', () => {
    render(<AccessibilityTab t={T_STUB} locale={LOCALE} />)
    expect(screen.getByText('Standard (défaut)')).toBeTruthy()
    expect(screen.getByText('Okabe-Ito')).toBeTruthy()
    expect(screen.getByText('Cividis')).toBeTruthy()
    expect(screen.getByText('Tol Bright')).toBeTruthy()
  })

  it('sélectionne la palette default par défaut', () => {
    render(<AccessibilityTab t={T_STUB} locale={LOCALE} />)
    const radios = screen.getAllByRole('radio')
    const defaultRadio = radios.find((r) => (r as HTMLInputElement).value === 'default')
    const okabeRadio = radios.find((r) => (r as HTMLInputElement).value === 'okabe-ito')
    expect((defaultRadio as HTMLInputElement).checked).toBe(true)
    expect((okabeRadio as HTMLInputElement).checked).toBe(false)
  })

  it('met à jour le store en cliquant sur Okabe-Ito', () => {
    render(<AccessibilityTab t={T_STUB} locale={LOCALE} />)
    const radios = screen.getAllByRole('radio')
    const okabeRadio = radios.find((r) => (r as HTMLInputElement).value === 'okabe-ito')!
    fireEvent.click(okabeRadio)
    expect(useSettingsDraftStore.getState().localUiPrefs.colorPalette).toBe('okabe-ito')
  })

  it('met à jour le store en cliquant sur Cividis', () => {
    render(<AccessibilityTab t={T_STUB} locale={LOCALE} />)
    const radios = screen.getAllByRole('radio')
    const cividisRadio = radios.find((r) => (r as HTMLInputElement).value === 'cividis')!
    fireEvent.click(cividisRadio)
    expect(useSettingsDraftStore.getState().localUiPrefs.colorPalette).toBe('cividis')
  })

  it('met à jour le store en cliquant sur Tol Bright', () => {
    render(<AccessibilityTab t={T_STUB} locale={LOCALE} />)
    const radios = screen.getAllByRole('radio')
    const tolRadio = radios.find((r) => (r as HTMLInputElement).value === 'tol-bright')!
    fireEvent.click(tolRadio)
    expect(useSettingsDraftStore.getState().localUiPrefs.colorPalette).toBe('tol-bright')
  })

  it('affiche la section aperçu', () => {
    render(<AccessibilityTab t={T_STUB} locale={LOCALE} />)
    expect(screen.getByText('Aperçu')).toBeTruthy()
  })
})
