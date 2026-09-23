/**
 * TacticalPage — un changement de FILTRE ne reconstruit plus la page (retours rejeu L2,
 * 2026-09-23).
 *
 * Constat : cocher une session (ou « Analyser » une période) créait une clé de périmètre
 * sans donnée ; la liste de `match_id` retombait à `null`, la grille à « Chargement… » —
 * chaque vignette (fond + mini-plan) démontée puis remontée — et, sur l'écran d'analyse,
 * le titre retombait un instant sur l'identifiant brut de la carte, le fond du plan étant
 * démonté avec la lecture.
 *
 * Ce que ces tests cadenassent, la résolution du NOUVEAU périmètre étant différée : sur la
 * grille, la vignette reste le même nœud (et la grille dit `aria-busy`) ; sur l'écran
 * d'analyse, le titre garde le nom de la carte, le fond reste le même `<img>`, et la vue dit
 * « Mise à jour… ». Et à l'inverse, changer de CARTE remet la vue à zéro : la réponse d'une
 * carte ne sert jamais de placeholder à une autre. Un seul `QueryClient` stable par test
 * (cf. la note de `TacticalAnalysisView.fond.test.tsx` sur `renderWithProviders`).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'

import type { FilterContextInput, TacticalMapsPage, TacticalRaster } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { createTestQueryClient } from '@/test/render-utils'

import { getTacticalText } from './i18n'
import { TacticalPage } from './TacticalPage'

let searchCourant: Record<string, unknown> = {}
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'JGtm', titleSlug: 'halo_infinite' }),
    useSearch: () => searchCourant,
  }
})

const get = vi.fn()
const post = vi.fn()
const getBlob = vi.fn()
vi.mock('@/lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api/client')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      get: (path: string) => get(path),
      post: (path: string, body: unknown) => post(path, body),
      getBlob: (path: string) => getBlob(path),
    },
  }
})

const t = getTacticalText('fr')
const URL_FOND = 'blob:tactique/streets'

const PAGE: TacticalMapsPage = {
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
  ],
}

const RASTER: TacticalRaster = {
  map_id: 'streets',
  question: 'morts',
  qui: 'moi',
  bornes: { min_x: 0, max_x: 100, min_y: 0, max_y: 50, valide: true },
  pas_m: 10,
  echelle: { p50: 1, p95: 5, borne: 5, n_cellules: 1, symetrique: false },
  cellules: [
    { col: 2, lig: 1, valeur: 3, brut: 3, matchs: 4, matchs_victoire: 2, matchs_defaite: 2, centre_x: 25, centre_y: 15 },
  ],
  matchs_filtres: 3,
  matchs_retenus: 3,
  matchs_victoire: 2,
  matchs_defaite: 1,
  evenements_journal: 10,
  evenements_localises: 10,
  points_ignores: 0,
}

let creerURL: ReturnType<typeof vi.spyOn>

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
  searchCourant = {}
  localStorage.clear()
  get.mockReset()
  get.mockResolvedValue({ teammates: [], enemies: [], total: 0 })
  getBlob.mockReset()
  getBlob.mockResolvedValue(new Blob(['png']))
  creerURL = vi.spyOn(URL, 'createObjectURL').mockReturnValue(URL_FOND)
  post.mockReset()
  post.mockImplementation((path: string, corps: unknown) => {
    if (path.endsWith('/filters/match-ids')) {
      // Le périmètre d'une SESSION épinglée ne se résout jamais pendant le test : c'est
      // la fenêtre de relecture qu'on observe.
      return (corps as FilterContextInput).filter_mode === 'sessions'
        ? new Promise(() => {})
        : Promise.resolve({ match_ids: ['m1', 'm2', 'm3'] })
    }
    if (path.endsWith('/tactical/maps')) return Promise.resolve(PAGE)
    if (path.endsWith('/tactical/streets/raster')) return Promise.resolve(RASTER)
    // La lecture d'une AUTRE carte ne répond pas pendant le test : on observe ce que la vue
    // montre en l'attendant.
    if (path.endsWith('/tactical/aquarius/raster')) return new Promise(() => {})
    return Promise.reject(new Error(`appel inattendu : ${path}`))
  })
})
afterEach(() => {
  creerURL.mockRestore()
})

function monter() {
  const client = createTestQueryClient()
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  const rendu = render(<TacticalPage />, { wrapper })
  return {
    /** Ouvre une autre carte (`?carte=`), comme un clic sur sa vignette. */
    ouvrirCarte: (carte: string) => {
      searchCourant = { ...searchCourant, carte }
      rendu.rerender(<TacticalPage />)
    },
    /** Épingle une session : la barre L2 écrit `ses` dans l'URL, la page se re-rend. */
    cocherSession: () => {
      searchCourant = { ...searchCourant, ses: 'Session du 3 mars' }
      rendu.rerender(<TacticalPage />)
    },
  }
}

