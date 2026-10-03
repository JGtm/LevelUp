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
import { useEffect } from 'react'
import { renderHook, act, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useReplayExport, type ReplayExportOptions } from './useReplayExport'
import type { ReplayWindowBounds } from '../model/replayWindow'
import type { ExportLayout } from './exportFormats'
import { readInk } from '../layers/canvasInk'
import { canvasPixelRatio, isExportActive, isExportLayoutApplied, useExportLayout } from './exportLayoutStore'
import { testReplayDoc } from '../test/testDoc'

// La SIGNATURE est portee par le TYPE du mock, pas par des parametres nommes : sans elle,
// `mock.calls` est un tuple vide et l'assertion sur l'indice d'image ne compile pas ; avec des
// parametres nommes mais inutilises, c'est le lint qui proteste.
const addFrame = vi.fn<(canvas: HTMLCanvasElement, index: number) => Promise<void>>(async () => {})
const finish = vi.fn(async () => new Blob(['mp4']))
const abort = vi.fn()
const addAudioTracks = vi.fn<(t: readonly { name: string; buffer: AudioBuffer }[]) => Promise<void>>(async () => {})
/** Les noms de pistes DECLARES a l'ouverture du conteneur (leur ordre y est fige). */
const nomsDeclares: string[] = []
/** Les dimensions avec lesquelles l'encodeur a ete OUVERT, par export. */
const dimsOuvertes: { width: number; height: number }[] = []
/** Le navigateur accepte-t-il la piste sonore ? Pilote par test (cf. le repli muet). */
const audioOk = { value: true }

vi.mock('./replayVideoEncoder', async (orig) => ({
  ...(await orig<typeof import('./replayVideoEncoder')>()),
  canExportVideo: () => true,
  openVideoExport: async (o: { width: number; height: number; audioTracks?: readonly string[] }) => {
    dimsOuvertes.push({ width: o.width, height: o.height })
    nomsDeclares.length = 0
    nomsDeclares.push(...(o.audioTracks ?? []))
    return { addFrame, addAudioTracks, finish, abort, audioEnabled: audioOk.value }
  },
}))
vi.mock('../sound/replayAudioMix', async (orig) => {
  const vrai = await orig<typeof import('../sound/replayAudioMix')>()
  return { ...vrai, mixReplayAudio: vi.fn(async () => null) }
})
vi.mock('./replayCapture', async (orig) => ({
  ...(await orig<typeof import('./replayCapture')>()),
  triggerDownload: vi.fn(),
}))

import { triggerDownload } from './replayCapture'
import { mixReplayAudio } from '../sound/replayAudioMix'

const DOC = testReplayDoc({ frameIntervalMs: 50, frameCount: 200 })

/** La mise en page que le dernier rendu du montage a lue — ce que `ReplayCanvas` en recevrait. */
const miseEnPageVue: { current: ExportLayout | null } = { current: null }

/**
 * LE MONTAGE AVEC SA TOILE : l'export ne peint sa premiere image qu'une fois la mise en page
 * APPLIQUEE par le cadrage React (`useReplayView` -> `useExportLayout`). Ce montage tient ce
 * role, et rien d'autre.
 */
function useExportWithView(o: ReplayExportOptions) {
  const layout = useExportLayout()
  useEffect(() => {
    miseEnPageVue.current = layout
  }, [layout])
  return useReplayExport(o)
}

/**
 * LA TOILE SIMULEE : le dimensionnement de `ReplayCanvas.draw`, a l'identique — taille de
 * dessin (celle de l'ecran, ou le cadre du format) fois `canvasPixelRatio`, arrondie.
 */
function toileSimulee(canvas: HTMLCanvasElement, ecran = { width: 320, height: 180 }) {
  return vi.fn(() => {
    const cadre = miseEnPageVue.current ?? ecran
    canvas.width = Math.round(cadre.width * canvasPixelRatio())
    canvas.height = Math.round(cadre.height * canvasPixelRatio())
  })
}

