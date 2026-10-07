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
 * Ce que ces tests cadenassent (écran unique) :
 *   - le NOUVEAU périmètre en cours de résolution : dans la colonne, la vignette reste le même
 *     nœud, la liste est `aria-busy`, estompée et dit « Mise à jour… » ; dans la lecture, le
 *     titre garde le nom de la carte, le fond reste le même `<img>`, la vue dit « Mise à jour… » ;
 *   - le nouveau périmètre RÉSOLU, la liste des cartes encore en relecture (revue L2-R3) : même
 *     vignette, même titre — c'est le placeholder de la liste des cartes qui répond ;
 *   - le nouveau périmètre en ÉCHEC (revue L2-R1) : le message d'échec SUR le fond, qui reste le
 *     même nœud — jamais une « Mise à jour… » qui ne viendra pas ;
 *   - un coéquipier INTROUVABLE (contrôle de parc L2-PARC-1) : « Coéquipier introuvable » SUR le
 *     fond, qui reste — jamais une relecture sans fin ;
 *   - changer de JOUEUR (revue L2-R2) : aucune réponse d'un joueur ne sert de placeholder à
 *     un autre, et aucune requête du nouveau joueur ne porte les `match_id` de l'ancien ;
 *   - changer de CARTE remet la vue à zéro.
 * Un seul `QueryClient` stable par test (cf. la note de `TacticalAnalysisView.fond.test.tsx`
 * sur `renderWithProviders`).
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
let paramsCourants: Record<string, string> = {}
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => paramsCourants,
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
const TITRE_RUELLES = 'Ruelles'

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
    {
      map_id: 'aquarius',
      map_name: 'Aquarius',
      map_name_fr: 'Aquarius',
      matchs: 12,
      victoires: 6,
      defaites: 6,
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
  points_ignores: 0,
}

/**
 * Le sort du périmètre d'une SESSION épinglée : `jamais` (il ne se résout pas pendant le
 * test — la fenêtre de relecture qu'on observe), `rejet` (la résolution échoue) ou `resolu`
 * (il rend `['m1']`, dont la GRILLE ne répond jamais).
 */
let perimetreSession: 'jamais' | 'rejet' | 'resolu' = 'jamais'

function perimetre(corps: FilterContextInput): Promise<unknown> {
  if (corps.filter_mode !== 'sessions') return Promise.resolve({ match_ids: ['m1', 'm2', 'm3'] })
  if (perimetreSession === 'rejet') return Promise.reject(new Error('résolution en échec'))
  if (perimetreSession === 'resolu') return Promise.resolve({ match_ids: ['m1'] })
  return new Promise(() => {})
}

let creerURL: ReturnType<typeof vi.spyOn>

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
  searchCourant = {}
  paramsCourants = { playerSlug: 'JGtm', titleSlug: 'halo_infinite' }
  perimetreSession = 'jamais'
  localStorage.clear()
  get.mockReset()
  get.mockResolvedValue({ teammates: [], enemies: [], total: 0 })
  getBlob.mockReset()
  getBlob.mockResolvedValue(new Blob(['png']))
  creerURL = vi.spyOn(URL, 'createObjectURL').mockReturnValue(URL_FOND)
  post.mockReset()
  post.mockImplementation((path: string, corps: unknown) => {
    // Le périmètre d'un AUTRE joueur ne se résout jamais pendant le test.
    if (path === '/players/Autre/filters/match-ids') return new Promise(() => {})
    if (path.endsWith('/filters/match-ids')) return perimetre(corps as FilterContextInput)
    if (path.endsWith('/tactical/maps')) {
      const ids = (corps as { match_ids: string[] }).match_ids
      return ids.length === 1 ? new Promise(() => {}) : Promise.resolve(PAGE)
    }
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
    /** Met un coéquipier dans la composition : la barre L2 écrit `eq` dans l'URL. */
    choisirCoequipier: (gamertag: string) => {
      searchCourant = { ...searchCourant, eq: gamertag }
      rendu.rerender(<TacticalPage />)
    },
    /** Passe sur un autre joueur : la route garde la page MONTÉE, seul le paramètre change. */
    changerDeJoueur: (playerSlug: string) => {
      paramsCourants = { ...paramsCourants, playerSlug }
      rendu.rerender(<TacticalPage />)
    },
  }
}

async function attendrePerimetreSession() {
  await waitFor(() =>
    expect(post).toHaveBeenCalledWith(
      '/players/JGtm/filters/match-ids',
      expect.objectContaining({ filter_mode: 'sessions' }),
    ),
  )
}

/** La lecture de Ruelles est chargée ; rend le `<img>` de son fond. */
async function analyseChargee(): Promise<HTMLImageElement> {
  expect(await screen.findByRole('region', { name: TITRE_RUELLES })).toBeInTheDocument()
  await screen.findByTestId('tactical-plan-canvas')
  await waitFor(() =>
    expect(screen.getByTestId('tactical-plan-frame').querySelector('img')).not.toBeNull(),
  )
  return screen.getByTestId('tactical-plan-frame').querySelector('img') as HTMLImageElement
}

