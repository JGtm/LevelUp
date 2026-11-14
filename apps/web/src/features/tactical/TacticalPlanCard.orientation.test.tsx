/**
 * TacticalPlanCard — l'orientation de la carte et le plan des murs viennent des préférences DU REJEU
 * (`replay-orientation.<mapId>`, `replay-show-wall-plan`), sans réglage propre : la carte est tournée
 * comme au rejeu, le plan des murs suit son interrupteur, un clic sur le plan rend la position MONDE
 * exacte, et l'image du fond tourne avec elle.
 *
 * Comme `TacticalPlanCard.zones.test` : jsdom n'a ni mise en page ni contexte 2D, le canvas reçoit une
 * taille et un contexte factices, `peindreLePlan` est moqué pour lire ce que la carte lui donne.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import type { TacticalRaster } from '@/lib/api/types'
import { orientAbout } from '@/lib/replay/orientation'
import type { WallPlanReady } from '@/lib/replay/wallPlan'
import { renderWithProviders } from '@/test/render-utils'

import { getTacticalText } from './i18n'
import { peindreLePlan } from './planPaint'
import { projectionDuPlan } from './tacticalView.logic'
import { TacticalPlanCard, type ReglagesDuPlan } from './TacticalPlanCard'

const PLAN: WallPlanReady = { edges: [[0, 0, 10, 10]], dropped: 0 }
const FOND = { originX: 0, originY: 50, widthM: 100, heightM: 50 }

const chargee = vi.hoisted(() => vi.fn<() => 'rendu' | 'plan'>(() => 'rendu'))
const planImage = vi.hoisted(() => vi.fn((): string | undefined => undefined))
vi.mock('./planPaint', () => ({ peindreLePlan: vi.fn() }))
vi.mock('./queries', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./queries')>()
  return {
    ...actual,
    useTacticalMapBackgroundFrame: () => FOND,
    useTacticalMapBackgroundPlan: () => PLAN,
    useTacticalMapBackgroundImage: () => ({ url: 'blob:fond', variante: chargee() }),
    useTacticalMapPlanImage: () => ({ planImage: planImage(), pending: false }),
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
  points_ignores: 0,
}

const REGLAGES: ReglagesDuPlan = {
  question: 'morts',
  onQuestionChange: () => {},
  qui: 'moi',
  onQuiChange: () => {},
  escouadeDisponible: false,
  escouadeEnAttente: false,
  spawn: '',
  onSpawnChange: () => {},
  grappes: [],
  zonesVisibles: true,
  onZonesVisiblesChange: () => {},
}

function rendre(onPointSelect: (p: { x: number; y: number }) => void = () => {}) {
  return renderWithProviders(
    <TacticalPlanCard
      t={t}
      locale="fr"
      playerSlug="JGtm"
      mapId="streets"
      titre="Ruelles"
      question="morts"
      lecture={LECTURE}
      etat="pret"
      inconnus={null}
      etiquette={null}
      reglages={REGLAGES}
      selected={null}
      onPointSelect={onPointSelect}
      contributions={[]}
      survol={null}
      onSurvol={() => {}}
      choisie={null}
      onChoisir={() => {}}
    />,
  )
}

const contexteFactice = { clearRect: vi.fn(), strokeRect: vi.fn() } as unknown as CanvasRenderingContext2D

/**
 * La hauteur du canvas suit le rapport du cadre : 200 x 100 pour la carte droite (100 x 50 m), 200 x 400
 * quand elle est tournée d'un quart ou de trois quarts de tour (50 x 100 m) — comme la boîte du plan
 * (`boiteDuPlan`) la ferait à l'écran.
 */
function hauteurDuCanvas(): number {
  const turns = Number(window.localStorage.getItem('replay-orientation.streets') ?? '0')
  return turns % 2 === 1 ? 400 : 100
}

beforeEach(() => {
  window.localStorage.clear()
  planImage.mockReset().mockReturnValue(undefined)
  chargee.mockReset().mockReturnValue('rendu')
  vi.spyOn(HTMLCanvasElement.prototype, 'clientWidth', 'get').mockReturnValue(200)
  vi.spyOn(HTMLCanvasElement.prototype, 'clientHeight', 'get').mockImplementation(hauteurDuCanvas)
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(contexteFactice as never)
  vi.spyOn(HTMLCanvasElement.prototype, 'getBoundingClientRect').mockImplementation(() => {
    const h = hauteurDuCanvas()
    return { left: 0, top: 0, right: 200, bottom: h, width: 200, height: h, x: 0, y: 0, toJSON: () => ({}) }
  })
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.mocked(peindreLePlan).mockClear()
})