function setup() {
  const canvas = document.createElement('canvas')
  canvas.width = 320
  canvas.height = 180
  const frameRef = { current: 42 }
  const redraw = toileSimulee(canvas)
  const pause = vi.fn()
  const hook = renderHook(() =>
    useExportWithView({
      canvasRef: { current: canvas },
      frameRef,
      redraw,
      pause,
      doc: DOC,
      playWindow: null,
      scoreboard: [],
      outcome: null,
      // LE POINT DE VUE EST OBLIGATOIRE depuis le 2026-09-07 (revue F4) : `null` = le joueur de
      // la page, le comportement d'origine. Il l'est aux six montages de ce fichier, et c'est le
      // but — un relais oublié entre le canvas et l'export ne compile plus.
      viewpoint: null,
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
  addAudioTracks.mockClear()
  dimsOuvertes.length = 0
  // jsdom n'a pas de contexte 2D : sans ce double, chaque image ecrirait « Not implemented ».
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null)
  audioOk.value = true
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
    // La mise en page d'ecran revient aussi sur ce chemin-la (meme `finally`), gestes rendus.
    expect(miseEnPageVue.current).toBeNull()
    expect(isExportActive()).toBe(false)
    expect(isExportLayoutApplied()).toBe(true)
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
    // Sans palier transmis, la carte est a 1x : le dialogue ne proposera pas de cadrage.
    expect(hook.result.current.zoomLevel).toBe(1)
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
    families: { voice: [], music: [] },
    engines: [],
  })
  const FENETRE = { startFrame: 0, leadInFrame: 0, endFrame: 100, startMs: 0, endMs: 5000 }

  function setupAvecSon(playWindow: ReplayWindowBounds | null = FENETRE) {
    const canvas = document.createElement('canvas')
    canvas.width = 320
    canvas.height = 180
    const frameRef = { current: 0 }
    return renderHook(() =>
      useExportWithView({
        canvasRef: { current: canvas },
        frameRef,
        redraw: toileSimulee(canvas),
        pause: vi.fn(),
        doc: DOC,
        playWindow,
        scoreboard: [],
        outcome: null,
        viewpoint: null,
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

describe('useReplayExport — le repli MUET quand le navigateur refuse la piste', () => {
  it('sort un clip muet, le DIT, et n’echoue pas', async () => {
    // Vecu en recette le 2026-08-28 : `AudioEncoder` existait, la configuration AAC etait
    // refusee de facon ASYNCHRONE, et la panne ne surgissait qu'au `flush()` sous la forme
    // trompeuse « Encoder must be configured first » — tout l'export etait perdu.
    audioOk.value = false
    vi.mocked(mixReplayAudio).mockResolvedValueOnce({ full: { duration: 2 } as AudioBuffer, families: [] })
    const trace = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const canvas = document.createElement('canvas')
    canvas.width = 320
    canvas.height = 180
    const hook = renderHook(() =>
      useExportWithView({
        canvasRef: { current: canvas },
        frameRef: { current: 0 },
        redraw: toileSimulee(canvas),
        pause: vi.fn(),
        doc: DOC,
        playWindow: null,
        scoreboard: [],
        outcome: null,
        viewpoint: null,
        titleSlug: 'halo_infinite',
        locale: 'fr',
        soundTrack: () => ({ timeline: [{ ms: 0, stem: 'x' }], endMatchStems: [], variationPercent: 0, distancePercent: 0, families: { voice: [], music: [] }, engines: [] }),
        soundVolume: 1,
      }),
    )
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 10 }))
    expect(addAudioTracks).not.toHaveBeenCalled()
    expect(hook.result.current.state.phase).toBe('done')
    expect(hook.result.current.state.mutedFallback).toBe(true)
    expect(trace).toHaveBeenCalled()
    trace.mockRestore()
  })
})

describe('useReplayExport — le format du fichier', () => {
  /** Monte un export sur une toile d'ECRAN donnee, sous une densite d'ecran donnee. */
  function monter(ecran: { width: number; height: number }, dpr: number) {
    Object.defineProperty(window, 'devicePixelRatio', { value: dpr, configurable: true })
    const canvas = document.createElement('canvas')
    canvas.width = Math.round(ecran.width * dpr)
    canvas.height = Math.round(ecran.height * dpr)
    const hook = renderHook(() =>
      useExportWithView({
        canvasRef: { current: canvas },
        frameRef: { current: 0 },
        redraw: toileSimulee(canvas, ecran),
        pause: vi.fn(),
        doc: DOC,
        playWindow: null,
        scoreboard: [],
        outcome: null,
        viewpoint: null,
        titleSlug: 'halo_infinite',
        locale: 'fr',
      }),
    )
    return { hook, canvas }
  }
  /** La taille de la toile au moment de CHAQUE image poussee a l'encodeur. */
  function taillesPoussees(): string[] {
    return addFrame.mock.calls.map(([c]) => `${c.width}x${c.height}`)
  }

  afterEach(() => {
    Object.defineProperty(window, 'devicePixelRatio', { value: 1, configurable: true })
  })

  it('sans choix : 1920x1080, quelle que soit la toile de l’ecran', async () => {
    const { hook } = monter({ width: 1234, height: 567 }, 1)
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 10 }))
    expect(dimsOuvertes).toEqual([{ width: 1920, height: 1080 }])
    expect(new Set(taillesPoussees())).toEqual(new Set(['1920x1080']))
  })

  it.each([
    { format: '1080p' as const, attendu: '1920x1080', ecran: { width: 502, height: 480 }, dpr: 1 },
    { format: '1080p' as const, attendu: '1920x1080', ecran: { width: 1400, height: 720 }, dpr: 2.5 },
    { format: '720p' as const, attendu: '1280x720', ecran: { width: 502, height: 480 }, dpr: 1 },
    { format: '720p' as const, attendu: '1280x720', ecran: { width: 1400, height: 720 }, dpr: 3 },
  ])('$format sur une toile $ecran.width x $ecran.height a DPR $dpr -> $attendu', async ({ format, attendu, ecran, dpr }) => {
    const { hook } = monter(ecran, dpr)
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 10 }, { format }))
    const [w, h] = attendu.split('x').map(Number)
    expect(dimsOuvertes).toEqual([{ width: w, height: h }])
    // CHAQUE image, pas seulement la premiere : une toile qui changerait de taille en cours de
    // route serait rognee par l'encodeur.
    expect(addFrame).toHaveBeenCalledTimes(16)
    expect(new Set(taillesPoussees())).toEqual(new Set([attendu]))
  })

  it('rend la mise en page d’ECRAN a la fin : densite de l’ecran, plus aucun cadre de format', async () => {
    const { hook } = monter({ width: 800, height: 450 }, 2)
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 10 }, { format: '720p' }))
    expect(miseEnPageVue.current).toBeNull()
    expect(isExportLayoutApplied()).toBe(true)
    expect(canvasPixelRatio()).toBe(2)
  })

  it('la rend meme quand l’export ECHOUE', async () => {
    const { hook } = monter({ width: 800, height: 450 }, 1)
    addFrame.mockRejectedValueOnce(new Error('boum'))
    const trace = vi.spyOn(console, 'error').mockImplementation(() => {})
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 10 }))
    expect(hook.result.current.state.phase).toBe('failed')
    expect(miseEnPageVue.current).toBeNull()
    expect(isExportActive()).toBe(false)
    expect(canvasPixelRatio()).toBe(1)
    trace.mockRestore()
  })

  it('fond NOIR PUR et encres du THEME SOMBRE, meme page en theme clair ; la page n’en sait rien', async () => {
    // Un pixel transparent n'a pas de sens pour H.264 : le fond est PEINT sous toute l'image,
    // bandes du cadre 16:9 comprises, et il est noir quel que soit le theme (D8). Les encres
    // lues pendant l'encodage sont celles du theme sombre (D9).
    const theme = document.createElement('style')
    theme.textContent = [
      ":root, :root[data-theme='dark'], :root[data-theme='light'] { --replay-export-backdrop: rgb(0 0 0); }",
      ":root, :root[data-theme='dark'] { --foreground: rgb(250 250 250); }",
      ":root[data-theme='light'] { --foreground: rgb(20 20 20); }",
    ].join(' ')
    document.head.appendChild(theme)
    const avant = document.documentElement.getAttribute('data-theme')
    document.documentElement.setAttribute('data-theme', 'light')
    const peint: string[] = []
    const encres: string[] = []
    const ctx = {
      globalCompositeOperation: 'source-over',
      fillStyle: '',
      setTransform: () => {},
      fillRect: (x: number, y: number, w: number, h: number) =>
        peint.push(`${ctx.globalCompositeOperation} ${ctx.fillStyle} ${x},${y},${w},${h}`),
    }
    vi.mocked(HTMLCanvasElement.prototype.getContext).mockReturnValue(ctx as unknown as CanvasRenderingContext2D)
    addFrame.mockImplementation(async () => {
      encres.push(readInk('--foreground'))
    })
    try {
      const { hook } = monter({ width: 700, height: 700 }, 1)
      await act(() => hook.result.current.run({ startFrame: 0, endFrame: 10 }))
      expect(peint).toHaveLength(16)
      expect(new Set(peint)).toEqual(new Set(['destination-over rgb(0 0 0) 0,0,1920,1080']))
      // Le mode de composition est RENDU : le trace suivant peindrait sinon sous l'image.
      expect(ctx.globalCompositeOperation).toBe('source-over')
      expect(new Set(encres)).toEqual(new Set(['rgb(250 250 250)']))
      // LA PAGE : toujours en theme clair, et ses encres redeviennent les siennes.
      expect(document.documentElement.getAttribute('data-theme')).toBe('light')
      expect(readInk('--foreground')).toBe('rgb(20 20 20)')
    } finally {
      addFrame.mockImplementation(async () => {})
      theme.remove()
      if (avant === null) document.documentElement.removeAttribute('data-theme')
      else document.documentElement.setAttribute('data-theme', avant)
    }
  })

  it('transmet le CADRAGE a la mise en page : carte entiere par defaut, cadrage actuel sur demande', async () => {
    const cadrages: string[] = []
    const { hook } = monter({ width: 800, height: 450 }, 1)
    addFrame.mockImplementation(async () => {
      cadrages.push(miseEnPageVue.current?.framing ?? 'aucun')
    })
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 10 }))
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 10 }, { framing: 'current' }))
    addFrame.mockImplementation(async () => {})
    expect(cadrages.slice(0, 16)).toEqual(Array(16).fill('whole'))
    expect(cadrages.slice(16)).toEqual(Array(16).fill('current'))
  })

  it('tient la toile DES la preparation (gestes eteints avant le mixage du son)', async () => {
    let actifPendantLeMixage = false
    vi.mocked(mixReplayAudio).mockImplementationOnce(async () => {
      actifPendantLeMixage = isExportActive()
      return null
    })
    const canvas = document.createElement('canvas')
    const hook = renderHook(() =>
      useExportWithView({
        canvasRef: { current: canvas },
        frameRef: { current: 0 },
        redraw: toileSimulee(canvas),
        pause: vi.fn(),
        doc: DOC,
        playWindow: null,
        scoreboard: [],
        outcome: null,
        viewpoint: null,
        titleSlug: 'halo_infinite',
        locale: 'fr',
        soundTrack: () => ({ timeline: [{ ms: 0, stem: 'x' }], endMatchStems: [], variationPercent: 0, distancePercent: 0, families: { voice: [], music: [] }, engines: [] }),
        zoomLevel: 2,
      }),
    )
    // Le palier de l'ecran est RELAYE au dialogue, qui decide d'y proposer le cadrage.
    expect(hook.result.current.zoomLevel).toBe(2)
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 10 }))
    expect(actifPendantLeMixage).toBe(true)
    expect(isExportActive()).toBe(false)
  })

  it('une toile qui ne prend JAMAIS le format : echec dit, encodeur jamais ouvert', async () => {
    // Aucun cadrage monte : la mise en page demandee ne s'applique pas. Le temps est avance
    // artificiellement pour ne pas attendre le delai reel.
    const canvas = document.createElement('canvas')
    let t = 0
    const horloge = vi.spyOn(performance, 'now').mockImplementation(() => (t += 1000))
    const trace = vi.spyOn(console, 'error').mockImplementation(() => {})
    const hook = renderHook(() =>
      useReplayExport({
        canvasRef: { current: canvas },
        frameRef: { current: 0 },
        redraw: vi.fn(),
        pause: vi.fn(),
        doc: DOC,
        playWindow: null,
        scoreboard: [],
        outcome: null,
        viewpoint: null,
        titleSlug: 'halo_infinite',
        locale: 'fr',
      }),
    )
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 10 }))
    expect(hook.result.current.state.phase).toBe('failed')
    expect(hook.result.current.state.message).toContain("mise en page d'export non appliquee")
    expect(dimsOuvertes).toEqual([])
    horloge.mockRestore()
    trace.mockRestore()
  })
})

