/**
 * Tests — le câblage du panneau de joueur au canvas (lot 4 du plan « maquettes du rejeu »).
 *
 * Ce qu'ils protègent :
 *  1. RIEN N'EST FILTRÉ sans Seul ni Masquer : `shownAtSlot` est `undefined`, les calques gardent
 *     exactement leur dessin d'avant.
 *  2. SEUL ET MASQUER donnent un prédicat qui désigne les vies du joueur.
 *  3. SUIVRE n'est actif que si l'action est allumée, et cadre le joueur choisi.
 *  4. LE POINT DE VUE N'EST PAS UN PARAMÈTRE : ces actions ne le lisent ni ne l'écrivent.
 */
import { renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi, type Mock } from 'vitest'

import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import type { PlayerFocus } from '../model/playerPanelLogic'
import { testReplayDoc } from '../test/testDoc'
import { useReplayPlayerFocus } from './useReplayPlayerFocus'
import type { ReplayZoom } from './useReplayZoom'

const DOC = testReplayDoc({ frameIntervalMs: 100, frameCount: 200 })

const PLAYER: ReplayPlayer = {
  xuid: 'A',
  lives: [
    { slot: 1, team: 0, xuid: 'A', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 0, y: 0 }, { t: 100, x: 100, y: 50 }] },
  ] as never,
}

function zoom(): ReplayZoom & { focusOn: Mock<ReplayZoom['focusOn']> } {
  return {
    level: 1, center: { x: 0, y: 0 }, canZoomIn: true, canZoomOut: false, canPan: false,
    zoomIn: vi.fn(), zoomOut: vi.fn(), reset: vi.fn(), panStep: vi.fn(), panBy: vi.fn(), zoomAt: vi.fn(),
    focusOn: vi.fn<ReplayZoom['focusOn']>(),
  }
}

beforeEach(() => {
  vi.stubGlobal('requestAnimationFrame', () => 1)
  vi.stubGlobal('cancelAnimationFrame', () => {})
})
afterEach(() => vi.unstubAllGlobals())

const focus = (over: Partial<PlayerFocus> = {}): PlayerFocus => ({
  player: PLAYER, follow: false, solo: false, hide: false, ...over,
})

describe('useReplayPlayerFocus', () => {
  it('panneau fermé, ou ni Seul ni Masquer : aucun prédicat', () => {
    const z = zoom()
    const frameRef = { current: 10 }
    expect(renderHook(() => useReplayPlayerFocus({ doc: DOC, focus: null, frameRef, zoom: z })).result.current.shownAtSlot).toBeUndefined()
    const f = focus({ follow: true })
    expect(renderHook(() => useReplayPlayerFocus({ doc: DOC, focus: f, frameRef, zoom: z })).result.current.shownAtSlot).toBeUndefined()
  })

  it('Masquer : le prédicat retire les vies du joueur', () => {
    const f = focus({ hide: true })
    const { result } = renderHook(() => useReplayPlayerFocus({ doc: DOC, focus: f, frameRef: { current: 10 }, zoom: zoom() }))
    expect(result.current.shownAtSlot?.(1, 10)).toBe(false)
    expect(result.current.shownAtSlot?.(2, 10)).toBe(true)
  })

  it('Seul : le prédicat n’affiche que les vies du joueur', () => {
    const f = focus({ solo: true })
    const { result } = renderHook(() => useReplayPlayerFocus({ doc: DOC, focus: f, frameRef: { current: 10 }, zoom: zoom() }))
    expect(result.current.shownAtSlot?.(1, 10)).toBe(true)
    expect(result.current.shownAtSlot?.(2, 10)).toBe(false)
  })

  describe('projectiles et grenades du joueur', () => {
    const DOC_P = testReplayDoc({
      frameIntervalMs: 100,
      frameCount: 200,
      shots: [{ slot: 1, t: 10, x: 0, y: 0, w: '0xW' }, { slot: 2, t: 10, x: 0, y: 0, w: '0xW' }],
      projectiles: [
        { t0: 10, p: [[0, 0, 0], [5, 1, 1]], detoT: 15, detoX: 1, detoY: 1, shot: 0 },
        { t0: 10, p: [[0, 0, 0], [5, 1, 1]], detoT: 15, detoX: 1, detoY: 1, shot: 1 },
        { t0: 10, p: [[0, 0, 0], [5, 1, 1]], detoT: 15, detoX: 1, detoY: 1 },
      ],
    })
    const shown = (f: PlayerFocus | null) =>
      renderHook(() => useReplayPlayerFocus({ doc: DOC_P, focus: f, frameRef: { current: 10 }, zoom: zoom() }))
        .result.current.shownProjectile

    it('rien n’est filtré sans Seul ni Masquer', () => {
      expect(shown(null)).toBeUndefined()
      expect(shown(focus({ follow: true }))).toBeUndefined()
    })

    it('Masquer retire les projectiles du joueur ; ceux des autres et sans auteur restent', () => {
      const s = shown(focus({ hide: true }))
      expect(s?.(0)).toBe(false)
      expect(s?.(1)).toBe(true)
      expect(s?.(2)).toBe(true)
    })

    it('Seul ne garde que les projectiles du joueur : un auteur inconnu n’est pas prouvé le sien, il est retiré', () => {
      const s = shown(focus({ solo: true }))
      expect(s?.(0)).toBe(true)
      expect(s?.(1)).toBe(false)
      expect(s?.(2)).toBe(false)
    })
  })

  it('Suivre allumé : la carte se cadre sur le joueur ; éteint, elle ne bouge pas', () => {
    const on = zoom()
    const f = focus({ follow: true })
    renderHook(() => useReplayPlayerFocus({ doc: DOC, focus: f, frameRef: { current: 20 }, zoom: on }))
    expect(on.focusOn).toHaveBeenCalledWith({ x: 20, y: 10 }, 2)

    const off = zoom()
    const g = focus({ solo: true })
    renderHook(() => useReplayPlayerFocus({ doc: DOC, focus: g, frameRef: { current: 20 }, zoom: off }))
    expect(off.focusOn).not.toHaveBeenCalled()
  })

  it('le prédicat garde son identité tant que le panneau ne change pas (aucun redessin inutile)', () => {
    const f = focus({ hide: true })
    const z = zoom()
    const frameRef = { current: 10 }
    const { result, rerender } = renderHook(() => useReplayPlayerFocus({ doc: DOC, focus: f, frameRef, zoom: z }))
    const first = result.current.shownAtSlot
    rerender()
    expect(result.current.shownAtSlot).toBe(first)
  })
})
