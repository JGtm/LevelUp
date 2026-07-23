/**
 * Tests divergentZeroGradient — fonction pure. `resolveToken` renvoie '' hors
 * runtime CSS : on vérifie la STRUCTURE (type linéaire, 4 arrêts, position de la
 * bascule `zeroRatio`), pas les couleurs résolues.
 */
import { describe, expect, it } from 'vitest'

import { divergentZeroGradient } from './divergentZeroGradient'

describe('divergentZeroGradient', () => {
  it('toujours un dégradé linéaire vertical à 4 arrêts', () => {
    const g = divergentZeroGradient([3, -1, 2])
    expect(g.type).toBe('linear')
    expect(g).toMatchObject({ x: 0, y: 0, x2: 0, y2: 1 })
    expect(g.colorStops).toHaveLength(4)
  })

  it('valeurs mixtes : bascule à la fraction top/span depuis le haut', () => {
    // top = max(5,0) = 5 ; bot = min(-5,0) = -5 ; span = 10 ; zeroRatio = 5/10 = 0.5
    const g = divergentZeroGradient([5, -5])
    expect(g.colorStops[1].offset).toBe(0.5)
    expect(g.colorStops[2].offset).toBe(0.5)
  })

  it('toutes positives : bascule en bas (zeroRatio = 1, entièrement positif)', () => {
    // top = 8 ; bot = min(...,0) = 0 ; span = 8 ; zeroRatio = 8/8 = 1
    const g = divergentZeroGradient([2, 5, 8])
    expect(g.colorStops[1].offset).toBe(1)
    expect(g.colorStops[2].offset).toBe(1)
  })

  it('toutes négatives : bascule en haut (zeroRatio = 0, entièrement négatif)', () => {
    // top = max(...,0) = 0 ; bot = -6 ; span = 6 ; zeroRatio = 0/6 = 0
    const g = divergentZeroGradient([-1, -3, -6])
    expect(g.colorStops[1].offset).toBe(0)
    expect(g.colorStops[2].offset).toBe(0)
  })

  it('série vide ou étendue nulle : zeroRatio = 1 (robuste, pas de NaN)', () => {
    expect(divergentZeroGradient([]).colorStops[1].offset).toBe(1)
    expect(divergentZeroGradient([0, 0]).colorStops[1].offset).toBe(1)
  })
})
