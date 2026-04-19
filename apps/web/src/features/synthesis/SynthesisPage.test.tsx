/**
 * Tests composant — SynthesisPage (Slice 7 — Synthèse).
 *
 * Smoke : monte, spinner, puis tableau de comparaison depuis MSW.
 */
import { describe, it, expect, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { SynthesisPage } from './SynthesisPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
  }
})

describe('SynthesisPage', () => {
  it('monte sans erreur', () => {
    const { container } = renderWithProviders(<SynthesisPage />)
    expect(container).toBeTruthy()
  })

  it('affiche le spinner pendant le chargement', () => {
    const { container } = renderWithProviders(<SynthesisPage />)
    expect(container.querySelector('.animate-spin')).toBeInTheDocument()
  })

  it('affiche le titre Synthese après chargement', async () => {
    renderWithProviders(<SynthesisPage />)
    await waitFor(() => {
      expect(screen.getByText('Synthèse')).toBeInTheDocument()
    })
  })

  it('affiche les selecteurs de période', async () => {
    renderWithProviders(<SynthesisPage />)
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Tout' })).toBeInTheDocument()
    })
  })

  it('affiche des placeholders explicites pour les sections vides', async () => {
    renderWithProviders(<SynthesisPage />)
    await waitFor(() => {
      expect(screen.getByText(/Activité indisponible/i)).toBeInTheDocument()
      expect(screen.getByText(/Aucune semaine remarquable/i)).toBeInTheDocument()
    })
  })

  describe('scope et overview (D9)', () => {
    it('affiche la barre scope-overview après chargement', async () => {
      renderWithProviders(<SynthesisPage />)
      await waitFor(() => {
        expect(screen.getByTestId('scope-overview-bar')).toBeInTheDocument()
      })
    })

    it('scope-overview affiche le nombre de matchs du fixture', async () => {
      renderWithProviders(<SynthesisPage />)
      await waitFor(() => {
        // Le fixture MSW retourne match_count: 5
        expect(screen.getByText('5')).toBeInTheDocument()
      })
    })

    it('scope-overview affiche la période "Tout"', async () => {
      renderWithProviders(<SynthesisPage />)
      // Le bouton de période "Tout" est déjà testé; ici on vérifie l'affichage dans le scope bar
      await waitFor(() => {
        const bar = screen.getByTestId('scope-overview-bar')
        expect(bar).toHaveTextContent('Tout')
      })
    })

    it('scope-overview est rendu avant le bloc Solo / Escouade', async () => {
      renderWithProviders(<SynthesisPage />)
      await waitFor(() => {
        const bar = screen.getByTestId('scope-overview-bar')
        expect(bar).toBeInTheDocument()
      })
      // Vérifier que scope-overview apparaît avant le texte "Solo"
      const bar = screen.getByTestId('scope-overview-bar')
      const soloCard = screen.getByText(/Solo \(/i).closest('[class*="Card"]') ??
                       screen.getByText(/Solo \(/i).closest('div')
      if (soloCard) {
        const position = bar.compareDocumentPosition(soloCard)
        // DOCUMENT_POSITION_FOLLOWING = 4 → soloCard est après bar
        expect(position & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
      }
    })
  })
})
