/**
 * Tests composant — MediaCard (réglages) : le réglage « supprimer les originaux après
 * conversion » n'existe pas quand le déploiement garde les originaux (bureau).
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { desktopCapabilities, testCapabilities } from '@/test/capabilities'
import { useAppShellStore } from '@/stores/appShellStore'
import { getSettingsText } from './i18n'
import { MediaCard } from './_settingsCards'

function renderCard(locale: 'fr' | 'en' = 'fr', handleChange = vi.fn()) {
  const t = getSettingsText(locale)
  renderWithProviders(<MediaCard merged={{}} handleChange={handleChange} t={t} />)
  return t
}

function setCapabilities(overrides: Parameters<typeof testCapabilities>[0]) {
  act(() => {
    useAppShellStore.setState({ capabilities: testCapabilities(overrides) })
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
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

describe('MediaCard — dossier des captures', () => {
  it('bureau : « Parcourir… » remplit le dossier des captures, une annulation ne change rien', async () => {
    setCapabilities(desktopCapabilities())
    const invoke = vi.fn().mockResolvedValueOnce(null).mockResolvedValueOnce('D:\\Captures')
    vi.stubGlobal('__TAURI_INTERNALS__', { invoke })
    const handleChange = vi.fn()
    renderCard('fr', handleChange)
    const parcourir = screen.getByRole('button', { name: 'Parcourir…' })

    fireEvent.click(parcourir)
    await waitFor(() => expect(invoke).toHaveBeenCalledWith('pick_folder', { initial: '' }))
    await waitFor(() => expect(parcourir).toBeEnabled())
    expect(handleChange).not.toHaveBeenCalled()

    fireEvent.click(parcourir)
    await waitFor(() => expect(handleChange).toHaveBeenCalledWith('media_captures_base_dir', 'D:\\Captures'))
  })

  it('serveur : pas de bouton « Parcourir… »', () => {
    setCapabilities({})
    renderCard()
    expect(screen.queryByRole('button', { name: 'Parcourir…' })).not.toBeInTheDocument()
  })
})
