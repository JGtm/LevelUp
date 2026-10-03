/**
 * LE FOND NE BOUGE PLUS (retours rejeu L2, 2026-09-23) — le test de COMPORTEMENT du lot.
 *
 * Constat utilisateur : changer de question (« Où je meurs » → « Où je tue ») ou cocher une
 * session faisait clignoter le fond de carte. Cause : la vue rendait tout son corps, fond
 * compris, sous `!raster.isPending`, et aucune lecture ne gardait sa réponse précédente —
 * chaque nouvelle clé de cache démontait le `<img>` puis en montait un autre.
 *
 * Ce que ces tests cadenassent, avec la VRAIE `useTacticalRaster` (seul `api` est moqué) et
 * une réponse DIFFÉRÉE : pendant l'attente, aucun indicateur ne REMPLACE le plan, le MÊME
 * nœud `<img>` reste connecté, l'ancien calque est dit « Mise à jour… », et l'image n'est
 * demandée qu'UNE fois. Et le détail d'une cellule choisie ne repart pas pendant la
 * relecture (revue L2-R5) : il attend la nouvelle réponse et son `pas_m`. Le fichier voisin
 * (`TacticalAnalysisView.test.tsx`) moque la
 * lecture : il ne joue aucune transition de clé, et c'est pourquoi le défaut y passait.
 *
 * UN SEUL `QueryClient` PAR TEST, stable d'un `rerender` à l'autre : le wrapper de
 * `renderWithProviders` en recrée un à chaque rendu, ce qui viderait le cache au premier
 * `rerender` et ferait passer n'importe quel code.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'

import type { TacticalRaster } from '@/lib/api/types'
import { createTestQueryClient } from '@/test/render-utils'

import { getTacticalText } from './i18n'
import { TacticalAnalysisView } from './TacticalAnalysisView'

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

function raster(question: string, p95: number): TacticalRaster {
  return {
    map_id: 'streets',
    question,
    qui: 'moi',
    bornes: { min_x: 0, max_x: 100, min_y: 0, max_y: 50, valide: true },
    pas_m: 10,
    echelle: { p50: 1, p95, borne: p95, n_cellules: 1, symetrique: false },
    cellules: [
      { col: 2, lig: 1, valeur: 3, brut: 3, matchs: 4, matchs_victoire: 2, matchs_defaite: 2, centre_x: 25, centre_y: 15 },
    ],
    matchs_filtres: 10,
    matchs_retenus: 9,
    matchs_victoire: 5,
    matchs_defaite: 4,
    evenements_journal: 50,
    evenements_localises: 48,
    points_ignores: 0,
  }
}

/** Une réponse que le test libère quand il le décide. */
function differe<T>() {
  let liberer!: (valeur: T) => void
  const promesse = new Promise<T>((resolve) => {
    liberer = resolve
  })
  return { promesse, liberer }
}

/** Corps posté au raster : la réponse dépend de la question et du périmètre. */
type CorpsRaster = { question?: string; match_ids: string[] }
let repondre: (corps: CorpsRaster) => Promise<TacticalRaster>

let creerURL: ReturnType<typeof vi.spyOn>

beforeEach(() => {
  get.mockReset()
  // Pas de calage de fond : le cadre prend le rapport des bornes. Sans incidence ici.
  get.mockRejectedValue(new Error('pas de calage'))
  getBlob.mockReset()
  getBlob.mockResolvedValue(new Blob(['png']))
  creerURL = vi.spyOn(URL, 'createObjectURL').mockReturnValue(URL_FOND)
  repondre = () => Promise.resolve(raster('morts', 5))
  post.mockReset()
  post.mockImplementation((path: string, corps: unknown) =>
    path.endsWith('/tactical/streets/raster')
      ? repondre(corps as CorpsRaster)
      : path.endsWith('/tactical/streets/cellule')
        ? Promise.resolve({ contributions: [], matchs_non_ouvrables: 0 })
        : Promise.reject(new Error(`appel inattendu : ${path}`)),
  )
})
afterEach(() => {
  creerURL.mockRestore()
})

function monter(matchIds: string[] | null) {
  const client = createTestQueryClient()
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  const vue = (ids: string[] | null) => (
    <TacticalAnalysisView
      playerSlug="JGtm"
      mapId="streets"
      mapName="Ruelles"
      locale="fr"
      t={t}
      matchIds={ids}
      coequipiers={[]}
    />
  )
  const rendu = render(vue(matchIds), { wrapper })
  return { rerender: (ids: string[] | null) => rendu.rerender(vue(ids)) }
}

/** Le `<img>` du fond, une fois l'URL d'objet servie. */
async function fondCharge(): Promise<HTMLImageElement> {
  await waitFor(() =>
    expect(screen.getByTestId('tactical-plan-frame').querySelector('img')).not.toBeNull(),
  )
  return screen.getByTestId('tactical-plan-frame').querySelector('img') as HTMLImageElement
}

/** Le fond est là, et c'est LE MÊME nœud qu'avant. */
function memeFond(img: HTMLImageElement) {
  expect(img.isConnected).toBe(true)
  expect(screen.getByTestId('tactical-plan-frame').querySelector('img')).toBe(img)
  expect(img).toHaveAttribute('src', URL_FOND)
  expect(getBlob).toHaveBeenCalledTimes(1)
}

/** Pendant la relecture : rien ne remplace le plan, l'ancien calque est dit périmé. */
function enRelecture(img: HTMLImageElement) {
  expect(screen.queryByTestId('tactical-analysis-pending')).toBeNull()
  memeFond(img)
  expect(screen.getByTestId('tactical-analysis-updating')).toHaveTextContent(t.analysisUpdating)
  expect(screen.getByTestId('tactical-analysis-body')).toHaveAttribute('aria-busy', 'true')
  expect(screen.getByTestId('kpi-strip')).toBeInTheDocument()
}

