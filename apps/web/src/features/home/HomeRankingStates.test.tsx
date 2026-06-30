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
      banner_image_url: undefined,
      emblem_image_url: undefined,
      backdrop_image_url: undefined,
      highest_csr: undefined,
      highest_lusr: undefined,
      career_rank: undefined,
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
    expect(screen.getByTestId('home-highest-lusr-detail')).toHaveTextContent('Non classé')
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

  it('affiche le placement CSR backend-driven via measurement_matches_remaining (mai 2026)', async () => {
    // Nouveau contrat : le backend renvoie un peak avec
    // measurement_matches_remaining > 0 et badge_image_url = unranked_N.png.
    // Le front ne devine plus via has_ranked_history.
    const response = buildHomeResponse()
    response.spartan_identity = {
      ...response.spartan_identity!,
      highest_csr: {
        rating_value: 0,
        tier_label: undefined,
        badge_image_url: '/static/ranks/halo_infinite/unranked_3.png',
        measurement_matches_remaining: 7,
      },
    }
    response.has_ranked_history = true

    server.use(
      http.get('/api/v1/players/:playerSlug/pages/home', () => HttpResponse.json(response)),
    )

    renderWithProviders(<HomePage />)

    await waitFor(() => {
      expect(screen.getByTestId('home-highest-csr-unranked')).toBeInTheDocument()
    })

    const badge = screen.getByTestId('home-highest-csr-unranked') as HTMLImageElement
    expect(badge.src).toContain('/static/ranks/halo_infinite/unranked_3.png')
    expect(screen.getByTestId('home-highest-csr-detail')).toHaveTextContent('En placement (3/10)')
    expect(screen.getByTestId('home-highest-csr-value')).toHaveTextContent('—')
  })

  it('affiche le placement LUSR backend-driven (régression bug "En placement faux")', async () => {
    // Avant mai 2026, LUSR ne pouvait JAMAIS être en placement (le front
    // passait mode='unranked' à resolveSkillPeakState → state='neutral').
    // Maintenant le LUSR peut être en placement (10 matchs par playlist_group).
    const response = buildHomeResponse()
    response.spartan_identity = {
      ...response.spartan_identity!,
      highest_lusr: {
        rating_value: 0,
        tier_label: undefined,
        badge_image_url: '/static/ranks/halo_infinite/unranked_6.png',
        measurement_matches_remaining: 4,
      },
    }
    response.has_unranked_history = true

    server.use(
      http.get('/api/v1/players/:playerSlug/pages/home', () => HttpResponse.json(response)),
    )

    renderWithProviders(<HomePage />)

    await waitFor(() => {
      expect(screen.getByTestId('home-highest-lusr-unranked')).toBeInTheDocument()
    })

    const badge = screen.getByTestId('home-highest-lusr-unranked') as HTMLImageElement
    expect(badge.src).toContain('/static/ranks/halo_infinite/unranked_6.png')
    expect(screen.getByTestId('home-highest-lusr-detail')).toHaveTextContent('En placement (6/10)')
  })

  // Phase 6 du plan pipeline CSR : seuil dynamique placement_total.
  it('affiche "En placement (3/5)" quand le backend renvoie placement_total=5 (saison S3+)', async () => {
    const response = buildHomeResponse()
    response.spartan_identity = {
      ...response.spartan_identity!,
      highest_csr: {
        rating_value: 0,
        tier_label: undefined,
        badge_image_url: '/static/ranks/halo_infinite/unranked_6.png',
        measurement_matches_remaining: 2,
        placement_total: 5, // S13 et postérieures
      },
    }
    response.has_ranked_history = true

    server.use(
      http.get('/api/v1/players/:playerSlug/pages/home', () => HttpResponse.json(response)),
    )

    renderWithProviders(<HomePage />)

    await waitFor(() => {
      expect(screen.getByTestId('home-highest-csr-detail')).toHaveTextContent('En placement (3/5)')
    })
    const badge = screen.getByTestId('home-highest-csr-unranked') as HTMLImageElement
    expect(badge.src).toContain('/static/ranks/halo_infinite/unranked_6.png')
  })

  it('affiche le rating + tier quand peak.measurement_matches_remaining=0 (matured)', async () => {
    const response = buildHomeResponse()
    response.spartan_identity = {
      ...response.spartan_identity!,
      highest_csr: {
        rating_value: 1450,
        tier_label: 'Onyx',
        badge_image_url: '/static/ranks/halo_infinite/120px-HINF-CSR_Onyx.png',
        measurement_matches_remaining: 0,
      },
    }
    response.has_ranked_history = true

    server.use(
      http.get('/api/v1/players/:playerSlug/pages/home', () => HttpResponse.json(response)),
    )

    renderWithProviders(<HomePage />)

    await waitFor(() => {
      expect(screen.getByTestId('home-highest-csr-badge')).toBeInTheDocument()
    })

    // Format fr-FR : espace insécable entre milliers (1 450, pas 1,450).
    expect(screen.getByTestId('home-highest-csr-value').textContent?.replace(/\s/g, '')).toBe('1450')
    expect(screen.getByTestId('home-highest-csr-tier')).toHaveTextContent('Onyx')
  })

  // Issue Halo 5 #2 : CSR « par paliers » sans valeur numérique (rating_value=0 mais tier
  // présent, matured) → on affiche le palier MAIS PAS le tiret « — » placeholder.
  it('CSR tier-only (Halo 5) : affiche le palier sans tiret « — »', async () => {
    const response = buildHomeResponse()
    response.spartan_identity = {
      ...response.spartan_identity!,
      highest_csr: {
        rating_value: 0,
        tier_label: 'Diamant 5',
        badge_image_url: '/static/ranks/halo_infinite/120px-HINF-CSR_Diamond5.png',
        measurement_matches_remaining: 0,
      },
    }
    response.has_ranked_history = true

    server.use(
      http.get('/api/v1/players/:playerSlug/pages/home', () => HttpResponse.json(response)),
    )

    renderWithProviders(<HomePage />)

    await waitFor(() => {
      expect(screen.getByTestId('home-highest-csr-tier')).toHaveTextContent('Diamant 5')
    })
    // La ligne de valeur (et son « — ») n'est PAS rendue quand le CSR est tier-only.
    expect(screen.queryByTestId('home-highest-csr-value')).toBeNull()
  })

  it('affiche la barre (sous-palier) + extrémités (rating gauche, sous-palier suivant droite) en CSR matured', async () => {
    const response = buildHomeResponse()
    response.spartan_identity = {
      ...response.spartan_identity!,
      highest_csr: {
        rating_value: 730,
        tier_label: 'Or III',
        badge_image_url: '/static/ranks/halo_infinite/120px-HINF-CSR_Gold3.png',
        measurement_matches_remaining: 0,
        tier_progress_pct: 50, // Or III → 3/6 = 50 %
        next_tier_label: 'Or IV', // sous-palier suivant (pas le palier suivant)
      },
    }
    response.has_ranked_history = true

    server.use(
      http.get('/api/v1/players/:playerSlug/pages/home', () => HttpResponse.json(response)),
    )

    renderWithProviders(<HomePage />)

    await waitFor(() => {
      expect(screen.getByTestId('home-highest-csr-tier-progress-fill')).toBeInTheDocument()
    })

    // Barre à 50 % ; extrémités = rating (gauche) + sous-palier suivant (droite) ; palier courant à gauche.
    expect(screen.getByTestId('home-highest-csr-tier-progress-fill')).toHaveStyle({ width: '50%' })
    expect(screen.getByTestId('home-highest-csr-value').textContent?.replace(/\s/g, '')).toBe('730')
    expect(screen.getByTestId('home-highest-csr-next-tier')).toHaveTextContent('Or IV')
    expect(screen.getByTestId('home-highest-csr-tier')).toHaveTextContent('Or III')
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
