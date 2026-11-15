/**
 * useAnchoredPanel.test.ts — la position du panneau de la vue : sous le bouton, bord droit
 * aligné, ramené dans la fenêtre, défilant en interne quand l'écran est bas.
 */
import { describe, expect, it } from 'vitest'

import { anchoredPosition } from './useAnchoredPanel'

const VIEWPORT = { width: 1280, height: 800 }

describe('anchoredPosition', () => {
  it('ouvre SOUS le bouton, bord droit aligné sur le sien', () => {
    const pos = anchoredPosition({ right: 1000, bottom: 100 }, VIEWPORT, 416)
    expect(pos.top).toBe(108)
    expect(pos.left).toBe(1000 - 416)
  })

  it('ne dépasse jamais à droite de la fenêtre', () => {
    const pos = anchoredPosition({ right: 1400, bottom: 100 }, VIEWPORT, 416)
    expect(pos.left + 416).toBeLessThanOrEqual(VIEWPORT.width - 8)
  })

  it('ne dépasse jamais à gauche : une fenêtre étroite garde la marge', () => {
    const pos = anchoredPosition({ right: 200, bottom: 100 }, { width: 360, height: 800 }, 416)
    expect(pos.left).toBe(8)
  })

  it('la hauteur utile est ce qui reste sous le bouton ; le panneau défile au-delà', () => {
    const pos = anchoredPosition({ right: 1000, bottom: 100 }, VIEWPORT, 416)
    expect(pos.maxHeight).toBe(800 - 100 - 8 - 8)
  })

  it('un écran bas garde une hauteur minimale lisible', () => {
    const pos = anchoredPosition({ right: 1000, bottom: 700 }, { width: 1280, height: 720 }, 416)
    expect(pos.maxHeight).toBe(160)
  })
})