function relectureFinie(img: HTMLImageElement) {
  expect(screen.queryByTestId('tactical-analysis-updating')).toBeNull()
  expect(screen.getByTestId('tactical-analysis-body')).toHaveAttribute('aria-busy', 'false')
  memeFond(img)
}

describe('TacticalAnalysisView — le fond de carte ne se démonte jamais après le premier chargement', () => {
  it('changement de QUESTION : même <img> pendant l’attente, ancien calque dit « Mise à jour… »', async () => {
    const kills = differe<TacticalRaster>()
    monter(['m1', 'm2'])
    await screen.findByTestId('kpi-strip')
    const img = await fondCharge()

    repondre = (corps) =>
      corps.question === 'kills' ? kills.promesse : Promise.resolve(raster('morts', 5))
    fireEvent.change(screen.getByRole('combobox', { name: t.questionLabel }), {
      target: { value: 'kills' },
    })
    await waitFor(() =>
      expect(post).toHaveBeenCalledWith(
        '/players/JGtm/tactical/streets/raster',
        expect.objectContaining({ question: 'kills' }),
      ),
    )

    enRelecture(img)
    // La légende décrit la réponse AFFICHÉE (l'ancienne question), jamais la demandée.
    expect(screen.getByTestId('tactical-plan-legend')).toHaveTextContent(t.units.morts)

    kills.liberer(raster('kills', 7))
    await waitFor(() =>
      expect(screen.getByTestId('tactical-plan-legend')).toHaveTextContent(t.units.kills),
    )
    relectureFinie(img)
  })

  it('changement de FILTRE (matchIds → null → autre liste) : même <img> tout du long', async () => {
    const autre = differe<TacticalRaster>()
    const vue = monter(['m1', 'm2'])
    await screen.findByTestId('kpi-strip')
    const img = await fondCharge()

    // Le périmètre se résout : la liste passe par `null` (non résolu).
    vue.rerender(null)
    enRelecture(img)

    // Le nouveau périmètre arrive, sa réponse est différée.
    repondre = (corps) =>
      corps.match_ids.length === 1 ? autre.promesse : Promise.resolve(raster('morts', 5))
    vue.rerender(['m1'])
    await waitFor(() =>
      expect(post).toHaveBeenCalledWith(
        '/players/JGtm/tactical/streets/raster',
        expect.objectContaining({ match_ids: ['m1'] }),
      ),
    )
    enRelecture(img)

    autre.liberer(raster('morts', 9))
    await waitFor(() =>
      expect(screen.queryByTestId('tactical-analysis-updating')).toBeNull(),
    )
    relectureFinie(img)
  })

  // Revue L2-R5 (a) : pendant une relecture, la réponse affichée est la PRÉCÉDENTE ; son
  // `pas_m` n'adresse pas forcément la même cellule dans la nouvelle. Le détail attend.
  it('détail de cellule : aucun /cellule pendant une relecture de filtre, reparti ensuite', async () => {
    // jsdom ne mesure rien : sans taille de canevas, le clic ne désigne aucune cellule.
    const largeur = vi.spyOn(Element.prototype, 'clientWidth', 'get').mockReturnValue(100)
    const hauteur = vi.spyOn(Element.prototype, 'clientHeight', 'get').mockReturnValue(50)
    const contexte = vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null)
    const detail = () =>
      post.mock.calls.filter(([path]) => (path as string).endsWith('/tactical/streets/cellule'))
    try {
      const autre = differe<TacticalRaster>()
      const vue = monter(['m1', 'm2'])
      await screen.findByTestId('kpi-strip')
      fireEvent.click(screen.getByTestId('tactical-plan-canvas'), { clientX: 30, clientY: 20 })
      await waitFor(() => expect(detail()).toHaveLength(1))

      repondre = (corps) =>
        corps.match_ids.length === 1 ? autre.promesse : Promise.resolve(raster('morts', 5))
      vue.rerender(null)
      vue.rerender(['m1'])
      await waitFor(() =>
        expect(post).toHaveBeenCalledWith(
          '/players/JGtm/tactical/streets/raster',
          expect.objectContaining({ match_ids: ['m1'] }),
        ),
      )
      expect(screen.getByTestId('tactical-analysis-updating')).toBeInTheDocument()
      expect(detail()).toHaveLength(1)

      autre.liberer(raster('morts', 9))
      await waitFor(() => expect(detail()).toHaveLength(2))
      expect(detail()[1][1]).toEqual(expect.objectContaining({ match_ids: ['m1'] }))
    } finally {
      largeur.mockRestore()
      hauteur.mockRestore()
      contexte.mockRestore()
    }
  })

  it('PREMIER CHARGEMENT : le fond est posé, l’indicateur PAR-DESSUS, puis le calque sur le même fond', async () => {
    const premiere = differe<TacticalRaster>()
    repondre = () => premiere.promesse
    monter(['m1', 'm2'])

    const img = await fondCharge()
    const cadre = screen.getByTestId('tactical-plan-frame')
    expect(within(cadre).getByTestId('tactical-analysis-pending')).toBeInTheDocument()
    expect(screen.queryByTestId('kpi-strip')).toBeNull()
    expect(screen.getByTestId('tactical-analysis-body')).toHaveAttribute('aria-busy', 'true')

    premiere.liberer(raster('morts', 5))
    await screen.findByTestId('kpi-strip')
    expect(screen.queryByTestId('tactical-analysis-pending')).toBeNull()
    memeFond(img)
  })
})
