import { describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { HomeRecentPlaylistsCard } from './HomeRecentPlaylistsCard'

describe('HomeRecentPlaylistsCard', () => {
  it('réserve Unranked aux playlists classées en placement et garde un fallback neutre ailleurs', () => {
    renderWithProviders(
      <HomeRecentPlaylistsCard
        recentPlaylistRanks={[
          {
            playlist_name: 'Ranked Slayer',
            is_ranked: true,
            rating_type: 'CSR',
            rating_value: null,
            tier_label: null,
            badge_image_url: null,
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
    expect(screen.getByTestId('home-rank-unranked-label')).toHaveTextContent('En placement')
    expect(screen.getAllByTestId('home-rank-unranked-image')).toHaveLength(1)
    expect(screen.getByText('Sans classement')).toBeInTheDocument()
    expect(screen.getAllByTestId('home-rank-neutral-placeholder')).toHaveLength(1)
  })
})
