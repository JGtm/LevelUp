/**
 * useVisibleUneFois — l'élément n'est « visible » qu'après être entré dans le champ de vision,
 * le reste ensuite, et l'observateur est débranché (à la première entrée et au démontage).
 */
import { act, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useVisibleUneFois } from './useVisibleUneFois'

type Rappel = (entrees: Array<Pick<IntersectionObserverEntry, 'isIntersecting'>>) => void

let rappel: Rappel
let options: IntersectionObserverInit | undefined
const disconnect = vi.fn()
const observe = vi.fn()

class FauxObservateur {
  constructor(cb: Rappel, opts?: IntersectionObserverInit) {
    rappel = cb
    options = opts
  }
  observe = observe
  disconnect = disconnect
}

function Sonde({ marge }: { marge?: number }) {
  const [ref, visible] = useVisibleUneFois<HTMLDivElement>(marge)
  return (
    <div ref={ref} data-testid="sonde">
      {visible ? 'vu' : 'pas vu'}
    </div>
  )
}

beforeEach(() => {
  disconnect.mockReset()
  observe.mockReset()
  options = undefined
  vi.stubGlobal('IntersectionObserver', FauxObservateur)
})
afterEach(() => vi.unstubAllGlobals())

describe('useVisibleUneFois', () => {
  it('hors du champ de vision, l’élément n’est pas visible ; il le devient à l’entrée', () => {
    render(<Sonde />)
    expect(screen.getByTestId('sonde')).toHaveTextContent('pas vu')
    expect(observe).toHaveBeenCalledTimes(1)
    act(() => rappel([{ isIntersecting: false }]))
    expect(screen.getByTestId('sonde')).toHaveTextContent('pas vu')
    act(() => rappel([{ isIntersecting: true }]))
    expect(screen.getByTestId('sonde')).toHaveTextContent('vu')
  })

  it('l’observateur est débranché à la première entrée, et la visibilité ne retombe pas', () => {
    render(<Sonde />)
    act(() => rappel([{ isIntersecting: true }]))
    expect(disconnect).toHaveBeenCalled()
    act(() => rappel([{ isIntersecting: false }]))
    expect(screen.getByTestId('sonde')).toHaveTextContent('vu')
  })

  it('le démontage débranche l’observateur', () => {
    const { unmount } = render(<Sonde />)
    unmount()
    expect(disconnect).toHaveBeenCalled()
  })

  it('la marge de chargement est celle demandée (100 px par défaut)', () => {
    const { unmount } = render(<Sonde />)
    expect(options?.rootMargin).toBe('100px')
    unmount()
    render(<Sonde marge={250} />)
    expect(options?.rootMargin).toBe('250px')
  })

  it('sans IntersectionObserver, l’élément est réputé visible', () => {
    vi.stubGlobal('IntersectionObserver', undefined)
    render(<Sonde />)
    expect(screen.getByTestId('sonde')).toHaveTextContent('vu')
  })
})
