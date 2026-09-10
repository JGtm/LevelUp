/**
 * TacticalAnalysisView — la vue d'analyse d'une carte (items 5.2-5.6).
 *
 * Ce que ces tests cadenassent, `useTacticalRaster` MOQUÉ (la lecture réseau est déjà
 * couverte côté contrat par les tests Go et par `queries.ts`) :
 *   - EN ATTENTE (`isPending`) -> un indicateur de chargement, aucun KPI ;
 *   - EN ÉCHEC (`isError`) -> le message d'échec, aucun KPI ;
 *   - VIDE (réponse reçue, aucune cellule au-dessus du plancher) -> le message du
 *     plancher dans la carte « Plan », le bandeau de KPI reste servi ;
 *   - NOMINAL -> les quatre tuiles de KPI, le canevas du plan, le placeholder de la
 *     carte « Cellule sélectionnée » tant qu'aucune cellule n'est cliquée.
 *
 * PAS DE TEST CANVAS (jsdom n'implémente pas le contexte 2D) : `TacticalPlanCard` garde
 * son `if (!ctx) return`, ces tests ne vérifient que le rendu React autour.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'
import type { UseQueryResult } from '@tanstack/react-query'

import type { TacticalRaster } from '@/lib/api/types'
import { renderWithProviders } from '@/test/render-utils'

import { getTacticalText } from './i18n'
import { TacticalAnalysisView } from './TacticalAnalysisView'
import { PLAN_ASPECT_DEFAUT, PLAN_HAUTEUR_MAX_PX } from './tacticalView.logic'

const getBlob = vi.fn()
vi.mock('@/lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api/client')>()
  return {
    ...actual,
    api: { ...actual.api, getBlob: (path: string) => getBlob(path) },
  }
})

const useTacticalRaster = vi.fn()
vi.mock('./queries', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./queries')>()
  return { ...actual, useTacticalRaster: (...args: unknown[]) => useTacticalRaster(...args) }
})

const t = getTacticalText('fr')

const BORNES = { min_x: 0, max_x: 100, min_y: 0, max_y: 50, valide: true }

const RASTER_NOMINAL: TacticalRaster = {
  map_id: 'streets',
  question: 'morts',
  qui: 'moi',
  bornes: BORNES,
  pas_m: 10,
  echelle: { p50: 1, p95: 5, borne: 5, n_cellules: 2, symetrique: false },
  cellules: [
    {
      col: 2,
      lig: 1,
      valeur: 3,
      brut: 3,
      matchs: 4,
      matchs_victoire: 2,
      matchs_defaite: 2,
      centre_x: 25,
      centre_y: 15,
    },
    {
      col: 5,
      lig: 2,
      valeur: 5,
      brut: 5,
      matchs: 6,
      matchs_victoire: 3,
      matchs_defaite: 3,
      centre_x: 55,
      centre_y: 25,
    },
  ],
  echange: { taux: 0.42, brut: 21, n: 50, par_match: 0.4, echantillon_faible: false },
  isolement: { taux: 0.18, brut: 9, n: 50, par_match: 0.18, echantillon_faible: false },
  grappes: [{ id: 'g1', nom_fr: 'Base Rouge', nom_en: 'Red Base', matchs: 10, x: 10, y: 10 }],
  matchs_filtres: 50,
  matchs_retenus: 45,
  matchs_victoire: 20,
  matchs_defaite: 25,
  matchs_en_attente: 0,
  matchs_non_cuisables: 0,
  evenements_journal: 500,
  evenements_localises: 480,
  points_ignores: 0,
}

const RASTER_VIDE: TacticalRaster = {
  ...RASTER_NOMINAL,
  cellules: [],
  echelle: { p50: 0, p95: 0, borne: 0, n_cellules: 0, symetrique: false },
  echange: undefined,
  isolement: undefined,
  matchs_retenus: 2,
}

function mockRaster(partial: Partial<UseQueryResult<TacticalRaster>>) {
  useTacticalRaster.mockReturnValue({
    data: undefined,
    isPending: false,
    isError: false,
    ...partial,
  } as UseQueryResult<TacticalRaster>)
}

function renderVue() {
  return renderWithProviders(
    <TacticalAnalysisView
      playerSlug="JGtm"
      mapId="streets"
      mapName="Ruelles"
      locale="fr"
      t={t}
      matchIds={['m1', 'm2']}
      coequipiers={[]}
    />,
  )
}

afterEach(() => {
  vi.clearAllMocks()
})

describe('TacticalAnalysisView — états de la lecture', () => {
  it('EN ATTENTE : un indicateur de chargement, aucun KPI', () => {
    mockRaster({ isPending: true })
    renderVue()
    expect(screen.getByTestId('tactical-analysis-pending')).toBeInTheDocument()
    expect(screen.queryByTestId('kpi-strip')).not.toBeInTheDocument()
  })

  it('EN ÉCHEC : le message d’échec, aucun KPI', () => {
    mockRaster({ isError: true })
    renderVue()
    expect(screen.getByText(t.analysisErrorTitle)).toBeInTheDocument()
    expect(screen.queryByTestId('kpi-strip')).not.toBeInTheDocument()
  })

  // VIDE — TROIS CAUSES, TROIS MESSAGES (point 21, lot 3.2). Le message générique « pas
  // assez de matchs mesurés » mentait quand les matchs étaient là mais dispersés.
  it('VIDE, matchs mesurés dispersés : « densité insuffisante », le KPI reste servi', () => {
    mockRaster({ data: RASTER_VIDE })
    renderVue()
    expect(screen.getByText(t.planEmptyDensityTitle)).toBeInTheDocument()
    expect(screen.queryByText(t.planEmptyTitle)).not.toBeInTheDocument()
    expect(screen.getByTestId('kpi-strip')).toBeInTheDocument()
  })

  it('VIDE, aucun match mesuré : le message historique reste servi', () => {
    mockRaster({ data: { ...RASTER_VIDE, matchs_retenus: 0, matchs_filtres: 12 } })
    renderVue()
    expect(screen.getByText(t.planEmptyTitle)).toBeInTheDocument()
    expect(screen.queryByText(t.planEmptyDensityTitle)).not.toBeInTheDocument()
  })

  it('VIDE, aucun match dans le filtre : le message du périmètre', () => {
    mockRaster({ data: { ...RASTER_VIDE, matchs_retenus: 0, matchs_filtres: 0 } })
    renderVue()
    expect(screen.getByText(t.planEmptyNoMatchTitle)).toBeInTheDocument()
    expect(screen.queryByText(t.planEmptyDensityTitle)).not.toBeInTheDocument()
  })

  // UN PLAN VIDE GARDE UN CADRE DE TAILLE NORMALE (lot 3.2) : ni canvas de 13 375 px, ni
  // carte qui se rétracte à la hauteur d'un message.
  it('VIDE : le cadre du plan reste posé, au rapport du fond, hauteur bornée', () => {
    mockRaster({ data: { ...RASTER_VIDE, bornes: { ...BORNES, valide: false } } })
    renderVue()
    const cadre = screen.getByTestId('tactical-plan-frame')
    // jsdom normalise `aspect-ratio: <n>` en « <n> / 1 ».
    expect(cadre.style.aspectRatio).toBe(`${PLAN_ASPECT_DEFAUT} / 1`)
    expect(cadre.style.maxWidth).toBe(`${PLAN_ASPECT_DEFAUT * PLAN_HAUTEUR_MAX_PX}px`)
    // Aucun calque de chaleur à peindre : le canevas n'est pas monté.
    expect(screen.queryByTestId('tactical-plan-canvas')).not.toBeInTheDocument()
  })

  it('des bornes très allongées ne rendent plus un cadre de 13 375 px', () => {
    mockRaster({
      data: {
        ...RASTER_NOMINAL,
        bornes: { min_x: 0, max_x: 8, min_y: 0, max_y: 100, valide: true },
      },
    })
    renderVue()
    const cadre = screen.getByTestId('tactical-plan-frame')
    // Largeur plafonnée à 0,08 x 720 px : la hauteur ne peut plus dépasser 720 px.
    expect(cadre.style.maxWidth).toBe(`${0.08 * PLAN_HAUTEUR_MAX_PX}px`)
  })

  it('NOMINAL : les KPI, le canevas du plan, le placeholder de la cellule', () => {
    mockRaster({ data: RASTER_NOMINAL })
    renderVue()

    expect(screen.getByTestId('kpi-strip')).toBeInTheDocument()
    const cartes = screen.getAllByTestId('kpi-card')
    expect(cartes).toHaveLength(4) // retenus, couverture, échange, isolement

    expect(screen.getByTestId('tactical-plan-canvas')).toBeInTheDocument()
    expect(screen.getByText(t.cellPlaceholder)).toBeInTheDocument()

    expect(
      screen.getByRole('heading', { name: 'Plan de Ruelles — Où je meurs' }),
    ).toBeInTheDocument()
  })

  // LE PAS RETENU EST AFFICHÉ, PAS DEVINÉ (lot 3.2, décision D6) : depuis le pas
  // adaptatif, deux cartes peuvent se lire à deux résolutions différentes — sans le dire,
  // elles ne se comparent plus.
  it('NOMINAL : le pied du plan annonce le pas de grille publié par le serveur', () => {
    mockRaster({ data: { ...RASTER_NOMINAL, pas_m: 2 } })
    renderVue()
    expect(screen.getByTestId('tactical-plan-grid-step')).toHaveTextContent('Grille : 2 m par cellule')
  })

  it('NOMINAL : un pas fractionnaire s’affiche dans la locale de la page', () => {
    mockRaster({ data: { ...RASTER_NOMINAL, pas_m: 0.5 } })
    renderVue()
    expect(screen.getByTestId('tactical-plan-grid-step')).toHaveTextContent('Grille : 0,5 m par cellule')
  })

  it('la carte Plan omet échange/isolement quand le contrat ne les publie pas', () => {
    mockRaster({ data: { ...RASTER_NOMINAL, echange: undefined, isolement: undefined } })
    renderVue()
    expect(screen.getAllByTestId('kpi-card')).toHaveLength(2)
  })
})

describe('TacticalAnalysisView — réserve d’échantillon faible (doctrine : interdit de comparer, pas de cacher)', () => {
  it('avec echantillon_faible=true, les tuiles Échange et Isolement rendent la réserve', () => {
    mockRaster({
      data: {
        ...RASTER_NOMINAL,
        echange: { ...RASTER_NOMINAL.echange!, echantillon_faible: true },
        isolement: { ...RASTER_NOMINAL.isolement!, echantillon_faible: true },
      },
    })
    renderVue()
    const cartes = screen.getAllByTestId('kpi-card')
    const tradeCard = cartes.find((c) => c.getAttribute('data-id') === 'tactical-trade')
    const isoCard = cartes.find((c) => c.getAttribute('data-id') === 'tactical-isolation')
    expect(tradeCard?.textContent).toContain(t.lowSample)
    expect(isoCard?.textContent).toContain(t.lowSample)
  })

  it('sans echantillon_faible, aucune tuile ne mentionne la réserve', () => {
    mockRaster({ data: RASTER_NOMINAL })
    renderVue()
    const cartes = screen.getAllByTestId('kpi-card')
    const tradeCard = cartes.find((c) => c.getAttribute('data-id') === 'tactical-trade')
    const isoCard = cartes.find((c) => c.getAttribute('data-id') === 'tactical-isolation')
    expect(tradeCard?.textContent).not.toContain(t.lowSample)
    expect(isoCard?.textContent).not.toContain(t.lowSample)
  })
})

describe('TacticalAnalysisView — note de couverture « matchs sans rayon connu » (tuile Isolement)', () => {
  it('matchs_sans_rayon > 0 : la note apparaît sur la tuile Isolement', () => {
    mockRaster({ data: { ...RASTER_NOMINAL, matchs_sans_rayon: 3 } })
    renderVue()
    expect(screen.getByTestId('tactical-isolation-no-radius')).toHaveTextContent(t.kpiNoRadiusNote(3))
  })

  it('matchs_sans_rayon absent ou nul : aucune note', () => {
    mockRaster({ data: { ...RASTER_NOMINAL, matchs_sans_rayon: 0 } })
    renderVue()
    expect(screen.queryByTestId('tactical-isolation-no-radius')).not.toBeInTheDocument()
  })
})
