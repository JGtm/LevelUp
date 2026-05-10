/**
 * Tests composant — CareerHubPage.
 *
 * Refacto 2026-05 : les onglets Progression / Citations / Pass saisonnier ont
 * été déplacés dans la NavL2 globale (cf. components/shell/NavL2.tsx). Cette
 * page délègue désormais directement à `<CareerProgressionTab>` ; les tests
 * d'onglets vivent désormais avec NavL2.
 *
 * On ne garde ici que les invariants vraiment liés au hub :
 *   - rendu sans erreur
 *   - aucune section "top matchs" ni "encounters" dans le hub
 */
import { beforeEach, describe, it, expect, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { CareerHubPage } from './CareerHubPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
    useSearch: () => ({ tab: 'progression' as const }),
  }
})

describe('CareerHubPage', () => {
  beforeEach(() => {
    // pas de state à reset — la page est purement déléguante
  })

  it('monte sans erreur', () => {
    const { container } = renderWithProviders(<CareerHubPage />)
    expect(container).toBeTruthy()
  })

  it("n'affiche pas de section 'top matchs' ou 'encounters' dans le hub", async () => {
    renderWithProviders(<CareerHubPage />)
    await waitFor(() => {
      expect(screen.queryByText(/top matchs/i)).not.toBeInTheDocument()
      expect(screen.queryByText(/Rencontres fréquentes/i)).not.toBeInTheDocument()
    })
  })
})
