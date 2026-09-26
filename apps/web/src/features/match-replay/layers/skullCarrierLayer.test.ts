import { describe, expect, it, vi } from 'vitest'

import { type CanvasView } from '../model/replayView'
import type { ReplaySkullCarry } from '../../../lib/replay/replayNormalize'
import { drawSkullCarrier, skullCarrierActiveAt, type SkullCarrierInput } from './skullCarrierLayer'
import { OFFSCREEN_MARGIN_PX } from '../model/edgeClamp'
import { skullPresenceAt } from '../model/skullPresence'
import type { ReplayObjectiveObjectReady } from '../../../lib/replay/replayNormalize'

const carry = (over: Partial<ReplaySkullCarry>): ReplaySkullCarry => ({
  xuid: '2533274806055812',
  t0: 0,
  t1: 100,
  closed: true,
  ...over,
})

describe('skullCarrierActiveAt', () => {
  it('rend les portages qui couvrent l’image, bornes incluses', () => {
    const a = carry({ xuid: 'a', t0: 0, t1: 50 })
    const b = carry({ xuid: 'b', t0: 60, t1: 100 })
    // Bornes incluses : t0 et t1 comptent.
    expect(skullCarrierActiveAt([a], 0).map((c) => c.xuid)).toEqual(['a'])
    expect(skullCarrierActiveAt([a], 50).map((c) => c.xuid)).toEqual(['a'])
    // Hors intervalle : rien.
    expect(skullCarrierActiveAt([a], 51)).toEqual([])
    // Un seul crâne : entre deux portages consécutifs, un seul est actif à un instant donné.
    expect(skullCarrierActiveAt([a, b], 70).map((c) => c.xuid)).toEqual(['b'])
  })
})

describe('drawSkullCarrier', () => {
  const view: CanvasView = {
    bounds: { minX: 0, minY: 0, maxX: 100, maxY: 100 },
    width: 200,
    height: 200,
    pad: 0,
  }

  it('ne dessine pas un porteur non localisable (aucune position propre à inventer)', () => {
    const ctx = {
      beginPath: vi.fn(), arc: vi.fn(), fill: vi.fn(), stroke: vi.fn(),
    } as unknown as CanvasRenderingContext2D
    const layer: SkullCarrierInput = {
      style: { ink: '#fff', outline: '#000', reducedMotion: true },
      posOf: () => null,
    }
    drawSkullCarrier(ctx, layer, [carry({ t0: 0, t1: 100 })], view, 10)
    expect(ctx.fill).not.toHaveBeenCalled()
  })

  it('dessine le crâne sur la position relue du porteur courant', () => {
    const fill = vi.fn()
    const ctx = {
      beginPath: vi.fn(), arc: vi.fn(), fill, stroke: vi.fn(),
    } as unknown as CanvasRenderingContext2D
    const layer: SkullCarrierInput = {
      style: { ink: '#fff', outline: '#000', reducedMotion: true },
      posOf: () => ({ x: 50, y: 50 }),
    }
    drawSkullCarrier(ctx, layer, [carry({ t0: 0, t1: 100 })], view, 10)
    // Le glyphe partagé remplit le disque puis ses deux orbites : trois remplissages pour UN crâne.
    expect(fill).toHaveBeenCalledTimes(3)
  })

  it('ne dessine rien hors de la fenêtre du portage', () => {
    const fill = vi.fn()
    const ctx = {
      beginPath: vi.fn(), arc: vi.fn(), fill, stroke: vi.fn(),
    } as unknown as CanvasRenderingContext2D
    const layer: SkullCarrierInput = {
      style: { ink: '#fff', outline: '#000', reducedMotion: true },
      posOf: () => ({ x: 50, y: 50 }),
    }
    drawSkullCarrier(ctx, layer, [carry({ t0: 0, t1: 40 })], view, 80)
    expect(fill).not.toHaveBeenCalled()
  })

  /**
   * BORNAGE HORS CADRE (plan escouade hors cadre, chantier B, phase B3, décision D3,
   * 2026-09-10). Même vue que les autres cas : 1 unité monde = 2 px canvas.
   */
  it('PORTÉ ET HORS CADRE : le crâne est PLAQUÉ À LA MARGE, jamais projeté hors toile', () => {
    const arc = vi.fn()
    const ctx = { beginPath: vi.fn(), arc, fill: vi.fn(), stroke: vi.fn() } as unknown as CanvasRenderingContext2D
    const layer: SkullCarrierInput = {
      style: { ink: '#fff', outline: '#000', reducedMotion: true },
      // Monde (1000, 25) -> canvas brut (2000, 150) : très au-delà de la toile (200x200).
      posOf: () => ({ x: 1000, y: 25 }),
    }
    drawSkullCarrier(ctx, layer, [carry({ t0: 0, t1: 100 })], view, 10)
    const [cx] = arc.mock.calls[0] as number[]
    expect(cx).toBeCloseTo(view.width - OFFSCREEN_MARGIN_PX, 5)
    expect(cx).toBeLessThan(2000)
  })
})

/**
 * UN SEUL GLYPHE PAR IMAGE (lot 6.7 phase B2, item 1 — audit causes C9/C10).
 *
 * Le crâne est le seul des trois objets portés dont le rendu est PARTAGÉ entre deux calques :
 * celui-ci dessine le crâne PORTÉ, `objectiveObjectsLayer` dessine le crâne LIBRE, et
 * `skullPresenceAt` arbitre. Le silence de ce calque sur un porteur non localisable n'est donc
 * correct QUE si la présence bascule sur `free` à la même image — sans quoi le crâne disparaît
 * (694 images du parc Oddball). Ce test tient les deux bouts ensemble : c'est la seule façon de
 * prouver « un glyphe, jamais zéro, jamais deux ».
 */
describe('crâne porté et crâne libre : exactement un glyphe à chaque image', () => {
  const view: CanvasView = {
    bounds: { minX: 0, minY: 0, maxX: 100, maxY: 100 },
    width: 200,
    height: 200,
    pad: 0,
  }
  const lives: ReplayObjectiveObjectReady[] = [
    { family: 'ball', en: 'Oddball', fr: 'Crâne', t0: 5, t1: 5, pts: [{ t: 5, x: 1, y: 1 }] },
  ]
  const carries = [carry({ xuid: 'a', t0: 10, t1: 40 })]
  /** Porteur localisable jusqu'à l'image 20, muet ensuite. */
  const posOf = (_x: string, f: number) => (f <= 20 ? { x: 7, y: 8 } : null)

  const paint = (frame: number) => {
    const fill = vi.fn()
    const ctx = {
      beginPath: vi.fn(), arc: vi.fn(), fill, stroke: vi.fn(),
    } as unknown as CanvasRenderingContext2D
    const layer: SkullCarrierInput = {
      style: { ink: '#fff', outline: '#000', reducedMotion: true }, posOf,
    }
    drawSkullCarrier(ctx, layer, carries, view, frame)
    return { porte: fill.mock.calls.length > 0, presence: skullPresenceAt(lives, carries, frame, null, posOf) }
  }

  it('porteur localisable : le calque PORTÉ dessine, le calque libre se tait', () => {
    const { porte, presence } = paint(15)
    expect(porte).toBe(true)
    expect(presence.state).toBe('carried')
  })

  it('porteur sans position : le calque porté se tait, le calque LIBRE prend le relais', () => {
    const { porte, presence } = paint(30)
    expect(porte).toBe(false)
    expect(presence).toEqual({ state: 'free', at: { x: 7, y: 8 }, rolling: false })
  })
})
