/**
 * TacticalAnalysisView — la lecture « Réapparitions adverses » : où l'adversaire réapparaît quand l'équipe
 * tient une zone.
 *
 * Ce que ces tests cadenassent, `useTacticalRaster` MOQUÉ (il porte le catalogue de zones) et l'API moquée :
 *   - la lecture est proposée dans la pilule « Lecture » seulement si le titre déclare `spawn_forecast` ;
 *   - elle remplace le plan et la colonne de zone ; en revenir rend la carte de placement ;
 *   - sans zone choisie, la requête part avec la zone de SONDE (la référence ne dépend pas de la zone) ; un clic
 *     sur une zone, ou la pilule, la choisit ; un re-clic la retire ;
 *   - servie : zones teintées, zone tenue hachurée, noms avec écart en points, barres comparées, légende ;
 *   - vide : bloc pointillé avec jauges, plan en gris, référence « pour mémoire », jamais d'action impossible ;
 *   - les zones sont rognées à l'image plan quand la carte en a une, entières sinon.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, waitFor, within } from '@testing-library/react'

import type { ReplayCalloutZone, TacticalRaster, TacticalReapparitionsAdverses } from '@/lib/api/types'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import type { CarteEffective } from './cockpit.logic'
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

const useTacticalRaster = vi.fn()
const useTacticalCellule = vi.fn<(...args: unknown[]) => { data?: unknown; isPending: boolean }>(() => ({
  data: undefined,
  isPending: true,
}))
vi.mock('./queries', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./queries')>()
  return {
    ...actual,
    useTacticalRaster: (...args: unknown[]) => useTacticalRaster(...args),
    useTacticalCellule: (...args: unknown[]) => useTacticalCellule(...args),
  }
})
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return { ...actual, Link: ({ children }: { children?: React.ReactNode }) => <a href="/rejeu">{children}</a> }
})

const t = getTacticalText('fr')
const LUE: CarteEffective = { mapId: 'streets', origine: 'url' }
const CALAGE = { metersPerPixel: 1, originX: 0, originY: 50, widthPx: 100, heightPx: 50, convention: 'xy' }

function zone(volume_index: number, fr: string, x0: number, big = true): ReplayCalloutZone {
  return {
    name: fr,
    fr,
    en: fr,
    volume_index,
    big,
    polygon: [
      [x0, 5],
      [x0 + 20, 5],
      [x0 + 20, 30],
      [x0, 30],
    ],
    x: x0 + 10,
    y: 17,
    z: 0,
    z_bottom: -1,
    z_top: 3,
  }
}

const RASTER: TacticalRaster = {
  map_id: 'streets',
  question: 'morts',
  qui: 'moi',
  bornes: { min_x: 0, max_x: 100, min_y: 0, max_y: 50, valide: true },
  pas_m: 10,
  echelle: { p50: 1, p95: 5, borne: 5, n_cellules: 1, symetrique: false },
  cellules: [{ col: 5, lig: 2, valeur: 5, brut: 5, matchs: 6, matchs_victoire: 3, matchs_defaite: 3, centre_x: 55, centre_y: 25 }],
  zones: [zone(7, 'Pont', 60), zone(3, 'Fosse', 10), zone(5, 'Hydro', 35)],
  matchs_filtres: 50,
  matchs_retenus: 45,
  matchs_victoire: 20,
  matchs_defaite: 25,
  matchs_en_attente: 0,
  points_ignores: 0,
}

function repartition(zones: Array<[number, string, number, number]>, extra = {}) {
  const total = zones.reduce((n, z) => n + z[2], 0)
  return {
    matchs: 9,
    reapparitions: total,
    sans_zone: 0,
    suffisante: true,
    zones: zones.map(([volume_index, nom, reapparitions, part]) => ({ volume_index, nom, reapparitions, part })),
    ...extra,
  }
}

const REFERENCE = repartition([
  [5, 'Hydro', 40, 0.4],
  [7, 'Pont', 30, 0.3],
  [3, 'Fosse', 30, 0.3],
])

function reponse(over: Partial<TacticalReapparitionsAdverses> = {}): TacticalReapparitionsAdverses {
  return {
    etat: 'servie',
    map_id: 'streets',
    matchs_filtres: 12,
    matchs_retenus: 10,
    matchs_sans_equipe: 0,
    plancher: { matchs: 3, reapparitions: 20, joueurs_pour_tenir: 2 },
    reference: REFERENCE,
    tenue: repartition([
      [7, 'Pont', 24, 0.6],
      [5, 'Hydro', 16, 0.4],
    ]),
    zone: { volume_index: 3, nom: 'Fosse' },
    ...over,
  }
}

/** La réponse du serveur à chaque zone demandée ; absente = la valeur par défaut `reponse()` de la zone. */
let reponses: Record<number, TacticalReapparitionsAdverses>

