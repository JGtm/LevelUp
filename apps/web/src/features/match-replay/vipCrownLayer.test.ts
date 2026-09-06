import { describe, expect, it, vi } from 'vitest'

import { type CanvasView } from './replayView'
import type { ReplayVipPeriod } from './replayNormalize'
import { drawVipCrown, vipActiveAt, type VipCrownInput } from './vipCrownLayer'

const period = (over: Partial<ReplayVipPeriod>): ReplayVipPeriod => ({
  xuid: '2533274806055812',
  t0: 0,
  t1: 100,
  closed: true,
  ...over,
})

describe('vipActiveAt', () => {
  it('rend les périodes qui couvrent l’image, bornes incluses', () => {
    const a = period({ xuid: 'a', t0: 0, t1: 50 })
    const b = period({ xuid: 'b', t0: 40, t1: 100 })
    // À l’image 45, les deux VIP (un par camp) portent la couronne — chevauchement légitime.
    expect(vipActiveAt([a, b], 45).map((p) => p.xuid)).toEqual(['a', 'b'])
    // Bornes incluses : t0 et t1 comptent.
    expect(vipActiveAt([a], 0).map((p) => p.xuid)).toEqual(['a'])
    expect(vipActiveAt([a], 50).map((p) => p.xuid)).toEqual(['a'])
    // Hors intervalle : rien.
    expect(vipActiveAt([a], 51)).toEqual([])
  })
})

describe('drawVipCrown', () => {
  const view: CanvasView = {
    bounds: { minX: 0, minY: 0, maxX: 100, maxY: 100 },
    width: 200,
    height: 200,
    pad: 0,
  }

  it('ne dessine pas un VIP non localisable (aucune position propre à inventer)', () => {
    const ctx = { beginPath: vi.fn(), moveTo: vi.fn(), lineTo: vi.fn(), closePath: vi.fn(), fill: vi.fn() } as unknown as CanvasRenderingContext2D
    const layer: VipCrownInput = { style: { ink: '#fff', reducedMotion: true }, posOf: () => null }
    drawVipCrown(ctx, layer, [period({ t0: 0, t1: 100 })], view, 10)
    expect(ctx.fill).not.toHaveBeenCalled()
  })

  it('dessine la couronne sur la position relue du VIP courant', () => {
    const fill = vi.fn()
    const ctx = { beginPath: vi.fn(), moveTo: vi.fn(), lineTo: vi.fn(), closePath: vi.fn(), fill } as unknown as CanvasRenderingContext2D
    const layer: VipCrownInput = {
      style: { ink: '#fff', reducedMotion: true },
      posOf: () => ({ x: 50, y: 50 }),
    }
    drawVipCrown(ctx, layer, [period({ t0: 0, t1: 100 })], view, 10)
    expect(fill).toHaveBeenCalledTimes(1)
  })
})
