/**
 * Tests — useReplayWheelZoom : la molette suit la toile, y compris quand celle-ci n'existe qu'une
 * fois les données du plan arrivées (Vue match « Occupation du terrain », Tactique).
 */
import { act, render } from '@testing-library/react'
import { useRef } from 'react'
import { describe, expect, it, vi } from 'vitest'

import type { CanvasView } from '../model/replayView'
import type { ReplayZoom } from './useReplayZoom'
import { useReplayWheelZoom, WHEEL_STEP } from './useReplayWheelZoom'

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

const VIEW: CanvasView = {
  bounds: { minX: 0, minY: 0, maxX: 100, maxY: 100 },
  width: 100,
  height: 100,
  pad: 0,
}

function Plan({ zoom, pret }: { zoom: ReplayZoom; pret: boolean }) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  useReplayWheelZoom(canvasRef, zoom, VIEW)
  return pret ? <canvas data-testid="toile" ref={canvasRef} /> : <p>chargement</p>
}

function molette(el: Element, deltaY: number) {
  const event = new WheelEvent('wheel', { deltaY, bubbles: true, cancelable: true })
  act(() => {
    el.dispatchEvent(event)
  })
  return event
}

describe('useReplayWheelZoom', () => {
  it('toile montée au premier rendu : un cran grossit, la page ne défile pas', () => {
    const zoom = spies()
    const { getByTestId } = render(<Plan zoom={zoom} pret />)
    expect(molette(getByTestId('toile'), -WHEEL_STEP).defaultPrevented).toBe(true)
    expect(zoom.zoomAt).toHaveBeenCalledTimes(1)
    expect(vi.mocked(zoom.zoomAt).mock.calls[0][0]).toBe(1)
  })

  it('toile montée APRÈS le premier rendu : la molette la suit', () => {
    const zoom = spies()
    const { getByTestId, rerender } = render(<Plan zoom={zoom} pret={false} />)
    rerender(<Plan zoom={zoom} pret />)
    molette(getByTestId('toile'), WHEEL_STEP)
    expect(zoom.zoomAt).toHaveBeenCalledTimes(1)
    expect(vi.mocked(zoom.zoomAt).mock.calls[0][0]).toBe(-1)
  })
})
