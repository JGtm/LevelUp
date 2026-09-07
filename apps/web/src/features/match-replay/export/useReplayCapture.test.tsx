/**
 * Tests — useReplayCapture (la couture React de la capture : image et vidéo).
 *
 * CE QU'ILS PROTÈGENT. La commande lit la toile et l'horloge AU MOMENT DU CLIC, pas au rendu :
 * le fichier doit porter l'instant qu'il montre. Quand la toile ne rend pas d'image — toile
 * absente, encodage refusé — il ne se télécharge RIEN : un fichier vide ferait croire à une
 * capture réussie. Et l'enregistrement n'a qu'UN chemin de sortie : second clic, pause, ou fin
 * du film mènent au même arrêt, donc à un seul clip remis une seule fois.
 *
 * TOUT EST DOUBLÉ, ET IL N'Y A RIEN À DOUBLER DE MOINS. jsdom n'a NI `MediaRecorder`, NI
 * `canvas.captureStream`, NI `toBlob` : il n'existe aucun enregistrement RÉEL à mesurer ici.
 * La question posée est donc « qui est appelé, quand, avec quoi » — la même que celle de
 * `useReplayPlayback.test.tsx`, qui remplace la boucle d'animation par une file qu'on vide à la
 * main. Ce que le module de sortie FAIT est vérifié chez lui (`replayCapture.test.ts`).
 */
import { act, renderHook } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'
import { createRef, type ReactNode, type RefObject } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { createTestQueryClient } from '@/test/render-utils'
import { fieldMappingsQueryKey } from '@/lib/i18n/fieldMappings'
import type { ReplayExportOptions } from './useReplayExport'
import { captureCanvasImage, triggerDownload } from './replayCapture'
import { pickVideoMimeType } from './replayRecording'
import { testReplayDoc } from '../test/testDoc'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { useReplayCapture } from './useReplayCapture'

/**
 * L'ESPION SUR `useReplayExport` (2026-09-07, revue ronde 2) — il DÉLÈGUE au vrai hook.
 *
 * POURQUOI IL FAUT EN PASSER PAR LÀ. `viewedLabel` — le mot que le panneau de l'export peindra —
 * est calculé DANS cette couture (`useViewedOutcome`) et ne ressort pas de `useReplayCapture` :
 * il descend directement dans les options de l'export. Le seul calcul de production n'avait donc
 * aucun test, et le remplacer par `label` laissait toute la suite verte. On intercepte les
 * options au passage, sans rien changer d'autre : le vrai hook est appelé, avec le même objet,
 * au même endroit de l'ordre des hooks.
 *
 * `vi.hoisted` parce que la fabrique de `vi.mock` est remontée en tête de module : une `const`
 * ordinaire y serait dans sa zone morte.
 */
const espionExport = vi.hoisted(() => ({ options: [] as ReplayExportOptions[] }))

vi.mock('./useReplayExport', async (importOriginal) => {
  const vrai = await importOriginal<typeof import('./useReplayExport')>()
  return {
    ...vrai,
    useReplayExport: (o: ReplayExportOptions) => {
      espionExport.options.push(o)
      return vrai.useReplayExport(o)
    },
  }
})

vi.mock('./replayCapture', async (importOriginal) => {
  // Le NOMMAGE reste le vrai : c'est lui qui prouve que l'instant lu est le bon.
  const vrai = await importOriginal<typeof import('./replayCapture')>()
  return {
    ...vrai,
    captureCanvasImage: vi.fn(),
    triggerDownload: vi.fn(),
  }
})

/** 20 images par seconde : l'image 40 tombe donc à la 2e seconde de match. */
const DOC = testReplayDoc({ matchId: 'match-témoin', frameIntervalMs: 50 })

function mount(
  frame: number,
  canvas: HTMLCanvasElement | null = document.createElement('canvas'),
  audioTrack?: () => MediaStreamTrack | null,
) {
  const canvasRef = createRef<HTMLCanvasElement>() as RefObject<HTMLCanvasElement | null>
  canvasRef.current = canvas
  const frameRef = createRef<number>() as RefObject<number>
  frameRef.current = frame
  const play = vi.fn()
  const view = renderHook(
    ({ playing }: { playing: boolean }) =>
      useReplayCapture({ canvasRef, doc: DOC, frameRef, playing, play, audioTrack, viewpoint: null }),
    { initialProps: { playing: true }, wrapper: Provider },
  )
  return { ...view, play }
}