describe('TacticalPage — un changement de filtre garde la page à l’écran', () => {
  it('GRILLE : la vignette reste le même nœud pendant la résolution du nouveau périmètre', async () => {
    const page = monter()
    const vignette = await screen.findByTestId('tactical-map-streets')

    page.cocherSession()
    await waitFor(() =>
      expect(post).toHaveBeenCalledWith(
        '/players/JGtm/filters/match-ids',
        expect.objectContaining({ filter_mode: 'sessions' }),
      ),
    )

    expect(screen.queryByText(t.loading)).toBeNull()
    expect(screen.getByTestId('tactical-map-streets')).toBe(vignette)
    expect(vignette.isConnected).toBe(true)
    expect(screen.getByTestId('tactical-grille')).toHaveAttribute('aria-busy', 'true')
  })

  it('ANALYSE : le titre garde le nom de la carte, le fond reste le même <img>', async () => {
    searchCourant = { carte: 'streets' }
    const page = monter()
    const titre = 'Plan de Ruelles — Où je meurs'
    expect(await screen.findByText(titre)).toBeInTheDocument()
    await screen.findByTestId('kpi-strip')
    await waitFor(() =>
      expect(screen.getByTestId('tactical-plan-frame').querySelector('img')).not.toBeNull(),
    )
    const img = screen.getByTestId('tactical-plan-frame').querySelector('img') as HTMLImageElement

    page.cocherSession()
    await waitFor(() =>
      expect(post).toHaveBeenCalledWith(
        '/players/JGtm/filters/match-ids',
        expect.objectContaining({ filter_mode: 'sessions' }),
      ),
    )

    // Jamais l'identifiant brut (« Plan de streets — … ») pendant la relecture.
    expect(screen.getByTestId('tactical-analysis-title')).toHaveTextContent(titre)
    expect(screen.queryByTestId('tactical-analysis-pending')).toBeNull()
    expect(img.isConnected).toBe(true)
    expect(screen.getByTestId('tactical-plan-frame').querySelector('img')).toBe(img)
    expect(screen.getByTestId('tactical-analysis-updating')).toHaveTextContent(t.analysisUpdating)
    expect(getBlob).toHaveBeenCalledTimes(1)
  })

  // `key={scope.carte}` : la réponse d'une carte ne sert JAMAIS de placeholder à une autre
  // (les grappes de spawn sont propres à une carte), et l'état local repart à zéro.
  it('CHANGER DE CARTE : la vue repart à zéro, aucune réponse de l’autre carte affichée', async () => {
    searchCourant = { carte: 'streets' }
    const page = monter()
    await screen.findByTestId('kpi-strip')
    const question = screen.getByRole('combobox', { name: t.questionLabel }) as HTMLSelectElement
    fireEvent.change(question, { target: { value: 'kills' } })
    expect(question.value).toBe('kills')

    page.ouvrirCarte('aquarius')
    await waitFor(() =>
      expect(post).toHaveBeenCalledWith('/players/JGtm/tactical/aquarius/raster', expect.anything()),
    )

    const questionApres = screen.getByRole('combobox', { name: t.questionLabel }) as HTMLSelectElement
    expect(questionApres.value).toBe('morts')
    expect(screen.queryByTestId('tactical-analysis-updating')).toBeNull()
    expect(screen.queryByTestId('kpi-strip')).toBeNull()
    expect(screen.getByTestId('tactical-analysis-pending')).toBeInTheDocument()
  })
})
