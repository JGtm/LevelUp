/**
 * useContentWidth.test.tsx — la largeur utile d'une zone : contenu seul, suivie à chaque changement de taille,
 * et une zone montée après coup est mesurée aussi.
 */
import { act, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useContentWidth } from './useContentWidth'

/** Les observateurs posés par le hook, rappelables à la main (jsdom ne met rien en page). */
const rappels: (() => void)[] = []

class ObservateurDeTaille {
  private readonly rappel: () => void
  constructor(rappel: () => void) {
    this.rappel = rappel
  }
  observe() {
    rappels.push(this.rappel)
  }
  disconnect() {
    const i = rappels.indexOf(this.rappel)
    if (i >= 0) rappels.splice(i, 1)
  }
}

let largeurClient = 0

function Zone({ montee = true, style }: { montee?: boolean; style?: React.CSSProperties }) {
  const { width, ref } = useContentWidth()
  return (
    <>
      <output data-testid="largeur">{width}</output>
      {montee ? <div ref={ref} style={style} /> : null}
    </>
  )
}

beforeEach(() => {
  rappels.length = 0
  largeurClient = 0
  vi.stubGlobal('ResizeObserver', ObservateurDeTaille)
  vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockImplementation(() => largeurClient)
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

function redimensionner(largeur: number) {
  largeurClient = largeur
  act(() => {
    for (const rappel of [...rappels]) rappel()
  })
}

/** La largeur que le hook a rendue, lue dans le DOM. */
const lue = () => Number(screen.getByTestId('largeur').textContent)

describe('useContentWidth', () => {
  it('rend 0 tant que la zone n’est pas mesurée', () => {
    largeurClient = 0
    render(<Zone />)
    expect(lue()).toBe(0)
  })

  it('lit la largeur du contenu, marges intérieures retirées, puis la suit', () => {
    largeurClient = 900
    render(<Zone style={{ paddingLeft: '12px', paddingRight: '12px' }} />)
    expect(lue()).toBe(876)
    redimensionner(1000)
    expect(lue()).toBe(976)
  })

  it('une zone sans marge rend sa largeur entière, arrondie à l’entier inférieur', () => {
    largeurClient = 641.9
    render(<Zone />)
    expect(lue()).toBe(641)
  })

  it('une zone montée après coup est mesurée aussi', () => {
    largeurClient = 500
    const { rerender } = render(<Zone montee={false} />)
    expect(lue()).toBe(0)
    rerender(<Zone montee />)
    expect(lue()).toBe(500)
  })

  it('une largeur plus petite que les marges rend 0, jamais un négatif', () => {
    largeurClient = 10
    render(<Zone style={{ paddingLeft: '12px', paddingRight: '12px' }} />)
    expect(lue()).toBe(0)
  })

  it('l’observateur est retiré au démontage', () => {
    largeurClient = 300
    const { unmount } = render(<Zone />)
    expect(rappels).toHaveLength(1)
    unmount()
    expect(rappels).toHaveLength(0)
  })
})
