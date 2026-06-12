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

  it('masque les colonnes enrichies tant qu’aucun joueur n’est backfillé', async () => {
    mockLeaderboard(ENTRIES) // entrées sans champs d'enrichissement
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => {
      expect(screen.getByText('LocalAce')).toBeInTheDocument()
    })
    // L'en-tête "Victoires" (col_win_rate) ne doit pas apparaître.
    expect(screen.queryByText('Victoires')).not.toBeInTheDocument()
    expect(screen.queryByText('Δ rang')).not.toBeInTheDocument()
  })

  it('affiche les colonnes enrichies (win rate, KDA, Δ rang) quand le joueur est backfillé', async () => {
    const enriched = [
      {
        // kda/accuracy sont des SOMMES brutes → affichées en moyenne (÷ match_count).
        rank: 1, xuid: 'x1', gamertag: 'Ace', csr_value: 1850, tier: 'Onyx', sub_tier: 0, is_local: false,
        match_count: 20, win_rate: 0.65, kda: 36, accuracy: 1050, win_rate_trend: 'up', kda_trend: 'down', rank_delta: 3,
        kills: 100, deaths: 80, assists: 30, damage_dealt: 50000, damage_taken: 48000,
      },
      {
        rank: 2, xuid: 'x2', gamertag: 'Rival', csr_value: 1700, tier: 'Diamond', sub_tier: 6, is_local: false,
        match_count: 15, win_rate: 0.4, kda: 16.5, accuracy: 600, win_rate_trend: 'stable', kda_trend: 'stable', rank_delta: -2,
        kills: 60, deaths: 70, assists: 12, damage_dealt: 30000, damage_taken: 35000,
      },
    ]
    mockLeaderboard(enriched as unknown as typeof ENTRIES)
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => {
      expect(screen.getByText('Ace')).toBeInTheDocument()
    })
    // En-têtes enrichis présents.
    expect(screen.getByText('Victoires')).toBeInTheDocument()
    expect(screen.getByText('Δ rang')).toBeInTheDocument()

    // Valeurs de la ligne Ace : win rate FR, KDA moyen (36/20=1.80), précision
    // moyenne (1050/20=52,5%), delta signé, nb matchs.
    const aceRow = screen.getByText('Ace').closest('tr')!
    expect(within(aceRow).getByText('65,0%')).toBeInTheDocument()
    expect(within(aceRow).getByText('1.80')).toBeInTheDocument()
    expect(within(aceRow).getByText('52,5%')).toBeInTheDocument()
    expect(within(aceRow).getByText('+3')).toBeInTheDocument()
    expect(within(aceRow).getByText('20')).toBeInTheDocument()

    // Delta négatif sur la ligne Rival.
    const rivalRow = screen.getByText('Rival').closest('tr')!
    expect(within(rivalRow).getByText('-2')).toBeInTheDocument()
  })
})
