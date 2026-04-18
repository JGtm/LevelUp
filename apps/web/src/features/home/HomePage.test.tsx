/**
 * Tests composant — HomePage (Slice 5).
 *
 * Smoke : monte, spinner, puis Hero KPIs affichés depuis MSW.
 */
import { describe, it, expect, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
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

  it('affiche le titre Mission Control', async () => {
    renderWithProviders(<HomePage />)
    await waitFor(() => {
      expect(screen.getByText(/Mission Control/i)).toBeInTheDocument()
    })
  })

  it('affiche les KPIs globaux (Parties, Win Rate, K/D)', async () => {
    renderWithProviders(<HomePage />)
    await waitFor(() => {
      expect(screen.getByText('Parties')).toBeInTheDocument()
      expect(screen.getByText('Win Rate')).toBeInTheDocument()
      expect(screen.getByText('K/D')).toBeInTheDocument()
    })
  })

  it('affiche le nom du joueur dans le titre', async () => {
    renderWithProviders(<HomePage />)
    await waitFor(() => {
      expect(screen.getByText(/TestPlayer/i)).toBeInTheDocument()
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