/** Les requêtes tactiques postées pour `joueur`. */
function lecturesTactiques(joueur: string) {
  return post.mock.calls.filter(([path]) =>
    (path as string).startsWith(`/players/${joueur}/tactical/`),
  )
}

describe('TacticalPage — un changement de filtre garde la page à l’écran', () => {
  it('GRILLE : la vignette reste le même nœud, la grille dit « Mise à jour… »', async () => {
    const page = monter()
    const vignette = await screen.findByTestId('tactical-map-streets')

    page.cocherSession()
    await attendrePerimetreSession()

    expect(screen.queryByText(t.loading)).toBeNull()
    expect(screen.getByTestId('tactical-map-streets')).toBe(vignette)
    expect(vignette.isConnected).toBe(true)
    const grille = screen.getByTestId('tactical-grille')
    expect(grille).toHaveAttribute('aria-busy', 'true')
    // Revue L2-R7 : les compteurs de l'ANCIEN périmètre ne se présentent pas comme courants.
    expect(grille.className).toContain('opacity-50')
    expect(screen.getByTestId('tactical-grille-updating')).toHaveTextContent(t.analysisUpdating)
  })

  it('GRILLE, relecture finie : ni estompage, ni mention', async () => {
    monter()
    await screen.findByTestId('tactical-map-streets')
    const grille = screen.getByTestId('tactical-grille')
    expect(grille).toHaveAttribute('aria-busy', 'false')
    expect(grille.className).not.toContain('opacity-50')
    expect(screen.queryByTestId('tactical-grille-updating')).toBeNull()
  })

  // Revue L2-R3 : le périmètre arrive, la GRILLE relit sous une clé neuve. C'est son
  // placeholder à elle qui garde la vignette montée et le nom de la carte au titre.
  it('GRILLE : le nouveau périmètre résolu, la grille en relecture — même vignette', async () => {
    perimetreSession = 'resolu'
    const page = monter()
    const vignette = await screen.findByTestId('tactical-map-streets')

    page.cocherSession()
    await waitFor(() =>
      expect(post).toHaveBeenCalledWith(
        '/players/JGtm/tactical/maps',
        expect.objectContaining({ match_ids: ['m1'] }),
      ),
    )

    expect(screen.queryByText(t.loading)).toBeNull()
    expect(screen.getByTestId('tactical-map-streets')).toBe(vignette)
    expect(screen.getByTestId('tactical-grille')).toHaveAttribute('aria-busy', 'true')
  })

  it('ANALYSE : le nouveau périmètre résolu, la grille en relecture — le titre garde le nom', async () => {
    perimetreSession = 'resolu'
    searchCourant = { carte: 'streets' }
    const page = monter()
    await analyseChargee()

    page.cocherSession()
    await waitFor(() =>
      expect(post).toHaveBeenCalledWith(
        '/players/JGtm/tactical/maps',
        expect.objectContaining({ match_ids: ['m1'] }),
      ),
    )

    expect(screen.getByTestId('tactical-plan-title')).toHaveTextContent(TITRE_RUELLES)
  })

  it('ANALYSE SANS ?carte= : la carte lue d’office garde le même <img> pendant la relecture', async () => {
    const page = monter()
    const img = await analyseChargee()

    page.cocherSession()
    await attendrePerimetreSession()

    expect(screen.getByTestId('tactical-plan-frame').querySelector('img')).toBe(img)
    expect(img.isConnected).toBe(true)
    expect(screen.getByTestId('tactical-plan-title')).toHaveTextContent(TITRE_RUELLES)
  })

  it('ANALYSE : le titre garde le nom de la carte, le fond reste le même <img>', async () => {
    searchCourant = { carte: 'streets' }
    const page = monter()
    const img = await analyseChargee()

    page.cocherSession()
    await attendrePerimetreSession()

    // Jamais l'identifiant brut (« Plan de streets — … ») pendant la relecture.
    expect(screen.getByTestId('tactical-plan-title')).toHaveTextContent(TITRE_RUELLES)
    expect(screen.queryByTestId('tactical-analysis-pending')).toBeNull()
    expect(img.isConnected).toBe(true)
    expect(screen.getByTestId('tactical-plan-frame').querySelector('img')).toBe(img)
    expect(screen.getByTestId('tactical-analysis-updating')).toHaveTextContent(t.analysisUpdating)
    // Le fond de Ruelles n'est lu qu'une fois (vignette et plan partagent la même lecture) ; la
    // vignette d'Aquarius lit le sien.
    expect(getBlob.mock.calls.filter(([p]) => (p as string).includes('/tactical/streets/'))).toHaveLength(1)
  })

  // Revue L2-R1 : le périmètre en échec se dit SUR le fond, qui reste le même nœud — ni une
  // « Mise à jour… » qui ne viendra pas, ni l'ancien calque.
  it('le nouveau périmètre ÉCHOUE — le message sur le fond, qui reste', async () => {
    perimetreSession = 'rejet'
    searchCourant = { carte: 'streets' }
    const page = monter()
    const img = await analyseChargee()

    page.cocherSession()
    expect(await screen.findByTestId('tactical-plan-avis')).toHaveTextContent(t.analysisErrorTitle)
    expect(screen.queryByTestId('tactical-analysis-updating')).toBeNull()
    expect(screen.queryByTestId('tactical-plan-canvas')).toBeNull()
    expect(screen.getByTestId('tactical-analysis-body')).toHaveAttribute('aria-busy', 'false')
    expect(screen.getByTestId('tactical-plan-frame').querySelector('img')).toBe(img)
  })

  // Contrôle de parc L2-PARC-1 : un coéquipier INTROUVABLE (URL, scope mémorisé, liste rechargée
  // sans lui) se dit SUR le fond, qui reste — jamais une « Mise à jour… » sans fin, ni le calque de
  // l'ancienne composition.
  it('un coéquipier introuvable — « Coéquipier introuvable » sur le fond, qui reste', async () => {
    searchCourant = { carte: 'streets' }
    const page = monter()
    const img = await analyseChargee()

    page.choisirCoequipier('Inconnu')
    const avis = await screen.findByTestId('tactical-plan-avis')
    expect(avis).toHaveTextContent(t.unknownTeammateTitle)
    expect(avis).toHaveTextContent(t.unknownTeammateDescription('Inconnu'))
    // Ce n'est pas une panne : le message générique mentirait.
    expect(screen.queryByText(t.analysisErrorTitle)).toBeNull()
    expect(screen.queryByTestId('tactical-analysis-updating')).toBeNull()
    expect(screen.queryByTestId('tactical-plan-canvas')).toBeNull()
    expect(screen.getByTestId('tactical-plan-frame').querySelector('img')).toBe(img)
  })

  it('ouverte sur un coéquipier introuvable : le message sur le fond, aucune attente ni lecture', async () => {
    searchCourant = { carte: 'streets', eq: 'Inconnu' }
    monter()
    expect(await screen.findByTestId('tactical-plan-avis')).toHaveTextContent(t.unknownTeammateTitle)
    expect(screen.queryByTestId('tactical-analysis-pending')).toBeNull()
    expect(screen.getByTestId('tactical-analysis-body')).toHaveAttribute('aria-busy', 'false')
    expect(lecturesTactiques('JGtm').filter(([path]) => (path as string).endsWith('/raster'))).toEqual([])
  })

  it('CHANGER DE CARTE : la vue repart à zéro, aucune réponse de l’autre carte affichée', async () => {
    searchCourant = { carte: 'streets' }
    const page = monter()
    await screen.findByTestId('tactical-plan-canvas')
    const question = screen.getByRole('combobox', { name: t.pillReading }) as HTMLSelectElement
    fireEvent.change(question, { target: { value: 'kills' } })
    expect(question.value).toBe('kills')

    page.ouvrirCarte('aquarius')
    await waitFor(() =>
      expect(post).toHaveBeenCalledWith('/players/JGtm/tactical/aquarius/raster', expect.anything()),
    )

    const questionApres = screen.getByRole('combobox', { name: t.pillReading }) as HTMLSelectElement
    expect(questionApres.value).toBe('morts')
    expect(screen.queryByTestId('tactical-analysis-updating')).toBeNull()
    expect(screen.queryByTestId('tactical-plan-canvas')).toBeNull()
    expect(screen.getByTestId('tactical-analysis-pending')).toBeInTheDocument()
  })
})

