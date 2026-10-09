/**
 * Tests — useReplayZoomKeys (le zoom au clavier des plans de la Vue match et de la Tactique).
 *
 * La règle protégée : une frappe ne zoome QUE si le plan est survolé ou a le focus — la Vue match
 * a ses propres raccourcis (flèches) et la page a des champs de saisie.
 */
import { act, render } from '@testing-library/react'
import { useRef } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { ReplayZoom } from './useReplayZoom'
import { useReplayZoomKeys } from './useReplayZoomKeys'

function spies(): ReplayZoom {
  return {
    level: 1,
    center: { x: 0, y: 0 },
    canZoomIn: true,
    canZoomOut: false,
    canPan: false,
    zoomIn: vi.fn(),
    zoomOut: vi.fn(),
    reset: vi.fn(),
    panStep: vi.fn(),
    zoomAt: vi.fn(),
    panBy: vi.fn(),
  }
}

function Plan({ zoom }: { zoom: ReplayZoom }) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  useReplayZoomKeys(canvasRef, zoom)
  return (
    <div>
      <div data-testid="cadre">
        <canvas ref={canvasRef} />
        <button type="button">+</button>
      </div>
      <input data-testid="champ" />
      <button type="button" data-testid="ailleurs">
        ailleurs
      </button>
    </div>
  )
}

function press(key: string, init: KeyboardEventInit = {}, target: EventTarget = window) {
  const event = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true, ...init })
  act(() => {
    target.dispatchEvent(event)
  })
  return event
}

function survoler(el: Element) {
  act(() => {
    el.dispatchEvent(new Event('pointerenter'))
  })
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('useReplayZoomKeys', () => {
  it('plan NI survolé NI focalisé : la frappe reste à la page', () => {
    const zoom = spies()
    render(<Plan zoom={zoom} />)
    expect(press('+').defaultPrevented).toBe(false)
    expect(zoom.zoomIn).not.toHaveBeenCalled()
  })

  it('plan survolé : + grossit, − réduit, 0 revoit toute la carte', () => {
    const zoom = spies()
    const { getByTestId } = render(<Plan zoom={zoom} />)
    survoler(getByTestId('cadre'))
    expect(press('+').defaultPrevented).toBe(true)
    press('-')
    press('0')
    expect(zoom.zoomIn).toHaveBeenCalledTimes(1)
    expect(zoom.zoomOut).toHaveBeenCalledTimes(1)
    expect(zoom.reset).toHaveBeenCalledTimes(1)
  })

  it('pointeur sorti du plan : plus rien', () => {
    const zoom = spies()
    const { getByTestId } = render(<Plan zoom={zoom} />)
    const cadre = getByTestId('cadre')
    survoler(cadre)
    act(() => {
      cadre.dispatchEvent(new Event('pointerleave'))
    })
    press('+')
    expect(zoom.zoomIn).not.toHaveBeenCalled()
  })

  it('focus dans le plan (commande de cadrage cliquée) : la frappe zoome', () => {
    const zoom = spies()
    const { getByText } = render(<Plan zoom={zoom} />)
    const bouton = getByText('+')
    act(() => bouton.focus())
    press('=', {}, bouton)
    expect(zoom.zoomIn).toHaveBeenCalledTimes(1)
  })

  it('focus ailleurs, champ de saisie, ou modificateur : jamais', () => {
    const zoom = spies()
    const { getByTestId } = render(<Plan zoom={zoom} />)
    const ailleurs = getByTestId('ailleurs')
    act(() => ailleurs.focus())
    press('+', {}, ailleurs)
    survoler(getByTestId('cadre'))
    press('+', {}, getByTestId('champ'))
    press('+', { ctrlKey: true })
    press('0', { metaKey: true })
    expect(zoom.zoomIn).not.toHaveBeenCalled()
    expect(zoom.reset).not.toHaveBeenCalled()
  })
})
