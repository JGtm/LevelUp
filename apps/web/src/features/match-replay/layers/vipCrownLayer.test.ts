import { describe, expect, it, vi } from 'vitest'

import { type CanvasView } from '../model/replayView'
import type { ReplayVipPeriod } from '../../../lib/replay/replayNormalize'
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

  // Un contexte qui consigne l'encre au moment de chaque remplissage.
  function recordingCtx() {
    const fills: string[] = []
    const ctx = {
      beginPath: vi.fn(), moveTo: vi.fn(), lineTo: vi.fn(), closePath: vi.fn(), rect: vi.fn(), arc: vi.fn(),
      stroke: vi.fn(),
      fillStyle: '',
      fill: vi.fn(function (this: { fillStyle: string }) { fills.push(this.fillStyle) }),
    }
    return { ctx: ctx as unknown as CanvasRenderingContext2D, fills, raw: ctx }
  }
  const style = (inkOf: (xuid: string) => string) => ({ inkOf, outline: 'fond', reducedMotion: true })

  it('ne dessine pas un VIP non localisable (aucune position propre à inventer)', () => {
    const { ctx, raw } = recordingCtx()
    const layer: VipCrownInput = { style: style(() => 'neutre'), posOf: () => null }
    drawVipCrown(ctx, layer, [period({ t0: 0, t1: 100 })], view, 10)
    expect(raw.fill).not.toHaveBeenCalled()
  })

  it('dessine la couronne sur la position relue du VIP courant, liseré puis forme', () => {
    const { ctx, raw } = recordingCtx()
    const layer: VipCrownInput = { style: style(() => 'neutre'), posOf: () => ({ x: 50, y: 50 }) }
    drawVipCrown(ctx, layer, [period({ t0: 0, t1: 100 })], view, 10)
    expect(raw.stroke).toHaveBeenCalledTimes(1)
    expect(raw.fill).toHaveBeenCalledTimes(1)
  })

  it('peint chaque couronne à l’encre du camp de son VIP', () => {
    const { ctx, fills } = recordingCtx()
    const inkOf = (xuid: string) => (xuid === 'allie' ? 'encre-alliee' : 'encre-adverse')
    const layer: VipCrownInput = { style: style(inkOf), posOf: () => ({ x: 50, y: 50 }) }
    drawVipCrown(ctx, layer, [period({ xuid: 'allie' }), period({ xuid: 'adverse' })], view, 10)
    expect(fills).toEqual(['encre-alliee', 'encre-adverse'])
  })
})
