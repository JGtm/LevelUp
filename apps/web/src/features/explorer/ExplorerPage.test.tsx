/**
 * Tests composant — ExplorerPage (Slice 4).
 *
 * Smoke : monte, affiche les onglets Matchs / Joueur.
 */
import { afterEach, describe, it, expect, vi } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { api } from '@/lib/api/client'
import { useAppShellStore } from '@/stores/appShellStore'
import type { ExplorerMatchesQueryResponse } from '@/lib/api/types'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { ExplorerPage } from './ExplorerPage'
import { ExplorerMatchesResultsBlock } from './ExplorerPage.matchesMode'

// Le search de l'URL est pilotable : le cas « ?replay=with sur un titre sans rejeu »
// (revue C-R1, constat C5) a besoin d'une URL qui porte le filtre.
const routerState = vi.hoisted(() => ({ search: {} as Record<string, unknown> }))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
    useSearch: () => routerState.search,
  }
})


/** Force les capabilities du titre courant (fail-open par défaut sinon). */
function setTitleCaps(caps: string[]) {
  useAppShellStore.setState({
    currentTitleSlug: 'test_title',
    availableTitles: [
      {
        slug: 'test_title', name: 'Test', status: 'active', capabilities: caps, is_default: true,
        effective_hp_to_kill: 225, provides_damage_taken: true, provides_team_mmr: true,
        provides_max_killing_spree: true, offensive_conversion_p80: 0.9,
        defensive_resistance_p80: 1.65,
      },
    ],
  })
}

// PORTE DE TITRE DU FILTRE « Avec rejeu / Sans rejeu » (2026-09-05, registre L5). Sur un
// titre sans décodeur de film, « Avec rejeu » rendrait toujours zéro ligne et « Sans rejeu »
// serait un synonyme de « tous » : un filtre qui ne filtre rien est pire qu'absent.
describe('ExplorerPage — filtre « rejeu » gaté par la capability replay', () => {
  const FILTRE_REJEU = 'Filtrer par présence de rejeu 2D'

  afterEach(() => {
    useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
    routerState.search = {}
  })

  it('présent quand le titre déclare `replay`', async () => {
    setTitleCaps(['replay'])
    renderWithProviders(<ExplorerPage />)
    await waitFor(() => expect(screen.getByLabelText(FILTRE_REJEU)).toBeInTheDocument())
  })

  // LE FILTRE MASQUE NE DOIT PAS RESTER ACTIF (revue C-R1, constat C5). La portee est
  // memorisee par JOUEUR, pas par titre : « Avec rejeu » pose sur halo_infinite se
  // reinjectait sur halo_5 au chargement — liste filtree a zero, controle invisible, et
  // rien pour le corriger sinon tout effacer. Ici l'URL joue le role du miroir : meme
  // chemin de lecture, meme correctif.
  it('titre sans `replay` + ?replay=with : le filtre est neutralise, pas seulement masque', async () => {
    routerState.search = { replay: 'with' }
    const post = vi.spyOn(api, 'post').mockRejectedValue(new Error('hors sujet ici'))
    setTitleCaps(['ranked'])
    renderWithProviders(<ExplorerPage />)
    await waitFor(() => expect(screen.getByText('Matchs')).toBeInTheDocument())

    // (a) aucun filtre actif annonce : sinon l'utilisateur voit « Reinitialiser les
    //     filtres » sans pouvoir identifier lequel.
    expect(screen.queryByText('Réinitialiser les filtres')).not.toBeInTheDocument()
    // (b) et surtout : `replay_scope` n'est pas envoye au backend.
    await waitFor(() => expect(post).toHaveBeenCalled())
    for (const [, body] of post.mock.calls) {
      expect((body as Record<string, unknown>).replay_scope).toBeUndefined()
    }
    post.mockRestore()
  })

  it('titre AVEC `replay` + ?replay=with : le filtre reste actif et part au backend', async () => {
    routerState.search = { replay: 'with' }
    const post = vi.spyOn(api, 'post').mockRejectedValue(new Error('hors sujet ici'))
    setTitleCaps(['replay'])
    renderWithProviders(<ExplorerPage />)
    await waitFor(() => expect(post).toHaveBeenCalled())
    expect(
      post.mock.calls.some(([, body]) => (body as Record<string, unknown>).replay_scope === 'with'),
    ).toBe(true)
    post.mockRestore()
  })

  it('absent quand le titre ne la déclare pas', async () => {
    setTitleCaps(['ranked'])
    renderWithProviders(<ExplorerPage />)
    await waitFor(() => expect(screen.getByText('Matchs')).toBeInTheDocument())
    expect(screen.queryByLabelText(FILTRE_REJEU)).not.toBeInTheDocument()
  })
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
