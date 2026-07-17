import type { ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'

import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useRelationsPrefsStore } from '@/stores/relationsPrefsStore'

import { PalmaresRelationsPage } from './PalmaresRelationsPage'

const EMPTY_RELATIONS = { overview: { distinct_players: 0, allies_count: 0, rivals_count: 0, core_count: 0, top_ally: null, top_nemesis: null }, relations: [] }

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
    Link: ({ children, to }: { children?: ReactNode; to?: string }) => <a href={to}>{children}</a>,
  }
})

describe('PalmaresRelationsPage', () => {
  // Isolation : le filtre de chips est un store persisté global — le remettre à
  // l'état par défaut après chaque test évite la pollution inter-tests.
  afterEach(() => {
    useRelationsPrefsStore.setState({ filter: 'all', includeNeverFaced: false })
  })

  it('monte sans erreur', () => {
    const { container } = renderWithProviders(<PalmaresRelationsPage />)
    expect(container).toBeTruthy()
  })

  it('affiche le hero et le tableau après chargement', async () => {
    renderWithProviders(<PalmaresRelationsPage />)

    await waitFor(() => {
      expect(screen.getByTestId('palmares-relations-overview')).toBeInTheDocument()
    })

    // Hero : binôme + bête noire (gamertags issus de l'overview du mock).
    expect(screen.getByText('Binôme')).toBeInTheDocument()
    expect(screen.getByText('Bête noire')).toBeInTheDocument()

    // Tableau : toutes les relations récurrentes.
    expect(screen.getAllByText('DuoAlpha').length).toBeGreaterThan(0)
    expect(screen.getAllByText('NemesisBravo').length).toBeGreaterThan(0)
    expect(screen.getAllByText('QueueGhost').length).toBeGreaterThan(0)
  })

  it('filtre les relations via les chips client', async () => {
    renderWithProviders(<PalmaresRelationsPage />)

    await waitFor(() => {
      expect(screen.getByTestId('palmares-relations-chips')).toBeInTheDocument()
    })

    // Le mock ne contient aucune relation strictement "alliée" pure mais tous
    // ont teammate_matches > 0, donc le filtre "Alliés" garde les 3 joueurs.
    fireEvent.click(screen.getByRole('button', { name: 'Alliés' }))
    expect(screen.getAllByText('DuoAlpha').length).toBeGreaterThan(0)
  })

  it('rend les badges solid (duo gagnant)', async () => {
    renderWithProviders(<PalmaresRelationsPage />)

    await waitFor(() => {
      expect(screen.getByTestId('palmares-relations-overview')).toBeInTheDocument()
    })

    // Le badge apparaît dans la card hero enrichie + le tableau ; tous en style solid.
    const badges = screen.getAllByText('Duo gagnant')
    expect(badges.length).toBeGreaterThan(0)
    expect(badges[0].closest('[data-testid="narrative-badge"]')?.getAttribute('data-solid')).toBe('true')
  })

  it('rend le badge cross-jeu en résolvant {game} depuis detail', async () => {
    renderWithProviders(<PalmaresRelationsPage />)

    await waitFor(() => {
      expect(screen.getByTestId('palmares-relations-overview')).toBeInTheDocument()
    })

    // narrative.encounter.cross_game = « {game} » ; le mock pose
    // detail.game = "Halo 5" → le label EST le nom de l'autre titre.
    expect(screen.getAllByText('Halo 5').length).toBeGreaterThan(0)
  })

  it('affiche le chip Multi-jeux et filtre sur les relations cross-jeu', async () => {
    renderWithProviders(<PalmaresRelationsPage />)

    await waitFor(() => {
      expect(screen.getByTestId('palmares-relations-chips')).toBeInTheDocument()
    })

    // DuoAlpha porte le badge cross-jeu (Halo 5) → le chip « Multi-jeux » apparaît.
    const crossChip = screen.getByRole('button', { name: 'Multi-jeux' })
    fireEvent.click(crossChip)

    // Seule la relation cross-jeu (DuoAlpha) reste ; QueueGhost (mono-titre)
    // disparaît du tableau.
    expect(screen.getAllByText('DuoAlpha').length).toBeGreaterThan(0)
    await waitFor(() => {
      expect(screen.queryByText('QueueGhost')).not.toBeInTheDocument()
    })
  })

  it('masque le chip Multi-jeux sans relation cross-jeu et neutralise un filtre cross persisté', async () => {
    // Filtre 'cross' persisté AVANT rendu : sans donnée cross-jeu, le garde-fou
    // doit retomber sur « Tous » sans masquer la relation.
    useRelationsPrefsStore.setState({ filter: 'cross' })
    server.use(
      http.post('/api/v1/players/:playerSlug/pages/palmares/relations', () =>
        HttpResponse.json({
          overview: {
            distinct_players: 1,
            allies_count: 1,
            rivals_count: 0,
            core_count: 0,
            top_ally: null,
            top_nemesis: null,
          },
          relations: [
            {
              xuid: '9',
              gamertag: 'SoloOnly',
              total_matches: 10,
              teammate_matches: 8,
              teammate_wins: 5,
              teammate_win_rate: 0.6,
              enemy_matches: 3,
              enemy_wins: 1,
              enemy_win_rate: 0.33,
              avg_kda_with: 1,
              avg_kda_against: 1,
              kills_dealt: 4,
              deaths_suffered: 4,
              duel_ratio: 1,
              first_seen_at: '2026-01-01T00:00:00Z',
              last_seen_at: '2026-06-01T00:00:00Z',
              category: 'mixed',
              badges: [],
            },
          ],
        }),
      ),
    )

    renderWithProviders(<PalmaresRelationsPage />)

    await waitFor(() => {
      expect(screen.getByTestId('palmares-relations-chips')).toBeInTheDocument()
    })

    // Profil mono-titre → aucun chip « Multi-jeux ».
    expect(screen.queryByRole('button', { name: 'Multi-jeux' })).not.toBeInTheDocument()
    // Garde-fou : le filtre 'cross' persisté est neutralisé → la relation reste visible.
    expect(screen.getAllByText('SoloOnly').length).toBeGreaterThan(0)
  })

  it('toggle « jamais affrontés » : défaut masqué + bascule du libellé', async () => {
    renderWithProviders(<PalmaresRelationsPage />)

    await waitFor(() => {
      expect(screen.getByTestId('palmares-relations-chips')).toBeInTheDocument()
    })

    // Défaut = masqué : le bouton propose de LES INCLURE.
    const toggle = screen.getByRole('button', { name: 'Inclure les jamais affrontés' })
    expect(toggle).toHaveAttribute('aria-pressed', 'false')

    // Bascule → libellé « inclus » + aria-pressed true.
    fireEvent.click(toggle)
    expect(
      screen.getByRole('button', { name: 'Jamais affrontés inclus' }),
    ).toHaveAttribute('aria-pressed', 'true')
  })

  it('rend la barre de segmentation serveur (Vue + Analyser)', async () => {
    renderWithProviders(<PalmaresRelationsPage />)

    await waitFor(() => {
      expect(screen.getByTestId('palmares-relations-overview')).toBeInTheDocument()
    })

    // Contrôle Vue solo/escouade + bouton Analyser présents.
    expect(screen.getByTestId('relations-view-dropdown')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Analyser' })).toBeInTheDocument()
  })

  it('affiche la section Moments & Rivalités en permanence', async () => {
    renderWithProviders(<PalmaresRelationsPage />)

    await waitFor(() => {
      expect(screen.getByTestId('palmares-relations-moments')).toBeInTheDocument()
    })

    // Permanent (plus de toggle) : la donnée du mock arrive d'office → titre Rivaux + rival.
    await waitFor(() => {
      expect(screen.getByText('Rivaux')).toBeInTheDocument()
    })
    expect(screen.getAllByText('NemesisBravo').length).toBeGreaterThan(0)
  })

  it('envoie un FilterContextInput segmenté (vue Escouade) après « Analyser »', async () => {
    const bodies: Array<Record<string, unknown>> = []
    server.use(
      http.post('/api/v1/players/:playerSlug/pages/palmares/relations', async ({ request }) => {
        bodies.push((await request.json()) as Record<string, unknown>)
        return HttpResponse.json(EMPTY_RELATIONS)
      }),
    )

    renderWithProviders(<PalmaresRelationsPage />)

    // Ouvre le dropdown Vue, sélectionne Escouade, puis Analyser.
    fireEvent.click(screen.getByTestId('relations-view-dropdown').querySelector('button')!)
    fireEvent.click(await screen.findByRole('button', { name: 'Escouade' }))
    fireEvent.click(screen.getByRole('button', { name: 'Analyser' }))

    await waitFor(() => {
      expect(bodies.some((b) => b.match_context === 'squad')).toBe(true)
    })
  })
})
