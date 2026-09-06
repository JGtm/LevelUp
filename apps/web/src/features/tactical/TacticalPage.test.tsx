/**
 * La GRILLE des cartes de l'onglet Tactique, rendue depuis une réponse de contrat.
 *
 * Ce que ces tests cadenassent :
 *   - la grille rend une vignette par carte, dans l'ordre décidé par la logique ;
 *   - une carte SOUS LE PLANCHER est désaturée, DÉSACTIVÉE (`aria-disabled`), et porte sa
 *     raison en clair — jamais une vignette qui promet une ouverture qui n'arrivera pas ;
 *   - une carte ouvrable est un bouton qui DIT ce qu'il fait, et son clic écrit la carte
 *     choisie dans l'URL (aucun lien mort vers la route de la phase 5, qui n'existe pas) ;
 *   - aucune carte -> `EmptyState`, jamais une grille vide muette ;
 *   - le pied de carte sert la couverture ET la phrase du plancher.
 *
 * La garde de capability (`replay`) est testée avec la route, dans
 * `routes/.../ascension/tactique.test.tsx` : c'est là qu'elle vit.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { TacticalMapsPage } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { TacticalPage } from './TacticalPage'

const navigate = vi.fn()
let searchCourant: Record<string, unknown> = {}

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => navigate,
    useParams: () => ({ playerSlug: 'JGtm' }),
    useSearch: () => searchCourant,
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
      // La vignette demande son fond : sans image, la grille s'affiche quand même.
      getBlob: () => Promise.reject(new Error('pas de fond')),
    },
  }
})

const page: TacticalMapsPage = {
  plancher_matchs: 10,
  cartes: [
    {
      map_id: 'streets',
      map_name: 'Streets',
      map_name_fr: 'Ruelles',
      matchs: 24,
      victoires: 14,
      defaites: 9,
      sous_plancher: false,
    },
    {
      map_id: 'aquarius',
      map_name: 'Aquarius',
      map_name_fr: 'Aquarius',
      matchs: 9,
      victoires: 4,
      defaites: 5,
      sous_plancher: true,
    },
  ],
}

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
  searchCourant = {}
  navigate.mockReset()
  get.mockReset()
  get.mockResolvedValue(page)
})
afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

describe('TacticalPage — la grille des cartes', () => {
  it('rend une vignette par carte, la plus jouée en tête', async () => {
    renderWithProviders(<TacticalPage />)
    const streets = await screen.findByTestId('tactical-map-streets')
    const aquarius = screen.getByTestId('tactical-map-aquarius')
    expect(streets).toBeInTheDocument()
    expect(aquarius).toBeInTheDocument()
    // L'ordre du DOM EST l'ordre affiché : Streets (24 matchs) avant Aquarius (9).
    expect(streets.compareDocumentPosition(aquarius) & Node.DOCUMENT_POSITION_FOLLOWING)
      .toBeTruthy()
    // Le nom FR vient du contrat, pas du nom canonique.
    expect(streets.textContent).toContain('Ruelles')
    expect(streets.textContent).toContain('24 matchs')
  })

  it('sert le compte de victoires ET de défaites — jamais un taux seul', async () => {
    renderWithProviders(<TacticalPage />)
    const streets = await screen.findByTestId('tactical-map-streets')
    expect(streets.textContent).toContain('14 victoires')
    expect(streets.textContent).toContain('9 défaites')
  })

  it('carte SOUS LE PLANCHER : désaturée, désactivée, avec sa raison en clair', async () => {
    renderWithProviders(<TacticalPage />)
    const aquarius = await screen.findByTestId('tactical-map-aquarius')
    expect(aquarius).toHaveAttribute('aria-disabled', 'true')
    expect(aquarius).toBeDisabled()
    // Désaturation par des utilitaires SANS couleur : aucun token détourné.
    expect(aquarius.className).toContain('grayscale')
    expect(aquarius.className).toContain('opacity-60')
    expect(screen.getByTestId('tactical-map-plancher-aquarius').textContent).toContain(
      '9 matchs sur 10 requis',
    )
  })

  it('carte ouvrable : bouton actif qui DIT ce qu’il fait', async () => {
    renderWithProviders(<TacticalPage />)
    const streets = await screen.findByTestId('tactical-map-streets')
    expect(streets).not.toBeDisabled()
    expect(streets).toHaveAttribute('aria-label', 'Sélectionner Ruelles')
    expect(streets).toHaveAttribute('aria-pressed', 'false')
  })

  it('le clic écrit la carte choisie dans l’URL — aucun lien mort', async () => {
    renderWithProviders(<TacticalPage />)
    fireEvent.click(await screen.findByTestId('tactical-map-streets'))
    expect(navigate).toHaveBeenCalledTimes(1)
    const arg = navigate.mock.calls[0][0] as {
      search: (p: Record<string, unknown>) => Record<string, unknown>
    }
    expect(arg.search({})).toEqual({ carte: 'streets' })
  })

  it('une carte sous le plancher ne navigue nulle part', async () => {
    renderWithProviders(<TacticalPage />)
    fireEvent.click(await screen.findByTestId('tactical-map-aquarius'))
    expect(navigate).not.toHaveBeenCalled()
  })

  it('la carte sélectionnée dans l’URL est marquée comme telle', async () => {
    searchCourant = { carte: 'streets' }
    renderWithProviders(<TacticalPage />)
    const streets = await screen.findByTestId('tactical-map-streets')
    expect(streets).toHaveAttribute('aria-pressed', 'true')
    expect(streets.textContent).toContain('Carte sélectionnée')
  })

  it('pied de carte : la couverture ET la phrase du plancher', async () => {
    renderWithProviders(<TacticalPage />)
    const couverture = await screen.findByTestId('tactical-couverture')
    expect(couverture.textContent).toContain('2 cartes jouées')
    expect(couverture.textContent).toContain('33 matchs')
    expect(screen.getByText(/à partir de 10 matchs/)).toBeInTheDocument()
  })

  it('aucune carte : état vide explicite, jamais une grille muette', async () => {
    get.mockResolvedValue({ cartes: [], plancher_matchs: 10 })
    renderWithProviders(<TacticalPage />)
    expect(await screen.findByText('Aucune carte jouée')).toBeInTheDocument()
    expect(screen.queryByTestId('tactical-couverture')).toBeNull()
  })

  it('cartes nulles au contrat (slice Go vide) : même état vide, aucun plantage', async () => {
    get.mockResolvedValue({ cartes: null, plancher_matchs: 10 })
    renderWithProviders(<TacticalPage />)
    expect(await screen.findByText('Aucune carte jouée')).toBeInTheDocument()
  })

  it('lecture en échec : on le dit, on ne rend pas une grille vide', async () => {
    get.mockRejectedValue(new Error('503'))
    renderWithProviders(<TacticalPage />)
    expect(await screen.findByTestId('tactical-erreur')).toBeInTheDocument()
    expect(screen.queryByText('Aucune carte jouée')).toBeNull()
  })

  it('en anglais, les libellés et le nom canonique', async () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<TacticalPage />)
    const streets = await screen.findByTestId('tactical-map-streets')
    expect(streets.textContent).toContain('Streets')
    expect(streets.textContent).toContain('24 matches')
  })
})
