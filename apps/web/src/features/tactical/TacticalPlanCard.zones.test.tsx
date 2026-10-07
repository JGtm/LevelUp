/**
 * TacticalPlanCard — les zones nommées servies avec la lecture atteignent le peintre du plan
 * (plan Tactique v2, lot L13 F7) : normalisées par le peintre du rejeu, à l'encre du texte de l'app ;
 * aucune quand la lecture n'en porte pas. L'ORDRE des calques est cadenassé par `planPaint.test.ts`.
 *
 * jsdom n'a ni taille de mise en page ni contexte 2D : le canvas reçoit ici une taille et un
 * contexte factices, et `peindreLePlan` est moqué pour lire ce que la carte lui donne.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import type { TacticalRaster } from '@/lib/api/types'
import { renderWithProviders } from '@/test/render-utils'

import { getTacticalText } from './i18n'
import { peindreLePlan } from './planPaint'
import { TacticalPlanCard, type ReglagesDuPlan } from './TacticalPlanCard'

vi.mock('./planPaint', () => ({ peindreLePlan: vi.fn() }))
vi.mock('./queries', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./queries')>()
  return {
    ...actual,
    useTacticalMapBackgroundFrame: () => null,
    useTacticalMapBackgroundUrl: () => null,
  }
})

const t = getTacticalText('fr')

const LECTURE: TacticalRaster = {
  map_id: 'streets',
  question: 'morts',
  qui: 'moi',
  bornes: { min_x: 0, max_x: 100, min_y: 0, max_y: 50, valide: true },
  pas_m: 10,
  echelle: { p50: 1, p95: 5, borne: 5, n_cellules: 1, symetrique: false },
  cellules: [{ col: 2, lig: 1, valeur: 3, brut: 3, matchs: 4, matchs_victoire: 2, matchs_defaite: 2, centre_x: 25, centre_y: 15 }],
  grappes: [],
  matchs_filtres: 50,
  matchs_retenus: 45,
  matchs_victoire: 20,
  matchs_defaite: 25,
  matchs_en_attente: 0,
  matchs_non_cuisables: 0,
  points_ignores: 0,
}

const ZONES: NonNullable<TacticalRaster['zones']> = [
  { name: '', fr: 'Base', en: 'Base', x: 20, y: 40, z: 1, z_bottom: 0, z_top: 2, volume_index: 1, big: true, polygon: [[10, 30], [30, 30], [30, 50], [10, 50]] },
  { name: '', fr: 'Pont', en: 'Bridge', x: 60, y: 10, z: 5, z_bottom: 4, z_top: 6, volume_index: 2, polygon: [[50, 0], [70, 0], [70, 20], [50, 20]] },
]

const REGLAGES: ReglagesDuPlan = {
  question: 'morts',
  onQuestionChange: () => {},
  qui: 'moi',
  onQuiChange: () => {},
  escouadeDisponible: false,
  spawn: '',
  onSpawnChange: () => {},
  grappes: [],
}

function rendre(lecture: TacticalRaster) {
  return renderWithProviders(
    <TacticalPlanCard
      t={t}
      locale="fr"
      playerSlug="JGtm"
      mapId="streets"
      titre="Ruelles"
      question="morts"
      lecture={lecture}
      etat="pret"
      inconnus={null}
      etiquette={null}
      reglages={REGLAGES}
      selected={null}
      onCellSelect={() => {}}
    />,
  )
}

const contexteFactice = {
  clearRect: vi.fn(),
  strokeRect: vi.fn(),
} as unknown as CanvasRenderingContext2D

beforeEach(() => {
  vi.spyOn(HTMLCanvasElement.prototype, 'clientWidth', 'get').mockReturnValue(200)
  vi.spyOn(HTMLCanvasElement.prototype, 'clientHeight', 'get').mockReturnValue(100)
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(contexteFactice as never)
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.mocked(peindreLePlan).mockClear()
})

describe('TacticalPlanCard — les zones nommées de la lecture', () => {
  it('zones servies : le plan les donne, normalisées, au peintre, avec l’encre du texte', () => {
    rendre({ ...LECTURE, zones: ZONES })
    const canvas = screen.getByTestId('tactical-plan-canvas')
    expect(peindreLePlan).toHaveBeenCalled()
    const [ctx, , largeur, peinture] = vi.mocked(peindreLePlan).mock.calls.at(-1) ?? []
    expect(ctx).toBe(contexteFactice)
    expect(largeur).toBe(200)
    expect(peinture?.zones.map((z) => z.fr)).toEqual(['Base', 'Pont'])
    expect(peinture?.zones[0]?.polygon).toHaveLength(4)
    expect(peinture?.locale).toBe('fr')
    expect(peinture?.encre).toBe(getComputedStyle(canvas).color)
  })

  it('zones absentes : le plan ne donne aucune zone au peintre', () => {
    rendre(LECTURE)
    expect(peindreLePlan).toHaveBeenCalled()
    expect(vi.mocked(peindreLePlan).mock.calls.at(-1)?.[3].zones).toEqual([])
  })
})