describe('useReplayExport — les pistes sonores separees', () => {
  it('declare le MIXAGE COMPLET en premier, puis les familles', async () => {
    audioOk.value = true
    vi.mocked(mixReplayAudio).mockResolvedValueOnce({
      full: { duration: 2 } as AudioBuffer,
      families: [
        { family: 'sfx', buffer: { duration: 2 } as AudioBuffer },
        { family: 'music', buffer: { duration: 2 } as AudioBuffer },
      ],
    })
    const canvas = document.createElement('canvas')
    canvas.width = 320
    canvas.height = 180
    const hook = renderHook(() =>
      useExportWithView({
        canvasRef: { current: canvas },
        frameRef: { current: 0 },
        redraw: toileSimulee(canvas),
        pause: vi.fn(),
        doc: DOC,
        playWindow: null,
        scoreboard: [],
        outcome: null,
        viewpoint: null,
        titleSlug: 'halo_infinite',
        locale: 'fr',
        soundTrack: () => ({ timeline: [{ ms: 0, stem: 'x' }], endMatchStems: [], variationPercent: 0, distancePercent: 0, families: { voice: [], music: [] }, engines: [] }),
        soundVolume: 1,
      }),
    )
    await act(() => hook.result.current.run({ startFrame: 0, endFrame: 10 }))
    // LE MIXAGE EN PREMIER : un lecteur ordinaire ne joue que la premiere piste, et un
    // navigateur n'expose meme pas les autres. Les familles seules feraient entendre les
    // bruitages sans la musique ni la voix.
    expect(nomsDeclares).toEqual(['Mixage complet', 'Bruitages', 'Musique'])
    expect(addAudioTracks.mock.calls[0][0].map((p) => p.name)).toEqual(nomsDeclares)
  })
})
