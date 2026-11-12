/**
 * Tests — « Suivre » : la carte se cadre sur le joueur à chaque image (lot 4 du plan « maquettes du
 * rejeu »).
 *
 * Ce qu'ils protègent :
 *  1. À L'ACTIVATION, la carte monte au palier minimal et centre sur le joueur, sans toucher au temps.
 *  2. À CHAQUE IMAGE, y compris fractionnaire et sans que le temps avance officiellement : la
 *     position se relit sur la cellule de dessin, et le cadrage la suit.
 *  3. UN JOUEUR IMMOBILE NE REDEMANDE RIEN : pas d'écriture du cadrage à chaque image d'écran.
 *  4. SANS POSITION, LA CARTE RESTE OÙ ELLE EST : aucune demande de cadrage.
 *  5. ÉTEINT, OU SANS JOUEUR : la boucle n'existe pas ; quitter le suivi l'arrête.
 */
import { renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi, type Mock } from 'vitest'

import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { testReplayDoc } from '../test/testDoc'
import { useReplayFollow } from './useReplayFollow'
import type { ReplayZoom } from './useReplayZoom'

const DOC = testReplayDoc({ frameIntervalMs: 100, frameCount: 200 })

const PLAYER: ReplayPlayer = {
  xuid: 'A',
  lives: [
    { slot: 1, team: 0, xuid: 'A', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 0, y: 0 }, { t: 100, x: 100, y: 50 }] },
  ] as never,
}

let queue: Array<{ id: number; cb: FrameRequestCallback }> = []
let nextId = 1

beforeEach(() => {
  queue = []
  nextId = 1
  vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => {
    const id = nextId++
    queue.push({ id, cb })
    return id
  })
  vi.stubGlobal('cancelAnimationFrame', (id: number) => {
    queue = queue.filter((q) => q.id !== id)
  })
})

afterEach(() => vi.unstubAllGlobals())

/** Joue UNE image d'écran : exécute les rappels en attente. */
function tick() {
  const due = queue
  queue = []
  for (const q of due) q.cb(0)
}

function fakeZoom(): ReplayZoom & { focusOn: Mock<ReplayZoom['focusOn']> } {
  return {
    level: 1, center: { x: 0, y: 0 }, canZoomIn: true, canZoomOut: false, canPan: false,
    zoomIn: vi.fn(), zoomOut: vi.fn(), reset: vi.fn(), panStep: vi.fn(), panBy: vi.fn(), zoomAt: vi.fn(),
    focusOn: vi.fn<ReplayZoom['focusOn']>(),
  }
}

describe('useReplayFollow', () => {
  it('à l’activation : cadre le joueur au palier minimal, tout de suite', () => {
    const zoom = fakeZoom()
    renderHook(() => useReplayFollow({ doc: DOC, player: PLAYER, enabled: true, frameRef: { current: 20 }, zoom }))
    expect(zoom.focusOn).toHaveBeenCalledTimes(1)
    expect(zoom.focusOn).toHaveBeenCalledWith({ x: 20, y: 10 }, 2)
  })

  it('suit l’image de la cellule de dessin à chaque image d’écran, fractionnaire comprise', () => {
    const zoom = fakeZoom()
    const frameRef = { current: 20 }
    renderHook(() => useReplayFollow({ doc: DOC, player: PLAYER, enabled: true, frameRef, zoom }))
    frameRef.current = 20.5
    tick()
    expect(zoom.focusOn).toHaveBeenLastCalledWith({ x: 20.5, y: 10.25 }, 2)
    frameRef.current = 21.25
    tick()
    expect(zoom.focusOn).toHaveBeenLastCalledWith({ x: 21.25, y: 10.625 }, 2)
    expect(zoom.focusOn).toHaveBeenCalledTimes(3)
  })

  it('un joueur qui ne bouge pas ne redemande rien', () => {
    const zoom = fakeZoom()
    renderHook(() => useReplayFollow({ doc: DOC, player: PLAYER, enabled: true, frameRef: { current: 20 }, zoom }))
    tick()
    tick()
    tick()
    expect(zoom.focusOn).toHaveBeenCalledTimes(1)
  })

  it('sans position (mort depuis longtemps) : la carte reste où elle est', () => {
    const zoom = fakeZoom()
    const frameRef = { current: 20 }
    renderHook(() => useReplayFollow({ doc: DOC, player: PLAYER, enabled: true, frameRef, zoom }))
    expect(zoom.focusOn).toHaveBeenCalledTimes(1)
    frameRef.current = 190 // la vie s'est close à l'image 100, la fenêtre après-mort est passée
    tick()
    expect(zoom.focusOn).toHaveBeenCalledTimes(1)
  })

  it('éteint ou sans joueur : aucune boucle, aucune demande', () => {
    const zoom = fakeZoom()
    renderHook(() => useReplayFollow({ doc: DOC, player: PLAYER, enabled: false, frameRef: { current: 20 }, zoom }))
    renderHook(() => useReplayFollow({ doc: DOC, player: null, enabled: true, frameRef: { current: 20 }, zoom }))
    expect(queue).toHaveLength(0)
    expect(zoom.focusOn).not.toHaveBeenCalled()
  })

  it('l’extinction arrête la boucle', () => {
    const zoom = fakeZoom()
    const frameRef = { current: 20 }
    const { rerender } = renderHook(
      ({ enabled }) => useReplayFollow({ doc: DOC, player: PLAYER, enabled, frameRef, zoom }),
      { initialProps: { enabled: true } },
    )
    rerender({ enabled: false })
    expect(queue).toHaveLength(0)
    frameRef.current = 50
    tick()
    expect(zoom.focusOn).toHaveBeenCalledTimes(1)
  })

  it('le démontage arrête la boucle', () => {
    const zoom = fakeZoom()
    const { unmount } = renderHook(() => useReplayFollow({ doc: DOC, player: PLAYER, enabled: true, frameRef: { current: 20 }, zoom }))
    unmount()
    expect(queue).toHaveLength(0)
  })
})
