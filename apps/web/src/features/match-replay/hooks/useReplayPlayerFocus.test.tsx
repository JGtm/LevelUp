/**
 * Tests — le câblage du panneau de joueur au canvas.
 *
 * Ce qu'ils protègent :
 *  1. RIEN N'EST FILTRÉ sans joueur masqué : `shownAtSlot` est `undefined`, les calques gardent
 *     exactement leur dessin d'avant.
 *  2. LES JOUEURS MASQUÉS (l'œil d'une tuile, « Seul » = tous les autres masqués) donnent un prédicat qui
 *     désigne leurs vies, indépendamment de la fiche ouverte.
 *  3. SUIVRE cadre le joueur de la CAMÉRA, fiche ouverte ou fermée.
 *  4. LE POINT DE VUE N'EST PAS UN PARAMÈTRE : ces actions ne le lisent ni ne l'écrivent.
 */
import { renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi, type Mock } from 'vitest'

import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import type { HiddenPlayers, PlayerFocus } from '../model/playerPanelLogic'
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
const OTHER: ReplayPlayer = {
  xuid: 'B',
  lives: [
    { slot: 2, team: 1, xuid: 'B', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 5, y: 5 }, { t: 100, x: 6, y: 5 }] },
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

const focus = (over: Partial<PlayerFocus> = {}): PlayerFocus => ({ player: PLAYER, solo: false, hidden: false, ...over })

/** Les options d'un rendu : ni fiche ni joueur masqué ni joueur suivi sauf ce que le cas pose. */
function options(over: Partial<Parameters<typeof useReplayPlayerFocus>[0]> = {}) {
  return { doc: DOC, followPlayer: null, focus: null, hidden: null, frameRef: { current: 10 }, zoom: zoom(), ...over }
}

describe('useReplayPlayerFocus', () => {
  it('aucun joueur masqué, fiche ouverte ou non : aucun prédicat', () => {
    expect(renderHook(() => useReplayPlayerFocus(options())).result.current.shownAtSlot).toBeUndefined()
    const f = focus()
    expect(renderHook(() => useReplayPlayerFocus(options({ focus: f }))).result.current.shownAtSlot).toBeUndefined()
  })

  it('un joueur masqué : le prédicat retire ses vies', () => {
    const hidden: HiddenPlayers = [PLAYER]
    const { result } = renderHook(() => useReplayPlayerFocus(options({ hidden })))
    expect(result.current.shownAtSlot?.(1, 10)).toBe(false)
    expect(result.current.shownAtSlot?.(2, 10)).toBe(true)
  })

  it('Seul A (B masqué) : le prédicat n’affiche que les vies de A', () => {
    const hidden: HiddenPlayers = [OTHER]
    const { result } = renderHook(() => useReplayPlayerFocus(options({ hidden })))
    expect(result.current.shownAtSlot?.(1, 10)).toBe(true)
    expect(result.current.shownAtSlot?.(2, 10)).toBe(false)
  })

  it('plusieurs joueurs masqués : les vies de chacun sont retirées', () => {
    const hidden: HiddenPlayers = [PLAYER, OTHER]
    const { result } = renderHook(() => useReplayPlayerFocus(options({ hidden })))
    expect(result.current.shownAtSlot?.(1, 10)).toBe(false)
    expect(result.current.shownAtSlot?.(2, 10)).toBe(false)
    expect(result.current.shownAtSlot?.(3, 10)).toBe(true)
  })

  it('le prédicat ne dépend pas de la fiche : ouvrir ou fermer la fiche ne le change pas', () => {
    const hidden: HiddenPlayers = [OTHER]
    const { result, rerender } = renderHook(({ f }) => useReplayPlayerFocus(options({ hidden, focus: f })), {
      initialProps: { f: null as PlayerFocus | null },
    })
    const sans = result.current.shownAtSlot
    rerender({ f: focus() })
    expect(result.current.shownAtSlot).toBe(sans)
    rerender({ f: focus({ player: OTHER, hidden: true }) })
    expect(result.current.shownAtSlot).toBe(sans)
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
    const shown = (hidden: HiddenPlayers | null) =>
      renderHook(() => useReplayPlayerFocus(options({ doc: DOC_P, hidden }))).result.current.shownProjectile

    it('rien n’est filtré sans joueur masqué', () => {
      expect(shown(null)).toBeUndefined()
    })

    it('un joueur masqué retire ses projectiles ; ceux des autres et sans auteur restent', () => {
      const s = shown([PLAYER])
      expect(s?.(0)).toBe(false)
      expect(s?.(1)).toBe(true)
      expect(s?.(2)).toBe(true)
    })

    it('Seul A (B masqué) garde les projectiles de A et ceux sans auteur lu ; il retire ceux de B', () => {
      const s = shown([OTHER])
      expect(s?.(0)).toBe(true)
      expect(s?.(1)).toBe(false)
      expect(s?.(2)).toBe(true)
    })
  })

  it('la caméra suit le joueur suivi : la carte se cadre sur lui ; sans joueur suivi, elle ne bouge pas', () => {
    const on = zoom()
    renderHook(() => useReplayPlayerFocus(options({ followPlayer: PLAYER, frameRef: { current: 20 }, zoom: on })))
    expect(on.focusOn).toHaveBeenCalledWith({ x: 20, y: 10 }, 2)

    const off = zoom()
    renderHook(() => useReplayPlayerFocus(options({ hidden: [OTHER], frameRef: { current: 20 }, zoom: off })))
    expect(off.focusOn).not.toHaveBeenCalled()
  })

  it('la caméra suit FICHE FERMÉE, et la fiche d’un AUTRE joueur ne la déplace pas', () => {
    const closed = zoom()
    renderHook(() => useReplayPlayerFocus(options({ followPlayer: PLAYER, frameRef: { current: 20 }, zoom: closed })))
    expect(closed.focusOn).toHaveBeenCalledTimes(1)

    const open = zoom()
    renderHook(() => useReplayPlayerFocus(options({
      followPlayer: PLAYER, focus: focus({ player: OTHER, solo: true }), frameRef: { current: 20 }, zoom: open,
    })))
    expect(open.focusOn).toHaveBeenCalledWith({ x: 20, y: 10 }, 2)
  })

  it('la fiche ouverte ne commande pas la caméra : seul le joueur suivi le fait', () => {
    const z = zoom()
    renderHook(() => useReplayPlayerFocus(options({ focus: focus(), frameRef: { current: 20 }, zoom: z })))
    expect(z.focusOn).not.toHaveBeenCalled()
  })

  it('l’anneau de la fiche : absent quand le joueur de la fiche est masqué', () => {
    const ouverte = renderHook(() => useReplayPlayerFocus(options({ focus: focus() }))).result.current.openPlayer
    expect(ouverte).toBeDefined()
    const masquee = renderHook(() => useReplayPlayerFocus(options({ focus: focus({ hidden: true }), hidden: [PLAYER] }))).result.current.openPlayer
    expect(masquee).toBeUndefined()
  })

  it('le prédicat garde son identité tant que les joueurs masqués ne changent pas (aucun redessin inutile)', () => {
    const hidden: HiddenPlayers = [PLAYER]
    const opts = options({ hidden })
    const { result, rerender } = renderHook(() => useReplayPlayerFocus(opts))
    const first = result.current.shownAtSlot
    rerender()
    expect(result.current.shownAtSlot).toBe(first)
  })
})
