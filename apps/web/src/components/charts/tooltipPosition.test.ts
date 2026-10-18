import { describe, expect, it } from 'vitest'

import { chartTooltipPosition, lastPointerPosition, placeTooltip, TOOLTIP_GAP, TOOLTIP_MARGIN } from './tooltipPosition'
import { getTooltipBase } from './_utils'
import { getEChartsThemeColors } from '@/lib/echarts/themeColors'

const VIEW: [number, number] = [1000, 700]
const SIZE: [number, number] = [200, 100]

function inside(pos: [number, number], size: readonly [number, number], view: readonly [number, number]) {
  return (
    pos[0] >= TOOLTIP_MARGIN &&
    pos[1] >= TOOLTIP_MARGIN &&
    pos[0] + size[0] <= view[0] - TOOLTIP_MARGIN &&
    pos[1] + size[1] <= view[1] - TOOLTIP_MARGIN
  )
}

describe('placeTooltip : sans marque (infobulle d axe)', () => {
  it('se pose en diagonale du pointeur quand la place existe', () => {
    expect(placeTooltip({ pointer: [300, 300], size: SIZE, viewport: VIEW })).toEqual([300 + TOOLTIP_GAP, 300 + TOOLTIP_GAP])
  })
  it('bascule a gauche au bord droit', () => {
    const [x] = placeTooltip({ pointer: [950, 300], size: SIZE, viewport: VIEW })
    expect(x).toBe(950 - TOOLTIP_GAP - SIZE[0])
  })
  it('bascule au-dessus au bord bas', () => {
    const [, y] = placeTooltip({ pointer: [300, 680], size: SIZE, viewport: VIEW })
    expect(y).toBe(680 - TOOLTIP_GAP - SIZE[1])
  })
  it.each([
    [0, 0],
    [999, 0],
    [0, 699],
    [999, 699],
    [500, 350],
  ])('reste dans la fenetre au pointeur (%i, %i)', (px, py) => {
    expect(inside(placeTooltip({ pointer: [px, py], size: SIZE, viewport: VIEW }), SIZE, VIEW)).toBe(true)
  })
  it('borne a la marge quand la fenetre est plus petite que la bulle', () => {
    const pos = placeTooltip({ pointer: [50, 50], size: [400, 300], viewport: [300, 200] })
    expect(pos).toEqual([TOOLTIP_MARGIN, TOOLTIP_MARGIN])
  })
})

describe('placeTooltip : avec marque survolee', () => {
  it('barre verticale : au-dessus de la barre, centree sur le pointeur', () => {
    const mark = { left: 400, top: 300, width: 30, height: 200 }
    const [x, y] = placeTooltip({ pointer: [415, 400], size: SIZE, viewport: VIEW, mark })
    expect(y + SIZE[1]).toBeLessThanOrEqual(mark.top - TOOLTIP_GAP)
    expect(x).toBe(415 - SIZE[0] / 2)
  })
  it('barre verticale proche du haut : au-dessous du pied de la barre', () => {
    const mark = { left: 400, top: 20, width: 30, height: 200 }
    const [, y] = placeTooltip({ pointer: [415, 100], size: SIZE, viewport: VIEW, mark })
    expect(y).toBe(mark.top + mark.height + TOOLTIP_GAP)
  })
  it('barre horizontale : a droite de la barre', () => {
    const mark = { left: 100, top: 300, width: 300, height: 20 }
    const [x] = placeTooltip({ pointer: [250, 310], size: SIZE, viewport: VIEW, mark })
    expect(x).toBe(mark.left + mark.width + TOOLTIP_GAP)
  })
  it('barre horizontale qui touche le bord droit : a gauche de la barre', () => {
    const mark = { left: 600, top: 300, width: 390, height: 20 }
    const [x] = placeTooltip({ pointer: [800, 310], size: SIZE, viewport: VIEW, mark })
    expect(x).toBe(mark.left - TOOLTIP_GAP - SIZE[0])
  })
  it('jamais hors fenetre, quelle que soit la marque', () => {
    const marks = [
      { left: 0, top: 0, width: 1000, height: 700 },
      { left: 950, top: 650, width: 40, height: 40 },
      { left: 0, top: 0, width: 20, height: 20 },
    ]
    for (const mark of marks) {
      const pos = placeTooltip({ pointer: [mark.left + 1, mark.top + 1], size: SIZE, viewport: VIEW, mark })
      expect(inside(pos, SIZE, VIEW)).toBe(true)
    }
  })
})

describe('getTooltipBase : placement', () => {
  it('ajoute l infobulle au body, sans transition, avec la fonction de placement', () => {
    const base = getTooltipBase(getEChartsThemeColors())
    expect(base.appendToBody).toBe(true)
    expect(base.transitionDuration).toBe(0)
    expect(base.position).toBe(chartTooltipPosition)
  })
})

describe('chartTooltipPosition : adaptateur ECharts', () => {
  it('sans pointeur connu : laisse ECharts placer', () => {
    expect(lastPointerPosition()).toBeNull()
    expect(chartTooltipPosition([10, 10], null, null, null, { contentSize: SIZE })).toBeUndefined()
  })
  it('rend des coordonnees locales au graphe (origine deduite du pointeur)', () => {
    // Pointeur a (500, 300) fenetre ; ECharts le voit en (100, 50) local : origine du graphe (400, 250).
    window.dispatchEvent(new MouseEvent('mousemove', { clientX: 500, clientY: 300 }))
    const local = chartTooltipPosition([100, 50], null, null, null, { contentSize: SIZE })
    const expected = placeTooltip({
      pointer: [500, 300],
      size: SIZE,
      viewport: [document.documentElement.clientWidth || window.innerWidth, document.documentElement.clientHeight || window.innerHeight],
    })
    expect(local).toEqual([expected[0] - 400, expected[1] - 250])
  })
})
