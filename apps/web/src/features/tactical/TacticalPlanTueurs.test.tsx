/**
 * La couche « autre joueur » du plan : absente sans zone choisie, présente avec, légende par
 * rôle, mise en avant par le survol (tuile -> plan) et survol d'un point (plan -> vue + infobulle).
 *
 * jsdom n'a ni mise en page ni contexte 2D : la toile reçoit une taille et un contexte factices,
 * et les peintres sont moqués pour lire ce que la carte leur donne.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, screen } from '@testing-library/react'

import type { TacticalContribution, TacticalRaster } from '@/lib/api/types'
import { renderWithProviders } from '@/test/render-utils'

import { getTacticalText } from './i18n'
import { peindreLePlan } from './planPaint'
import { TacticalPlanCard, type ReglagesDuPlan } from './TacticalPlanCard'
import { peindreLaCouche } from './tueursPaint'

vi.mock('./planPaint', () => ({ peindreLePlan: vi.fn() }))
vi.mock('./tueursPaint', () => ({ peindreLaCouche: vi.fn() }))
vi.mock('./queries', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./queries')>()
  return { ...actual, useTacticalMapBackgroundFrame: () => null, useTacticalMapBackgroundUrl: () => null }
})

const t = getTacticalText('fr')

const LECTURE: TacticalRaster = {
  map_id: 'streets',
  question: 'solde',
  qui: 'moi',
  bornes: { min_x: 0, max_x: 100, min_y: 0, max_y: 50, valide: true },
  pas_m: 10,
  echelle: { p50: 1, p95: 5, borne: 5, n_cellules: 1, symetrique: true },
  cellules: [{ col: 2, lig: 1, valeur: 3, brut: 3, matchs: 4, matchs_victoire: 2, matchs_defaite: 2, centre_x: 25, centre_y: 15 }],
  grappes: [],
  matchs_filtres: 50,
  matchs_retenus: 45,
  matchs_victoire: 20,
  matchs_defaite: 25,
  matchs_en_attente: 0,
  points_ignores: 0,
}

const REGLAGES: ReglagesDuPlan = {
  question: 'solde',
  onQuestionChange: () => {},
  qui: 'moi',
  onQuiChange: () => {},
  escouadeDisponible: false,
  escouadeEnAttente: false,
  spawn: '',
  onSpawnChange: () => {},
  zonesVisibles: true,
  onZonesVisiblesChange: () => {},
  grappes: [],
}

function contribution(extra: Partial<TacticalContribution>): TacticalContribution {
  return {
    match_id: 'm1',
    instant_ms: 1000,
    clock: 'match',
    xuid: '1',
    match_started_at: '2026-09-01T22:30:00Z',
    replay_available: true,
    ...extra,
  }
}

// Toile de 200 x 100 pour des bornes de 100 x 50 : 2 px par mètre, Y inversé (le point (50, 25) est en (100, 50)).
const MORT = contribution({ face: 'mort', autre_gamertag: 'Rival', point: { x: 20, y: 20 }, autre_point: { x: 50, y: 25 } })
const FRAG = contribution({ face: 'frag', autre_gamertag: 'Cible', point: { x: 20, y: 20 }, autre_point: { x: 80, y: 10 } })
const ENTREE = contribution({ face: 'entree' })

interface Reglage {
  selected?: { col: number; row: number } | null
  contributions?: TacticalContribution[]
  survol?: number | null
  choisie?: number | null
  onZone?: (point: { x: number; y: number }) => void
}

function carte(props: Reglage, onSurvol: (i: number | null) => void, onChoisir: (i: number) => void = () => {}) {
  return (
    <TacticalPlanCard
      t={t}
      locale="fr"
      playerSlug="JGtm"
      mapId="streets"
      titre="Ruelles"
      question="solde"
      lecture={LECTURE}
      etat="pret"
      inconnus={null}
      etiquette={null}
      reglages={REGLAGES}
      selected={props.selected === undefined ? { col: 2, row: 1 } : props.selected}
      onPointSelect={props.onZone ?? (() => {})}
      contributions={props.contributions ?? [MORT, FRAG, ENTREE]}
      survol={props.survol ?? null}
      onSurvol={onSurvol}
      choisie={props.choisie ?? null}
      onChoisir={onChoisir}
    />
  )
}

function rendre(
  props: Reglage = {},
  onSurvol: (i: number | null) => void = () => {},
  onChoisir: (i: number) => void = () => {},
) {
  return renderWithProviders(carte(props, onSurvol, onChoisir))
}

beforeEach(() => {
  vi.spyOn(HTMLCanvasElement.prototype, 'clientWidth', 'get').mockReturnValue(200)
  vi.spyOn(HTMLCanvasElement.prototype, 'clientHeight', 'get').mockReturnValue(100)
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({
    clearRect: vi.fn(),
    strokeRect: vi.fn(),
    beginPath: vi.fn(),
    arc: vi.fn(),
    stroke: vi.fn(),
  } as unknown as CanvasRenderingContext2D)
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.mocked(peindreLePlan).mockClear()
  vi.mocked(peindreLaCouche).mockClear()
})

describe('TacticalPlanCard — la couche des tueurs et des victimes', () => {
  it('sans zone choisie : ni couche ni légende', () => {
    rendre({ selected: null })
    expect(screen.queryByTestId('tactical-plan-tueurs')).toBeNull()
    expect(screen.queryByTestId('tactical-plan-formes')).toBeNull()
  })

  it('avec une zone : la couche, et une entrée de légende par rôle présent', () => {
    rendre()
    expect(screen.getByTestId('tactical-plan-tueurs')).toBeInTheDocument()
    const legende = screen.getByTestId('tactical-plan-formes')
    expect(legende).toHaveTextContent('Tueurs')
    expect(legende).toHaveTextContent('Victimes')
    // La légende dit les formes : losange pour les tueurs, rond pour les victimes.
    expect(legende.querySelector('[data-forme="losange"]')).not.toBeNull()
    expect(legende.querySelector('[data-forme="rond"]')).not.toBeNull()
    const points = vi.mocked(peindreLaCouche).mock.calls.at(-1)?.[1]
    expect(points?.map((p) => [p.index, p.role])).toEqual([
      [0, 'tueur'],
      [1, 'victime'],
    ])
  })

  it('une seule face : une seule entrée de légende', () => {
    rendre({ contributions: [MORT] })
    const legende = screen.getByTestId('tactical-plan-formes')
    expect(legende).toHaveTextContent('Tueurs')
    expect(legende).not.toHaveTextContent('Victimes')
  })

  it('lecture sans autre joueur (temps, routes) : aucune couche', () => {
    rendre({ contributions: [ENTREE] })
    expect(screen.queryByTestId('tactical-plan-tueurs')).toBeNull()
    expect(screen.queryByTestId('tactical-plan-formes')).toBeNull()
  })

  it('survol d’une tuile : la couche reçoit l’index mis en avant', () => {
    rendre({ survol: 1 })
    expect(vi.mocked(peindreLaCouche).mock.calls.at(-1)?.[5]).toBe(1)
  })

  it('la contribution choisie reste en relief sans survol, et le survol d’une autre prime', () => {
    rendre({ choisie: 0 })
    expect(vi.mocked(peindreLaCouche).mock.calls.at(-1)?.[5]).toBe(0)
    vi.mocked(peindreLaCouche).mockClear()
    rendre({ choisie: 0, survol: 1 })
    expect(vi.mocked(peindreLaCouche).mock.calls.at(-1)?.[5]).toBe(1)
  })

  it('les tailles passées au peintre suivent l’échelle du plan', () => {
    rendre()
    const dim = vi.mocked(peindreLaCouche).mock.calls.at(-1)?.[4]
    expect(dim?.rayonPoint).toBeGreaterThanOrEqual(4)
    expect(dim?.rayonPointAvant).toBeGreaterThan(dim?.rayonPoint ?? 0)
  })

  it('clic sur un point du plan : la vue choisit sa tuile, sans changer de zone', () => {
    const onChoisir = vi.fn()
    const onChoisir_ = onChoisir
    const surZone = vi.fn()
    rendre({ onZone: surZone }, () => {}, onChoisir_)
    const canvas = screen.getByTestId('tactical-plan-canvas')
    act(() => {
      canvas.dispatchEvent(new MouseEvent('click', { bubbles: true, clientX: 102, clientY: 51 }))
    })
    expect(onChoisir_).toHaveBeenCalledWith(0)
    // Le clic n'a pas atteint la toile de la chaleur (il aurait changé de zone).
    expect(surZone).not.toHaveBeenCalled()
    // Loin de tout point : le clic passe à la toile (changement de zone), aucun choix.
    onChoisir_.mockClear()
    act(() => {
      canvas.dispatchEvent(new MouseEvent('click', { bubbles: true, clientX: 10, clientY: 90 }))
    })
    expect(onChoisir_).not.toHaveBeenCalled()
    expect(surZone).toHaveBeenCalledTimes(1)
  })

  it('survol d’un point du plan : la vue est prévenue, l’infobulle nomme l’autre joueur et la distance', () => {
    const onSurvol = vi.fn()
    const { rerender } = rendre({}, onSurvol)
    const cadre = screen.getByTestId('tactical-plan-frame')
    act(() => {
      cadre.dispatchEvent(new MouseEvent('pointermove', { bubbles: true, clientX: 102, clientY: 51 }))
    })
    expect(onSurvol).toHaveBeenLastCalledWith(0)
    // La vue répond en posant l'index : l'infobulle s'affiche (30 m de (20, 20) à (50, 25)).
    rerender(carte({ survol: 0 }, onSurvol))
    expect(screen.getByTestId('tactical-tueurs-tip')).toHaveTextContent('Rival · à 30 m')
    act(() => {
      cadre.dispatchEvent(new MouseEvent('pointermove', { bubbles: true, clientX: 10, clientY: 90 }))
    })
    expect(onSurvol).toHaveBeenLastCalledWith(null)
  })
})
