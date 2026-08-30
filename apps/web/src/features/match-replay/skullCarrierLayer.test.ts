import { describe, expect, it, vi } from 'vitest'

import type { CanvasView } from './objectivesLayer'
import type { ReplaySkullCarry } from './replayNormalize'
import { drawSkullCarrier, skullCarrierActiveAt, type SkullCarrierInput } from './skullCarrierLayer'

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
})
