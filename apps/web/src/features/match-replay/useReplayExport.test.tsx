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
import type { ReplayWindowBounds } from './replayWindow'
import { testReplayDoc } from './test/testDoc'

// La SIGNATURE est portee par le TYPE du mock, pas par des parametres nommes : sans elle,
// `mock.calls` est un tuple vide et l'assertion sur l'indice d'image ne compile pas ; avec des
// parametres nommes mais inutilises, c'est le lint qui proteste.
const addFrame = vi.fn<(canvas: HTMLCanvasElement, index: number) => Promise<void>>(async () => {})
const finish = vi.fn(async () => new Blob(['mp4']))
const abort = vi.fn()
const addAudioBuffer = vi.fn<(buffer: AudioBuffer) => Promise<void>>(async () => {})

vi.mock('./replayVideoEncoder', async (orig) => ({
  ...(await orig<typeof import('./replayVideoEncoder')>()),
  canExportVideo: () => true,
  openVideoExport: async () => ({ addFrame, addAudioBuffer, finish, abort }),
}))
vi.mock('./replayAudioMix', async (orig) => {
  const vrai = await orig<typeof import('./replayAudioMix')>()
  return { ...vrai, mixReplayAudio: vi.fn(async () => null) }
})
vi.mock('./replayCapture', async (orig) => ({
  ...(await orig<typeof import('./replayCapture')>()),
  triggerDownload: vi.fn(),
}))

import { triggerDownload } from './replayCapture'
import { mixReplayAudio } from './replayAudioMix'

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
  // Le mock du mixage est PARTAGE entre les tests : sans ce nettoyage, `mock.calls[0]`
  // rendrait l'appel d'un test precedent.
  vi.mocked(mixReplayAudio).mockClear()
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
    // LE NOM PORTE LES DEUX BORNES (décision D9) : sans la seconde, deux exports de plages
    // différentes qui partagent leur fin s'écraseraient dans le dossier de téléchargements.
    expect(triggerDownload).toHaveBeenCalledWith(expect.any(Blob), 'rejeu-m-0m00s-0m00s.mp4')
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
    await waitFor(() => expect(hook.result.current.state.phase).toBe('done'))
    // La FIN se dit, avec le nom du fichier : le clip part dans les telechargements, ou
    // rien ne le rattache au geste qui vient d'etre fait.
    expect(hook.result.current.state.filename).toBe('rejeu-m-0m00s-0m00s.mp4')
  })

  it('propose par défaut le film entier quand il n’y a pas de cadrage', () => {
    const { hook } = setup()
    expect(hook.result.current.defaultBounds()).toEqual({ startFrame: 0, endFrame: 199 })
  })
})

/**
 * LES TROIS DÉFAUTS TROUVÉS PAR LA REVUE ADVERSARIALE DU 2026-08-28, verrouillés ici.
 *
 * Aucun n'était couvert : le premier faisait mentir le son d'un extrait, le deuxième laissait
 * l'utilisateur devant une barre disparue sans un mot, le troisième fuyait un encodeur.
 */
describe('useReplayExport — non-régressions de la revue adversariale', () => {
  const PISTE = () => ({
    timeline: [{ ms: 100, stem: 'tir' }],
    endMatchStems: ['fanfare'],
    variationPercent: 0,
    distancePercent: 0,
  })
  const FENETRE = { startFrame: 0, endFrame: 100, startMs: 0, endMs: 5000 }

  function setupAvecSon(playWindow: ReplayWindowBounds | null = FENETRE) {
    const canvas = document.createElement('canvas')
    canvas.width = 320
    canvas.height = 180
    const frameRef = { current: 0 }
    return renderHook(() =>
      useReplayExport({
        canvasRef: { current: canvas },
        frameRef,
        redraw: vi.fn(),
        pause: vi.fn(),
        doc: DOC,
        playWindow,
        scoreboard: [],
        outcome: null,
        titleSlug: 'halo_infinite',
        locale: 'fr',
        soundTrack: PISTE,
        soundVolume: 1,
      }),
    )
  }

  it('n’attache PAS la fanfare de fin à un extrait de milieu de match', async () => {
    const hook = setupAvecSon()
    // La plage s'arrête bien avant la fin : un extrait ne se termine pas sur la voix de
    // l'annonceur et la fanfare de victoire — le son affirmerait un fait faux.
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 40 }))
    expect(vi.mocked(mixReplayAudio).mock.calls[0]?.[2].endMatchStems).toEqual([])
  })

  it('attache la fanfare quand la plage VA jusqu’au bout du match', async () => {
    const hook = setupAvecSon()
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 100 }))
    expect(vi.mocked(mixReplayAudio).mock.calls[0]?.[2].endMatchStems).toEqual(['fanfare'])
  })

  it('sans fenêtre de gameplay, on ne suppose PAS que la plage est la fin', async () => {
    const hook = setupAvecSon(null)
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 199 }))
    expect(vi.mocked(mixReplayAudio).mock.calls[0]?.[2].endMatchStems).toEqual([])
  })

  it('un échec est DIT, tracé, et ne dépose aucun fichier', async () => {
    const hook = setupAvecSon()
    const erreur = new Error('encodeur hors service')
    addFrame.mockRejectedValueOnce(erreur)
    const trace = vi.spyOn(console, 'error').mockImplementation(() => {})
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 40 }))
    expect(hook.result.current.state.phase).toBe('failed')
    expect(hook.result.current.state.message).toBe('encodeur hors service')
    expect(trace).toHaveBeenCalled()
    expect(triggerDownload).not.toHaveBeenCalled()
    trace.mockRestore()
  })

  it('un échec REFERME l’encodeur au lieu de le laisser ouvert', async () => {
    const hook = setupAvecSon()
    addFrame.mockRejectedValueOnce(new Error('boum'))
    vi.spyOn(console, 'error').mockImplementation(() => {})
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 40 }))
    // Sans ce `abort`, le VideoEncoder et son muxeur restaient en mémoire jusqu'à la fermeture
    // de l'onglet.
    expect(abort).toHaveBeenCalledTimes(1)
  })
})