function servirCapabilities(capabilities: string[] | null) {
  useAppShellStore.setState({
    currentTitleSlug: 'un_titre',
    availableTitles: (capabilities === null
      ? []
      : [{ slug: 'un_titre', name: 'T', status: 'active', capabilities, is_default: false, effective_hp_to_kill: 225 }]) as unknown as ReturnType<
      typeof useAppShellStore.getState
    >['availableTitles'],
  })
}

beforeEach(() => {
  reponses = {}
  useTacticalRaster.mockReturnValue({ data: RASTER, isPending: false, isError: false })
  get.mockResolvedValue({ calibration: CALAGE, planImage: 'streets.plan.webp' })
  getBlob.mockResolvedValue(new Blob(['webp']))
  post.mockImplementation((_chemin: string, corps: { volume_index: number }) =>
    Promise.resolve(reponses[corps.volume_index] ?? reponse({ zone: { volume_index: corps.volume_index, nom: `Zone ${corps.volume_index}` } })),
  )
  vi.spyOn(URL, 'createObjectURL').mockImplementation(() => 'blob:plan')
})

afterEach(() => {
  vi.clearAllMocks()
  vi.restoreAllMocks()
  servirCapabilities(null)
})

function monter() {
  return renderWithProviders(
    <TacticalAnalysisView
      playerSlug="JGtm"
      carte={LUE}
      mapName="Ruelles"
      locale="fr"
      t={t}
      matchIds={['m1', 'm2']}
      coequipiers={['x9']}
    />,
  )
}

/** Monte la vue et bascule la pilule « Lecture » sur « Réapparitions adverses ». */
async function ouvrirLaLecture(attendreLesZones = true) {
  monter()
  fireEvent.change(screen.getByRole('combobox', { name: t.pillReading }), { target: { value: 'reapparitions_adverses' } })
  await screen.findByTestId('tactical-reapp-plan')
  // Les zones ne se peignent qu'une fois la décision de rognage prise (plan chargé, ou absent).
  if (attendreLesZones) await screen.findByTestId('tactical-reapp-zones')
}

const corpsPostes = () => post.mock.calls.map((c) => c[1] as { match_ids: string[]; coequipiers: string[]; volume_index: number })
const chemin = (volume: number) => document.querySelector(`path[data-zone="${volume}"]`) as SVGPathElement

describe('la pilule « Lecture »', () => {
  it('propose « Réapparitions adverses » quand le titre déclare spawn_forecast, et pas sinon', () => {
    servirCapabilities(['replay', 'spawn_forecast'])
    const { unmount } = monter()
    const options = within(screen.getByRole('combobox', { name: t.pillReading })).getAllByRole('option')
    expect(options.at(-1)).toHaveTextContent('Réapparitions adverses')
    unmount()
    servirCapabilities(['replay'])
    monter()
    expect(within(screen.getByRole('combobox', { name: t.pillReading })).queryByRole('option', { name: 'Réapparitions adverses' })).toBeNull()
  })

  it('la lecture remplace le plan de placement ; une autre lecture le rend', async () => {
    await ouvrirLaLecture()
    expect(screen.queryByTestId('tactical-plan-canvas')).toBeNull()
    expect(screen.getByTestId('tactical-reapp-colonne')).toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: t.pillReading })).toHaveValue('reapparitions_adverses')
    fireEvent.change(screen.getByRole('combobox', { name: t.pillReading }), { target: { value: 'kills' } })
    expect(screen.queryByTestId('tactical-reapp-plan')).toBeNull()
    expect(screen.getByTestId('tactical-plan-canvas')).toBeInTheDocument()
  })
})

describe('sans zone tenue choisie', () => {
  it('lit la référence par la zone de sonde (le plus petit indice), avec le périmètre et la composition', async () => {
    await ouvrirLaLecture()
    await screen.findByTestId('tactical-reapp-barres')
    expect(corpsPostes()).toEqual([{ match_ids: ['m1', 'm2'], coequipiers: ['x9'], volume_index: 3 }])
    expect(post.mock.calls[0][0]).toBe('/players/JGtm/tactical/streets/reapparitions-adverses')
  })

  it('le bloc pointillé dit comment choisir la zone ; les barres sont celles de la référence seule', async () => {
    await ouvrirLaLecture()
    const bloc = await screen.findByTestId('tactical-reapp-vide')
    expect(bloc).toHaveTextContent(t.reappEmptyNoneTitle)
    const barres = screen.getByTestId('tactical-reapp-barres')
    expect(within(barres).getAllByRole('listitem').map((l) => l.getAttribute('data-testid'))).toEqual([
      'tactical-reapp-barre-5',
      'tactical-reapp-barre-3',
      'tactical-reapp-barre-7',
    ])
    expect(screen.getByTestId('tactical-reapp-zone-nom')).toHaveTextContent(t.reappHeldNone)
    expect(screen.getByTestId('tactical-reapp-legende')).toHaveTextContent(t.reappLegendGreyNoZone)
  })
})

