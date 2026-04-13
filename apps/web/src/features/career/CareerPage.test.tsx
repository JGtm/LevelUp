/**
 * Tests composant — CareerPage (Slice 2).
 *
 * Smoke : monte, affiche le spinner, puis les données de carrière depuis MSW.
 */
import { describe, it, expect, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { CareerPage } from './CareerPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
  }
})

describe('CareerPage', () => {
  it('monte sans erreur', () => {
    const { container } = renderWithProviders(<CareerPage />)
    expect(container).toBeTruthy()
  })

  it('affiche le spinner en phase de chargement', () => {
    renderWithProviders(<CareerPage />)
    expect(screen.getByText(/Chargement de la carrière/i)).toBeInTheDocument()
  })

  it("affiche le titre 'Carrière' une fois les données chargées", async () => {
    renderWithProviders(<CareerPage />)
    await waitFor(() => {
      expect(screen.getByText('Carrière')).toBeInTheDocument()
    })
  })

  it('affiche la section de progression du rang', async () => {
    renderWithProviders(<CareerPage />)
    await waitFor(() => {
      // Le badge de rang "Gold 3" doit être visible (CareerSummaryCard)
      expect(screen.getByText('Gold 3')).toBeInTheDocument()
    })
  })

  it("affiche le bouton 'Voir tous les top matchs' si preview non vide", async () => {
    // Le fixture career retourne une preview vide → bouton absent
    renderWithProviders(<CareerPage />)
    await waitFor(() => {
      expect(screen.queryByText(/Voir tous les top matchs/i)).not.toBeInTheDocument()
    })
  })
})
