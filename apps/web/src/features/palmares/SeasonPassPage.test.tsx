import { describe, expect, it, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import type { ReactNode } from 'react'

import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'

import { SeasonPassPage } from './SeasonPassPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useParams: () => ({ playerSlug: 'test-player' }),
    Link: ({ children }: { children?: ReactNode }) => <a>{children}</a>,
  }
})

describe('SeasonPassPage', () => {
  it('affiche le hero actif, le carrousel de paliers et la progression du palier courant', async () => {
    const originalScrollTo = HTMLElement.prototype.scrollTo
    const originalScrollIntoView = HTMLElement.prototype.scrollIntoView
    const scrollToMock = vi.fn()
    const scrollIntoViewMock = vi.fn()
    Object.defineProperty(HTMLElement.prototype, 'scrollTo', {
      configurable: true,
      writable: true,
      value: scrollToMock,
    })
    Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', {
      configurable: true,
      writable: true,
      value: scrollIntoViewMock,
    })

    try {
    server.use(
      http.get('/api/v1/players/:playerSlug/pages/palmares/season-pass', () => HttpResponse.json({
        title_slug: 'halo_infinite',
        available: true,
        active_track_path: 'RewardTracks/TrackA',
        challenges: {
          available: true,
          total: 4,
          completed: 1,
          xp_available: 2000,
          next_expiry: null,
          items: [],
          error_hint: null,
        },
        passes: [
          {
            reward_track_path: 'RewardTracks/TrackA',
            name: 'Operation Alpha',
            description: 'Escalade principale',
            status: 'active',
            is_active: true,
            is_owned: true,
            premium_owned: true,
            has_reached_max_rank: false,
            current_rank: 12,
            partial_progress: 300,
            xp_per_rank: 1000,
            max_rank: 20,
            completion_percent: 60,
            active_tier_rank: 13,
            active_tier_progress_percent: 30,
            image_url: 'https://example.com/track-a.png',
            background_image_url: 'https://example.com/bg-a.png',
            tiers: [
              {
                rank: 12,
                title: 'Récompense 12',
                description: null,
                image_url: 'https://example.com/tier-12.png',
                is_obtained: true,
                is_current: false,
                is_premium: false,
              },
              {
                rank: 13,
                title: 'Récompense 13',
                description: 'Effet rare',
                image_url: 'https://example.com/tier-13.png',
                is_obtained: false,
                is_current: true,
                is_premium: true,
              },
              {
                rank: 14,
                title: 'Récompense 14',
                description: null,
                image_url: null,
                is_obtained: false,
                is_current: false,
                is_premium: false,
              },
            ],
          },
          {
            reward_track_path: 'RewardTracks/TrackB',
            name: 'Operation Beta',
            description: null,
            status: 'completed',
            is_active: false,
            is_owned: true,
            premium_owned: true,
            has_reached_max_rank: true,
            current_rank: 20,
            partial_progress: 0,
            xp_per_rank: 1000,
            max_rank: 20,
            completion_percent: 100,
            active_tier_rank: 20,
            active_tier_progress_percent: 100,
            image_url: 'https://example.com/track-b.png',
            background_image_url: 'https://example.com/bg-b.png',
            tiers: [],
          },
        ],
      })),
    )

    renderWithProviders(<SeasonPassPage />)

    await waitFor(() => {
      expect(screen.getAllByText('Operation Alpha').length).toBeGreaterThan(0)
      // Refacto 2026-05 (commit 83295929 "hero overlay allégé") : la description
      // textuelle du pass ("Escalade principale") n'est plus rendue sur le hero.
      // Elle reste exposée dans la lightbox d'une récompense (cf. card.description
      // mappé dans allRewards), pas sur la vue d'accueil.
      expect(screen.getAllByText('Récompense 13').length).toBeGreaterThan(0)
    })

    expect(scrollToMock).toHaveBeenCalledWith(expect.objectContaining({ behavior: 'smooth' }))
    expect(scrollIntoViewMock).not.toHaveBeenCalled()

    expect(screen.getByAltText('Operation Alpha')).toBeInTheDocument()

    const tierCards = screen.getAllByTestId('battle-pass-tier-card')
    expect(tierCards).toHaveLength(3)
    expect(tierCards[0]).toHaveAttribute('data-obtained', 'true')
    expect(tierCards[1]).toHaveAttribute('data-current', 'true')

    expect(screen.getByTestId('season-pass-active-tier-progress-fill')).toHaveStyle({ width: '30%' })
    expect(screen.getByTestId('season-pass-active-tier-progress-current')).toHaveTextContent('300 XP')
    expect(screen.getByTestId('season-pass-active-tier-progress-target')).toHaveTextContent('1 000 XP')
    expect(screen.getByText('Autres passes')).toBeInTheDocument()
    expect(screen.getByText('Operation Beta')).toBeInTheDocument()
    } finally {
      Object.defineProperty(HTMLElement.prototype, 'scrollTo', {
        configurable: true,
        writable: true,
        value: originalScrollTo,
      })
      Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', {
        configurable: true,
        writable: true,
        value: originalScrollIntoView,
      })
    }
  })
})
