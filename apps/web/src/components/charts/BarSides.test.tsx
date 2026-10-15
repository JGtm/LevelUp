/**
 * BarSides.test — la valeur qui ne tient pas dans son segment se lit À CÔTÉ de la barre, sur sa
 * ligne (15/10) : structure figée (jsdom ne mesure aucune géométrie ; la mesure au navigateur
 * rend 0 px d'écart entre le centre de la valeur et celui de sa barre).
 *
 *   - rangée `flex items-center` : la valeur se centre sur la hauteur de la barre ;
 *   - gauche : collée à la barre (`justify-end`), droite : `justify-start` ;
 *   - la place est réservée même sans valeur (barres de largeur égale d'une ligne à l'autre) ;
 *   - la valeur reste dans le DOM, avec la pastille de la couleur du segment.
 */
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { AfterBar, BarWithSides, SIDE_GAP_PX, SideValue, sideSlotPx } from './BarSides'

describe('sideSlotPx', () => {
  it('rien à écrire : rien à réserver ; sinon la place du plus long texte, pastille comprise, plafonnée', () => {
    expect(sideSlotPx([])).toBe(0)
    expect(sideSlotPx(['', ''])).toBe(0)
    const court = sideSlotPx(['12'])
    const long = sideSlotPx(['12', '123 · 45,6 %'])
    expect(long).toBeGreaterThan(court)
    expect(court).toBeGreaterThan(14)
    expect(sideSlotPx(['x'.repeat(200)])).toBeLessThanOrEqual(112)
  })
})

describe('BarWithSides', () => {
  function mount(props: { left?: boolean; right?: boolean; leftPx?: number; rightPx?: number }) {
    render(
      <BarWithSides
        leftPx={props.leftPx ?? 60}
        rightPx={props.rightPx ?? 60}
        left={props.left && <SideValue color="#3b82f6" testId="g">1 · 10 %</SideValue>}
        right={props.right && <SideValue color="#ef4444" testId="d">9</SideValue>}
      >
        <div data-testid="bar" className="h-[22px]" />
      </BarWithSides>,
    )
    return screen.getByTestId('bar').parentElement!.parentElement!
  }

  it('rangée en flex items-center : la valeur se centre sur la hauteur de la barre', () => {
    const row = mount({ left: true, right: true })
    expect(row.className).toContain('flex')
    expect(row.className).toContain('items-center')
    for (const side of row.querySelectorAll('[data-side]')) expect(side.className).toContain('items-center')
  })

  it('gauche collée à la barre, droite collée à la barre, valeur dans le DOM avec sa pastille', () => {
    mount({ left: true, right: true })
    const left = screen.getByTestId('g').closest('[data-side]') as HTMLElement
    const right = screen.getByTestId('d').closest('[data-side]') as HTMLElement
    expect(left.getAttribute('data-side')).toBe('left')
    expect(left.className).toContain('justify-end')
    expect(left.style.paddingRight).toBe(`${SIDE_GAP_PX}px`)
    expect(right.getAttribute('data-side')).toBe('right')
    expect(right.className).toContain('justify-start')
    expect(screen.getByTestId('g').textContent).toBe('1 · 10 %')
    expect(screen.getByTestId('g').querySelector('[aria-hidden]')).not.toBeNull()
  })

  it('la place est réservée même sans valeur ; à 0, pas de gouttière', () => {
    const row = mount({})
    const [left, right] = [...row.querySelectorAll<HTMLElement>('[data-side]')]
    expect(left.style.width).toBe('60px')
    expect(right.style.width).toBe('60px')
  })

  it('à 0, pas de gouttière', () => {
    const none = mount({ leftPx: 0, rightPx: 0 })
    expect(none.querySelectorAll('[data-side]')).toHaveLength(0)
  })
})

describe('AfterBar', () => {
  it('posée juste après le bout de la barre, sur sa hauteur, dans la place réservée', () => {
    render(
      <div className="relative" style={{ paddingRight: 40 }}>
        <AfterBar endPct={50} slotPx={40}>
          <SideValue color="#3b82f6" testId="v">3</SideValue>
        </AfterBar>
      </div>,
    )
    const after = screen.getByTestId('v').closest('[data-side]') as HTMLElement
    expect(after.getAttribute('data-side')).toBe('after')
    expect(after.className).toContain('inset-y-0')
    expect(after.className).toContain('items-center')
    expect(after.style.left).toContain('0.5 * (100% - 40px)')
  })
})
