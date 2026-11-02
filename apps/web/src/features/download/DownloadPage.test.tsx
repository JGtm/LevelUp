/**
 * Tests — DownloadPage : rendu par capacité, par langue, mention Microsoft, SmartScreen, lien de version.
 */
import { describe, it, expect, afterEach } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { CapabilityMap } from '@/lib/api/types'
import { GITHUB_LATEST_RELEASE_URL, MICROSOFT_CONTENT_NOTICE, XBOX_GAME_CONTENT_RULES_URL } from '@/lib/appLinks'
import { KNOWN_LOCALES } from '@/lib/i18n/generated/locales'
import { DownloadPage } from './DownloadPage'
import { DOWNLOAD_TEXT } from './i18n'

function offerDownload(offered: boolean) {
  useAppShellStore.setState({
    // Seule la capacité testée est posée : les autres retombent sur leurs défauts d'avant hydratation.
    capabilities: { can_offer_windows_download: offered } as CapabilityMap,
  })
}

describe('DownloadPage', () => {
  afterEach(() => {
    useAppShellStore.setState({ locale: 'fr', capabilities: null })
  })

  it.each(KNOWN_LOCALES)(
    'capacité vraie : titre, lien de la dernière version, SmartScreen et mention Microsoft en %s',
    (locale) => {
      useAppShellStore.setState({ locale })
      offerDownload(true)
      renderWithProviders(<DownloadPage />)
      const t = DOWNLOAD_TEXT[locale]

      expect(screen.getByRole('heading', { level: 1, name: t.title })).toBeInTheDocument()
      expect(screen.getByRole('link', { name: t.action })).toHaveAttribute('href', GITHUB_LATEST_RELEASE_URL)
      expect(screen.getByRole('heading', { level: 3, name: t.smartScreenHeading })).toBeInTheDocument()
      for (const step of t.smartScreenSteps) {
        expect(screen.getByText(step)).toBeInTheDocument()
      }
      // La mention imposée reste en anglais mot pour mot dans toutes les langues, avec le lien vers les règles.
      expect(screen.getByTestId('microsoft-notice')).toHaveTextContent(MICROSOFT_CONTENT_NOTICE)
      const rules = screen.getAllByRole('link').find((a) => a.getAttribute('href') === XBOX_GAME_CONTENT_RULES_URL)
      expect(rules).toBeDefined()
      expect(screen.queryByText(t.unavailable)).toBeNull()
    },
  )

  it.each(KNOWN_LOCALES)('capacité fausse (application de bureau) : seule la phrase de retrait en %s', (locale) => {
    useAppShellStore.setState({ locale })
    offerDownload(false)
    renderWithProviders(<DownloadPage />)
    expect(screen.getByText(DOWNLOAD_TEXT[locale].unavailable)).toBeInTheDocument()
    expect(screen.queryByRole('heading', { level: 1 })).toBeNull()
    expect(screen.queryByTestId('microsoft-notice')).toBeNull()
  })

  it('SmartScreen : les deux étapes et les libellés de Windows sont dits en français', () => {
    offerDownload(true)
    renderWithProviders(<DownloadPage />)
    expect(screen.getByText('Cliquez sur « Informations complémentaires ».')).toBeInTheDocument()
    expect(screen.getByText('Cliquez sur « Exécuter quand même ».')).toBeInTheDocument()
  })

  it('le lien de version s’ouvre dans un nouvel onglet sans fuite de referrer', () => {
    offerDownload(true)
    renderWithProviders(<DownloadPage />)
    const link = screen.getByRole('link', { name: DOWNLOAD_TEXT.fr.action })
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  })

  it('un bootstrap sans capacités garde l’ancien défaut : la page est proposée', () => {
    useAppShellStore.setState({ capabilities: null })
    renderWithProviders(<DownloadPage />)
    expect(screen.getByRole('heading', { level: 1, name: DOWNLOAD_TEXT.fr.title })).toBeInTheDocument()
  })
})
