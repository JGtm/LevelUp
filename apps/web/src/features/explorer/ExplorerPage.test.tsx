/**
 * Tests composant — ExplorerPage (Slice 4).
 *
 * Smoke : monte, affiche les onglets Matchs / Joueur.
 */
import { describe, it, expect, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { ExplorerPage } from './ExplorerPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
  }
})

describe('ExplorerPage', () => {
  it('monte sans erreur', () => {
    const { container } = renderWithProviders(<ExplorerPage />)
    expect(container).toBeTruthy()
  })

  it('affiche le titre Explorer', () => {
    renderWithProviders(<ExplorerPage />)
    expect(screen.getByText('Explorer')).toBeInTheDocument()
  })

  it('affiche les onglets Matchs et Joueur', () => {
    renderWithProviders(<ExplorerPage />)
    expect(screen.getByText('Matchs')).toBeInTheDocument()
    expect(screen.getByText('Joueur')).toBeInTheDocument()
  })

  it('affiche les résultats vides après chargement', async () => {
    renderWithProviders(<ExplorerPage />)
    await waitFor(() => {
      // Le fixture retourne 0 matchs — aucune ligne de résultat
      expect(screen.queryByRole('row')).not.toBeInTheDocument()
    })
  })
})