/** Le dernier appel du peintre : le repère, la largeur et la peinture. */
function dernierAppel() {
  const [, repere, largeur, peinture] = vi.mocked(peindreLePlan).mock.calls.at(-1) ?? []
  return { repere, largeur, peinture }
}

describe('TacticalPlanCard — orientation et plan des murs du rejeu', () => {
  it('par défaut : carte non tournée, plan des murs donné au peintre, image sans rotation', () => {
    rendre()
    const { repere, peinture } = dernierAppel()
    expect(repere?.orient).toBeUndefined()
    expect(peinture?.plan).toBe(PLAN)
    expect(document.querySelector('img')?.style.transform).toBe('')
  })

  it('une orientation retenue au rejeu pour CETTE carte tourne le repère, l’image et le plan', () => {
    window.localStorage.setItem('replay-orientation.streets', '1')
    rendre()
    const { repere, peinture } = dernierAppel()
    expect(repere?.orient?.turns).toBe(1)
    // Cadre du fond 100 x 50 m, tourné d'un quart de tour : 50 x 100 m.
    expect((repere?.maxX ?? 0) - (repere?.minX ?? 0)).toBe(50)
    expect((repere?.maxY ?? 0) - (repere?.minY ?? 0)).toBe(100)
    expect(peinture?.plan).toBe(PLAN)
    expect(document.querySelector('img')?.style.transform).toBe('rotate(90deg)')
  })

  it('l’orientation d’une AUTRE carte ne tourne pas celle-ci', () => {
    window.localStorage.setItem('replay-orientation.ridgeline', '2')
    rendre()
    expect(dernierAppel().repere?.orient).toBeUndefined()
  })

  it('plan des murs éteint au rejeu (replay-show-wall-plan) : aucun plan donné au peintre', () => {
    window.localStorage.setItem('replay-show-wall-plan', 'false')
    rendre()
    expect(dernierAppel().peinture?.plan).toBeNull()
  })

  it('fond plan (réglage plan, carte avec image plan) : le plan des murs est déjà dans l’image, aucun plan donné au peintre', () => {
    window.localStorage.setItem('replay-background-style', 'plan')
    planImage.mockReturnValue('plan.png')
    chargee.mockReturnValue('plan')
    rendre()
    expect(dernierAppel().peinture?.plan).toBeNull()
  })

  it('plan demandé mais repli sur le rendu (l’image plan a échoué) : les murs sont dessinés', () => {
    window.localStorage.setItem('replay-background-style', 'plan')
    planImage.mockReturnValue('plan.png')
    chargee.mockReturnValue('rendu')
    rendre()
    expect(dernierAppel().peinture?.plan).toBe(PLAN)
  })

  it('réglage plan mais carte SANS image plan : repli sur le rendu, le plan des murs reste tracé', () => {
    window.localStorage.setItem('replay-background-style', 'plan')
    rendre()
    expect(dernierAppel().peinture?.plan).toBe(PLAN)
  })

  it('réglage rendu avec une carte qui a une image plan : le plan des murs reste tracé', () => {
    planImage.mockReturnValue('plan.png')
    rendre()
    expect(dernierAppel().peinture?.plan).toBe(PLAN)
  })

  it.each([0, 1, 2, 3] as const)('un clic sur le plan, à %i quart(s) de tour, rend la position MONDE du point visé', (turns) => {
    window.localStorage.setItem('replay-orientation.streets', String(turns))
    const onPointSelect = vi.fn()
    rendre(onPointSelect)
    const { repere } = dernierAppel()
    expect(repere).toBeTruthy()
    // Le point monde visé, projeté par le plan tourné : le clic sur ce pixel doit le rendre.
    const monde = { x: 31, y: 12 }
    const proj = projectionDuPlan(repere!, 200)!
    const px = proj(monde)
    // Le point est bien tourné (sauf à 0 quart de tour) : la projection n'est pas celle d'une carte droite.
    if (turns !== 0) {
      const droit = { x: monde.x, y: monde.y }
      expect(orientAbout(droit, turns, { x: 50, y: 25 })).not.toEqual(droit)
    }
    fireEvent.click(screen.getByTestId('tactical-plan-canvas'), { clientX: px.x, clientY: px.y })
    expect(onPointSelect).toHaveBeenCalledTimes(1)
    const recu = onPointSelect.mock.calls[0][0] as { x: number; y: number }
    expect(recu.x).toBeCloseTo(monde.x, 6)
    expect(recu.y).toBeCloseTo(monde.y, 6)
  })
})
