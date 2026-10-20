/**
 * segmentLabelFit.test.ts — une valeur s'écrit dans son segment seulement avec 6 px de marge
 * de chaque côté (règle S3 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26).
 */
import { describe, expect, it } from 'vitest'
import { labelFitsInSegment, measureHiddenSegments, SEGMENT_LABEL_MARGIN_PX } from './segmentLabelFit'

describe('labelFitsInSegment', () => {
  it('6 px de marge de chaque côté, bornes comprises', () => {
    expect(SEGMENT_LABEL_MARGIN_PX).toBe(6)
    expect(labelFitsInSegment(10, 22)).toBe(true)
    expect(labelFitsInSegment(10, 21.9)).toBe(false)
  })
  it('largeur d’étiquette inconnue (0) → ne tient pas : la valeur part au repli', () => {
    expect(labelFitsInSegment(0, 500)).toBe(false)
  })
})

describe('measureHiddenSegments', () => {
  it('rend les clés des segments dont l’étiquette ne tient pas', () => {
    const root = document.createElement('div')
    root.innerHTML =
      '<div data-fit-key="a"><span data-fit-label>12</span></div>' +
      '<div data-fit-key="b"><span data-fit-label>3</span></div>'
    const widths: Record<string, number> = { a: 40, b: 10 }
    root.querySelectorAll<HTMLElement>('[data-fit-key]').forEach((el) => {
      el.getBoundingClientRect = () => ({ width: widths[el.dataset.fitKey ?? ''] }) as DOMRect
      const label = el.querySelector<HTMLElement>('[data-fit-label]')!
      label.getBoundingClientRect = () => ({ width: 8 }) as DOMRect
    })
    expect([...measureHiddenSegments(root)]).toEqual(['b'])
  })
})