/**
 * LE PROVIDER DE REQUÊTES EST DEVENU NÉCESSAIRE LE 2026-09-07 (revue F2) : la couture résout
 * désormais le MOT du verdict par les mappings du titre (`useOutcomeMapping`, TanStack Query)
 * quand le point de vue permute la lecture. La plupart des cas d'ici n'exercent pas ce chemin —
 * ils montent la capture sans verdict — mais le hook s'appelle sans condition, comme tout hook.
 *
 * `outcomes` À `true` SÈME LES MAPPINGS DU TITRE dans le cache, sous la clé que
 * `useFieldMappings` construit à partir des défauts du store (`halo_infinite`, `fr`). La requête
 * reste désactivée en test (`isBootstrapped` est faux) et lit quand même la donnée. À `false`,
 * c'est le cas dégradé : aucun libellé canonique.
 */
function providerWith(outcomes: boolean) {
  const qc = createTestQueryClient()
  if (outcomes) {
    qc.setQueryData(fieldMappingsQueryKey('halo_infinite', 'fr'), {
      title_slug: 'halo_infinite',
      schema_version: 1,
      locale: 'fr',
      fields: {},
      outcomes: {
        win: { label: 'Victoire', color_token: 'outcome.positive' },
        loss: { label: 'Défaite', color_token: 'outcome.negative' },
      },
    })
  }
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  }
}

/** Le provider des cas qui n'ont pas de verdict à peindre : sans mappings, ils n'en lisent pas. */
const Provider = providerWith(false)

beforeEach(() => {
  vi.mocked(captureCanvasImage).mockReset()
  vi.mocked(triggerDownload).mockReset()
})

describe('useReplayCapture — capturer l’image', () => {
  it('télécharge un PNG nommé sur l’instant de match affiché', async () => {
    const blob = new Blob(['png'])
    vi.mocked(captureCanvasImage).mockResolvedValue(blob)
    const { result } = mount(40)
    await act(async () => {
      result.current.captureImage()
    })
    expect(captureCanvasImage).toHaveBeenCalledTimes(1)
    // L'identifiant du match est assaini au passage (l'accent ne survit pas au nom de fichier).
    expect(triggerDownload).toHaveBeenCalledWith(blob, 'rejeu-match-t-moin-0m02s.png')
  })

  it('une toile qui ne rend pas d’image ne télécharge RIEN', async () => {
    vi.mocked(captureCanvasImage).mockResolvedValue(null)
    const { result } = mount(40)
    await act(async () => {
      result.current.captureImage()
    })
    expect(triggerDownload).not.toHaveBeenCalled()
  })

  it('sans toile montée, la commande ne fait rien et ne lève pas', async () => {
    const { result } = mount(40, null)
    await act(async () => {
      result.current.captureImage()
    })
    expect(captureCanvasImage).not.toHaveBeenCalled()
    expect(triggerDownload).not.toHaveBeenCalled()
  })
})

describe('pickVideoMimeType — le conteneur le plus portable d’abord', () => {
  it('MP4/H.264 l’emporte dès qu’il est accepté', () => {
    expect(pickVideoMimeType(() => true)).toEqual({ mime: 'video/mp4;codecs=avc1', ext: 'mp4' })
  })

  it('sans MP4, on descend sur WebM/VP9 — et l’extension SUIT le type retenu', () => {
    // Un `.mp4` qui contiendrait du WebM ne s'ouvrirait nulle part : c'est le couple
    // (type, extension) qui est la décision, jamais le type seul.
    const choix = pickVideoMimeType((t) => t.startsWith('video/webm'))
    expect(choix).toEqual({ mime: 'video/webm;codecs=vp9', ext: 'webm' })
  })

  it('aucun conteneur accepté : `null`, et surtout pas un repli inventé', () => {
    expect(pickVideoMimeType(() => false)).toBeNull()
  })
})

