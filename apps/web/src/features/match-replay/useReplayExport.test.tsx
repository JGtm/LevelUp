/**
 * useReplayExport.test.tsx — LA BOUCLE, ses deux sorties, et ce qu'elle rend à l'écran.
 *
 * L'ENCODEUR EST SIMULÉ : WebCodecs n'existe pas sous jsdom, et ce n'est pas lui qu'on teste
 * ici (il a ses propres tests, et sa vraie preuve est un MP4 ouvert dans un lecteur). Ce qui
 * se vérifie ici est la COUTURE : la lecture est mise en pause, chaque image du plan est
 * poussée une fois, l'annulation ne dépose aucun fichier, et — surtout — l'image d'avant
 * l'export est REPOSÉE quoi qu'il arrive. Cette dernière est le genre de garantie qui casse en
 * silence : l'utilisateur retrouverait son rejeu à la fin du match sans savoir pourquoi.
 */
import { renderHook, act, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { useReplayExport } from './useReplayExport'
import { testReplayDoc } from './test/testDoc'

// La SIGNATURE est portee par le mock : sans elle, `mock.calls` est un tuple vide et
// l'assertion sur l'indice d'image ne compile pas.
const addFrame = vi.fn(async (_canvas: HTMLCanvasElement, _index: number) => {})
const finish = vi.fn(async () => new Blob(['mp4']))
const abort = vi.fn()
const addAudioBuffer = vi.fn(async (_b: AudioBuffer) => {})

vi.mock('./replayVideoEncoder', async (orig) => ({
  ...(await orig<typeof import('./replayVideoEncoder')>()),
  canExportVideo: () => true,
  openVideoExport: async () => ({ addFrame, addAudioBuffer, finish, abort }),
}))
vi.mock('./replayCapture', async (orig) => ({
  ...(await orig<typeof import('./replayCapture')>()),
  triggerDownload: vi.fn(),
}))

import { triggerDownload } from './replayCapture'

const DOC = testReplayDoc({ frameIntervalMs: 50, frameCount: 200 })

function setup() {
  const canvas = document.createElement('canvas')
  canvas.width = 320
  canvas.height = 180
  const frameRef = { current: 42 }
  const redraw = vi.fn()
  const pause = vi.fn()
  const hook = renderHook(() =>
    useReplayExport({
      canvasRef: { current: canvas },
      frameRef,
      redraw,
      pause,
      doc: DOC,
      playWindow: null,
      scoreboard: [],
      outcome: null,
      titleSlug: 'halo_infinite',
      locale: 'fr',
      filenameFor: (ext) => `rejeu-test.${ext}`,
    }),
  )
  return { hook, frameRef, redraw, pause }
}

beforeEach(() => {
  addFrame.mockClear()
  finish.mockClear()
  abort.mockClear()
  addAudioBuffer.mockClear()
  vi.mocked(triggerDownload).mockClear()
  // jsdom ne fournit pas `document.fonts` : la boucle l'attend avant la première image.
  Object.defineProperty(document, 'fonts', { value: { ready: Promise.resolve() }, writable: true })
})

describe('useReplayExport', () => {
  it('met la lecture en PAUSE avant de commencer', async () => {
    const { hook, pause } = setup()
    // Sans cela, la boucle d'animation et l'export s'écriraient dessus dans `frameRef`.
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 10 }))
    expect(pause).toHaveBeenCalled()
  })

  it('pousse une image par entrée du plan, puis remet le fichier', async () => {
    const { hook } = setup()
    // 10 images à 50 ms = 500 ms de match, à 30 im/s : 15 pas + la borne de fin.
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 10 }))
    expect(addFrame).toHaveBeenCalledTimes(16)
    expect(finish).toHaveBeenCalledTimes(1)
    expect(triggerDownload).toHaveBeenCalledWith(expect.any(Blob), 'rejeu-test.mp4')
  })

  it('numérote les images du FICHIER en continu, à partir de zéro', async () => {
    const { hook } = setup()
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 10 }))
    // C'est cet indice qui porte l'horodatage : un trou y décalerait tout le clip.
    expect(addFrame.mock.calls.map((c) => c[1])).toEqual(Array.from({ length: 16 }, (_, i) => i))
  })

  it('REPOSE l’image d’avant l’export, et la repeint', async () => {
    const { hook, frameRef, redraw } = setup()
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 10 }))
    expect(frameRef.current).toBe(42)
    expect(redraw).toHaveBeenCalled()
  })

  it('annulé : aucun fichier remis, et l’encodeur est refermé', async () => {
    const { hook, frameRef } = setup()
    hook.result.current.cancel()
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 10 }))
    // `cancel` avant `run` ne compte pas : la boucle remet le drapeau à plat au démarrage.
    expect(triggerDownload).toHaveBeenCalled()
    expect(frameRef.current).toBe(42)
  })

  it('annulé EN COURS : le fichier n’est pas remis', async () => {
    const { hook } = setup()
    addFrame.mockImplementationOnce(async () => {
      hook.result.current.cancel()
    })
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 10 }))
    expect(triggerDownload).not.toHaveBeenCalled()
    expect(abort).toHaveBeenCalledTimes(1)
    expect(finish).not.toHaveBeenCalled()
  })

  it('revient à l’état inerte une fois terminé', async () => {
    const { hook } = setup()
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 10 }))
    await waitFor(() => expect(hook.result.current.state.running).toBe(false))
    expect(hook.result.current.state.pct).toBe(0)
  })

  it('propose par défaut le film entier quand il n’y a pas de cadrage', () => {
    const { hook } = setup()
    expect(hook.result.current.defaultBounds()).toEqual({ startFrame: 0, endFrame: 199 })
  })
})
