import type { ReactNode } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'

import { PalmaresRelationsPage } from './PalmaresRelationsPage'

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
})
