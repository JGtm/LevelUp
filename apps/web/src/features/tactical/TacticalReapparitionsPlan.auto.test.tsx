/**
 * TacticalReapparitionsPlan — L'ORIENTATION AUTO du rejeu s'applique à la zone de CE plan (celui de la lecture
 * « Réapparitions adverses »), comme au plan de placement : une carte haute est tournée dans une zone large et
 * reste droite dans une zone haute ; un choix retenu pour la carte prime toujours.
 *
 * La zone est la largeur utile du conteneur du cadre par la hauteur que la fenêtre laisse au plan
 * (`useHauteurDuPlan`, simulée ici). La largeur est posée par le test et suivie par un observateur de taille simulé.
 */
import { act } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { TacticalRaster } from '@/lib/api/types'
import { renderWithProviders } from '@/test/render-utils'

import { getTacticalText } from './i18n'
import { TacticalReapparitionsPlan } from './TacticalReapparitionsPlan'

// Une carte HAUTE : 50 x 100 m.
const FOND = { originX: 0, originY: 100, widthM: 50, heightM: 100 }
const hauteur = vi.hoisted(() => ({ value: 500 }))
const MARGE_LEGENDE_PX = 90

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
  cellules: [],
  grappes: [],
  matchs_filtres: 50,
  matchs_retenus: 45,
  matchs_victoire: 20,
  matchs_defaite: 25,
  matchs_en_attente: 0,
  points_ignores: 0,
}

function rendre() {
  return renderWithProviders(
    <TacticalReapparitionsPlan
      t={t}
      locale="fr"
      playerSlug="JGtm"
      mapId="ronin"
      titre="Ronin"
      lecture={LECTURE}
      etat="pret"
      inconnus={null}
      modele={null}
      zones={[]}
      pilule={null}
      zonesProposees={[]}
      choisie={null}
      onChoisir={() => {}}
      plan={{ href: null, decide: true }}
    />,
  )
}

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

/** La zone change de taille : sa largeur utile et la hauteur laissée au plan, observateurs avertis. */
function redimensionner(largeurUtile: number, hauteurLaissee: number) {
  largeurZone = largeurUtile + MARGE_LEGENDE_PX
  hauteur.value = hauteurLaissee
  act(() => {
    for (const rappel of [...observateurs]) rappel()
  })
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
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('TacticalReapparitionsPlan — l’Auto du rejeu sur la zone du plan', () => {
  it('zone large (1000 x 500), aucun choix retenu : la carte haute est tournée (image, rapport du cadre)', () => {
    rendre()
    expect(image()).toBe('rotate(90deg)')
    expect(rapport()).toBe('2 / 1')
  })

  it('zone haute (300 x 900), aucun choix retenu : la même carte reste droite', () => {
    redimensionner(300, 900)
    rendre()
    expect(image()).toBe('')
    expect(rapport()).toBe('0.5 / 1')
  })

  it('zone large puis zone haute : la carte tourne, puis revient droite', () => {
    rendre()
    expect(image()).toBe('rotate(90deg)')
    redimensionner(300, 900)
    expect(image()).toBe('')
    expect(rapport()).toBe('0.5 / 1')
    redimensionner(1000, 500)
    expect(image()).toBe('rotate(90deg)')
  })

  it('zone pas encore mesurée : le plan reste droit', () => {
    largeurZone = 0
    rendre()
    expect(image()).toBe('')
  })

  it('un choix retenu pour la carte prime sur l’Auto, dans les deux sens', () => {
    window.localStorage.setItem('replay-orientation.ronin', '0')
    const a = rendre()
    expect(image()).toBe('')
    a.unmount()
    window.localStorage.setItem('replay-orientation.ronin', '2')
    redimensionner(300, 900)
    rendre()
    expect(image()).toBe('rotate(180deg)')
  })
})
