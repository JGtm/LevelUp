/**
 * Tests — useReplayStaticLayers : combien de fois les calques statiques sont cuits.
 *
 * Ce qu'ils protègent : (1) SUIVRE ne recuit pas les calques à chaque image — le cadrage glisse en
 * translation pure, la cuisson est gardée et le calque recopié décalé, comme pendant un glisser ;
 * (2) la cuisson reprend dès que le décalage sort de la marge, dès que l'échelle change (zoom),
 * et à l'arrêt du suivi ; (3) le glisser garde son gel, sans marge ; (4) une donnée qui change
 * pendant le suivi se cuit quand même (contre le cadrage gardé) ; (5) `cookedRef` dit toujours
 * contre quel cadrage les calques ont été cuits, c'est ce qui rend le décalage exact.
 */
import { renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { layerOffset } from '../../../lib/replay/replayLogic'
import type { CanvasView } from '../model/replayView'
import { recordingContext } from '../test/recordingContext'
import type { ObjectiveElementReady } from './objectivesLayer'
import { useReplayStaticLayers, type StaticLayersInput } from './useReplayStaticLayers'

const paintObjectives = vi.hoisted(() => vi.fn())
vi.mock('./objectivesLayer', () => ({ drawObjectivesLayer: paintObjectives }))

/** Un cadrage de 400 x 400 px sur 100 x 100 unités : 4 px par unité, marge de 10 % = 40 px. */
function viewAt(dx: number, span = 100): CanvasView {
  return { bounds: { minX: dx, minY: 0, maxX: dx + span, maxY: span }, width: 400, height: 400, pad: 0 }
}

// Les entrées stables entre deux rendus, comme celles du canvas : une identité neuve relance la cuisson.
const REDRAW = () => {}
const ZONES = { zones: [], bigColors: [], fineInk: 'i', locale: 'fr' } as const
const HEAT = { grid: null, ramp: [] }
const COLOR_OF_TEAM = () => 'c'
const Z = { min: 0, max: 1 }
const ELEMENT: ObjectiveElementReady = { role: 'zone', team: 0, x: 10, y: 10, z: 0, kind: 'zone', halfX: 1, halfY: 1 } as ObjectiveElementReady
const ELEMENTS = [ELEMENT]

function entree(view: CanvasView, over: Partial<StaticLayersInput> = {}): StaticLayersInput {
  return {
    view,
    redraw: REDRAW,
    zones: ZONES as never,
    heat: HEAT,
    objectives: { elements: ELEMENTS, colorOfTeam: COLOR_OF_TEAM, neutralOutline: 'o', z: Z },
    ...over,
  }
}

beforeEach(() => {
  paintObjectives.mockReset()
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(
    () => recordingContext().ctx as never,
  )
})

afterEach(() => {
  vi.restoreAllMocks()
})

function monter(initial: StaticLayersInput) {
  return renderHook((p: StaticLayersInput) => useReplayStaticLayers(p), { initialProps: initial })
}

/** Fait glisser le cadrage de `pas` unités par image pendant `n` images, et rend le nombre de cuissons. */
function suivre(n: number, pas: number, over: Partial<StaticLayersInput> & { follow?: boolean }) {
  const { follow, ...rest } = over
  const base = (dx: number) => entree(viewAt(dx), { ...rest, follow })
  const h = monter(base(0))
  for (let i = 1; i <= n; i++) h.rerender(base(i * pas))
  return { ...h, cuissons: paintObjectives.mock.calls.length, base }
}

describe('Suivre : le cadrage glisse, la cuisson est gardée', () => {
  it('sans suivi, chaque image de cadrage recuit (le défaut d’origine)', () => {
    expect(suivre(20, 1.5, {}).cuissons).toBe(21)
  })

  it('pendant le suivi, 20 images de glissement ne recuisent que quand le décalage sort de la marge', () => {
    // 6 px par image, marge 40 px : réutilisable jusqu'à 6 images, recuit à la 7e, puis à la 14e.
    expect(suivre(20, 1.5, { follow: true }).cuissons).toBe(3)
  })

  it('dans la marge, une seule cuisson : le calque est recopié décalé', () => {
    const h = suivre(5, 1.5, { follow: true })
    expect(h.cuissons).toBe(1)
    const cuit = h.result.current.cookedRef.current as CanvasView
    expect(cuit).toEqual(viewAt(0))
    // Le décalage est exact : 5 x 1.5 unités x 4 px, et le sens est celui de la translation.
    expect(Math.abs(layerOffset(cuit, viewAt(7.5)).x)).toBeCloseTo(30, 6)
  })

  it('un changement d’échelle (zoom) recuit même pendant le suivi', () => {
    const h = suivre(3, 1.5, { follow: true })
    expect(h.cuissons).toBe(1)
    h.rerender(entree(viewAt(4.5, 50), { follow: true }))
    expect(paintObjectives).toHaveBeenCalledTimes(2)
    expect(h.result.current.cookedRef.current).toEqual(viewAt(4.5, 50))
  })

  it('l’arrêt du suivi recuit contre le cadrage courant', () => {
    const h = suivre(4, 1.5, { follow: true })
    expect(h.cuissons).toBe(1)
    h.rerender(entree(viewAt(6), { follow: false }))
    expect(paintObjectives).toHaveBeenCalledTimes(2)
    expect(h.result.current.cookedRef.current).toEqual(viewAt(6))
  })

  it('une donnée qui change pendant le suivi se cuit, contre le cadrage gardé', () => {
    const h = suivre(3, 1.5, { follow: true })
    const autres = { ...ELEMENT }
    h.rerender(
      entree(viewAt(4.5), {
        follow: true,
        objectives: {
          elements: [autres],
          colorOfTeam: COLOR_OF_TEAM,
          neutralOutline: 'o',
          z: Z,
        },
      }),
    )
    expect(paintObjectives).toHaveBeenCalledTimes(2)
    expect(h.result.current.cookedRef.current).toEqual(viewAt(0))
  })

  it('un glisser garde son gel, sans marge', () => {
    // 100 images de 1.5 unité : bien au-delà de la marge, et pourtant aucune recuisson.
    const h = monter(entree(viewAt(0)))
    for (let i = 1; i <= 100; i++) h.rerender(entree(viewAt(i * 1.5), { frozen: true }))
    expect(paintObjectives).toHaveBeenCalledTimes(1)
    expect(h.result.current.cookedRef.current).toEqual(viewAt(0))
  })

  it('le relâchement du glisser recuit', () => {
    const h = monter(entree(viewAt(0)))
    for (let i = 1; i <= 10; i++) h.rerender(entree(viewAt(i * 1.5), { frozen: true }))
    h.rerender(entree(viewAt(15), { frozen: false }))
    expect(paintObjectives).toHaveBeenCalledTimes(2)
    expect(h.result.current.cookedRef.current).toEqual(viewAt(15))
  })
})
