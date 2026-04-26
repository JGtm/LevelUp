/**
 * Tests composant — MatchHistoryPage (Slice 3).
 *
 * Smoke : monte, spinner, puis tableau chargé depuis MSW.
 */
import { describe, it, expect, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { MatchHistoryPage } from './MatchHistoryPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
  }
})

describe('MatchHistoryPage', () => {
  it('monte sans erreur', () => {
    const { container } = renderWithProviders(<MatchHistoryPage />)
    expect(container).toBeTruthy()
  })

  it('ne rend pas de loader plein écran pendant le chargement (TopProgressBar globale)', () => {
    renderWithProviders(<MatchHistoryPage />)
    expect(screen.queryByText(/Chargement de l'historique/i)).not.toBeInTheDocument()
  })

  it("affiche le titre 'Historique' une fois les données chargées", async () => {
    renderWithProviders(<MatchHistoryPage />)
    await waitFor(() => {
      expect(screen.getByText('Historique des parties')).toBeInTheDocument()
    })
  })

  it('affiche 0 résultats pour le fixture vide', async () => {
    renderWithProviders(<MatchHistoryPage />)
    await waitFor(() => {
      expect(screen.getByText(/0 parties dans la période/i)).toBeInTheDocument()
    })
  })
})
