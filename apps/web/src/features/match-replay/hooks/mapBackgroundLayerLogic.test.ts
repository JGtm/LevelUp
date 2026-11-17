/**
 * mapBackgroundLayerLogic — le fond assemblé pour le canvas et la règle du rognage au plan : plan choisi
 * et image présente, plan choisi sans image (ou en échec), rendu affiché, fond absent.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayMapBackground } from '@/lib/api/types'

import { buildMapBackgroundLayer, zonesClipOf, type MapBackgroundInputs } from './mapBackgroundLayerLogic'

const CALAGE = { widthPx: 100, heightPx: 80 } as ReplayMapBackground['calibration']
const FOND = { calibration: CALAGE, planImage: 'plan.webp', plan: null } as unknown as ReplayMapBackground
const FOND_SANS_PLAN = { calibration: CALAGE, planImage: null, plan: null } as unknown as ReplayMapBackground
const RENDU = { nom: 'rendu' } as unknown as HTMLImageElement
const PLAN = { nom: 'plan' } as unknown as HTMLImageElement

function inputs(over: Partial<MapBackgroundInputs>): MapBackgroundInputs {
  return { background: FOND, image: PLAN, variante: 'plan', planImage: PLAN, planVariante: 'plan', ...over }
}

describe('buildMapBackgroundLayer', () => {
  it('plan choisi et image présente : le plan est posé et sert de masque aux zones', () => {
    const layer = buildMapBackgroundLayer(inputs({}))
    expect(layer).toMatchObject({ variante: 'plan', planAvailable: true, image: PLAN })
    expect(layer?.zonesClip).toEqual({ calibration: CALAGE, image: PLAN })
  })

  it('plan choisi sans image (pas encore arrivée) : le rendu est posé, zones entières', () => {
    const layer = buildMapBackgroundLayer(inputs({ image: RENDU, variante: 'rendu', planImage: null, planVariante: 'plan' }))
    expect(layer).toMatchObject({ variante: 'rendu', image: RENDU })
    expect(layer?.zonesClip).toBeNull()
  })

  it('plan choisi mais en échec (le rendu est servi à sa place) : pas de masque faux', () => {
    const layer = buildMapBackgroundLayer(inputs({ image: RENDU, variante: 'rendu', planImage: RENDU, planVariante: 'rendu' }))
    expect(layer?.zonesClip).toBeNull()
    expect(layer?.planAvailable).toBe(true)
  })

  it('rendu affiché, plan chargé pour le masque : les zones se rognent au plan quand même', () => {
    const layer = buildMapBackgroundLayer(inputs({ image: RENDU, variante: 'rendu' }))
    expect(layer).toMatchObject({ variante: 'rendu', image: RENDU })
    expect(layer?.zonesClip).toEqual({ calibration: CALAGE, image: PLAN })
  })

  it('un fond sans image plan : la commande « Fond » n’est pas proposée', () => {
    const layer = buildMapBackgroundLayer(
      inputs({ background: FOND_SANS_PLAN, image: RENDU, variante: 'rendu', planImage: null, planVariante: 'rendu' }),
    )
    expect(layer?.planAvailable).toBe(false)
    expect(layer?.zonesClip).toBeNull()
  })

  it('sans calage ou sans image posée : pas de fond', () => {
    expect(buildMapBackgroundLayer(inputs({ background: undefined }))).toBeNull()
    expect(buildMapBackgroundLayer(inputs({ image: null }))).toBeNull()
  })
})

describe('zonesClipOf', () => {
  it('le masque suit le calage du fond', () => {
    expect(zonesClipOf(FOND, PLAN, 'plan')).toEqual({ calibration: CALAGE, image: PLAN })
    expect(zonesClipOf(FOND, null, 'plan')).toBeNull()
    expect(zonesClipOf(FOND, PLAN, 'rendu')).toBeNull()
  })
})
