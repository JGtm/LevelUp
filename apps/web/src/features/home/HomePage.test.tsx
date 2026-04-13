/**
 * Tests composant — HomePage (Slice 5).
 *
 * Smoke : monte, spinner, puis Hero KPIs affichés depuis MSW.
 */
import { describe, it, expect, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { HomePage } from './HomePage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
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
})