/**
 * L'ENREGISTREUR DOUBLÉ : il note ce qu'on lui demande, et rend une tranche unique à l'arrêt.
 * `stop()` émet la tranche PUIS `onstop`, dans cet ordre — c'est celui du navigateur, et c'est
 * ce qui fait que le clip contient bien sa dernière image.
 */
class FakeMediaRecorder {
  static supported: string[] = ['video/mp4;codecs=avc1']
  static last: FakeMediaRecorder | null = null
  static isTypeSupported(type: string): boolean {
    return FakeMediaRecorder.supported.includes(type)
  }
  state: 'inactive' | 'recording' = 'inactive'
  ondataavailable: ((e: { data: Blob }) => void) | null = null
  onstop: (() => void) | null = null
  stream: { getTracks: () => unknown[] }
  options: { mimeType: string }
  constructor(
    stream: { getTracks: () => unknown[] },
    options: { mimeType: string },
  ) {
    this.stream = stream
    this.options = options
    FakeMediaRecorder.last = this
  }
  start() {
    this.state = 'recording'
  }
  stop() {
    this.state = 'inactive'
    this.ondataavailable?.({ data: new Blob(['clip']) })
    this.onstop?.()
  }
}

/** Les pistes du flux : on vérifie qu'elles sont bien coupées à la fin, pas avant. */
let tracksStopped = 0

/** Le flux à plusieurs pistes que l'assemblage image + son fabrique. */
class FakeMediaStream {
  tracks: unknown[]
  constructor(tracks: unknown[]) {
    this.tracks = tracks
  }
  getTracks() {
    return this.tracks
  }
}

/** La piste vidéo de la toile — celle qui doit être coupée à l'arrêt, et elle seule. */
const PISTE_VIDEO = { kind: 'video', stop: () => (tracksStopped += 1) }

function armerLEnregistrement(supported = ['video/mp4;codecs=avc1']) {
  FakeMediaRecorder.supported = supported
  FakeMediaRecorder.last = null
  tracksStopped = 0
  vi.stubGlobal('MediaRecorder', FakeMediaRecorder)
  vi.stubGlobal('MediaStream', FakeMediaStream)
  HTMLCanvasElement.prototype.captureStream = () =>
    ({ getTracks: () => [PISTE_VIDEO] }) as unknown as MediaStream
}

afterEach(() => {
  vi.unstubAllGlobals()
  delete (HTMLCanvasElement.prototype as { captureStream?: unknown }).captureStream
})

