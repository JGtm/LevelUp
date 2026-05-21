import { describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { HomeRecentPlaylistsCard } from './HomeRecentPlaylistsCard'

describe('HomeRecentPlaylistsCard', () => {
  it('affiche le badge unranked_N.png fourni par le backend pour les placements et un fallback neutre pour les non classées', () => {
    renderWithProviders(
      <HomeRecentPlaylistsCard
        recentPlaylistRanks={[
          {
            playlist_name: 'Ranked Slayer',
            is_ranked: true,
            rating_type: 'CSR',
            rating_value: null,
            tier_label: null,
            // Backend émet désormais unranked_N.png pour les playlists classées en placement.
            badge_image_url: '/static/ranks/halo_infinite/unranked_4.png',
            measurement_matches_remaining: 6,
          },
          {
            playlist_name: 'Quick Play',
            is_ranked: false,
            rating_type: null,
            rating_value: null,
            tier_label: null,
            badge_image_url: null,
          },
        ]}
      />,
    )

    expect(screen.getByText('Ranked Slayer')).toBeInTheDocument()
    expect(screen.getByText('Quick Play')).toBeInTheDocument()
    expect(screen.getByTestId('home-rank-unranked-label')).toHaveTextContent('En placement (4/10)')
    const unrankedImg = screen.getByTestId('home-rank-unranked-image') as HTMLImageElement
    expect(unrankedImg.getAttribute('src')).toBe('/static/ranks/halo_infinite/unranked_4.png')
    expect(screen.getByText('Sans classement')).toBeInTheDocument()
    expect(screen.getAllByTestId('home-rank-neutral-placeholder')).toHaveLength(1)
  })

  it('affiche "En placement" sans progression quand le backend ne fournit pas measurement_matches_remaining', () => {
    renderWithProviders(
      <HomeRecentPlaylistsCard
        recentPlaylistRanks={[
          {
            playlist_name: 'Ranked Doubles',
            is_ranked: true,
            rating_type: 'CSR',
            rating_value: null,
            tier_label: null,
            badge_image_url: '/static/ranks/halo_infinite/unranked_0.png',
          },
        ]}
      />,
    )

    expect(screen.getByTestId('home-rank-unranked-label')).toHaveTextContent('En placement')
    expect(screen.getByTestId('home-rank-unranked-label').textContent).not.toMatch(/\d+\/10/)
  })
})
