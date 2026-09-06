/**
 * objectiveObjectsLayer.test.ts — LE CALQUE DU CRÂNE LIBRE.
 *
 * Les quatre propriétés testées ici sont celles qui font qu'il ne MENT pas :
 *   - il dessine l'objet là où le document dit qu'il est ;
 *   - il DISPARAÎT pendant les portages, parce que le document ne dit pas qui porte ;
 *   - il ne prolonge jamais une vie au-delà de sa dernière position émise ;
 *   - le survol vise exactement la forme dessinée.
 */
import { describe, expect, it, vi } from 'vitest'

import {
  drawFreeSkull,
  objectiveObjectAt,
  objectiveObjectHitAt,
  objectiveObjectsAt,
  type ObjectiveObjectsInput,
} from './objectiveObjectsLayer'
import { SKULL_GLYPH_RADIUS } from './skullGlyph'

import { type CanvasView } from '../replayView'
import type { ReplayObjectiveObjectReady } from '../replayNormalize'

/** Une vie qui roule de (0,0) à (2,0) entre les images 10 et 12. */
const vieQuiRoule: ReplayObjectiveObjectReady = {
  family: 'ball', en: 'Oddball', fr: 'Crâne', t0: 10, t1: 12,
  pts: [{ t: 10, x: 0, y: 0 }, { t: 11, x: 1, y: 0 }, { t: 12, x: 2, y: 0 }],
}

/** Une vie IMMOBILE : née à son socle, jamais bougée. Un seul point, et c'est réel. */
const vieImmobile: ReplayObjectiveObjectReady = {
  family: 'ball', en: 'Oddball', fr: 'Crâne', t0: 30, t1: 30,
  pts: [{ t: 30, x: 5, y: 5 }],
}

const view: CanvasView = {
  bounds: { minX: -10, maxX: 10, minY: -10, maxY: 10 },
  width: 200, height: 200, pad: 0,
} as CanvasView

const layer: ObjectiveObjectsInput = { style: { ink: 'var(--ink)', outline: 'var(--outline)' } }

describe('objectiveObjectAt — la position lue à une image', () => {
  it('rend la dernière position ÉMISE, sans interpoler', () => {
    expect(objectiveObjectAt(vieQuiRoule, 11)?.at).toEqual({ x: 1, y: 0 })
  })

  it('ne rend RIEN hors de la vie — un trou est un portage, et le porteur est inconnu', () => {
    expect(objectiveObjectAt(vieQuiRoule, 9)).toBeNull()
    expect(objectiveObjectAt(vieQuiRoule, 13)).toBeNull()
  })

  it('ne prolonge pas la dernière position au-delà de t1', () => {
    // C'EST LA PROPRIÉTÉ LA PLUS IMPORTANTE DU CALQUE. Laisser le crâne posé au sol après la
    // fin de sa vie le montrerait immobile pendant qu'un joueur court avec.
    expect(objectiveObjectAt(vieQuiRoule, 100)).toBeNull()
  })

  it('rend une vie IMMOBILE à son unique image', () => {
    const now = objectiveObjectAt(vieImmobile, 30)
    expect(now?.at).toEqual({ x: 5, y: 5 })
    expect(now?.rolling).toBe(false)
  })

  it('dit que l objet BOUGE encore quand un point suit dans la même vie', () => {
    expect(objectiveObjectAt(vieQuiRoule, 10)?.rolling).toBe(true)
    expect(objectiveObjectAt(vieQuiRoule, 12)?.rolling).toBe(false)
  })
})

describe('objectiveObjectsAt — toutes les vies d une image', () => {
  it('ne retient que celles qui répliquent', () => {
    const lives = [vieQuiRoule, vieImmobile]
    expect(objectiveObjectsAt(lives, 11)).toHaveLength(1)
    expect(objectiveObjectsAt(lives, 30)).toHaveLength(1)
    expect(objectiveObjectsAt(lives, 20)).toHaveLength(0)
  })
})

describe('drawFreeSkull — le tracé d une présence libre', () => {
  function ctxEspion() {
    return {
      globalAlpha: 1, fillStyle: '', strokeStyle: '', lineWidth: 0,
      beginPath: vi.fn(), arc: vi.fn(), fill: vi.fn(), stroke: vi.fn(),
    } as unknown as CanvasRenderingContext2D & { arc: ReturnType<typeof vi.fn> }
  }

  it('trace le glyphe partagé du crâne à la position monde servie (liseré, disque, orbites)', () => {
    const ctx = ctxEspion()
    drawFreeSkull(ctx, layer, { x: 1, y: 0 }, view, false)
    // Le glyphe du crâne pose son disque au rayon partagé, plus le liseré au même rayon et les
    // deux orbites (quatre arcs en tout).
    expect(ctx.arc).toHaveBeenCalledWith(
      expect.any(Number), expect.any(Number), SKULL_GLYPH_RADIUS, 0, Math.PI * 2,
    )
    expect(ctx.arc).toHaveBeenCalledTimes(4)
  })

  it('projette la position monde puis dessine (centre au monde = centre du canvas centré)', () => {
    const ctx = ctxEspion()
    // (0,0) monde est le centre d'une vue centrée sur zéro : (100, 100) canvas.
    drawFreeSkull(ctx, layer, { x: 0, y: 0 }, view, true)
    expect(ctx.arc).toHaveBeenCalledWith(100, 100, SKULL_GLYPH_RADIUS, 0, Math.PI * 2)
  })
})

describe('objectiveObjectHitAt — le survol', () => {
  it('vise exactement la forme dessinée', () => {
    // (0,0) monde est le centre d une vue centrée sur zéro : (100, 100) canvas.
    const hit = objectiveObjectHitAt([vieQuiRoule], view, 10, { x: 100, y: 100 })
    expect(hit?.now.life.family).toBe('ball')
    expect(hit?.at).toEqual({ x: 100, y: 100 })
  })

  it('ne touche rien loin du glyphe, ni hors de la vie', () => {
    expect(objectiveObjectHitAt([vieQuiRoule], view, 10, { x: 10, y: 10 })).toBeNull()
    expect(objectiveObjectHitAt([vieQuiRoule], view, 50, { x: 100, y: 100 })).toBeNull()
  })
})
