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
  previewLabel: 'Aperçu',
} as Parameters<typeof AccessibilityTab>[0]['t']

beforeEach(() => {
  useSettingsDraftStore.setState((s) => ({
    ...s,
    localUiPrefs: { ...s.localUiPrefs, colorPalette: 'default' },
  }))
})

describe('AccessibilityTab', () => {
  it('affiche les deux options de palette', () => {
    render(<AccessibilityTab t={T_STUB} />)
    expect(screen.getByText('Standard (défaut)')).toBeTruthy()
    expect(screen.getByText('Okabe-Ito')).toBeTruthy()
  })

  it('sélectionne la palette default par défaut', () => {
    render(<AccessibilityTab t={T_STUB} />)
    const radios = screen.getAllByRole('radio')
    const defaultRadio = radios.find((r) => (r as HTMLInputElement).value === 'default')
    const okabeRadio = radios.find((r) => (r as HTMLInputElement).value === 'okabe-ito')
    expect((defaultRadio as HTMLInputElement).checked).toBe(true)
    expect((okabeRadio as HTMLInputElement).checked).toBe(false)
  })

  it('met à jour le store en cliquant sur Okabe-Ito', () => {
    render(<AccessibilityTab t={T_STUB} />)
    const radios = screen.getAllByRole('radio')
    const okabeRadio = radios.find((r) => (r as HTMLInputElement).value === 'okabe-ito')!
    fireEvent.click(okabeRadio)
    expect(useSettingsDraftStore.getState().localUiPrefs.colorPalette).toBe('okabe-ito')
  })

  it('affiche la section aperçu', () => {
    render(<AccessibilityTab t={T_STUB} />)
    expect(screen.getByText('Aperçu')).toBeTruthy()
  })
})
