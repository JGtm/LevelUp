/**
 * bombGlyph.test.ts — LE GLYPHE PARTAGÉ DE LA BOMBE (portée + posée au sol).
 *
 * CE QU'IL PROTÈGE : l'habillage du crâne et du drapeau — un liseré à l'encre du FOND posé
 * SOUS la silhouette (corps + collier + mèche), la silhouette à l'encre neutre par-dessus.
 * Et l'opacité servie est bien appliquée, et le centre est le point servi.
 */
import { describe, expect, it, vi } from 'vitest'

import { BOMB_GLYPH_RADIUS, drawBombGlyph } from './bombGlyph'

function ctxEspion() {
  return {
    globalAlpha: 1,
    fillStyle: '',
    strokeStyle: '',
    lineWidth: 0,
    lineCap: 'butt',
    beginPath: vi.fn(),
    arc: vi.fn(),
    rect: vi.fn(),
    moveTo: vi.fn(),
    quadraticCurveTo: vi.fn(),
    fill: vi.fn(),
    stroke: vi.fn(),
  } as unknown as CanvasRenderingContext2D & {
    arc: ReturnType<typeof vi.fn>
    rect: ReturnType<typeof vi.fn>
    fill: ReturnType<typeof vi.fn>
    stroke: ReturnType<typeof vi.fn>
  }
}

describe('drawBombGlyph — l habillage du crane et du drapeau', () => {
  it('pose le liseré (strokes) SOUS la silhouette, puis un seul remplissage', () => {
    const ctx = ctxEspion()
    drawBombGlyph(ctx, { x: 40, y: 30 }, { ink: 'var(--ink)', outline: 'var(--bg)', alpha: 0.9 })
    // Trois traits : liseré de la silhouette, liseré de la mèche, mèche à l'encre.
    expect(ctx.stroke).toHaveBeenCalledTimes(3)
    // Un seul remplissage : la silhouette (corps + collier, un chemin).
    expect(ctx.fill).toHaveBeenCalledTimes(1)
    // Le corps est au rayon publié.
    expect(ctx.arc).toHaveBeenCalledWith(40, 30, BOMB_GLYPH_RADIUS, 0, Math.PI * 2)
  })

  it('applique l opacité servie (un portage « ouvert » pulse par cette valeur)', () => {
    const ctx = ctxEspion()
    drawBombGlyph(ctx, { x: 0, y: 0 }, { ink: 'a', outline: 'b', alpha: 0.42 })
    expect(ctx.globalAlpha).toBe(0.42)
  })

  it('centre le glyphe sur le point servi (l appelant applique son propre décalage)', () => {
    const ctx = ctxEspion()
    drawBombGlyph(ctx, { x: 12, y: 34 }, { ink: 'a', outline: 'b', alpha: 1 })
    expect(ctx.arc).toHaveBeenCalledWith(12, 34, BOMB_GLYPH_RADIUS, 0, Math.PI * 2)
  })
})
