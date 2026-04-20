/**
 * Tests composant — HomePage (Slice 5).
 *
 * Smoke : monte, spinner, puis Hero KPIs affichés depuis MSW.
 */
import { describe, it, expect, vi } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import type { ReactNode } from 'react'
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

describe('HomePage', () => {
  it('monte sans erreur', () => {
    const { container } = renderWithProviders(<HomePage />)
    expect(container).toBeTruthy()
  })

  it("affiche le spinner pendant le chargement", () => {
    renderWithProviders(<HomePage />)
    expect(screen.getByText(/Chargement de l'accueil/i)).toBeInTheDocument()
  })

  it('affiche la section Performance globale après chargement', async () => {
    renderWithProviders(<HomePage />)
    await waitFor(() => {
      expect(screen.getByText(/Performance globale/i)).toBeInTheDocument()
    })
  })

  it('affiche les défis actifs détaillés triés du plus avancé au moins avancé', async () => {
    server.use(
      http.get('/api/v1/players/:playerSlug/challenges', () => HttpResponse.json({
        available: true,
        total: 3,
        completed: 0,
        xp_available: 4500,
        next_expiry: '2026-04-20T18:00:00Z',
        items: [
          {
            challenge_path: 'challenge/not-started',
            tracking_id: 'c3',
            title: 'Défi pas commencé',
            description: 'Commence ce défi.',
            image_url: 'https://example.com/challenge-3.png',
            progress_current: 0,
            progress_target: 5,
            progress_percent: 0,
          },
          {
            challenge_path: 'ChallengeContent/ClientChallengeDefinitions/WeeklyChallenges/Action/ch1.json',
            tracking_id: 'c1',
            title: 'Défi avancé',
            description: 'Presque terminé.',
            image_url: 'https://example.com/challenge-1.png',
            progress_current: 7,
            progress_target: 10,
            progress_percent: 70,
          },
          {
            challenge_path: 'ChallengeContent/ClientChallengeDefinitions/DailyChallenges/ch2.json',
            tracking_id: 'c2',
            title: 'Défi en cours',
            description: 'Continue la progression.',
            image_url: 'https://example.com/challenge-2.png',
            progress_current: 1,
            progress_target: 3,
            progress_percent: 33.3,
          },
        ],
        error_hint: null,
      })),
    )

    renderWithProviders(<HomePage />)

    await waitFor(() => {
      expect(screen.getByText(/Défis actifs/i)).toBeInTheDocument()
      expect(
        screen.getByText((content) => content.replace(/\s+/g, ' ').includes('4 500 XP disponibles')),
      ).toBeInTheDocument()
    })

    const titles = screen.getAllByTestId('home-challenge-title')
    expect(titles.map((node) => node.textContent)).toEqual([
      'Défi avancé',
      'Défi en cours',
      'Défi pas commencé',
    ])

    const kinds = screen.getAllByTestId('home-challenge-kind')
    expect(kinds.map((node) => node.textContent)).toEqual(['Hebdo', 'Quotidien'])
    expect(kinds[0]).toHaveClass('bg-sky-500/14')
    expect(kinds[1]).toHaveClass('bg-amber-500/14')

    expect(screen.getByText('Défi avancé')).toHaveClass('font-semibold')
    expect(screen.getByText('Presque terminé.')).toHaveClass('italic')

    const fills = screen.getAllByTestId('home-challenge-progress-fill')
    expect(fills[0]).toHaveStyle({ width: '70%' })
    expect(fills[2]).toHaveStyle({ width: '0%' })
  })

  it('affiche les KPIs globaux (Parties, Taux de victoire, K/D)', async () => {
    renderWithProviders(<HomePage />)
    await waitFor(() => {
      expect(screen.getByText('Parties')).toBeInTheDocument()
      expect(screen.getByText('Taux de victoire')).toBeInTheDocument()
      expect(screen.getByText('K/D')).toBeInTheDocument()
    })
  })

  it("n'affiche pas de warning privacy quand la payload n'en contient pas", async () => {
    renderWithProviders(<HomePage />)
    await waitFor(() => {
      expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    })
  })

  it('affiche des messages explicites pour les sections vides', async () => {
    renderWithProviders(<HomePage />)
    await waitFor(() => {
      expect(screen.getByText(/Aucune session récente disponible/i)).toBeInTheDocument()
      expect(screen.getByText(/Aucun point saillant disponible/i)).toBeInTheDocument()
      expect(screen.getByText(/Aucun média récent disponible/i)).toBeInTheDocument()
    })
  })

  it('affiche la bannière visuelle en tête de la home', async () => {
    renderWithProviders(<HomePage />)

    await waitFor(() => {
      const stickyShell = screen.getByTestId('home-hero-banner-sticky')
      const banner = screen.getByTestId('home-hero-banner')
      const image = banner.querySelector('img')

      expect(stickyShell).toHaveClass('sticky', 'top-0')
      expect(banner).toBeInTheDocument()
      expect(banner).not.toHaveClass('sticky', 'top-0')
      expect(image).not.toBeNull()
      expect(image).toHaveClass('h-36', 'sm:h-48', 'lg:h-56')
    })
  })

  it('garde la bannière avant le warning privacy quand il existe', async () => {
    server.use(
      http.get('/api/v1/players/:playerSlug/pages/home', () => HttpResponse.json({
        hero: {
          player_name: 'TestPlayer',
          kpis: {
            win_rate: 55.0,
            global_ratio: 1.2,
            avg_accuracy: 42.0,
            total_matches: 120,
            wins: 66,
            losses: 54,
          },
          trend: null,
        },
        highlights: [],
        recent_matches: [],
        recent_media: [],
        solo_session: null,
        squad_session: null,
        privacy_warning: {
          level: 'partial',
          message: 'Certaines données Halo ne sont pas accessibles.',
        },
      })),
    )

    renderWithProviders(<HomePage />)

    await waitFor(() => {
      const banner = screen.getByTestId('home-hero-banner')
      const alert = screen.getByRole('alert')

      expect(
        banner.compareDocumentPosition(alert) & Node.DOCUMENT_POSITION_FOLLOWING,
      ).toBeTruthy()
    })
  })

  it('affiche un rail média enrichi et ouvre la lightbox au clic', async () => {
    server.use(
      http.post('/api/v1/players/:playerSlug/pages/media', () => HttpResponse.json({
        items: {
          items: [
            {
              basename: 'clip-epic.mp4',
              file_path: '/media/clip-epic.mp4',
              kind: 'clip',
              thumbnail_path: '/media/thumb-clip-epic.jpg',
              match_id: 'match-123',
              capture_end_utc: '2026-04-18T12:00:00Z',
              match_start_time: '2026-04-18T11:55:00Z',
              section: 'match',
              owner_gamertag: 'TestPlayer',
              map_name: 'Live Fire',
            },
          ],
          pagination: { page: 1, page_size: 4, total: 1, total_pages: 1, has_next: false, has_prev: false },
          freshness: null,
        },
        total_mine: 1,
        total_teammates: 0,
        total_unassigned: 0,
      })),
    )

    renderWithProviders(<HomePage />)

    await waitFor(() => {
      expect(screen.getByText('clip-epic.mp4')).toBeInTheDocument()
      expect(screen.getByText(/Aperçu au survol/i)).toBeInTheDocument()
    })

    fireEvent.click(screen.getByText('clip-epic.mp4'))

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Fermer/i })).toBeInTheDocument()
      expect(screen.getByText(/Voir le match/i)).toBeInTheDocument()
    })
  })

  it('affiche un message clair si la payload home est absente', async () => {
    server.use(
      http.get('/api/v1/players/:playerSlug/pages/home', () => HttpResponse.json(null)),
    )

    renderWithProviders(<HomePage />)

    await waitFor(() => {
      expect(screen.getByText(/Accueil vide/i)).toBeInTheDocument()
    })
  })
})
