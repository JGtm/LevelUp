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

  // Phase 6 du plan pipeline CSR : seuil dynamique placement_total.
  it('affiche "En placement (3/5)" pour la saison S13 (placement_total=5)', () => {
    renderWithProviders(
      <HomeRecentPlaylistsCard
        recentPlaylistRanks={[
          {
            playlist_name: 'Assassin classé',
            is_ranked: true,
            rating_type: 'CSR',
            rating_value: null,
            tier_label: null,
            badge_image_url: '/static/ranks/halo_infinite/unranked_6.png',
            measurement_matches_remaining: 2,
            placement_total: 5, // S3+
          },
        ]}
      />,
    )

    expect(screen.getByTestId('home-rank-unranked-label')).toHaveTextContent('En placement (3/5)')
    const img = screen.getByTestId('home-rank-unranked-image') as HTMLImageElement
    expect(img.getAttribute('src')).toBe('/static/ranks/halo_infinite/unranked_6.png')
  })

  it('affiche "En placement (4/10)" pour la saison historique S2 (placement_total=10)', () => {
    renderWithProviders(
      <HomeRecentPlaylistsCard
        recentPlaylistRanks={[
          {
            playlist_name: 'Ranked Arena (S2 archive)',
            is_ranked: true,
            rating_type: 'CSR',
            rating_value: null,
            tier_label: null,
            badge_image_url: '/static/ranks/halo_infinite/unranked_4.png',
            measurement_matches_remaining: 6,
            placement_total: 10,
          },
        ]}
      />,
    )

    expect(screen.getByTestId('home-rank-unranked-label')).toHaveTextContent('En placement (4/10)')
  })

  it('fallback à placement_total=10 quand le backend ne fournit pas le champ (back-compat legacy)', () => {
    renderWithProviders(
      <HomeRecentPlaylistsCard
        recentPlaylistRanks={[
          {
            playlist_name: 'Ranked Arena',
            is_ranked: true,
            rating_type: 'CSR',
            rating_value: null,
            tier_label: null,
            badge_image_url: '/static/ranks/halo_infinite/unranked_3.png',
            measurement_matches_remaining: 7,
            // placement_total absent
          },
        ]}
      />,
    )

    expect(screen.getByTestId('home-rank-unranked-label')).toHaveTextContent('En placement (3/10)')
  })

  // GH2-A3 : une playlist non résolue côté backend (asset_translations sans entrée
  // pour la locale) retombe sur le playlist_id brut = un UUID. Il ne doit JAMAIS
  // s'afficher tel quel — libellé neutre localisé à la place.
  it("n'affiche jamais un UUID brut de playlist — libellé neutre à la place (GH2-A3)", () => {
    const uuid = '96f32b0a-f89b-4507-83b1-bc07dd458dfa'
    renderWithProviders(
      <HomeRecentPlaylistsCard
        recentPlaylistRanks={[
          {
            playlist_name: uuid,
            is_ranked: false,
            rating_type: 'LUSR',
            rating_value: 1570,
            tier_label: 'Or VI',
            badge_image_url: '/static/ranks/halo_infinite/120px-HINF-CSR_Gold6.png',
            tier_progress_pct: 100,
            next_tier_label: 'Platine I',
          },
        ]}
      />,
    )

    // Le UUID ne doit JAMAIS apparaître à l'écran.
    expect(screen.queryByText(uuid)).not.toBeInTheDocument()
    // Libellé neutre localisé (FR par défaut dans les tests).
    expect(screen.getByTestId('home-recent-playlist-name')).toHaveTextContent('Sélection inconnue')
  })
})
