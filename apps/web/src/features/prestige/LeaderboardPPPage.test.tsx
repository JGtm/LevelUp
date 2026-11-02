/**
 * Tests — Leaderboard PP de la communauté : sur une application de bureau (un seul compte suivi, capacité
 * `can_compare_accounts` fausse), la page dit honnêtement qu'elle n'a rien à comparer, au lieu d'un
 * classement vide.
 */
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { desktopCapabilities, testCapabilities } from '@/test/capabilities'
import { LeaderboardPPPage } from './LeaderboardPPPage'

type State = ReturnType<typeof useAppShellStore.getState>

describe('LeaderboardPPPage — comparaison entre comptes', () => {
  beforeEach(() => {
    useAppShellStore.setState({
      locale: 'fr',
      currentPlayer: { player_slug: 'p', gamertag: 'P' } as unknown as State['currentPlayer'],
    })
  })
  afterEach(() => useAppShellStore.setState({ locale: 'fr', capabilities: null, currentPlayer: null }))

  it('bureau : bloc d’explication affiché, aucun classement', () => {
    useAppShellStore.setState({ capabilities: testCapabilities(desktopCapabilities()) })
    renderWithProviders(<LeaderboardPPPage />)
    expect(
      screen.getByText("Cette page compare plusieurs comptes ; l'application de bureau n'en suit qu'un."),
    ).toBeInTheDocument()
    expect(screen.queryByText(/Classement Prestige entre amis/)).not.toBeInTheDocument()
  })

  it('bureau, en anglais', () => {
    useAppShellStore.setState({ locale: 'en', capabilities: testCapabilities(desktopCapabilities()) })
    renderWithProviders(<LeaderboardPPPage />)
    expect(
      screen.getByText('This page compares several accounts; the desktop application follows only one.'),
    ).toBeInTheDocument()
  })

  it('site : la page garde son classement', () => {
    useAppShellStore.setState({ capabilities: testCapabilities({}) })
    renderWithProviders(<LeaderboardPPPage />)
    expect(screen.getByText(/Classement Prestige entre amis/)).toBeInTheDocument()
    expect(screen.queryByText(/compare plusieurs comptes/)).not.toBeInTheDocument()
  })
})
