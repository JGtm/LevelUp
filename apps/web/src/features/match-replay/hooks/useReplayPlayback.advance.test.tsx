/**
 * useReplayPlayback.advance.test.tsx : `onAdvance`, LE SIGNAL « LA LECTURE VIENT DE FRANCHIR CES IMAGES ».
 *
 * Un pas de la boucle d'animation annonce l'intervalle qu'il vient de parcourir ; aucun déplacement
 * direct du curseur (glissé, saut, pas d'image, retour au début) ne l'émet. C'est ce qui distingue,
 * pour une pulsation d'écran, le temps qui passe d'un curseur qu'on déplace.
 */
import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createRef, type ChangeEvent, type RefObject } from 'react'

import { testReplayDoc } from '../test/testDoc'
import { AUTOPLAY_KEY } from '../settings/useReplaySettings'
import { useReplayPlayback } from './useReplayPlayback'

const DOC = testReplayDoc({ frameCount: 1001 })

let file: Array<(ts: number) => void> = []
/** Joue l'image d'animation suivante à l'instant `ts` (ms). */
const etape = (ts: number) => act(() => file.shift()?.(ts))

beforeEach(() => {
  localStorage.clear()
  localStorage.setItem(AUTOPLAY_KEY, 'true')
  file = []
  vi.stubGlobal('requestAnimationFrame', (cb: (ts: number) => void) => file.push(cb))
  vi.stubGlobal('cancelAnimationFrame', () => {
    file = []
  })
})
afterEach(() => vi.unstubAllGlobals())

function monter(speed: number) {
  const frameRef = createRef<number>() as RefObject<number>
  frameRef.current = 100
  const onAdvance = vi.fn()
  const view = renderHook(() =>
    useReplayPlayback({
      doc: DOC,
      playWindow: null,
      baseFps: 10,
      speed,
      renderWidth: 480,
      frameRef,
      draw: vi.fn(),
      soundTick: vi.fn(),
      soundSeek: vi.fn(),
      onEnded: vi.fn(),
      onTransportGesture: vi.fn(),
      onAdvance,
    }),
  )
  return { ...view, frameRef, onAdvance }
}

describe('useReplayPlayback : onAdvance', () => {
  it('un pas de la boucle annonce l’intervalle franchi, à la vitesse choisie', () => {
    const { frameRef, onAdvance } = monter(8)
    etape(1000)
    etape(1100) // 100 ms d'horloge murale à ×8 : 0,8 s de film, 8 images à 10 images par seconde
    expect(onAdvance).toHaveBeenCalledTimes(2)
    expect(onAdvance).toHaveBeenLastCalledWith(100, 108)
    expect(frameRef.current).toBe(108)
  })

  it('un glissé ou un saut du curseur n’annonce rien', () => {
    const { result, onAdvance } = monter(1)
    act(() => result.current.seekBy(30))
    act(() => result.current.seekToFrame(500))
    act(() => result.current.stepFrames(1))
    act(() =>
      result.current.onScrub({ currentTarget: { value: '700' } } as unknown as ChangeEvent<HTMLInputElement>),
    )
    expect(onAdvance).not.toHaveBeenCalled()
  })

  it('après un saut en cours de lecture, le pas suivant part de l’image du saut', () => {
    const { result, onAdvance } = monter(1)
    etape(1000)
    act(() => result.current.seekToFrame(500))
    onAdvance.mockClear()
    etape(1100)
    expect(onAdvance.mock.calls[0][0]).toBe(500)
  })
})
