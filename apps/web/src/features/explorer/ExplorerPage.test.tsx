/**
 * Tests composant — ExplorerPage (Slice 4).
 *
 * Smoke : monte, affiche les onglets Matchs / Joueur.
 */
import { describe, it, expect, vi } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import type { ExplorerMatchesQueryResponse } from '@/lib/api/types'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { ExplorerPage } from './ExplorerPage'
import { ExplorerMatchesResultsBlock } from './ExplorerPage.matchesMode'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
    useSearch: () => ({}),
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

// Stub i18n : renvoie la clé (contrôle structurel du rendu).
const tStub = ((key: string) => key) as (
  key: ExplorerManifestKey,
  values?: Record<string, string | number>,
) => string

function fakeMatchesQuery(): {
  data: ExplorerMatchesQueryResponse | undefined
  isLoading: boolean
  isError: boolean
  refetch: () => void
} {
  return {
    data: {
      briefing: null,
      table: { items: [] },
      export_hint: { token: 'tok-123' },
    } as unknown as ExplorerMatchesQueryResponse,
    isLoading: false,
    isError: false,
    refetch: () => {},
  }
}

describe('ExplorerMatchesResultsBlock — compteur/CSV (DP-7)', () => {
  it('retire le compteur « N matchs trouvés » du haut et rend l’export CSV SOUS le tableau', () => {
    const { container } = renderWithProviders(
      <ExplorerMatchesResultsBlock
        playerSlug="test-player"
        t={tStub}
        matchesQuery={fakeMatchesQuery()}
        matchesContextDescriptor={undefined}
      />,
    )
    // Le bloc résultats (stub `t` → renvoie la clé) ne rend PLUS le compteur redondant
    // du haut : la clé count_label est absente (le pied de pagination du VRAI tableau
    // utilise son propre `t` interne et rend du français, jamais la clé brute).
    expect(container.textContent ?? '').not.toContain('explorer.matches.count_label')
    // L'export CSV est rendu SOUS le tableau : son conteneur (flex justify-end) est le
    // DERNIER enfant du bloc résultats (le Bandeau rend null ici → tableau puis CSV).
    const csv = screen.getByText('explorer.matches.export_csv')
    const root = container.querySelector('.space-y-2')
    expect(root?.lastElementChild?.contains(csv)).toBe(true)
    // Deux cellules distinctes (tableau + CSV) → le CSV vient bien APRÈS le tableau.
    expect(root?.firstElementChild).not.toBe(root?.lastElementChild)
  })
})
