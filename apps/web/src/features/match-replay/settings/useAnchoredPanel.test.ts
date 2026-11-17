/**
 * useAnchoredPanel.test.ts — la position du panneau de la vue : sous le bouton, bord droit
 * aligné, ramené dans la fenêtre, défilant en interne quand l'écran est bas.
 */
import { renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { anchoredPosition, useAnchoredPanel } from './useAnchoredPanel'

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

describe('anchoredPosition : côté d’alignement', () => {
  it('par défaut, le bord droit : un déclencheur muni d’un bord gauche ne change rien', () => {
    const sansOption = anchoredPosition({ right: 1000, bottom: 100 }, VIEWPORT, 330)
    const avecBordGauche = anchoredPosition({ left: 700, right: 1000, bottom: 100 }, VIEWPORT, 330)
    expect(avecBordGauche).toEqual(sansOption)
    expect(sansOption.left).toBe(1000 - 330)
  })

  it('« start » aligne le bord GAUCHE sur celui du déclencheur', () => {
    const pos = anchoredPosition({ left: 140, right: 400, bottom: 100 }, VIEWPORT, 330, { align: 'start' })
    expect(pos.left).toBe(140)
    expect(pos.top).toBe(108)
  })

  it('« start » près du bord droit de la page : le panneau s’aligne par la droite du déclencheur', () => {
    const pos = anchoredPosition({ left: 1100, right: 1270, bottom: 100 }, VIEWPORT, 330, { align: 'start' })
    expect(pos.left).toBe(1270 - 330)
    expect(pos.left + 330).toBeLessThanOrEqual(VIEWPORT.width - 8)
  })

  it('« start » : la dernière position qui tient à gauche garde le bord gauche, un pixel de plus bascule', () => {
    const maxLeft = VIEWPORT.width - 330 - 8
    expect(anchoredPosition({ left: maxLeft, right: maxLeft + 100, bottom: 0 }, VIEWPORT, 330, { align: 'start' }).left).toBe(maxLeft)
    expect(anchoredPosition({ left: maxLeft + 1, right: maxLeft + 101, bottom: 0 }, VIEWPORT, 330, { align: 'start' }).left).toBe(
      Math.min(maxLeft + 101 - 330, maxLeft),
    )
  })

  it('« start » sans bord gauche lu : repli sur le bord droit, jamais de panneau perdu', () => {
    const pos = anchoredPosition({ right: 1000, bottom: 100 }, VIEWPORT, 330, { align: 'start' })
    expect(pos.left).toBe(1000 - 330)
  })
})

describe('anchoredPosition : élément borne', () => {
  it('la hauteur s’arrête au bas de l’élément borne, moins la marge', () => {
    const pos = anchoredPosition({ right: 1000, bottom: 100 }, VIEWPORT, 330, { boundBottom: 600 })
    expect(pos.maxHeight).toBe(600 - 100 - 8 - 8)
  })

  it('un élément borne plus bas que la fenêtre ne la dépasse pas', () => {
    const pos = anchoredPosition({ right: 1000, bottom: 100 }, VIEWPORT, 330, { boundBottom: 5000 })
    expect(pos.maxHeight).toBe(800 - 100 - 8 - 8)
  })

  it('un élément borne trop proche garde la hauteur minimale lisible', () => {
    const pos = anchoredPosition({ right: 1000, bottom: 100 }, VIEWPORT, 330, { boundBottom: 150 })
    expect(pos.maxHeight).toBe(160)
  })

  it('sans élément borne, la fenêtre borne comme avant', () => {
    expect(anchoredPosition({ right: 1000, bottom: 100 }, VIEWPORT, 330, {}).maxHeight).toBe(800 - 100 - 8 - 8)
  })
})

describe('useAnchoredPanel : les deux options, lues sur le DOM', () => {
  /** Les options sont lues par leurs valeurs, mais la référence de la borne doit rester la même d'un rendu à l'autre. */
  const SANS_BORNE = { current: null }

  function boite(rect: Partial<DOMRect>): HTMLElement {
    const el = document.createElement('div')
    el.getBoundingClientRect = () => ({ left: 0, right: 0, top: 0, bottom: 0, width: 0, height: 0, x: 0, y: 0, toJSON: () => ({}), ...rect })
    return el
  }

  it('« start » et l’élément borne : bord gauche du déclencheur, hauteur jusqu’au bas de l’élément borne', () => {
    const trigger = { current: boite({ left: 140, right: 400, bottom: 60 }) }
    const bound = { current: boite({ bottom: 500 }) }
    const { result } = renderHook(() => useAnchoredPanel(trigger, 330, { align: 'start', boundRef: bound }))
    expect(result.current.left).toBe(140)
    expect(result.current.top).toBe(68)
    expect(result.current.maxHeight).toBe(500 - 60 - 8 - 8)
  })

  it('sans options, le panneau garde le bord droit et la hauteur de la fenêtre', () => {
    const trigger = { current: boite({ left: 140, right: 1000, bottom: 60 }) }
    const { result } = renderHook(() => useAnchoredPanel(trigger, 330))
    expect(result.current.left).toBe(1000 - 330)
    expect(result.current.maxHeight).toBe(window.innerHeight - 60 - 8 - 8)
  })

  it('un élément borne absent retombe sur la fenêtre', () => {
    const trigger = { current: boite({ left: 140, right: 400, bottom: 60 }) }
    const { result } = renderHook(() => useAnchoredPanel(trigger, 330, { align: 'start', boundRef: SANS_BORNE }))
    expect(result.current.maxHeight).toBe(window.innerHeight - 60 - 8 - 8)
  })
})
