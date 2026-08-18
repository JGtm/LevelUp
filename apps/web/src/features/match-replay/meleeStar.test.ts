/**
 * Tests — L'ÉTOILE DU COUP DE MÊLÉE FATAL (D3/R2-3, verdict du 2026-08-18).
 *
 * Ce qu'ils tiennent :
 *  1. LA FORME. Huit branches alternées, jamais quatre (ce serait une croix — le verdict est
 *     explicite : « une étoile, pas une croix ») ; un contour fermé, un noyau.
 *  2. LA DURÉE. 400 ms, en TEMPS RÉEL — pas en images.
 *  3. L'AIGUILLAGE. Une mort de famille `melee` rend une étoile ET RIEN D'AUTRE ; les autres
 *     familles ne rendent jamais d'étoile.
 *  4. LE LIEU. Au point de la MORT (la victime dès qu'elle est relue), jamais entre deux points.
 */
import { describe, expect, it } from 'vitest'

import { drawKillFxLayer, type KillFxStyle } from './replayDraw'
import type { KillFxEntry } from './killFx'
import { drawMeleeStar, MELEE_STAR_MS, meleeStarProgress } from './meleeStar'
import { recordingContext, type CanvasOp } from './test/recordingContext'

const VIEW = { bounds: { minX: 0, minY: 0, maxX: 20, maxY: 20 }, width: 400, height: 400, pad: 0 }
const FRAME_MS = 100
const STYLE: KillFxStyle = {
  colorOfSlot: () => 'tueur',
  fallback: 'repli',
  reducedMotion: false,
  k: 1,
}

function entry(over: Partial<KillFxEntry> = {}): KillFxEntry {
  return {
    frame: 0, x: 5, y: 5, vx: null, vy: null, deathX: 15, deathY: 15,
    dist: null, fam: 'melee', slot: 3, seed: 7, ...over,
  }
}

function trace(fx: KillFxEntry, frame = 1, style: Partial<KillFxStyle> = {}): CanvasOp[] {
  const { ops, ctx } = recordingContext()
  drawKillFxLayer(ctx, [fx], VIEW, { frame, hold: 14, frameMs: FRAME_MS }, { ...STYLE, ...style })
  return ops
}

describe('meleeStarProgress — la durée est en TEMPS, pas en images', () => {
  it('vit 400 ms puis n’existe plus', () => {
    expect(meleeStarProgress(0, false)).toBe(0)
    expect(meleeStarProgress(200, false)).toBeCloseTo(0.5, 6)
    expect(meleeStarProgress(MELEE_STAR_MS, false)).toBe(1)
    expect(meleeStarProgress(MELEE_STAR_MS + 1, false)).toBeNull()
    expect(meleeStarProgress(-1, false)).toBeNull()
  })

  it('sous mouvement réduit, elle est FIGÉE à son plein éclat — elle ne jaillit pas', () => {
    expect(meleeStarProgress(0, true)).toBe(meleeStarProgress(399, true))
    expect(meleeStarProgress(MELEE_STAR_MS + 1, true)).toBeNull()
  })
})

describe('drawMeleeStar — une ÉTOILE, pas une croix', () => {
  it('trace HUIT branches alternées, refermées, plus un noyau', () => {
    const { ops, ctx } = recordingContext()
    drawMeleeStar(ctx, { x: 50, y: 50 }, 0.5, 1, 'encre')
    // 16 sommets = 8 pointes et 8 creux : un `moveTo` puis 15 `lineTo`.
    expect(ops.filter((o) => o.op === 'moveTo')).toHaveLength(1)
    expect(ops.filter((o) => o.op === 'lineTo')).toHaveLength(15)
    expect(ops.filter((o) => o.op === 'closePath')).toHaveLength(1)
    // Le noyau : le seul `arc` du tracé.
    expect(ops.filter((o) => o.op === 'arc')).toHaveLength(1)
    // Une croix se serait dite en 4 branches : la règle est ici, chiffrée.
    expect(ops.filter((o) => o.op === 'lineTo').length).not.toBe(3)
  })

  it('les sommets ALTERNENT long / court — c’est ce qui fait l’étoile', () => {
    const { ops, ctx } = recordingContext()
    drawMeleeStar(ctx, { x: 0, y: 0 }, 1, 1, 'encre')
    const rayons = ops
      .filter((o) => o.op === 'moveTo' || o.op === 'lineTo')
      .map((o) => Math.hypot(Number(o.args[0]), Number(o.args[1])))
    for (let i = 1; i < rayons.length; i++) {
      if (i % 2 === 1) expect(rayons[i]).toBeLessThan(rayons[i - 1])
      else expect(rayons[i]).toBeGreaterThan(rayons[i - 1])
    }
  })

  it('elle JAILLIT puis s’éteint : elle n’apparaît pas à sa taille finale', () => {
    const rayonMax = (p: number) => {
      const { ops, ctx } = recordingContext()
      drawMeleeStar(ctx, { x: 0, y: 0 }, p, 1, 'encre')
      return Math.max(
        ...ops.filter((o) => o.op === 'lineTo').map((o) => Math.hypot(Number(o.args[0]), Number(o.args[1]))),
      )
    }
    expect(rayonMax(0.1)).toBeLessThan(rayonMax(0.33))
    expect(rayonMax(0.33)).toBeCloseTo(rayonMax(0.9), 6)
  })
})

describe('drawKillFxLayer — l’aiguillage de la mêlée', () => {
  it('une mort de MÊLÉE rend l’étoile, et aucun anneau pointillé', () => {
    const ops = trace(entry())
    expect(ops.filter((o) => o.op === 'closePath')).toHaveLength(1)
    expect(ops.some((o) => o.op === 'setLineDash')).toBe(false)
  })

  it('elle se pose au LIEU DE LA MORT, pas à l’origine de l’effet', () => {
    // Victime à (15,15) sur une emprise 20x20 projetée en 400x400 : x = 300.
    const ops = trace(entry())
    const premier = ops.find((o) => o.op === 'moveTo')!
    expect(Number(premier.args[0])).toBeCloseTo(300, 6)
  })

  it('victime NON localisée : l’étoile retombe sur l’origine, jamais nulle part', () => {
    const ops = trace(entry({ deathX: null, deathY: null }))
    const premier = ops.find((o) => o.op === 'moveTo')!
    expect(Number(premier.args[0])).toBeCloseTo(100, 6)
  })

  it('passé 400 ms, l’étoile ne se dessine plus — la fenêtre du calque dure plus longtemps', () => {
    expect(trace(entry(), 3).some((o) => o.op === 'closePath')).toBe(true)
    expect(trace(entry(), 5).some((o) => o.op === 'closePath')).toBe(false)
  })

  it('une AUTRE famille ne rend jamais d’étoile', () => {
    const ops = trace(entry({ fam: 'ballistic' }))
    expect(ops.some((o) => o.op === 'closePath')).toBe(false)
  })

  it('elle porte la couleur du TUEUR, comme tout effet de mort', () => {
    const encres = trace(entry()).filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0])
    expect(encres).toContain('tueur')
  })
})
