import type { ReactNode } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'

import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'

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
    expect(screen.getByText('Ton binôme')).toBeInTheDocument()
    expect(screen.getByText('Ta bête noire')).toBeInTheDocument()

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

    expect(screen.getByText('Duo gagnant')).toBeInTheDocument()
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

  it('affiche la section Moments & Rivalités repliée puis la déplie au clic', async () => {
    renderWithProviders(<PalmaresRelationsPage />)

    await waitFor(() => {
      expect(screen.getByTestId('palmares-relations-moments')).toBeInTheDocument()
    })

    // Repliée : le titre de section revanche n'est pas monté tant que fermée.
    expect(screen.queryByText('Revanche')).not.toBeInTheDocument()

    fireEvent.click(screen.getByText('Afficher les moments et rivalités'))

    // Dépliée : la donnée du mock arrive → titre Revanche + rival NemesisBravo.
    await waitFor(() => {
      expect(screen.getByText('Revanche')).toBeInTheDocument()
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