describe('une zone tenue servie', () => {
  async function choisirPont() {
    await ouvrirLaLecture()
    await screen.findByTestId('tactical-reapp-barres')
    reponses[7] = reponse({ zone: { volume_index: 7, nom: 'Pont' } })
    fireEvent.click(chemin(7))
    await waitFor(() => expect(screen.getByTestId('tactical-reapp-zone-nom')).toHaveTextContent('Pont'))
  }

  it('un clic sur le plan choisit la zone : la requête repart avec son indice, la pilule la suit', async () => {
    await choisirPont()
    expect(corpsPostes().at(-1)?.volume_index).toBe(7)
    expect(screen.getByTestId('tactical-reapp-zone-select')).toHaveValue('7')
  })

  it('les barres comparent la tenue (épaisse) et le sans condition (fine), triées sur la tenue', async () => {
    await choisirPont()
    const barres = await screen.findByTestId('tactical-reapp-barres')
    const lignes = within(barres).getAllByRole('listitem')
    expect(lignes.map((l) => l.getAttribute('data-testid'))).toEqual(['tactical-reapp-barre-7', 'tactical-reapp-barre-5', 'tactical-reapp-barre-3'])
    // Pont : 60 % pendant la tenue, 30 % sans condition ; Fosse : absente de la tenue.
    expect(lignes[0]).toHaveTextContent(/60\s%/)
    expect(lignes[0]).toHaveTextContent(/30\s%/)
    expect(lignes[2]).toHaveTextContent(/0\s%/)
    expect(screen.queryByTestId('tactical-reapp-vide')).toBeNull()
    expect(screen.getByTestId('tactical-reapp-colonne')).toHaveTextContent(t.reappBarHeld('Pont', 40, 9))
  })

  it('le plan nomme les zones principales avec leur écart en points, et la zone tenue', async () => {
    await choisirPont()
    const hydro = await screen.findByTestId('tactical-reapp-etiquette-5')
    expect(hydro).toHaveTextContent('Hydro')
    expect(hydro).toHaveTextContent(/40\s%/)
    // 40 % pendant la tenue, 40 % sans condition : écart nul.
    expect(within(hydro).getByText('=')).toBeInTheDocument()
    expect(screen.getByTestId('tactical-reapp-etiquette-tenue')).toHaveTextContent(t.reappLabelHeld('Pont'))
    expect(screen.getByTestId('tactical-reapp-legende')).toHaveTextContent(t.reappLegendHeld(2))
  })

  it('la zone tenue est hachurée, les autres teintées par leur part', async () => {
    await choisirPont()
    await screen.findByTestId('tactical-reapp-etiquette-tenue')
    expect(chemin(7).getAttribute('style')).toMatch(/fill: url\(["']?#[^)]*hachure/)
    const hydro = Number(chemin(5).style.fillOpacity)
    const fosse = Number(chemin(3).style.fillOpacity)
    expect(hydro).toBeGreaterThan(fosse)
  })

  it('un re-clic sur la zone tenue la retire : retour à la sonde, plus de zone nommée', async () => {
    await choisirPont()
    fireEvent.click(chemin(7))
    await waitFor(() => expect(screen.getByTestId('tactical-reapp-zone-nom')).toHaveTextContent(t.reappHeldNone))
    expect(screen.getByTestId('tactical-reapp-zone-select')).toHaveValue('')
    expect(corpsPostes().at(-1)?.volume_index).toBe(3)
  })

  it('la pilule « Zone tenue » choisit une zone au clavier, par nom', async () => {
    await ouvrirLaLecture()
    await screen.findByTestId('tactical-reapp-barres')
    const pilule = screen.getByRole('combobox', { name: t.reappPillHeld })
    expect(within(pilule).getAllByRole('option').map((o) => o.textContent)).toEqual([t.reappHeldNone, 'Fosse', 'Hydro', 'Pont'])
    fireEvent.change(pilule, { target: { value: '5' } })
    await waitFor(() => expect(corpsPostes().at(-1)?.volume_index).toBe(5))
  })
})

describe('l’état vide', () => {
  it('sous les planchers : bloc pointillé, deux jauges, plan en gris, référence pour mémoire', async () => {
    reponses[7] = reponse({
      etat: 'vide',
      raison: 'sous_plancher_matchs',
      zone: { volume_index: 7, nom: 'Pont' },
      tenue: repartition([], { matchs: 2, reapparitions: 12, suffisante: false, raison: 'sous_plancher_matchs' }),
    })
    await ouvrirLaLecture()
    await screen.findByTestId('tactical-reapp-barres')
    fireEvent.click(chemin(7))
    const bloc = await screen.findByText(t.reappEmptyFloorTitle('Pont'))
    expect(bloc.closest('[data-testid="tactical-reapp-vide"]')).not.toBeNull()
    expect(screen.getByTestId('tactical-reapp-jauge-matchs')).toHaveTextContent('2 sur 3 matchs')
    expect(screen.getByTestId('tactical-reapp-jauge-matchs')).toHaveAttribute('data-atteint', 'false')
    expect(screen.getByTestId('tactical-reapp-jauge-reapparitions')).toHaveTextContent('12 sur 20 réapparitions')
    expect(screen.getByTestId('tactical-reapp-plan-vide')).toHaveTextContent(t.reappPlanEmpty('Pont'))
    expect(screen.getByText(t.reappSubReference)).toBeInTheDocument()
    expect(screen.getByTestId('tactical-reapp-legende')).toHaveTextContent(t.reappLegendGrey)
    // Aucune barre « tenue » (épaisse) : la référence seule, en gris.
    expect(within(screen.getByTestId('tactical-reapp-barres')).getAllByRole('listitem')).toHaveLength(3)
  })

  it('une cause nommée (équipe inconnue) : un titre et une phrase, sans jauge', async () => {
    reponses[3] = reponse({ etat: 'vide', raison: 'equipe_inconnue', tenue: repartition([], { suffisante: false }) })
    await ouvrirLaLecture()
    await screen.findByTestId('tactical-reapp-barres')
    fireEvent.change(screen.getByRole('combobox', { name: t.reappPillHeld }), { target: { value: '3' } })
    expect(await screen.findByText(t.reappEmptyTeamTitle)).toBeInTheDocument()
    expect(screen.queryByTestId('tactical-reapp-jauge-matchs')).toBeNull()
  })

  it('une panne du serveur est dite, sans faux état vide', async () => {
    post.mockRejectedValue(new Error('503'))
    await ouvrirLaLecture()
    expect(await screen.findByRole('alert')).toHaveTextContent(t.reappError)
  })

  it('une carte sans zone nommée : le bloc le dit, aucune requête ne part', async () => {
    useTacticalRaster.mockReturnValue({ data: { ...RASTER, zones: [] }, isPending: false, isError: false })
    await ouvrirLaLecture(false)
    expect(await screen.findByText(t.reappEmptyNoZonesTitle)).toBeInTheDocument()
    expect(post).not.toHaveBeenCalled()
  })
})

describe('le rognage au plan', () => {
  it('une carte avec vue plan : les zones portent le masque alpha de l’image plan', async () => {
    await ouvrirLaLecture()
    const svg = await screen.findByTestId('tactical-reapp-zones')
    await waitFor(() => expect(svg.getAttribute('data-rogne')).toBe('plan'))
    const masque = svg.querySelector('mask')
    expect(masque?.querySelector('image')?.getAttribute('href')).toBe('blob:plan')
    expect(svg.querySelector('g')?.getAttribute('mask')).toMatch(/^url\(#plan-mask-/)
    // L'image plan demandée est celle de la vue plan, quel que soit le fond affiché.
    expect(getBlob).toHaveBeenCalledWith('/players/JGtm/tactical/streets/background.png?variante=plan')
  })

  it('une carte sans vue plan : zones entières, aucun masque', async () => {
    get.mockResolvedValue({ calibration: CALAGE })
    await ouvrirLaLecture()
    const svg = await screen.findByTestId('tactical-reapp-zones')
    expect(svg.getAttribute('data-rogne')).toBe('non')
    expect(svg.querySelector('mask')).toBeNull()
    expect(svg.querySelector('g')?.hasAttribute('mask')).toBe(false)
    expect(getBlob.mock.calls.map((c) => c[0] as string).filter((p) => p.includes('variante=plan'))).toEqual([])
  })

  it('image plan en échec : zones entières, sans erreur affichée', async () => {
    getBlob.mockImplementation((p: string) => (p.includes('variante=plan') ? Promise.reject(new Error('404')) : Promise.resolve(new Blob(['png']))))
    await ouvrirLaLecture()
    const svg = await screen.findByTestId('tactical-reapp-zones')
    expect(svg.getAttribute('data-rogne')).toBe('non')
    expect(screen.queryByRole('alert')).toBeNull()
  })
})
