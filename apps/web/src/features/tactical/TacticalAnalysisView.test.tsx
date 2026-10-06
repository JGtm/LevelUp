/**
 * TacticalAnalysisView — la carte du plan et la zone choisie (plan Tactique v2, L5).
 *
 * Ce que ces tests cadenassent, `useTacticalRaster` MOQUÉ (les TRANSITIONS de clé, avec la vraie
 * lecture, sont dans `TacticalAnalysisView.fond.test.tsx`) :
 *   - la carte du plan est TOUJOURS rendue : premier chargement → l'indicateur sur le cadre ;
 *     échec (lecture, périmètre) et composition impossible → le message SUR le fond ; relecture →
 *     l'ancien calque et la légende estompés sous « Mise à jour… » ; carte hors du filtre → la
 *     dire, sans lecture ; sans carte encore → cadre au rapport par défaut, sans titre ;
 *   - le bandeau : nom de la carte, aide ⓘ, pilules « Lecture » (ordre des lectures),
 *     « Joueurs » (« Escouade » désactivé sans composition, avec son infobulle), « Réapparition » ;
 *   - le bandeau d'état des lectures d'artefact ; les trois états vides en titre seul ;
 *   - la rampe verticale : une unité par lecture, divergente pour les lectures signées.
 *
 * PAS DE TEST CANVAS (jsdom n'implémente pas le contexte 2D) : le calque garde son
 * `if (!ctx) return`, ces tests ne vérifient que le rendu React autour.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { screen, within } from '@testing-library/react'
import type { UseQueryResult } from '@tanstack/react-query'

import type { TacticalRaster } from '@/lib/api/types'
import { renderWithProviders } from '@/test/render-utils'

import type { CarteEffective } from './cockpit.logic'
import { getTacticalText } from './i18n'
import { TacticalAnalysisView } from './TacticalAnalysisView'
import { PLAN_ASPECT_DEFAUT } from './tacticalView.logic'

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
    { col: 2, lig: 1, valeur: 3, brut: 3, matchs: 4, matchs_victoire: 2, matchs_defaite: 2, centre_x: 25, centre_y: 15 },
    { col: 5, lig: 2, valeur: 5, brut: 5, matchs: 6, matchs_victoire: 3, matchs_defaite: 3, centre_x: 55, centre_y: 25 },
  ],
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
  matchs_retenus: 2,
}

const LUE: CarteEffective = { mapId: 'streets', origine: 'url' }

function mockRaster(partial: Partial<UseQueryResult<TacticalRaster>>) {
  useTacticalRaster.mockReturnValue({
    data: undefined,
    isPending: false,
    isError: false,
    ...partial,
  } as UseQueryResult<TacticalRaster>)
}

function renderVue(
  options: {
    carte?: CarteEffective
    mapName?: string
    perimetreEnEchec?: boolean
    coequipiersInconnus?: string[] | null
    coequipiers?: string[]
  } = {},
) {
  const { carte = LUE, mapName = 'Ruelles', coequipiers = [], ...reste } = options
  return renderWithProviders(
    <TacticalAnalysisView
      playerSlug="JGtm"
      carte={carte}
      mapName={mapName}
      locale="fr"
      t={t}
      matchIds={['m1', 'm2']}
      coequipiers={coequipiers}
      {...reste}
    />,
  )
}

const cadre = () => screen.getByTestId('tactical-plan-frame')

afterEach(() => {
  vi.clearAllMocks()
})

describe('TacticalAnalysisView — la carte du plan, toujours rendue', () => {
  it('PREMIER CHARGEMENT : le cadre est posé, l’indicateur par-dessus, aucun calque', () => {
    mockRaster({ isPending: true })
    renderVue()
    expect(within(cadre()).getByTestId('tactical-analysis-pending')).toBeInTheDocument()
    expect(screen.queryByTestId('tactical-plan-canvas')).toBeNull()
    expect(screen.getByTestId('tactical-analysis-body')).toHaveAttribute('aria-busy', 'true')
    expect(screen.getByRole('region', { name: 'Ruelles' })).toBeInTheDocument()
  })

  it('EN ÉCHEC : le message SUR le fond, aucun calque, le cadre reste', () => {
    mockRaster({ isError: true })
    renderVue()
    expect(within(cadre()).getByText(t.analysisErrorTitle)).toBeInTheDocument()
    expect(screen.queryByTestId('tactical-plan-canvas')).toBeNull()
  })

  it('EN ÉCHEC avec une réponse gardée : ni calque, ni légende chiffrée, ni carte de la zone', () => {
    mockRaster({ isError: true, data: RASTER_NOMINAL })
    renderVue()
    expect(screen.queryByTestId('tactical-plan-canvas')).toBeNull()
    expect(screen.queryByRole('img', { name: /Échelle de la lecture/ })).toBeNull()
    expect(screen.queryByText(t.cellTitle)).toBeNull()
  })

  it('PÉRIMÈTRE EN ÉCHEC : le message sur le fond, jamais « Mise à jour… »', () => {
    mockRaster({ data: RASTER_NOMINAL, isPlaceholderData: true })
    renderVue({ perimetreEnEchec: true })
    expect(within(cadre()).getByText(t.analysisErrorTitle)).toBeInTheDocument()
    expect(screen.queryByTestId('tactical-analysis-updating')).toBeNull()
  })

  it('COMPOSITION IMPOSSIBLE : « Coéquipier introuvable » sur le fond, jamais la panne générique', () => {
    mockRaster({ data: RASTER_NOMINAL, isPlaceholderData: true })
    renderVue({ coequipiersInconnus: ['Inconnu'] })
    expect(within(cadre()).getByText(t.unknownTeammateTitle)).toBeInTheDocument()
    expect(screen.queryByText(t.analysisErrorTitle)).toBeNull()
  })

  it('RELECTURE : calque et légende de SA lecture estompés sous « Mise à jour… »', () => {
    mockRaster({ data: { ...RASTER_NOMINAL, question: 'kills' }, isPlaceholderData: true })
    renderVue()
    expect(screen.getByTestId('tactical-analysis-updating')).toHaveTextContent(t.analysisUpdating)
    expect(screen.getByTestId('tactical-plan-canvas').className).toContain('opacity-50')
    const legende = screen.getByTestId('tactical-plan-legend')
    expect(legende.className).toContain('opacity-50')
    // L'unité est celle de la lecture À LAQUELLE la réponse affichée répond (« kills »).
    expect(legende).toHaveTextContent(t.units.kills)
  })

  it('RÉPONSE COURANTE : ni mention, ni estompage, corps non occupé', () => {
    mockRaster({ data: RASTER_NOMINAL })
    renderVue()
    expect(screen.queryByTestId('tactical-analysis-updating')).toBeNull()
    expect(screen.getByTestId('tactical-plan-canvas').className).not.toContain('opacity-50')
    expect(screen.getByTestId('tactical-analysis-body')).toHaveAttribute('aria-busy', 'false')
  })

  it('CARTE HORS DU FILTRE : « Aucun match sur cette carte dans ce filtre », sans lecture', () => {
    mockRaster({})
    renderVue({ carte: { mapId: 'aquarius', origine: 'hors_filtre' }, mapName: 'Aquarius' })
    expect(within(cadre()).getByTestId('tactical-carte-hors-filtre')).toHaveTextContent(t.planEmptyNoMatchTitle)
    expect(screen.queryByTestId('tactical-analysis-pending')).toBeNull()
    const params = useTacticalRaster.mock.calls[0][2] as { match_ids: string[] | null }
    expect(params.match_ids).toBeNull()
  })

  it('SANS CARTE ENCORE : le cadre au rapport par défaut sous l’indicateur, aucun titre', () => {
    mockRaster({})
    renderVue({ carte: { mapId: '', origine: 'attente' }, mapName: '' })
    expect(within(cadre()).getByTestId('tactical-analysis-pending')).toBeInTheDocument()
    expect(cadre().style.maxWidth).toBe(`${PLAN_ASPECT_DEFAUT * 800}px`)
    expect(screen.queryByRole('region', { name: 'Ruelles' })).toBeNull()
  })

  it('AUCUNE CARTE OUVRABLE : rien sur le fond (la colonne le dit)', () => {
    mockRaster({})
    renderVue({ carte: { mapId: '', origine: 'aucune' }, mapName: '' })
    expect(screen.queryByTestId('tactical-analysis-pending')).toBeNull()
    expect(screen.queryByTestId('tactical-plan-avis')).toBeNull()
  })
})

describe('TacticalAnalysisView — états vides en titre seul', () => {
  it('matchs mesurés dispersés : « densité insuffisante », sans conseil', () => {
    mockRaster({ data: RASTER_VIDE })
    renderVue()
    expect(within(cadre()).getByTestId('tactical-plan-vide')).toHaveTextContent(t.planEmptyDensityTitle)
    expect(screen.queryByText(/Élargis/)).toBeNull()
  })

  it('aucun match mesuré : le titre de l’absence de mesure', () => {
    mockRaster({ data: { ...RASTER_VIDE, matchs_retenus: 0 } })
    renderVue()
    expect(screen.getByTestId('tactical-plan-vide')).toHaveTextContent(t.planEmptyTitle)
  })

  it('aucun match dans le filtre : le titre du périmètre vide', () => {
    mockRaster({ data: { ...RASTER_VIDE, matchs_retenus: 0, matchs_filtres: 0 } })
    renderVue()
    expect(screen.getByTestId('tactical-plan-vide')).toHaveTextContent(t.planEmptyNoMatchTitle)
  })
})

describe('TacticalAnalysisView — le cadre du fond', () => {
  it('au rapport des bornes, jamais plus de 800 px de haut', () => {
    mockRaster({ data: RASTER_VIDE })
    renderVue()
    expect(cadre().style.maxWidth).toBe('1600px')
  })

  it('des bornes très allongées ne rendent pas un cadre démesuré', () => {
    mockRaster({ data: { ...RASTER_VIDE, bornes: { min_x: 0, max_x: 4, min_y: 0, max_y: 50, valide: true } } })
    renderVue()
    expect(cadre().style.maxWidth).toBe(`${0.08 * 800}px`)
  })
})

describe('TacticalAnalysisView — le bandeau', () => {
  it('le nom de la carte, l’aide ⓘ et la pilule « Lecture » dans l’ordre des lectures', () => {
    mockRaster({ data: RASTER_NOMINAL })
    renderVue()
    const titre = screen.getByTestId('tactical-plan-title')
    expect(titre).toHaveTextContent('Ruelles')
    expect(within(titre).getByRole('button')).toBeInTheDocument()
    const lecture = screen.getByRole('combobox', { name: t.pillReading })
    const options = within(lecture).getAllByRole('option').map((o) => o.getAttribute('value'))
    expect(options).toEqual(['morts', 'kills', 'solde', 'gagne', 'temps', 'routes', 'isole'])
  })

  it('« Joueurs » : Moi pressé, « Escouade » désactivé sans composition, avec son infobulle', () => {
    mockRaster({ data: RASTER_NOMINAL })
    renderVue()
    const joueurs = screen.getByRole('group', { name: t.pillPlayers })
    expect(within(joueurs).getByRole('button', { name: t.whoMe })).toHaveAttribute('aria-pressed', 'true')
    const escouade = within(joueurs).getByRole('button', { name: t.whoSquad })
    expect(escouade).toBeDisabled()
    expect(escouade).toHaveAttribute('title', t.planSquadDisabled)
  })

  it('« Escouade » actif avec une composition', () => {
    mockRaster({ data: RASTER_NOMINAL })
    renderVue({ coequipiers: ['xuid(42)'] })
    expect(screen.getByRole('button', { name: t.whoSquad })).not.toBeDisabled()
  })

  it('« Réapparition » : « Toutes » puis les grappes de la lecture', () => {
    mockRaster({ data: RASTER_NOMINAL })
    renderVue()
    const reapparition = screen.getByRole('combobox', { name: t.pillRespawn })
    expect(within(reapparition).getAllByRole('option').map((o) => o.textContent)).toEqual([t.pillRespawnAll, 'Base Rouge'])
  })

  it('bandeau d’état des lectures d’artefact : les matchs en attente se disent', () => {
    mockRaster({ data: { ...RASTER_NOMINAL, question: 'temps', matchs_en_attente: 2 } })
    renderVue()
    expect(screen.getByTestId('tactical-plan-status')).toHaveTextContent(t.statusPending(2))
  })
})

describe('TacticalAnalysisView — la rampe verticale, une unité par lecture', () => {
  const SIGNEE = { p50: 0, p95: 0, borne: 2, n_cellules: 2, symetrique: true }
  it.each([
    ['morts', false],
    ['kills', false],
    ['solde', true],
    ['gagne', true],
    ['temps', false],
    ['routes', false],
    ['isole', false],
  ] as const)('%s : son unité, rampe %s', (question, signee) => {
    mockRaster({ data: { ...RASTER_NOMINAL, question, echelle: signee ? SIGNEE : RASTER_NOMINAL.echelle } })
    renderVue()
    const legende = screen.getByTestId('tactical-plan-legend')
    expect(legende).toHaveTextContent(t.units[question])
    const rampe = within(legende).getByRole('img', { name: /Échelle de la lecture/ })
    expect(rampe).toHaveAttribute('data-mode', signee ? 'divergent' : 'intensity')
  })
})
