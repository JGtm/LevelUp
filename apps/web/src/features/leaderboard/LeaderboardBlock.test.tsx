/**
 * Tests composant — LeaderboardBlock (refonte Classement).
 *
 * Catégorie csr-world par défaut : tableau CSR mondial avec badges tier,
 * highlight du joueur local, états vide / erreur.
 */
import { describe, it, expect } from 'vitest'
import { screen, waitFor, within } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { http, HttpResponse } from 'msw'
import { server } from '@/test/setup'
import { LeaderboardBlock } from './LeaderboardBlock'

const p = (path: string) => `/api/v1${path}`

const ENTRIES = [
  { rank: 1, xuid: 'x1', gamertag: 'LocalAce', csr_value: 1850, tier: 'Onyx', sub_tier: 0, is_local: true },
  { rank: 2, xuid: 'x2', gamertag: 'RemoteRival', csr_value: 1720, tier: 'Diamond', sub_tier: 6, is_local: false },
  { rank: 3, xuid: 'x3', gamertag: 'LocalVet', csr_value: 1600, tier: 'Diamond', sub_tier: 3, is_local: true },
]

function mockLeaderboard(entries: typeof ENTRIES) {
  server.use(
    http.get(p('/players/:playerSlug/pages/leaderboard'), () =>
      HttpResponse.json({
        entries,
        category: 'csr-world',
        season_id: 'csrseason13-2',
        playlist_id: 'p',
        title_slug: 'halo_infinite',
        total: entries.length,
      }),
    ),
  )
}

describe('LeaderboardBlock', () => {
  it('affiche le spinner pendant le chargement', () => {
    mockLeaderboard(ENTRIES)
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)
    expect(document.querySelector('[class*="animate-spin"]')).toBeTruthy()
  })

  it('affiche le titre du classement CSR mondial', () => {
    mockLeaderboard(ENTRIES)
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)
    expect(screen.getByText('Classement CSR mondial')).toBeInTheDocument()
  })

  it('marque les joueurs locaux avec le badge Local', async () => {
    mockLeaderboard(ENTRIES)
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => {
      expect(screen.getByText('LocalAce')).toBeInTheDocument()
    })
    expect(screen.getAllByText('Local')).toHaveLength(2) // LocalAce + LocalVet

    const remoteRow = screen.getByText('RemoteRival').closest('tr')!
    expect(within(remoteRow).queryByText('Local')).not.toBeInTheDocument()
  })

  it('affiche les lignes dans l’ordre du rang', async () => {
    mockLeaderboard(ENTRIES)
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => {
      expect(screen.getByText('LocalAce')).toBeInTheDocument()
    })
    const rows = screen.getAllByRole('row')
    expect(within(rows[1]).getByText('LocalAce')).toBeInTheDocument()
    expect(within(rows[2]).getByText('RemoteRival')).toBeInTheDocument()
    expect(within(rows[3]).getByText('LocalVet')).toBeInTheDocument()
  })

  it('affiche le CSR et le tier dérivé', async () => {
    mockLeaderboard(ENTRIES)
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => {
      expect(screen.getByText('Onyx')).toBeInTheDocument()
    })
    expect(screen.getByText(/1\s?850/)).toBeInTheDocument()
    expect(screen.getByText(/1\s?720/)).toBeInTheDocument()
    expect(screen.getByText(/1\s?600/)).toBeInTheDocument()
    expect(screen.getByText('Diamond VI')).toBeInTheDocument()
    expect(screen.getByText('Diamond III')).toBeInTheDocument()
  })

  it('affiche un état vide si le classement est vide', async () => {
    mockLeaderboard([])
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => {
      expect(screen.getByText('Classement vide')).toBeInTheDocument()
    })
  })

  it('affiche une erreur en cas de réponse serveur 500', async () => {
    server.use(
      http.get(p('/players/:playerSlug/pages/leaderboard'), () =>
        HttpResponse.json({ error: 'internal' }, { status: 500 }),
      ),
    )
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => {
      expect(screen.getByText('Erreur de chargement')).toBeInTheDocument()
    })
  })
})
