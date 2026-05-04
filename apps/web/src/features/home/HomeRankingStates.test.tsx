import { describe, expect, it, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import type { ReactNode } from 'react'
import type { HomePageResponse } from '@/lib/api/types'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { HomePage } from './HomePage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
    Link: ({ children }: { children?: ReactNode }) => <a>{children}</a>,
  }
})

function buildHomeResponse(): HomePageResponse {
  return {
    hero: {
      player_name: 'TestPlayer',
      kpis: {
        win_rate: 0.55,
        global_ratio: 1.2,
        avg_kda: 1.4,
        avg_accuracy: 42,
        total_matches: 120,
        wins: 66,
        draws: 0,
        dnfs: 0,
        losses: 54,
        total_playtime_secs: 36000,
        favorite_weapon_name: '',
        favorite_weapon_kills: 0,
        favorite_playlist_name: '',
        favorite_playlist_count: 0,
        avg_offensive_conversion: null,
        avg_defensive_resistance: null,
      },
      trend: null,
    },
    spartan_identity: {
      spartan_id: 'JGTM',
      banner_image_url: null,
      emblem_image_url: null,
      backdrop_image_url: null,
      highest_csr: null,
      highest_lusr: null,
      career_rank: null,
    },
    highlights: [],
    recent_matches: [],
    favorite_matches: [],
    recent_media: [],
    solo_session: null,
    squad_session: null,
    has_ranked_history: false,
    has_unranked_history: false,
    recent_playlist_ranks: [],
  }
}

describe('Home ranking states', () => {
  it('affiche le placement CSR et le neutre LUSR quand seul le classé existe', async () => {
    const response = buildHomeResponse()
    response.has_ranked_history = true

    server.use(
      http.get('/api/v1/players/:playerSlug/pages/home', () => HttpResponse.json(response)),
    )

    renderWithProviders(<HomePage />)

    await waitFor(() => {
      expect(screen.getByTestId('home-highest-csr-unranked')).toBeInTheDocument()
    })

    expect(screen.getByTestId('home-highest-csr-detail')).toHaveTextContent('En placement')
    expect(screen.getByTestId('home-highest-lusr-detail')).toHaveTextContent('Aucune partie non classée')
  })

  it('affiche un état vide explicite quand aucun classement n’est disponible', async () => {
    const response = buildHomeResponse()

    server.use(
      http.get('/api/v1/players/:playerSlug/pages/home', () => HttpResponse.json(response)),
    )

    renderWithProviders(<HomePage />)

    await waitFor(() => {
      expect(screen.getByTestId('home-skill-peaks-empty')).toBeInTheDocument()
    })

    expect(screen.getByText('Aucun classement disponible')).toBeInTheDocument()
  })

  it('affiche un état indisponible si la privacy rend l’historique incomplet', async () => {
    const response = buildHomeResponse()
    response.privacy_warning = {
      level: 'partial',
      message: 'Historique partiel.',
    }

    server.use(
      http.get('/api/v1/players/:playerSlug/pages/home', () => HttpResponse.json(response)),
    )

    renderWithProviders(<HomePage />)

    await waitFor(() => {
      expect(screen.getByTestId('home-skill-peaks-empty')).toBeInTheDocument()
    })

    expect(screen.getByText('Classements indisponibles')).toBeInTheDocument()
  })
})