describe('useReplayCapture — enregistrer la vidéo', () => {
  beforeEach(() => armerLEnregistrement())

  it('un clic ouvre l’enregistrement dans le conteneur retenu', () => {
    const { result } = mount(40)
    expect(result.current.recordingSupported).toBe(true)
    act(() => {
      result.current.toggleRecording()
    })
    expect(result.current.recording).toBe(true)
    expect(FakeMediaRecorder.last?.options.mimeType).toBe('video/mp4;codecs=avc1')
    expect(FakeMediaRecorder.last?.state).toBe('recording')
  })

  it('le second clic arrête, assemble, et télécharge UNE fois — avec la bonne extension', () => {
    const { result } = mount(40)
    act(() => {
      result.current.toggleRecording()
    })
    act(() => {
      result.current.toggleRecording()
    })
    expect(result.current.recording).toBe(false)
    expect(triggerDownload).toHaveBeenCalledTimes(1)
    const [blob, nom] = vi.mocked(triggerDownload).mock.calls[0]
    expect(nom).toBe('rejeu-match-t-moin-0m02s.mp4')
    expect(blob.type).toBe('video/mp4;codecs=avc1')
    // Les pistes de la toile ne se coupent qu'APRÈS la dernière tranche.
    expect(tracksStopped).toBe(1)
  })

  it('sans MP4 accepté, le clip sort en WebM — nom et conteneur d’accord', () => {
    armerLEnregistrement(['video/webm'])
    const { result } = mount(40)
    act(() => {
      result.current.toggleRecording()
    })
    act(() => {
      result.current.toggleRecording()
    })
    expect(vi.mocked(triggerDownload).mock.calls[0][1]).toBe('rejeu-match-t-moin-0m02s.webm')
  })

  // DÉCISION 3 — un seul chemin de sortie. La fin du film met la lecture en pause
  // (useReplayPlayback) : c'est cette RETOMBÉE qui clôt le clip, sans second clic.
  it('la lecture qui s’arrête clôt l’enregistrement et télécharge', () => {
    const { result, rerender } = mount(40)
    act(() => {
      result.current.toggleRecording()
    })
    expect(triggerDownload).not.toHaveBeenCalled()
    act(() => {
      rerender({ playing: false })
    })
    expect(result.current.recording).toBe(false)
    expect(triggerDownload).toHaveBeenCalledTimes(1)
  })

  // Le piège que la garde de TRANSITION évite : au démarrage depuis une pause, `playing` vaut
  // encore `false` le temps d'un rendu. Lire l'ÉTAT plutôt que sa retombée refermerait le clip
  // aussitôt ouvert — et rendrait le bouton inutilisable sur un rejeu en pause.
  it('démarrer sur une PAUSE relance la lecture, et ne se referme pas aussitôt', () => {
    const { result, rerender, play } = mount(40)
    act(() => {
      rerender({ playing: false })
    })
    act(() => {
      result.current.toggleRecording()
    })
    expect(play).toHaveBeenCalledTimes(1)
    expect(result.current.recording).toBe(true)
    expect(triggerDownload).not.toHaveBeenCalled()
    act(() => {
      rerender({ playing: true })
    })
    expect(result.current.recording).toBe(true)
  })

  // DÉCISION 6 — le son rejoint la vidéo AU DÉMARRAGE, et seulement s'il est déjà actif.
  it('avec le son actif : DEUX pistes, l’image et le son', () => {
    const piste = { kind: 'audio', stop: vi.fn() } as unknown as MediaStreamTrack
    const { result } = mount(40, document.createElement('canvas'), () => piste)
    act(() => {
      result.current.toggleRecording()
    })
    expect(FakeMediaRecorder.last?.stream.getTracks()).toEqual([PISTE_VIDEO, piste])
  })

  it('sans son : UNE piste, et le flux de la toile passe tel quel', () => {
    const { result } = mount(40, document.createElement('canvas'), () => null)
    act(() => {
      result.current.toggleRecording()
    })
    expect(FakeMediaRecorder.last?.stream.getTracks()).toEqual([PISTE_VIDEO])
  })

  it('à l’arrêt, la piste AUDIO survit — elle appartient au lecteur de son', () => {
    // La couper ici rendrait muet tout enregistrement suivant : le lecteur vit plus
    // longtemps que le clip, et c'est lui qui possède cette piste.
    const stop = vi.fn()
    const piste = { kind: 'audio', stop } as unknown as MediaStreamTrack
    const { result } = mount(40, document.createElement('canvas'), () => piste)
    act(() => {
      result.current.toggleRecording()
    })
    act(() => {
      result.current.toggleRecording()
    })
    expect(tracksStopped).toBe(1) // la piste de la toile, elle, est bien coupée
    expect(stop).not.toHaveBeenCalled()
  })

  it('quitter la page pendant un enregistrement referme SANS déposer de fichier', () => {
    const { result, unmount } = mount(40)
    act(() => {
      result.current.toggleRecording()
    })
    act(() => {
      unmount()
    })
    expect(FakeMediaRecorder.last?.state).toBe('inactive')
    expect(triggerDownload).not.toHaveBeenCalled()
  })
})

describe('useReplayCapture — navigateur sans enregistrement', () => {
  it('`recordingSupported` est faux quand l’API manque, et le bouton image reste utile', async () => {
    // Ni `MediaRecorder`, ni `captureStream` : l'état par défaut de jsdom, et celui d'un vieux
    // navigateur. La capture d'image, elle, ne dépend d'aucun des deux.
    const blob = new Blob(['png'])
    vi.mocked(captureCanvasImage).mockResolvedValue(blob)
    const { result } = mount(40)
    expect(result.current.recordingSupported).toBe(false)
    await act(async () => {
      result.current.captureImage()
    })
    expect(triggerDownload).toHaveBeenCalledTimes(1)
  })
})

