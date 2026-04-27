/**
 * Tests composant — ExplorerPage (Slice 4).
 *
 * Smoke : monte, affiche les onglets Matchs / Joueur.
 */
import { describe, it, expect, vi } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'
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

  // Titre h1 "Explorer" retiré du composant lors du refacto post-84ae65ca
  // (la NavL1 expose déjà le label de section). Le rendu de la page est validé
  // par les tests onglets + messages "Aucun match trouvé" ci-dessous.

  it('affiche les onglets Matchs et Joueur', () => {
    renderWithProviders(<ExplorerPage />)
    expect(screen.getByText('Matchs')).toBeInTheDocument()
    expect(screen.getByText('Joueur')).toBeInTheDocument()
  })

  it('affiche un message explicite quand aucun match n’est trouvé', async () => {
    renderWithProviders(<ExplorerPage />)
    await waitFor(() => {
      expect(screen.getByText(/Aucun match trouvé/i)).toBeInTheDocument()
    })
  })

  it('affiche un message explicite tant qu’aucun joueur n’est sélectionné', async () => {
    renderWithProviders(<ExplorerPage />)
    fireEvent.click(screen.getByText('Joueur'))

    await waitFor(() => {
      expect(screen.getByText(/Aucun joueur sélectionné/i)).toBeInTheDocument()
    })
  })
})
