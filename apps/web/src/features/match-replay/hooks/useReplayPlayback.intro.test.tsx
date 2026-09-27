/**
 * Tests — useReplayPlayback, LE DÉPART DEPUIS LE PRÉAMBULE (item 7 du backlog, 2026-09-27).
 *
 * CE QU'ILS PROTÈGENT, et c'est la décision D-8 : la musique d'intro part quand l'utilisateur
 * LANCE le rejeu depuis sa seconde de préambule — « Lecture » à l'ouverture, « Recommencer »,
 * ou « Lecture » sur un rejeu terminé (qui rembobine au préambule). Elle ne part JAMAIS sur une
 * reprise en cours de match, un glissé, un saut, un lien tactique ou une lecture automatique :
 * une intro au milieu d'un échange de tirs serait un contresens, et une lecture automatique n'a
 * pas de geste (donc pas de son, cf. `useAudioUnlock`).
 *
 * `onStarted` est le signal ; ce que le son en fait (préférence, vitesse) se teste côté son
 * (`sound/introSound.test.tsx`). Même boucle pilotée à la main que `useReplayPlayback.test.tsx`.
 */
import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createRef, type ChangeEvent, type RefObject } from 'react'

import type { ReplayWindowBounds } from '../model/replayWindow'
import { testReplayDoc } from '../test/testDoc'
import { AUTOPLAY_KEY } from '../settings/useReplaySettings'
import { useReplayPlayback } from './useReplayPlayback'

/** 51 images ; fenêtre de gameplay 10 → 40, préambule sur la 9 (même fixture que le voisin). */
const DOC = testReplayDoc({ frameCount: 51 })
const FENETRE: ReplayWindowBounds = { startFrame: 10, leadInFrame: 9, endFrame: 40, startMs: 10_000, endMs: 40_000 }

let pending: FrameRequestCallback[] = []

beforeEach(() => {
  localStorage.clear() // lecture automatique ÉTEINTE : le défaut du produit
  pending = []
  vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => {
    pending.push(cb)
    return pending.length
  })
  vi.stubGlobal('cancelAnimationFrame', () => {})
})

afterEach(() => {
  vi.unstubAllGlobals()
})

/** Un pas de boucle (le premier n'avance jamais : il amorce l'horloge). */
function tick(ts: number) {
  const next = pending.shift()
  if (!next) throw new Error('aucune image demandée — la boucle ne tourne pas')
  act(() => {
    next(ts)
  })
}

function mount(opts: { openAtFrame?: number | null } = {}) {
  const frameRef = createRef<number>() as RefObject<number>
  frameRef.current = 0
  const calls: string[] = []
  const onStarted = vi.fn(() => calls.push('started'))
  const onTransportGesture = vi.fn(() => calls.push('gesture'))
  const view = renderHook(() =>
    useReplayPlayback({
      doc: DOC,
      playWindow: FENETRE,
      baseFps: 10,
      speed: 1,
      renderWidth: 480,
      frameRef,
      draw: vi.fn(),
      soundTick: vi.fn(),
      soundSeek: vi.fn(),
      onEnded: vi.fn(),
      onTransportGesture,
      onStarted,
      openAtFrame: opts.openAtFrame,
    }),
  )
  return { ...view, frameRef, onStarted, calls }
}

/** Lecture puis quelques pas : la lecture a quitté le préambule. */
function playAway(r: ReturnType<typeof mount>) {
  act(() => r.result.current.togglePlay())
  tick(1)
  tick(1001) // +1 s à 10 i/s = +10 images
}

describe('useReplayPlayback — l’intro part au départ depuis le préambule', () => {
  it('« Lecture » à l’ouverture : un seul signal, APRÈS le geste de transport', () => {
    const r = mount()
    expect(r.frameRef.current).toBe(FENETRE.leadInFrame)
    act(() => r.result.current.togglePlay())
    expect(r.onStarted).toHaveBeenCalledTimes(1)
    // Le geste ouvre le lecteur audio ; l'intro ne peut partir qu'ensuite.
    expect(r.calls).toEqual(['gesture', 'started'])
  })

  it('« Pause » n’est pas un départ : aucun signal', () => {
    const r = mount()
    playAway(r)
    act(() => r.result.current.togglePlay()) // pause
    expect(r.onStarted).toHaveBeenCalledTimes(1)
  })

  it('pause puis reprise en cours de match : aucun nouveau signal', () => {
    const r = mount()
    playAway(r)
    act(() => r.result.current.togglePlay()) // pause
    act(() => r.result.current.togglePlay()) // reprise
    expect(r.onStarted).toHaveBeenCalledTimes(1)
  })

  it('glissé de frise puis « Lecture » : aucun signal', () => {
    const r = mount()
    act(() => r.result.current.onScrub({ currentTarget: { value: '20' } } as unknown as ChangeEvent<HTMLInputElement>))
    act(() => r.result.current.togglePlay())
    expect(r.onStarted).not.toHaveBeenCalled()
  })

  it('saut de secondes ou pas d’image puis « Lecture » : aucun signal', () => {
    const r = mount()
    act(() => r.result.current.seekBy(1)) // préambule 9 → 19, en deçà de la fin
    act(() => r.result.current.togglePlay())
    act(() => r.result.current.togglePlay()) // pause
    act(() => r.result.current.stepFrames(1))
    act(() => r.result.current.togglePlay())
    expect(r.onStarted).not.toHaveBeenCalled()
  })

  it('ouvert par un lien tactique : ni à l’ouverture, ni à la « Lecture » qui suit', () => {
    const r = mount({ openAtFrame: 25 })
    expect(r.frameRef.current).toBe(25)
    expect(r.onStarted).not.toHaveBeenCalled()
    act(() => r.result.current.togglePlay())
    expect(r.onStarted).not.toHaveBeenCalled()
  })

  it('lecture automatique : la lecture tourne sans aucun signal', () => {
    localStorage.setItem(AUTOPLAY_KEY, 'true')
    const r = mount()
    expect(r.result.current.playing).toBe(true)
    tick(1)
    tick(1001)
    expect(r.onStarted).not.toHaveBeenCalled()
  })

  it('« Recommencer » en plein match : le signal repart', () => {
    const r = mount()
    playAway(r)
    act(() => r.result.current.restart())
    expect(r.frameRef.current).toBe(FENETRE.leadInFrame)
    expect(r.onStarted).toHaveBeenCalledTimes(2)
  })

  it('« Lecture » sur un rejeu terminé : le rembobinage ramène au préambule, le signal repart', () => {
    const r = mount()
    act(() => r.result.current.togglePlay())
    tick(1)
    tick(10_000) // bien au-delà de la fin : la boucle s'arrête sur la borne
    expect(r.result.current.playing).toBe(false)
    expect(r.frameRef.current).toBe(FENETRE.endFrame)
    act(() => r.result.current.togglePlay())
    expect(r.frameRef.current).toBe(FENETRE.leadInFrame)
    expect(r.onStarted).toHaveBeenCalledTimes(2)
  })
})
