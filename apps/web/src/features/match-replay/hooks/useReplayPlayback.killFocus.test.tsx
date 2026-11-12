/**
 * Tests — useReplayPlayback face à la mise en valeur d'une élimination (plan « maquettes du
 * rejeu », lot 2) : le canal de saut, et les gestes qui effacent la mise en valeur.
 *
 * # CE QU'ILS PROTÈGENT
 *
 * (1) Une demande de saut de la page est exécutée par le lecteur — pause, puis pose sur l'image,
 * bornée à la fenêtre de gameplay — et ne passe PAS pour un geste de l'utilisateur : le clic qui
 * pose la mise en valeur ne doit pas l'effacer aussitôt. (2) Tout ce qui déplace ou relance la
 * lecture à la main (Lecture, Recommencer, glissé de frise, sauts, pas d'image, saut vers un
 * repère) prévient `onUserMove`, d'où le canvas efface la mise en valeur. Mettre en PAUSE ne
 * l'efface pas : seule la reprise le fait.
 */
import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createRef, type ChangeEvent, type RefObject } from 'react'

import { createPlaybackStore } from '../model/playbackStore'
import type { ReplayWindowBounds } from '../model/replayWindow'
import { testReplayDoc } from '../test/testDoc'
import { AUTOPLAY_KEY } from '../settings/useReplaySettings'
import { useReplayPlayback } from './useReplayPlayback'

const DOC = testReplayDoc({ frameCount: 51 })

const FENETRE: ReplayWindowBounds = {
  startFrame: 10,
  leadInFrame: 9,
  endFrame: 40,
  startMs: 10_000,
  endMs: 40_000,
}

beforeEach(() => {
  localStorage.clear()
  vi.stubGlobal('requestAnimationFrame', () => 1)
  vi.stubGlobal('cancelAnimationFrame', () => {})
})

afterEach(() => {
  vi.unstubAllGlobals()
})

function monter(frame: number, enLecture: boolean) {
  localStorage.setItem(AUTOPLAY_KEY, String(enLecture))
  const frameRef = createRef<number>() as RefObject<number>
  frameRef.current = frame
  const draw = vi.fn()
  const onUserMove = vi.fn()
  const store = createPlaybackStore()
  const view = renderHook(() =>
    useReplayPlayback({
      doc: DOC,
      playWindow: FENETRE,
      baseFps: 10,
      speed: 1,
      renderWidth: 480,
      frameRef,
      draw,
      soundTick: vi.fn(),
      soundSeek: vi.fn(),
      onEnded: vi.fn(),
      onTransportGesture: vi.fn(),
      onSeekRequest: store.onSeekRequest,
      onUserMove,
    }),
  )
  return { ...view, frameRef, draw, onUserMove, store }
}

describe('le canal de saut — exécuté par le lecteur', () => {
  it('met en PAUSE et pose la lecture sur l’image demandée', () => {
    const { result, frameRef, store } = monter(20, true)
    expect(result.current.playing).toBe(true)
    act(() => store.requestSeek(33))
    expect(result.current.playing).toBe(false)
    expect(frameRef.current).toBe(33)
  })

  it('peint l’image posée : un curseur déplacé sans repeindre montrerait l’instant précédent', () => {
    const { store, draw } = monter(20, false)
    draw.mockClear()
    act(() => store.requestSeek(33))
    expect(draw).toHaveBeenCalledTimes(1)
  })

  it('reste borné à la fenêtre de gameplay, comme tout saut', () => {
    const { store, frameRef } = monter(20, false)
    act(() => store.requestSeek(9_999))
    expect(frameRef.current).toBe(FENETRE.endFrame)
    act(() => store.requestSeek(-5))
    expect(frameRef.current).toBe(FENETRE.leadInFrame)
  })

  it('n’est PAS un geste de l’utilisateur : il n’efface pas la mise en valeur qu’il vient de poser', () => {
    const { store, onUserMove } = monter(20, true)
    act(() => store.requestSeek(33))
    expect(onUserMove).not.toHaveBeenCalled()
  })

  it('se désabonne au démontage', () => {
    const { store, frameRef, unmount } = monter(20, false)
    unmount()
    act(() => store.requestSeek(33))
    expect(frameRef.current).toBe(20)
  })
})

describe('onUserMove — ce qui efface la mise en valeur', () => {
  function glisser(frame: number) {
    return { currentTarget: { value: String(frame) } } as unknown as ChangeEvent<HTMLInputElement>
  }

  it('la LECTURE (reprise) l’efface', () => {
    const { result, onUserMove } = monter(20, false)
    act(() => result.current.togglePlay())
    expect(result.current.playing).toBe(true)
    expect(onUserMove).toHaveBeenCalledTimes(1)
  })

  it('mettre en PAUSE ne l’efface pas', () => {
    const { result, onUserMove } = monter(20, true)
    act(() => result.current.togglePlay())
    expect(result.current.playing).toBe(false)
    expect(onUserMove).not.toHaveBeenCalled()
  })

  it('« Recommencer » l’efface', () => {
    const { result, onUserMove } = monter(20, false)
    act(() => result.current.restart())
    expect(onUserMove).toHaveBeenCalledTimes(1)
  })

  it('le glissé de la frise l’efface', () => {
    const { result, onUserMove } = monter(20, false)
    act(() => result.current.onScrub(glisser(30)))
    expect(onUserMove).toHaveBeenCalledTimes(1)
  })

  it('les sauts de ±N secondes (flèches, J et L) l’effacent', () => {
    const { result, onUserMove } = monter(20, false)
    act(() => result.current.seekBy(5))
    expect(onUserMove).toHaveBeenCalledTimes(1)
  })

  it('le pas d’image (virgule et point) l’efface', () => {
    const { result, onUserMove } = monter(20, false)
    act(() => result.current.stepFrames(1))
    expect(onUserMove).toHaveBeenCalledTimes(1)
  })

  it('le saut vers un repère de la frise l’efface', () => {
    const { result, onUserMove } = monter(20, false)
    act(() => result.current.seekToFrame(33))
    expect(onUserMove).toHaveBeenCalledTimes(1)
  })
})
