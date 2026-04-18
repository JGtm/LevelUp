/**
 * Tests composant — CompareDrawer (Sprint 54 F6).
 *
 * Vérifie : skeleton pendant fetch, états limites (absent, privé, identique), KPIs cohérents.
 */
import { describe, it, expect, vi } from 'vitest'
import { screen, waitFor, fireEvent } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { CompareDrawer } from './CompareDrawer'

// CompareDrawer n'utilise pas le router directement — pas besoin de mock routeur.

const makePlayer = (gamertag: string, isLocal: boolean) => ({
  title_slug: 'halo_infinite',
  xuid: isLocal ? 'xuid-a' : 'xuid-b',
  gamertag,
  matches_played: isLocal ? 120 : 80,
  win_rate: isLocal ? 0.58 : 0.50,
  kd_ratio: isLocal ? 1.5 : 1.2,
  accuracy: isLocal ? 0.42 : 0.38,
  avg_kills: isLocal ? 12.3 : 10.1,
  avg_deaths: isLocal ? 8.2 : 8.5,
  avg_assists: isLocal ? 4.1 : 3.2,
  avg_damage: isLocal ? 3200 : 2900,
  avg_ps_per_min: 0,
  avg_playtime_min: 0,
  avg_performance_score: 0,
  is_local: isLocal,
})

const baseCompareResponse = {
  player_a: makePlayer('LocalPlayer', true),
  player_b: makePlayer('RemoteRival', false),
  metrics: [
    { metric: 'win_rate', label_fr: 'Taux de victoire', value_a: 58, value_b: 50, delta: -8, winner: 'a' },
    { metric: 'kdr', label_fr: 'K/D', value_a: 1.5, value_b: 1.2, delta: -0.3, winner: 'a' },
  ],
  title_slug: 'halo_infinite',
  privacy_warning: null,
  player_b_partial: false,
}

function renderDrawer(open = true) {
  return renderWithProviders(
    <CompareDrawer playerSlug="test-player" open={open} onClose={vi.fn()} />,
  )
}

describe('CompareDrawer', () => {
  it('ne monte rien quand open=false', () => {
    const { container } = renderWithProviders(
      <CompareDrawer playerSlug="test-player" open={false} onClose={vi.fn()} />,
    )
    expect(container.innerHTML).toBe('')
  })

  it('affiche le formulaire de recherche à l\'ouverture', () => {
    renderDrawer()
    expect(screen.getByText('Comparer avec un joueur')).toBeInTheDocument()
    expect(screen.getByPlaceholderText(/HaloPlayer123/i)).toBeInTheDocument()
    expect(screen.getByText('Entrez un gamertag pour lancer la comparaison.')).toBeInTheDocument()
  })

  it('affiche un spinner pendant le fetch', async () => {
    // Handler qui ne répond jamais (on vérifie le spinner pendant le fetch)
    server.use(
      http.post('/api/v1/players/:playerSlug/pages/compare', async () => {
        await new Promise(() => {}) // ne résout jamais
      }),
    )

    renderDrawer()
    fireEvent.change(screen.getByPlaceholderText(/HaloPlayer123/i), {
      target: { value: 'RemoteRival' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Comparer' }))

    // Le spinner doit apparaître pendant le chargement
    await waitFor(() => {
      expect(screen.getByRole('button', { name: '…' })).toBeInTheDocument()
    })
  })

  it('affiche les KPIs des deux joueurs après un fetch réussi', async () => {
    server.use(
      http.post('/api/v1/players/:playerSlug/pages/compare', () =>
        HttpResponse.json(baseCompareResponse),
      ),
    )

    renderDrawer()
    fireEvent.change(screen.getByPlaceholderText(/HaloPlayer123/i), {
      target: { value: 'RemoteRival' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Comparer' }))

    await waitFor(() => {
      // Les deux gamertags sont visibles (peuvent apparaître dans plusieurs éléments)
      expect(screen.getAllByText(/LocalPlayer/i).length).toBeGreaterThan(0)
      expect(screen.getAllByText(/RemoteRival/i).length).toBeGreaterThan(0)
      // Métriques affichées
      expect(screen.getByText('Taux de victoire')).toBeInTheDocument()
      expect(screen.getByText('K/D')).toBeInTheDocument()
    })
  })

  it('affiche "joueur introuvable" sur une 404', async () => {
    server.use(
      http.post('/api/v1/players/:playerSlug/pages/compare', () =>
        HttpResponse.json({ code: 'not_found', message: '404: joueur introuvable' }, { status: 404 }),
      ),
    )

    renderDrawer()
    fireEvent.change(screen.getByPlaceholderText(/HaloPlayer123/i), {
      target: { value: 'InexistantXXX' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Comparer' }))

    await waitFor(() => {
      expect(screen.getByText('Joueur introuvable')).toBeInTheDocument()
    })
  })

  it('affiche une erreur générique sur une 500', async () => {
    server.use(
      http.post('/api/v1/players/:playerSlug/pages/compare', () =>
        HttpResponse.json({ code: 'compare_error', message: 'internal server error' }, { status: 500 }),
      ),
    )

    renderDrawer()
    fireEvent.change(screen.getByPlaceholderText(/HaloPlayer123/i), {
      target: { value: 'BrokenPlayer' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Comparer' }))

    await waitFor(() => {
      expect(screen.getByText('Erreur')).toBeInTheDocument()
    })
  })

  it('affiche le bandeau privacy_warning si présent', async () => {
    server.use(
      http.post('/api/v1/players/:playerSlug/pages/compare', () =>
        HttpResponse.json({
          ...baseCompareResponse,
          privacy_warning: {
            level: 'partial',
            message: 'Les données de ce joueur sont limitées par ses paramètres de confidentialité.',
          },
        }),
      ),
    )

    renderDrawer()
    fireEvent.change(screen.getByPlaceholderText(/HaloPlayer123/i), {
      target: { value: 'PrivatePlayer' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Comparer' }))

    await waitFor(() => {
      expect(screen.getByText(/confidentialité/i)).toBeInTheDocument()
    })
  })

  it('affiche le warning données partielles si player_b_partial=true sans privacy_warning', async () => {
    server.use(
      http.post('/api/v1/players/:playerSlug/pages/compare', () =>
        HttpResponse.json({
          ...baseCompareResponse,
          player_b_partial: true,
          privacy_warning: null,
        }),
      ),
    )

    renderDrawer()
    fireEvent.change(screen.getByPlaceholderText(/HaloPlayer123/i), {
      target: { value: 'PartialPlayer' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Comparer' }))

    await waitFor(() => {
      expect(screen.getByText(/données.*partielles/i)).toBeInTheDocument()
    })
  })

  it('le bouton Comparer est désactivé si le champ est vide', () => {
    renderDrawer()
    const btn = screen.getByRole('button', { name: 'Comparer' })
    expect(btn).toBeDisabled()
  })

  it('le bouton fermer appelle onClose', () => {
    const onClose = vi.fn()
    renderWithProviders(
      <CompareDrawer playerSlug="test-player" open={true} onClose={onClose} />,
    )
    fireEvent.click(screen.getByLabelText('Fermer'))
    expect(onClose).toHaveBeenCalledOnce()
  })
})
