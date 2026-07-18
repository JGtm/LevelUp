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
      table: {
        // Une ligne réelle : sinon le tableau court-circuite sur son empty-state
        // (aucun pied rendu) et le slot CSV ne pourrait pas s'afficher.
        items: [
          {
            match_id: 'm1',
            start_time: '2026-05-26T12:00:00Z',
            map_ui: 'Alpha',
            mode_ui: 'Slayer',
            playlist_label: 'Quick Play',
            outcome_code: 2,
            score_label: '50-30',
            is_with_friends: false,
            kills: 10,
            deaths: 5,
            assists: 3,
            kda: 2.1,
          },
        ],
      },
      export_hint: { token: 'tok-123' },
    } as unknown as ExplorerMatchesQueryResponse,
    isLoading: false,
    isError: false,
    refetch: () => {},
  }
}

describe('ExplorerMatchesResultsBlock — compteur retiré / CSV dans le pied du tableau', () => {
  it('retire le compteur « N matchs trouvés » et ancre l’export CSV à gauche du pied du tableau', () => {
    const { container } = renderWithProviders(
      <ExplorerMatchesResultsBlock
        playerSlug="test-player"
        t={tStub}
        matchesQuery={fakeMatchesQuery()}
        matchesContextDescriptor={undefined}
      />,
    )
    // Le compteur redondant n'est plus rendu : ni la clé stub `count_label` (bloc
    // résultats), ni le libellé FR « … trouvé(s) » du `t` interne du tableau — le pied
    // porte le bouton CSV à sa place.
    expect(container.textContent ?? '').not.toContain('explorer.matches.count_label')
    expect(container.textContent ?? '').not.toContain('trouvé')
    // L'export CSV (stub `t` → clé brute) est présent ET ancré DANS le pied du tableau
    // (composant `data-testid="explorer-matches-table"`), non plus dans un bloc séparé.
    const csv = screen.getByText('explorer.matches.export_csv')
    const tableRoot = screen.getByTestId('explorer-matches-table')
    expect(tableRoot.contains(csv)).toBe(true)
  })
})
