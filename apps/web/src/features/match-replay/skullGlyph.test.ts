/**
 * skullGlyph.test.ts — LE GLYPHE PARTAGÉ DU CRÂNE (crâne libre + crâne porté).
 *
 * CE QU'IL PROTÈGE : l'habillage « comme le drapeau » (retour utilisateur du 2026-08-28) — un
 * liseré à l'encre du FOND posé SOUS le disque, le disque à l'encre neutre par-dessus, et deux
 * orbites qui font lire « crâne ». Et l'opacité servie est bien appliquée.
 */
import { describe, expect, it, vi } from 'vitest'

import { drawSkullGlyph, SKULL_GLYPH_RADIUS } from './skullGlyph'

function ctxEspion() {
  return {
    globalAlpha: 1,
    fillStyle: '',
    strokeStyle: '',
    lineWidth: 0,
    beginPath: vi.fn(),
    arc: vi.fn(),
    fill: vi.fn(),
    stroke: vi.fn(),
  } as unknown as CanvasRenderingContext2D & {
    arc: ReturnType<typeof vi.fn>
    fill: ReturnType<typeof vi.fn>
    stroke: ReturnType<typeof vi.fn>
  }
}

describe('drawSkullGlyph — l habillage du drapeau', () => {
  it('pose un liseré (stroke) SOUS le disque, puis le disque et deux orbites (fill)', () => {
    const ctx = ctxEspion()
    drawSkullGlyph(ctx, { x: 40, y: 30 }, { ink: 'var(--ink)', outline: 'var(--bg)', alpha: 0.9 })
    // Le liseré : un seul trait, au rayon du disque.
    expect(ctx.stroke).toHaveBeenCalledTimes(1)
    expect(ctx.arc).toHaveBeenCalledWith(40, 30, SKULL_GLYPH_RADIUS, 0, Math.PI * 2)
    // Le disque + les deux orbites : trois remplissages.
    expect(ctx.fill).toHaveBeenCalledTimes(3)
    // Quatre arcs en tout : liseré, disque, deux orbites.
    expect(ctx.arc).toHaveBeenCalledTimes(4)
  })

  it('applique l opacité servie (un portage « ouvert » pulse par cette valeur)', () => {
    const ctx = ctxEspion()
    drawSkullGlyph(ctx, { x: 0, y: 0 }, { ink: 'a', outline: 'b', alpha: 0.42 })
    expect(ctx.globalAlpha).toBe(0.42)
  })

  it('centre le glyphe sur le point servi (l appelant applique son propre décalage)', () => {
    const ctx = ctxEspion()
    drawSkullGlyph(ctx, { x: 12, y: 34 }, { ink: 'a', outline: 'b', alpha: 1 })
    // Le disque est centré exactement sur (12, 34) — aucun décalage interne.
    expect(ctx.arc).toHaveBeenCalledWith(12, 34, SKULL_GLYPH_RADIUS, 0, Math.PI * 2)
  })
})