// Revue L2-R2 : la page reste MONTÉE quand le joueur change (aucune `key` sur la route). La
// réponse précédente n'est gardée qu'au MÊME joueur : sinon la liste de `match_id` du joueur
// A partait sur les lectures du joueur B, et s'affichait comme sa réponse.
describe('TacticalPage — changer de joueur ne garde rien de l’ancien', () => {
  it('GRILLE : aucune lecture du nouveau joueur, aucune vignette de l’ancien', async () => {
    const page = monter()
    await screen.findByTestId('tactical-map-streets')

    page.changerDeJoueur('Autre')
    await waitFor(() =>
      expect(post).toHaveBeenCalledWith('/players/Autre/filters/match-ids', expect.anything()),
    )

    expect(lecturesTactiques('Autre')).toEqual([])
    expect(screen.queryByTestId('tactical-map-streets')).toBeNull()
    expect(screen.getByText(t.loading)).toBeInTheDocument()
  })

  it('ANALYSE : aucune lecture du nouveau joueur, aucun calque de l’ancien', async () => {
    searchCourant = { carte: 'streets' }
    const page = monter()
    await analyseChargee()

    page.changerDeJoueur('Autre')
    await waitFor(() =>
      expect(post).toHaveBeenCalledWith('/players/Autre/filters/match-ids', expect.anything()),
    )

    expect(lecturesTactiques('Autre')).toEqual([])
    expect(screen.queryByTestId('tactical-plan-canvas')).toBeNull()
    expect(screen.queryByTestId('tactical-analysis-updating')).toBeNull()
    // La carte de l'URL reste affichée : sans les cartes du nouveau joueur, rien n'est lu, et le
    // plan attend sous son indicateur.
    expect(screen.getByTestId('tactical-analysis-pending')).toBeInTheDocument()
    expect(screen.getByText(t.loading)).toBeInTheDocument()
  })
})
