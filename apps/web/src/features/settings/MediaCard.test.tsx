/**
 * Tests composant — MediaCard (réglages) : le réglage « supprimer les originaux après
 * conversion » n'existe pas quand le déploiement garde les originaux (bureau).
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { desktopCapabilities, testCapabilities } from '@/test/capabilities'
import { useAppShellStore } from '@/stores/appShellStore'
import { getSettingsText } from './i18n'
import { MediaCard } from './_settingsCards'

function renderCard(locale: 'fr' | 'en' = 'fr') {
  const t = getSettingsText(locale)
  renderWithProviders(<MediaCard merged={{}} handleChange={vi.fn()} t={t} />)
  return t
}

function setCapabilities(overrides: Parameters<typeof testCapabilities>[0]) {
  act(() => {
    useAppShellStore.setState({ capabilities: testCapabilities(overrides) })
  })
}

afterEach(() => {
  act(() => {
    useAppShellStore.setState({ capabilities: null })
  })
})

describe('MediaCard — suppression des originaux', () => {
  it.each(['fr', 'en'] as const)('serveur (%s) : le réglage et son aide sont affichés', (locale) => {
    setCapabilities({})
    const t = renderCard(locale)
    expect(screen.getByText(t.mediaDeleteSource)).toBeInTheDocument()
    expect(screen.getByText(t.mediaDeleteSourceHint)).toBeInTheDocument()
  })

  it.each(['fr', 'en'] as const)('bureau (%s) : réglage et aide masqués, le reste de la carte demeure', (locale) => {
    setCapabilities(desktopCapabilities())
    const t = renderCard(locale)
    expect(screen.queryByText(t.mediaDeleteSource)).not.toBeInTheDocument()
    expect(screen.queryByText(t.mediaDeleteSourceHint)).not.toBeInTheDocument()
    expect(screen.getByText(t.mediaWatcherEnabled)).toBeInTheDocument()
    expect(screen.getByText(t.mediaToleranceLabel)).toBeInTheDocument()
  })

  it('avant hydratation (aucune capacité) : le réglage est affiché', () => {
    const t = renderCard()
    expect(screen.getByText(t.mediaDeleteSource)).toBeInTheDocument()
  })
})
