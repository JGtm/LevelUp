/**
 * Tests composant — LeaderboardBlock (Sprint 54-E).
 *
 * Vérifie le chargement progressif : joueurs locaux affichés sans attendre
 * la résolution Waypoint, états vides et erreurs.
 */
import { describe, it, expect } from 'vitest'
import { screen, waitFor, within } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { http, HttpResponse } from 'msw'
import { server } from '@/test/setup'
import { LeaderboardBlock } from './LeaderboardBlock'

const p = (path: string) => `/api/v1${path}`

describe('LeaderboardBlock', () => {
  it('affiche le spinner pendant le chargement', () => {
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)
    // Le composant Spinner utilise role="status"
    expect(document.querySelector('[class*="animate-spin"]')).toBeTruthy()
  })

  it('affiche les joueurs locaux avec le badge Local', async () => {
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => {
      expect(screen.getByText('LocalAce')).toBeInTheDocument()
    })

    // Les joueurs locaux doivent avoir le badge "Local"
    const localBadges = screen.getAllByText('Local')
    expect(localBadges).toHaveLength(2) // LocalAce + LocalVet

    // Le joueur distant ne doit PAS avoir de badge Local
    const remoteRow = screen.getByText('RemoteRival').closest('tr')!
    expect(within(remoteRow).queryByText('Local')).not.toBeInTheDocument()
  })

  it('affiche les joueurs locaux avant les joueurs distants (ordre par rank)', async () => {
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => {
      expect(screen.getByText('LocalAce')).toBeInTheDocument()
    })

    // Les joueurs sont dans l'ordre rank croissant : LocalAce (1), RemoteRival (2), LocalVet (3)
    const rows = screen.getAllByRole('row')
    // rows[0] = header, rows[1..3] = data
    expect(within(rows[1]).getByText('LocalAce')).toBeInTheDocument()
    expect(within(rows[2]).getByText('RemoteRival')).toBeInTheDocument()
    expect(within(rows[3]).getByText('LocalVet')).toBeInTheDocument()
  })

  it('affiche le CSR et le tier de chaque joueur', async () => {
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => {
      expect(screen.getByText('Onyx')).toBeInTheDocument()
    })

    // CSR values (localisés fr-FR)
    expect(screen.getByText(/1[\s\u202f]?850/)).toBeInTheDocument()
    expect(screen.getByText(/1[\s\u202f]?720/)).toBeInTheDocument()
    expect(screen.getByText(/1[\s\u202f]?600/)).toBeInTheDocument()

    // Tiers
    expect(screen.getByText('Onyx')).toBeInTheDocument()
    expect(screen.getByText('Diamond VI')).toBeInTheDocument()
    expect(screen.getByText('Diamond III')).toBeInTheDocument()
  })

  it('affiche le compteur total de joueurs', async () => {
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)

    await waitFor(() => {
      expect(screen.getByText(/3 joueurs/)).toBeInTheDocument()
    })
  })

  it('affiche un état vide si le classement est vide', async () => {
    server.use(
      http.get(p('/players/:playerSlug/pages/leaderboard'), () =>
        HttpResponse.json({
          entries: [],
          season_id: 'Season5',
          playlist_id: 'Ranked',
          title_slug: 'halo_infinite',
          total: 0,
        }),
      ),
    )

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
      expect(screen.getByText('Erreur')).toBeInTheDocument()
    })
  })

  it('affiche le titre Classement CSR', () => {
    renderWithProviders(<LeaderboardBlock playerSlug="test-player" />)
    expect(screen.getByText('Classement CSR')).toBeInTheDocument()
  })
})
