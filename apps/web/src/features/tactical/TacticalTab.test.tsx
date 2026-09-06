/**
 * LA SECONDE DES DEUX PORTES : la route elle-même.
 *
 * Masquer l'onglet dans la barre ne suffit pas — une URL s'ouvre directement, se
 * partage, et reste dans un signet. Ces tests vérifient que pour un titre SANS `replay` :
 *   - la page n'est PAS montée (donc aucune requête n'est émise pour un titre qui n'a
 *     rien à servir) ;
 *   - l'utilisateur lit une explication, pas une page vide.
 * Et qu'à l'inverse, un titre qui déclare `replay` obtient bien la grille.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { TacticalTab } from './TacticalTab'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'JGtm' }),
    useSearch: () => ({}),
  }
})

const get = vi.fn()
vi.mock('@/lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api/client')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      get: (path: string) => get(path),
      getBlob: () => Promise.reject(new Error('pas de fond')),
    },
  }
})

function poserTitre(capabilities: string[]) {
  useAppShellStore.setState({
    locale: 'fr',
    currentTitleSlug: 'un_titre',
    availableTitles: [
      { slug: 'un_titre', name: 'Un titre', capabilities },
    ] as never,
  })
}

beforeEach(() => {
  get.mockReset()
  get.mockResolvedValue({ cartes: [], plancher_matchs: 10 })
})
afterEach(() => {
  useAppShellStore.setState({ locale: 'fr', availableTitles: [] })
})

describe('TacticalTab — la porte de titre sur la route', () => {
  it("titre SANS `replay` : la page n'est pas montée, et aucune requête n'est émise", () => {
    poserTitre(['matchmaking'])
    renderWithProviders(<TacticalTab />)
    expect(screen.queryByText('Cartes jouées')).toBeNull()
    expect(get).not.toHaveBeenCalled()
  })

  it('titre SANS `replay` : on explique, on ne laisse pas une page vide', () => {
    poserTitre(['matchmaking'])
    renderWithProviders(<TacticalTab />)
    expect(screen.getByText('Indisponible pour ce titre')).toBeInTheDocument()
    expect(screen.getByText(/ne fournit pas le rejeu 2D des matchs/)).toBeInTheDocument()
  })

  it('titre AVEC `replay` : la grille est servie', async () => {
    poserTitre(['matchmaking', 'replay'])
    renderWithProviders(<TacticalTab />)
    expect(await screen.findByText('Cartes jouées')).toBeInTheDocument()
    expect(get).toHaveBeenCalled()
  })
})
