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
  matches: isLocal ? 120 : 80,
  win_rate: isLocal ? 0.58 : 0.50,
  kda: isLocal ? 2.1 : 1.7,
  kdr: isLocal ? 1.5 : 1.2,
  kills_per_game: isLocal ? 12.3 : 10.1,
  deaths_per_game: isLocal ? 8.2 : 8.5,
  assists_per_game: isLocal ? 4.1 : 3.2,
  accuracy: isLocal ? 0.42 : 0.38,
  damage_per_game: isLocal ? 3200 : 2900,
  career_rank: isLocal ? 82 : 64,
  csr_current: isLocal ? 1620 : 1485,
  csr_best: isLocal ? 1705 : 1510,
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
    expect(screen.getByText('Face-à-face avec un joueur')).toBeInTheDocument()
    expect(screen.getByPlaceholderText(/HaloPlayer123/i)).toBeInTheDocument()
    expect(screen.getByText('Entrez un gamertag pour lancer le face-à-face.')).toBeInTheDocument()
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
    fireEvent.click(screen.getByRole('button', { name: 'Face-à-face' }))

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
    fireEvent.click(screen.getByRole('button', { name: 'Face-à-face' }))

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
    fireEvent.click(screen.getByRole('button', { name: 'Face-à-face' }))

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
    fireEvent.click(screen.getByRole('button', { name: 'Face-à-face' }))

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
    fireEvent.click(screen.getByRole('button', { name: 'Face-à-face' }))

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
    fireEvent.click(screen.getByRole('button', { name: 'Face-à-face' }))

    await waitFor(() => {
      expect(screen.getByText(/données.*partielles/i)).toBeInTheDocument()
    })
  })

  it('le bouton Face-à-face est désactivé si le champ est vide', () => {
    renderDrawer()
    const btn = screen.getByRole('button', { name: 'Face-à-face' })
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
