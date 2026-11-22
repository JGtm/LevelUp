/**
 * Tests — LE BLOC DE LA TOILE (`ReplayCanvasBlock`) selon la place de la barre de lecture : dans le bloc (Standard,
 * sans cadre propre), posée sur le plan (cadre arrondi, barre hors du flux, dans le cadre), ou SOUS le bloc (le bloc
 * arrondi ne porte que le plan, la barre est son frère, sur le fond de la page, à l'écart voulu). L'arbre est le même
 * dans les trois cas, et la colonne que mesure la page contient le plan ET la barre.
 */
import { render } from '@testing-library/react'
import { createRef } from 'react'
import { describe, expect, it } from 'vitest'

import { COTES_BARRE_COMPACTE as C } from '../model/cotesBarreCompacte'
import type { BarPlacement } from '../model/transportBarLogic'
import { ReplayCanvasBlock } from './ReplayCanvasBlock'

function monter(placement: BarPlacement) {
  const containerRef = createRef<HTMLDivElement>()
  const view = render(
    <ReplayCanvasBlock containerRef={containerRef} placement={placement} transport={<div data-testid="barre" />}>
      <div data-testid="plan" />
    </ReplayCanvasBlock>,
  )
  const bloc = view.container.querySelector<HTMLElement>('[data-replay-plan-block]') as HTMLElement
  const barre = view.getByTestId('barre')
  return { ...view, bloc, containerRef, barre, plan: view.getByTestId('plan'), emplacement: barre.parentElement as HTMLElement }
}

describe('sous le bloc (Cinéma, Plein cadre sans barre fine)', () => {
  it('le bloc arrondi porte le plan et PAS la barre ; la barre est la suivante dans la colonne mesurée', () => {
    const { bloc, barre, plan, containerRef, emplacement } = monter('below')
    expect(bloc.className).toContain('rounded-lg')
    expect(bloc.className).toContain('border')
    expect(bloc.contains(plan)).toBe(true)
    expect(bloc.contains(barre)).toBe(false)
    expect(containerRef.current?.contains(bloc)).toBe(true)
    expect(containerRef.current?.contains(barre)).toBe(true)
    expect(bloc.nextElementSibling).toBe(emplacement)
  })

  it('le bloc se ferme au-dessus de la barre : marge basse du bloc, écart voulu avant la barre, barre dans le flux', () => {
    const { bloc, emplacement } = monter('below')
    expect(bloc.className).toContain('pb-3')
    expect(emplacement.style.marginTop).toBe(`${C.blocBarreEcartPx}px`)
    expect(emplacement.className).not.toContain('absolute')
  })
})

describe('sur le plan (Plein cadre, barre fine)', () => {
  it('le bloc est arrondi ; la barre est hors du flux, collée au bas du cadre, dans son liseré, sans marge ni écart', () => {
    const { bloc, emplacement } = monter('overlay')
    expect(bloc.className).toContain('rounded-lg')
    expect(bloc.className).not.toContain('pb-3')
    expect(emplacement.className).toContain('absolute')
    expect(emplacement.className).toContain('bottom-px')
    expect(emplacement.className).toContain('inset-x-px')
    expect(emplacement.className).toContain('rounded-b')
    expect(emplacement.style.marginTop).toBe('')
  })
})

describe('dans le bloc (Standard)', () => {
  it('aucun cadre propre : le cadre pose UN bloc autour de la carte et de sa colonne ; la barre est dans le flux', () => {
    const { bloc, emplacement } = monter('in-block')
    expect(bloc.className).not.toContain('rounded')
    expect(bloc.className).not.toContain('border')
    expect(emplacement.className).not.toContain('absolute')
    expect(emplacement.style.marginTop).toBe('')
  })
})

describe("l'arbre est le même d'une place à l'autre", () => {
  it("la colonne mesurée porte toujours le bloc, puis l'emplacement de la barre, qui porte la barre", () => {
    for (const placement of ['in-block', 'overlay', 'below'] as const) {
      const { containerRef, bloc, plan, barre, emplacement, unmount } = monter(placement)
      expect([...(containerRef.current as HTMLElement).children]).toEqual([bloc, emplacement])
      expect(bloc.contains(plan)).toBe(true)
      expect(emplacement.contains(barre)).toBe(true)
      expect(emplacement.getAttribute('data-replay-bar-slot')).toBe(placement)
      unmount()
    }
  })
})
