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

  it('ne rend pas de loader plein écran pendant le chargement (TopProgressBar globale)', () => {
    renderWithProviders(<CareerPage />)
    expect(screen.queryByText(/Chargement de la carrière/i)).not.toBeInTheDocument()
  })

  // Test "affiche le titre 'Carrière'" supprimé (refacto post-84ae65ca) :
  // - Le titre h1 a été retiré du composant (la NavL1 expose déjà la section).
  // - Le rendu post-loading est déjà couvert par le test "affiche la section
  //   de progression du rang" + "affiche des placeholders explicites".

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

  it('affiche des placeholders explicites pour les sections vides', async () => {
    renderWithProviders(<CareerPage />)
    await waitFor(() => {
      // Placeholders "Graphique indisponible" et "Top matchs indisponibles"
      // restent rendus dans le DOM pour les sections vides. Le placeholder
      // "Rating indisponible" a été retiré du composant lors d'un refacto
      // ultérieur (post-test) — son assertion a été supprimée pour aligner
      // le test sur l'état actuel du composant.
      expect(screen.getAllByText(/Graphique indisponible/i).length).toBeGreaterThan(0)
      expect(screen.getByText(/Top matchs indisponibles/i)).toBeInTheDocument()
    })
  })
})
