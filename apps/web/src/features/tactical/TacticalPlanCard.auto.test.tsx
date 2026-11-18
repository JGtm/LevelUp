/**
 * TacticalPlanCard — L'ORIENTATION AUTO du rejeu s'applique à la zone de CE plan : sans choix retenu pour la
 * carte, une carte haute est tournée dans une zone large et reste droite dans une zone haute ; un choix retenu
 * prime toujours.
 *
 * La zone est la largeur utile du conteneur du cadre (sa marge de légende retirée) par la hauteur que la fenêtre
 * laisse au plan (`useHauteurDuPlan`, simulée ici : sa règle a ses propres tests). jsdom n'a ni mise en page ni
 * contexte 2D : la largeur de la zone est posée par le test et suivie par un observateur de taille simulé.
 */
import { act } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { TacticalRaster } from '@/lib/api/types'
import { renderWithProviders } from '@/test/render-utils'

import { getTacticalText } from './i18n'
import { peindreLePlan } from './planPaint'
import { TacticalPlanCard, type ReglagesDuPlan } from './TacticalPlanCard'

// Une carte HAUTE : 50 x 100 m.
const FOND = { originX: 0, originY: 100, widthM: 50, heightM: 100 }
const hauteur = vi.hoisted(() => ({ value: 500 }))
const MARGE_LEGENDE_PX = 90

vi.mock('./planPaint', () => ({ peindreLePlan: vi.fn() }))
vi.mock('./useHauteurDuPlan', () => ({ useHauteurDuPlan: () => hauteur.value }))
vi.mock('./queries', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./queries')>()
  return {
    ...actual,
    useTacticalMapBackgroundFrame: () => FOND,
    useTacticalMapBackgroundPlan: () => null,
    useTacticalMapBackgroundImage: () => ({ url: 'blob:fond', variante: 'rendu' as const }),
    useTacticalMapPlanImage: () => ({ planImage: undefined, pending: false }),
  }
})

const t = getTacticalText('fr')

const LECTURE: TacticalRaster = {
  map_id: 'ronin',
  question: 'morts',
  qui: 'moi',
  bornes: { min_x: 0, max_x: 50, min_y: 0, max_y: 100, valide: true },
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
  vies: 'depart',
  onViesChange: () => {},
  grappes: [],
  zonesVisibles: true,
  onZonesVisiblesChange: () => {},
}

function rendre() {
  return renderWithProviders(
    <TacticalPlanCard
      t={t}
      locale="fr"
      playerSlug="JGtm"
      mapId="ronin"
      titre="Ronin"
      question="morts"
      lecture={LECTURE}
      etat="pret"
      inconnus={null}
      etiquette={null}
      reglages={REGLAGES}
      selected={null}
      onPointSelect={() => {}}
      contributions={[]}
      survol={null}
      onSurvol={() => {}}
      choisie={null}
      onChoisir={() => {}}
    />,
  )
}

/** La largeur UTILE de la zone que le test pose (marge de légende comprise : `clientWidth` la porte). */
let largeurZone = 0
const observateurs: (() => void)[] = []

class ObservateurDeTaille {
  private readonly rappel: () => void
  constructor(rappel: () => void) {
    this.rappel = rappel
  }
  observe() {
    observateurs.push(this.rappel)
  }
  disconnect() {
    const i = observateurs.indexOf(this.rappel)
    if (i >= 0) observateurs.splice(i, 1)
  }
}

/** La zone change de taille : sa largeur et la hauteur laissée au plan, observateurs avertis. */
function redimensionner(largeurUtile: number, hauteurLaissee: number) {
  largeurZone = largeurUtile + MARGE_LEGENDE_PX
  hauteur.value = hauteurLaissee
  act(() => {
    for (const rappel of [...observateurs]) rappel()
  })
}

/** Les quarts de tour que la carte a donnés au repère du plan. */
function quartsDeTour(): number {
  const [, repere] = vi.mocked(peindreLePlan).mock.calls.at(-1) ?? []
  return repere?.orient?.turns ?? 0
}

const image = () => document.querySelector('img')?.style.transform ?? ''
const rapport = () => (document.querySelector('[data-testid="tactical-plan-frame"]') as HTMLElement).style.aspectRatio

beforeEach(() => {
  window.localStorage.clear()
  observateurs.length = 0
  largeurZone = 1000 + MARGE_LEGENDE_PX
  hauteur.value = 500
  vi.stubGlobal('ResizeObserver', ObservateurDeTaille)
  vi.spyOn(HTMLDivElement.prototype, 'clientWidth', 'get').mockImplementation(() => largeurZone)
  vi.spyOn(HTMLCanvasElement.prototype, 'clientWidth', 'get').mockReturnValue(200)
  vi.spyOn(HTMLCanvasElement.prototype, 'clientHeight', 'get').mockReturnValue(400)
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(
    { clearRect: vi.fn(), strokeRect: vi.fn() } as unknown as CanvasRenderingContext2D,
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  vi.mocked(peindreLePlan).mockClear()
})

describe('TacticalPlanCard — l’Auto du rejeu sur la zone du plan', () => {
  it('zone large (1000 x 500), aucun choix retenu : la carte haute est tournée (repère, image, rapport du cadre)', () => {
    rendre()
    expect(quartsDeTour()).toBe(1)
    expect(image()).toBe('rotate(90deg)')
    expect(rapport()).toBe('2 / 1')
  })

  it('zone haute (300 x 900), aucun choix retenu : la même carte reste droite', () => {
    redimensionner(300, 900)
    rendre()
    expect(quartsDeTour()).toBe(0)
    expect(image()).toBe('')
    expect(rapport()).toBe('0.5 / 1')
  })

  it('zone large puis zone haute : la carte tourne, puis revient droite', () => {
    rendre()
    expect(quartsDeTour()).toBe(1)
    redimensionner(300, 900)
    expect(quartsDeTour()).toBe(0)
    expect(image()).toBe('')
    redimensionner(1000, 500)
    expect(quartsDeTour()).toBe(1)
  })

  it('zone pas encore mesurée : le plan reste droit', () => {
    largeurZone = 0
    rendre()
    expect(quartsDeTour()).toBe(0)
  })

  it('un choix retenu pour la carte prime sur l’Auto, dans les deux sens', () => {
    window.localStorage.setItem('replay-orientation.ronin', '0')
    const a = rendre()
    expect(quartsDeTour()).toBe(0)
    a.unmount()
    vi.mocked(peindreLePlan).mockClear()
    window.localStorage.setItem('replay-orientation.ronin', '2')
    redimensionner(300, 900)
    rendre()
    expect(quartsDeTour()).toBe(2)
  })
})
