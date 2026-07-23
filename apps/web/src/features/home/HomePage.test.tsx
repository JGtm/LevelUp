/**
 * Tests composant — HomePage (Slice 5).
 *
 * Smoke : monte, spinner, puis Hero KPIs affichés depuis MSW.
 */
import { describe, it, expect, vi } from 'vitest'
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import type { ReactNode } from 'react'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { TITLE_CAPABILITIES } from '@/lib/capabilities/capabilities'
import type { TitleSummary } from '@/lib/api/types'
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

  it("ne rend pas de loader plein écran pendant le chargement (TopProgressBar globale)", () => {
    renderWithProviders(<HomePage />)
    expect(screen.queryByText(/Chargement de l'accueil/i)).not.toBeInTheDocument()
  })

  it('affiche la section Performance globale après chargement', async () => {
    // "Taux de victoire" : le backend field-mappings (TOML) prime, mais quand il
    // est absent (flag MULTI_TITLE_API_ENABLED off / fields vides en test),
    // labelOf retombe sur les libellés locaux kpi.i18n.ts (pas la clé brute).
    renderWithProviders(<HomePage />)
    await waitFor(() => {
      expect(screen.getByText('Taux de victoire')).toBeInTheDocument()
    })
  })

  it('affiche le Spartan ID et le rang carrière localisé dans Performance globale', async () => {
    const previousLocale = useAppShellStore.getState().locale
    act(() => {
      // Titre courant SANS spartan_customizer → bandeau d'identité normal (emblème <img>),
      // pas la synthèse Halo 5 (useCapability fail-open si availableTitles vide).
      useAppShellStore.setState({
        locale: 'en',
        currentTitleSlug: 'halo_infinite',
        availableTitles: [
          {
            slug: 'halo_infinite',
            name: 'Halo Infinite',
            status: 'active',
            // Toutes les capabilities SAUF spartan_customizer → bandeau normal (emblème
            // <img>) + panneau skill peaks (ranked/lusr) visible.
            capabilities: TITLE_CAPABILITIES.filter((c) => c !== 'spartan_customizer'),
            is_default: true,
            effective_hp_to_kill: 225,
          } as unknown as TitleSummary,
        ],
      })
    })

    server.use(
      http.get('/api/v1/players/:playerSlug/pages/home', () => HttpResponse.json({
        hero: {
          player_name: 'TestPlayer',
          kpis: {
            win_rate: 0.55,
            global_ratio: 1.2,
            avg_accuracy: 42,
            total_matches: 120,
            wins: 66,
            losses: 54,
          },
          trend: null,
        },
        spartan_identity: {
          spartan_id: 'JGTM',
          banner_image_url: 'https://example.test/identity/nameplate.png',
          emblem_image_url: 'https://example.test/identity/emblem.png',
          backdrop_image_url: 'https://example.test/identity/backdrop.png',
          highest_csr: {
            rating_value: 1525,
            tier_label: 'Gold 3',
            badge_image_url: '/static/ranks/halo_infinite/120px-HINF-CSR_Gold3.png',
          },
          highest_lusr: {
            rating_value: 1750,
            tier_label: 'Platinum V',
            badge_image_url: '/static/ranks/halo_infinite/120px-HINF-CSR_Platinum5.png',
          },
          career_rank: {
            rank_number: 25,
            rank_title: 'Lance Corporal',
            rank_image_url: 'https://example.test/ranks/lance-corporal.png',
            adornment_image_url: 'https://example.test/ranks/lance-corporal-adornment.png',
            current_xp: 5000,
            xp_for_next_rank: 10000,
            progress_pct: 50,
            is_max_rank: false,
          },
        },
        highlights: [],
        recent_matches: [],
        favorite_matches: [],
        recent_media: [],
        solo_session: null,
        squad_session: null,
      })),
    )

    try {
      renderWithProviders(<HomePage />)

      await waitFor(() => {
        const spartanBanner = screen.getByTestId('home-spartan-identity-banner')
        expect(screen.getByTestId('home-spartan-gamertag')).toHaveTextContent('TestPlayer')
        expect(screen.getByTestId('home-spartan-id-value')).toHaveTextContent('JGTM')
        expect(screen.getByTestId('home-spartan-emblem-image')).toHaveAttribute('src', 'https://example.test/identity/emblem.png')
        expect(screen.getByTestId('home-skill-peaks-panel')).toBeInTheDocument()
        expect(screen.getByTestId('home-highest-csr-value')).toHaveTextContent('1,525')
        expect(screen.getByTestId('home-highest-csr-tier')).toHaveTextContent('Gold 3')
        expect(screen.getByTestId('home-highest-csr-badge')).toHaveAttribute('src', '/static/ranks/halo_infinite/120px-HINF-CSR_Gold3.png')
        expect(screen.getByTestId('home-highest-lusr-value')).toHaveTextContent('1,750')
        expect(screen.getByTestId('home-highest-lusr-tier')).toHaveTextContent('Platinum V')
        expect(screen.getByTestId('home-highest-lusr-badge')).toHaveAttribute('src', '/static/ranks/halo_infinite/120px-HINF-CSR_Platinum5.png')
        expect(within(spartanBanner).queryByTestId('home-highest-csr-card')).not.toBeInTheDocument()
        expect(within(spartanBanner).queryByTestId('home-highest-lusr-card')).not.toBeInTheDocument()
        expect(screen.getByTestId('home-career-rank-title')).toHaveTextContent('Lance Corporal')
        expect(screen.getByTestId('home-spartan-adornment-image')).toHaveAttribute('src', 'https://example.test/ranks/lance-corporal-adornment.png')
        // Note : le label "Career rank" a été retiré du composant lors de la
        // migration useFieldLabel (Phase D-bis, commit 84ae65ca) — seul le
        // rank_title est rendu (testid home-career-rank-title vérifié au-dessus).
        expect(screen.getByText('Highest CSR')).toBeInTheDocument()
        expect(screen.getByText('Highest LUSR')).toBeInTheDocument()
      })

      expect(screen.getByTestId('home-spartan-banner-shell')).toBeInTheDocument()
      expect(screen.getByTestId('home-spartan-banner-surface')).toBeInTheDocument()
      expect(screen.queryByText('Spartan ID')).not.toBeInTheDocument()

      // Note : labels textuels "Rank 25" et "Current progress" retirés du
      // composant lors du refacto Phase D-bis (commit 84ae65ca) au profit des
      // testid dédiés vérifiés ci-dessous.
      expect(screen.getByTestId('home-career-rank-progress-current')).toHaveTextContent('5,000 XP')
      expect(screen.getByTestId('home-career-rank-progress-target')).toHaveTextContent('10,000 XP')
      expect(screen.getByTestId('home-career-rank-progress-fill')).toHaveStyle({ width: '50%' })
    } finally {
      act(() => {
        useAppShellStore.setState({ locale: previousLocale, availableTitles: [] })
      })
    }
  })

  it('ne réutilise pas le backdrop comme bannière quand la nameplate est absente', async () => {
    server.use(
      http.get('/api/v1/players/:playerSlug/pages/home', () => HttpResponse.json({
        hero: {
          player_name: 'TestPlayer',
          kpis: {
            win_rate: 0.55,
            global_ratio: 1.2,
            avg_accuracy: 42,
            total_matches: 120,
            wins: 66,
            losses: 54,
          },
          trend: null,
        },
        spartan_identity: {
          spartan_id: 'JGTM',
          banner_image_url: null,
          emblem_image_url: 'https://example.test/identity/emblem.png',
          backdrop_image_url: 'https://example.test/identity/backdrop.png',
          career_rank: null,
        },
        highlights: [],
        recent_matches: [],
        favorite_matches: [],
        recent_media: [],
        solo_session: null,
        squad_session: null,
      })),
    )

    renderWithProviders(<HomePage />)

    await waitFor(() => {
      expect(screen.getByTestId('home-spartan-id-value')).toHaveTextContent('JGTM')
    })

    expect(screen.getByTestId('home-spartan-banner-shell')).toBeInTheDocument()
    expect(screen.getByTestId('home-spartan-banner-surface')).toBeInTheDocument()
    expect(screen.queryByText('Spartan ID')).not.toBeInTheDocument()
  })

  it('affiche le visuel battle pass, le carrousel de paliers et la progression du palier actif sur la home', async () => {
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
        ],
      })),
    )

    renderWithProviders(<HomePage />)

    await waitFor(() => {
      expect(screen.getByText('Operation Alpha')).toBeInTheDocument()
      expect(screen.getByAltText('Illustration de Operation Alpha')).toBeInTheDocument()
      expect(screen.getAllByText('Récompense 13').length).toBeGreaterThan(0)
    })

    expect(scrollToMock).toHaveBeenCalledWith(expect.objectContaining({ behavior: 'smooth' }))
    expect(scrollIntoViewMock).not.toHaveBeenCalled()

    const tierCards = screen.getAllByTestId('battle-pass-tier-card')
    expect(tierCards).toHaveLength(3)
    expect(tierCards[0]).toHaveAttribute('data-obtained', 'true')
    expect(tierCards[1]).toHaveAttribute('data-current', 'true')
    expect(screen.getByTestId('home-battle-pass-active-tier-progress-fill')).toHaveStyle({ width: '30%' })
    expect(screen.getByTestId('home-battle-pass-active-tier-progress-current')).toHaveTextContent('300 XP')
    expect(screen.getByTestId('home-battle-pass-active-tier-progress-target')).toHaveTextContent('1 000 XP')
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

  it('affiche les défis actifs détaillés triés du plus avancé au moins avancé', async () => {
    server.use(
      http.get('/api/v1/players/:playerSlug/pages/palmares/season-pass', () => HttpResponse.json({
        title_slug: 'halo_infinite',
        available: true,
        active_track_path: 'RewardTracks/TrackA',
        challenges: {
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
        },
        passes: [
          {
            reward_track_path: 'RewardTracks/TrackA',
            name: 'Operation Alpha',
            description: null,
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
            tiers: [],
          },
        ],
      })),
    )

    renderWithProviders(<HomePage />)

    await waitFor(() => {
      expect(screen.getByText(/Défis actifs/i)).toBeInTheDocument()
      expect(screen.getByTestId('home-challenges-completed')).toHaveTextContent('0 / 3 complétés')
      expect(screen.getByTestId('home-challenges-card')).toHaveClass('min-h-[14rem]')
      expect(
        screen.getByText((content) => content.replace(/\s+/g, ' ').includes('4 500 XP disponibles')),
      ).toBeInTheDocument()
    })

    const sectionTitles = screen.getAllByTestId('home-challenge-section-title')
    expect(sectionTitles.map((node) => node.textContent)).toEqual(['Quotidien', 'Hebdo'])

    const dailySection = screen.getByTestId('home-challenge-section-daily')
    const weeklySection = screen.getByTestId('home-challenge-section-weekly')

    expect(within(dailySection).getAllByTestId('home-challenge-title').map((node) => node.textContent)).toEqual([
      'Défi en cours',
    ])
    expect(within(weeklySection).getAllByTestId('home-challenge-title').map((node) => node.textContent)).toEqual([
      'Défi avancé',
      'Défi pas commencé',
    ])
    expect(screen.queryByTestId('home-challenge-kind')).not.toBeInTheDocument()

    const thumbs = screen.getAllByTestId('home-challenge-thumb')
    expect(thumbs[0]).not.toHaveClass('bg-sky-500/8')
    expect(thumbs[1]).not.toHaveClass('bg-amber-500/8')
    expect(thumbs[2]).not.toHaveClass('bg-muted/35')

    expect(screen.getByText('Défi avancé')).toHaveClass('font-semibold')
    expect(screen.getByText('Presque terminé.')).toHaveClass('italic')

    const dailyProgressRow = within(dailySection).getByTestId('home-challenge-progress-row')
    expect(within(dailyProgressRow).getByTestId('home-challenge-progress-current')).toHaveTextContent('1 / 3')
    expect(within(dailyProgressRow).getByTestId('home-challenge-progress-percent')).toHaveTextContent('33%')
    expect(within(dailyProgressRow).getByTestId('home-challenge-progress-track')).toBeInTheDocument()

    const weeklyFills = within(weeklySection).getAllByTestId('home-challenge-progress-fill')
    expect(weeklyFills[0]).toHaveStyle({ width: '70%' })
    expect(weeklyFills[1]).toHaveStyle({ width: '0%' })
  })

  it('affiche un état vide compact pour les défis quand aucun item actif n’est détaillé', async () => {
    server.use(
      http.get('/api/v1/players/:playerSlug/pages/palmares/season-pass', () => HttpResponse.json({
        title_slug: 'halo_infinite',
        available: true,
        active_track_path: 'RewardTracks/TrackA',
        challenges: {
          available: true,
          total: 5,
          completed: 5,
          xp_available: 0,
          next_expiry: null,
          items: [],
          error_hint: null,
        },
        passes: [
          {
            reward_track_path: 'RewardTracks/TrackA',
            name: 'Operation Alpha',
            description: null,
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
            tiers: [],
          },
        ],
      })),
    )

    renderWithProviders(<HomePage />)

    await waitFor(() => {
      expect(screen.getByTestId('home-challenges-card')).toHaveClass('min-h-[14rem]')
      expect(screen.getByText('Aucun défi actif')).toBeInTheDocument()
      expect(screen.getByText(/Tous les défis visibles sont terminés/i)).toBeInTheDocument()
    })

    expect(screen.queryByTestId('home-challenge-item')).not.toBeInTheDocument()
    expect(screen.getByTestId('home-challenges-completed')).toHaveTextContent('5 / 5 complétés')
  })

  it('affiche les KPIs globaux (libellés locaux fallback quand TOML absent)', async () => {
    // labelOf privilégie le backend field-mappings (TOML) ; quand il est absent
    // (fields vides en test, ou flag MULTI_TITLE_API_ENABLED off en prod), il
    // retombe sur les libellés locaux kpi.i18n.ts — JAMAIS la clé brute.
    renderWithProviders(<HomePage />)
    await waitFor(() => {
      expect(screen.getByText('Matchs')).toBeInTheDocument()
      expect(screen.getByText('Taux de victoire')).toBeInTheDocument()
      expect(screen.getByText('KDA')).toBeInTheDocument()
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

      expect(stickyShell).toHaveClass('sticky', 'top-0')
      expect(banner).toBeInTheDocument()
      expect(banner).not.toHaveClass('sticky', 'top-0')
      // Le composant HomeHeroBanner a été refactoré (post-84ae65ca) : il n'utilise
      // plus de balise <img> mais des <div> avec backgroundImage en style inline
      // pour permettre des transitions cross-fade entre plusieurs visuels. La
      // hauteur responsive est portée par le wrapper interne (querySelectable).
      const heightWrapper = banner.querySelector('.h-36.w-full')
      expect(heightWrapper).not.toBeNull()
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
              kind: 'screenshot',
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

    // MediaThumbnailCard ne rend plus le basename comme texte visible (refacto
    // post-84ae65ca) — il l'utilise uniquement comme alt sur l'<img>. On
    // cherche donc l'image par son alt pour identifier la card.
    const thumbnail = await screen.findByAltText('clip-epic.mp4')
    expect(thumbnail).toBeInTheDocument()

    // Le card lui-meme est role="button" (parent de l'image).
    const card = thumbnail.closest('[role="button"]')
    expect(card).not.toBeNull()
    fireEvent.click(card as HTMLElement)

    await waitFor(() => {
      // La lightbox CoverFlowModal expose un bouton "Fermer" — assertion minimale
      // que le clic sur la card a bien ouvert la lightbox. Le lien "Voir le match"
      // n'est plus rendu dans la lightbox actuelle.
      expect(screen.getByRole('button', { name: /Fermer/i })).toBeInTheDocument()
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
