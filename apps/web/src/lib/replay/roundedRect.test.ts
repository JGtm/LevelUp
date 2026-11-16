/**
 * roundedRectPath — `ctx.roundRect` quand il existe ; sinon un chemin fermé de quatre `arcTo`,
 * au rayon borné. Dans les deux cas, un chemin neuf.
 */
import { describe, expect, it, vi } from 'vitest'

import { roundedRectPath } from './roundedRect'

function fakeCtx(withRoundRect: boolean) {
  const ctx = {
    beginPath: vi.fn(),
    closePath: vi.fn(),
    moveTo: vi.fn(),
    lineTo: vi.fn(),
    arcTo: vi.fn(),
    ...(withRoundRect ? { roundRect: vi.fn() } : {}),
  }
  return ctx as typeof ctx & { roundRect?: ReturnType<typeof vi.fn> }
}

const RECT = { x: 10, y: 20, w: 100, h: 40 }

describe('roundedRectPath', () => {
  it('avec ctx.roundRect : un chemin neuf et un seul appel, sans arcTo', () => {
    const ctx = fakeCtx(true)
    roundedRectPath(ctx as unknown as CanvasRenderingContext2D, RECT, 8)
    expect(ctx.beginPath).toHaveBeenCalledOnce()
    expect(ctx.roundRect).toHaveBeenCalledExactlyOnceWith(10, 20, 100, 40, 8)
    expect(ctx.arcTo).not.toHaveBeenCalled()
  })

  it('sans ctx.roundRect : le repli trace les quatre coins et ferme le chemin', () => {
    const ctx = fakeCtx(false)
    roundedRectPath(ctx as unknown as CanvasRenderingContext2D, RECT, 8)
    expect(ctx.beginPath).toHaveBeenCalledOnce()
    expect(ctx.moveTo).toHaveBeenCalledExactlyOnceWith(18, 20)
    expect(ctx.arcTo.mock.calls).toEqual([
      [110, 20, 110, 60, 8],
      [110, 60, 10, 60, 8],
      [10, 60, 10, 20, 8],
      [10, 20, 110, 20, 8],
    ])
    expect(ctx.closePath).toHaveBeenCalledOnce()
  })

  it('sans ctx.roundRect, le rayon est borné à la demi-hauteur et jamais négatif', () => {
    const ctx = fakeCtx(false)
    roundedRectPath(ctx as unknown as CanvasRenderingContext2D, RECT, 999)
    expect(ctx.arcTo.mock.calls.every((c) => c[4] === 20)).toBe(true)
    const neg = fakeCtx(false)
    roundedRectPath(neg as unknown as CanvasRenderingContext2D, RECT, -3)
    expect(neg.arcTo.mock.calls.every((c) => c[4] === 0)).toBe(true)
  })
})
