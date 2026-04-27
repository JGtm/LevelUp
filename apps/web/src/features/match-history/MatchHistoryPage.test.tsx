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

  // Test "affiche le titre 'Historique'" supprimé : le titre h1 a été retiré
  // du composant lors du refacto post-84ae65ca (NavL1 expose la section).

  it('affiche le compteur de parties après chargement', async () => {
    // Format simplifié post-84ae65ca : "X partie(s)" dans MatchHistoryTable au
    // lieu de "0 parties dans la période". Fixture vide → "0 partie".
    renderWithProviders(<MatchHistoryPage />)
    await waitFor(() => {
      expect(screen.getByText(/0 partie/i)).toBeInTheDocument()
    })
  })
})