/**
 * AJOUT DU 2026-09-07 (revue ronde 2) — LE MOT QUE LE CLIP PEINDRA.
 *
 * `useViewedOutcome` est le SEUL calcul de production de `viewedLabel`, et il n'était couvert
 * par rien : le remplacer par `label` laissait toute la suite verte, et le clip exporté depuis
 * le point de vue d'un adversaire aurait annoncé « Victoire » sur l'équipe qui a perdu — le
 * défaut F2, revenu par la porte de l'export. Ces cas fixent les deux régimes, et le dégradé.
 *
 * ILS N'OBSERVENT PAS LE PANNEAU mais l'ENTRÉE de l'export : ce que `exportOverlayPanels` en
 * fait est éprouvé chez lui (`exportOverlayPanels.test.ts`, quatre cas sur `viewedLabel`). Ici,
 * la seule question est « la couture a-t-elle résolu le bon mot ? ».
 */
describe('useReplayCapture — le mot du verdict transmis à l’export', () => {
  const SB = [
    { xuid: 'moi', team_side: 't0', is_me: true },
    { xuid: 'eux', team_side: 't1', is_me: false },
  ] as MatchScoreboardRow[]

  /** Le verdict servi par l'en-tête : le joueur de la page a GAGNÉ (code 2). */
  const OUTCOME = { code: 2, label: 'Victoire' }

  /** Le même lobby, plus un COÉQUIPIER du joueur de la page. */
  const SB_TRIO = [...SB, { xuid: 'pote', team_side: 't0', is_me: false } as MatchScoreboardRow]

  /** Monte la capture avec un verdict et un point de vue, et rend les options vues par l'export. */
  function optionsPour(
    viewpoint: string | null,
    { mappings = true, board = SB }: { mappings?: boolean; board?: MatchScoreboardRow[] } = {},
  ) {
    espionExport.options.length = 0
    const canvasRef = createRef<HTMLCanvasElement>() as RefObject<HTMLCanvasElement | null>
    canvasRef.current = document.createElement('canvas')
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 0
    renderHook(
      () =>
        useReplayCapture({
          canvasRef,
          doc: DOC,
          frameRef,
          playing: false,
          play: vi.fn(),
          // `redraw` est ce qui rend l'export DISPONIBLE (cf. `useExportSeam`) : sans lui le
          // hook tourne quand même, mais son résultat est jeté — et les options nous échappent.
          redraw: vi.fn(),
          scoreboard: board,
          outcome: OUTCOME,
          viewpoint,
        }),
      { wrapper: providerWith(mappings) },
    )
    return espionExport.options[espionExport.options.length - 1]
  }

  it('point de vue ADVERSE : le libellé CANONIQUE de l’issue permutée', () => {
    // Le joueur de la page a gagné (code 2) ; vu de `eux`, la lecture est une défaite, et c'est
    // ce mot-là — celui d'`outcomes.toml` — que le clip doit peindre, pas « Victoire ».
    expect(optionsPour('eux').outcome?.viewedLabel).toBe('Défaite')
  })

  it('point de vue = LE JOUEUR DE LA PAGE : le libellé du backend, tel quel', () => {
    expect(optionsPour('moi').outcome?.viewedLabel).toBe('Victoire')
  })

  it('point de vue ABSENT (`null`) : le libellé du backend, comme avant le chantier', () => {
    expect(optionsPour(null).outcome?.viewedLabel).toBe('Victoire')
  })

  it('COÉQUIPIER du joueur de la page : même camp, donc rien de permuté', () => {
    expect(optionsPour('pote', { board: SB_TRIO }).outcome?.viewedLabel).toBe('Victoire')
  })

  it('issue permutée SANS mappings du titre : `null` — jamais une clé brute dans le clip', () => {
    // `useOutcomeMapping` rend `undefined` : le panneau se taira, comme sans `outcome_label`.
    // C'est exactement ce que `useOutcomeLabel` ne saurait pas dire — il rendrait « loss ».
    expect(optionsPour('eux', { mappings: false }).outcome?.viewedLabel).toBeNull()
  })

  it('le reste du verdict traverse INCHANGÉ : seul le mot vu s’ajoute', () => {
    const outcome = optionsPour('eux').outcome
    expect(outcome).toMatchObject({ code: 2, label: 'Victoire', viewedLabel: 'Défaite' })
  })
})
