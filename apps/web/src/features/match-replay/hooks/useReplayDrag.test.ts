/**
 * Tests — useReplayDrag.
 *
 * CE QU'ILS PROTÈGENT, et aucun n'est cosmétique :
 *
 * 1. LE SENS. Tirer vers la droite doit amener vers soi ce qui était à gauche — donc déplacer la
 *    FENÊTRE vers la gauche. Un signe inversé donne une carte qui fuit la main, et c'est le
 *    genre de défaut qu'on ne voit qu'en l'essayant.
 * 2. LA CONVERSION. Le pointeur parle en pixels, le cadrage en unités monde. Le même geste doit
 *    parcourir la même distance de CARTE quel que soit le grossissement.
 * 3. LE GEL. `dragging` est ce qui empêche les quatre calques statiques de recuire à chaque
 *    mouvement de pointeur. S'il restait faux, le rejeu se figerait pendant le geste.
 */
import { describe, expect, it, vi } from 'vitest'
import { act, renderHook } from '@testing-library/react'

import { sceneCenter, visibleBounds } from '../replayLogic'
import { useReplayDrag } from './useReplayDrag'
import type { ReplayZoom } from './useReplayZoom'

const SCENE = { minX: 0, minY: 0, maxX: 100, maxY: 60 }
const VIEW = { bounds: SCENE, width: 900, height: 480, pad: 24 }

function fakeZoom(over: Partial<ReplayZoom> = {}): ReplayZoom {
  return {
    level: 2,
    center: sceneCenter(SCENE),
    canZoomIn: true,
    canZoomOut: true,
    canPan: true,
    zoomIn: vi.fn(),
    zoomOut: vi.fn(),
    reset: vi.fn(),
    panStep: vi.fn(),
    panBy: vi.fn(),
    zoomAt: vi.fn(),
    ...over,
  }
}

/** Un événement de pointeur réduit à ce que le hook lit. */
function evt(x: number, y: number) {
  return {
    clientX: x,
    clientY: y,
    button: 0,
    pointerId: 1,
    currentTarget: {
      setPointerCapture: vi.fn(),
      releasePointerCapture: vi.fn(),
      hasPointerCapture: () => true,
    },
  } as unknown as Parameters<ReturnType<typeof useReplayDrag>['onPointerDown']>[0]
}

describe('useReplayDrag', () => {
  it('à 1x, le clic ne démarre rien — le survol garde la balise', () => {
    const zoom = fakeZoom({ canPan: false })
    const { result } = renderHook(() => useReplayDrag(zoom, VIEW))
    act(() => result.current.onPointerDown(evt(100, 100)))
    expect(result.current.dragging).toBe(false)
    act(() => result.current.onPointerMove(evt(150, 100)))
    expect(zoom.panBy).not.toHaveBeenCalled()
  })

  it('tirer vers la DROITE déplace la fenêtre vers la GAUCHE', () => {
    const zoom = fakeZoom()
    const { result } = renderHook(() => useReplayDrag(zoom, VIEW))
    act(() => result.current.onPointerDown(evt(100, 100)))
    act(() => result.current.onPointerMove(evt(150, 100)))
    expect(zoom.panBy).toHaveBeenCalledTimes(1)
    expect(vi.mocked(zoom.panBy).mock.calls[0][0]).toBeLessThan(0)
  })

  it('tirer vers le BAS déplace la fenêtre vers le HAUT du monde', () => {
    const zoom = fakeZoom()
    const { result } = renderHook(() => useReplayDrag(zoom, VIEW))
    act(() => result.current.onPointerDown(evt(100, 100)))
    act(() => result.current.onPointerMove(evt(100, 160)))
    expect(vi.mocked(zoom.panBy).mock.calls[0][1]).toBeGreaterThan(0)
  })

  // LE MÊME GESTE PARCOURT LA MÊME DISTANCE DE CARTE À TOUS LES GROSSISSEMENTS : la fenêtre
  // rétrécit, l'échelle grandit, et la division par l'échelle annule l'un par l'autre.
  it('convertit les pixels en unités monde selon l échelle du cadrage', () => {
    const c = sceneCenter(SCENE)
    const large = { ...VIEW, bounds: visibleBounds(SCENE, 1, c.x, c.y) }
    const serre = { ...VIEW, bounds: visibleBounds(SCENE, 3, c.x, c.y) }

    const a = fakeZoom()
    const ra = renderHook(() => useReplayDrag(a, large))
    act(() => ra.result.current.onPointerDown(evt(0, 0)))
    act(() => ra.result.current.onPointerMove(evt(90, 0)))

    const b = fakeZoom()
    const rb = renderHook(() => useReplayDrag(b, serre))
    act(() => rb.result.current.onPointerDown(evt(0, 0)))
    act(() => rb.result.current.onPointerMove(evt(90, 0)))

    const dxLarge = Math.abs(vi.mocked(a.panBy).mock.calls[0][0])
    const dxSerre = Math.abs(vi.mocked(b.panBy).mock.calls[0][0])
    expect(dxSerre).toBeLessThan(dxLarge)
    expect(dxSerre).toBeCloseTo(dxLarge / 3, 6)
  })

  it('gèle la cuisson pendant le geste, et la relâche à la fin', () => {
    const { result } = renderHook(() => useReplayDrag(fakeZoom(), VIEW))
    act(() => result.current.onPointerDown(evt(10, 10)))
    expect(result.current.dragging).toBe(true)
    act(() => result.current.onPointerUp(evt(60, 10)))
    expect(result.current.dragging).toBe(false)
  })

  it('après le relâchement, un mouvement ne déplace plus rien', () => {
    const zoom = fakeZoom()
    const { result } = renderHook(() => useReplayDrag(zoom, VIEW))
    act(() => result.current.onPointerDown(evt(10, 10)))
    act(() => result.current.onPointerUp(evt(60, 10)))
    vi.mocked(zoom.panBy).mockClear()
    act(() => result.current.onPointerMove(evt(200, 10)))
    expect(zoom.panBy).not.toHaveBeenCalled()
  })
})
