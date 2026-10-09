/**
 * Tests CoverFlowModal — lecture HLS : hls.js est branché pour les .m3u8, le
 * sélecteur de piste apparaît après MANIFEST_PARSED, et les mp4 restent en
 * lecture directe. hls.js est mocké (jsdom n'a ni MediaSource ni décodeur).
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { act, screen, fireEvent } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { CoverFlowModal } from './CoverFlowModal'
import type { MediaItemRow } from '@/lib/api/types'

interface MockHlsInstance {
  loadSource: ReturnType<typeof vi.fn>
  attachMedia: ReturnType<typeof vi.fn>
  destroy: ReturnType<typeof vi.fn>
  startLoad: ReturnType<typeof vi.fn>
  stopLoad: ReturnType<typeof vi.fn>
  handlers: Record<string, ((evt?: unknown, data?: unknown) => void) | undefined>
  audioTracks: { name: string; lang: string }[]
  audioTrack: number
}

const { instances } = vi.hoisted(() => ({ instances: [] as MockHlsInstance[] }))

vi.mock('hls.js', () => {
  class Hls {
    static isSupported() {
      return true
    }
    static Events = { MANIFEST_PARSED: 'manifestParsed', AUDIO_TRACKS_UPDATED: 'audioTracksUpdated', ERROR: 'error' }
    audioTracks = [
      { name: 'Game', lang: 'fra' },
      { name: 'Mic', lang: 'fra' },
    ]
    audioTrack = 0
    handlers: Record<string, ((evt?: unknown, data?: unknown) => void) | undefined> = {}
    loadSource = vi.fn()
    attachMedia = vi.fn()
    destroy = vi.fn()
    // autoStartLoad:false côté prod → le composant pilote start/stopLoad selon
    // le centrage du clip (chargement des segments réservé au clip centré).
    startLoad = vi.fn()
    stopLoad = vi.fn()
    on(evt: string, cb: (evt?: unknown, data?: unknown) => void) {
      this.handlers[evt] = cb
    }
    constructor() {
      instances.push(this as unknown as MockHlsInstance)
    }
  }
  return { default: Hls }
})

function makeClip(filePath: string): MediaItemRow {
  return {
    basename: 'clip',
    file_path: filePath,
    kind: 'clip',
    thumbnail_path: null,
    match_id: null,
    capture_end_utc: null,
    match_start_time: null,
    section: 'mine',
    owner_gamertag: 'me',
    map_name: null,
    mode_name: null,
    liked: false,
    like_count: 0,
  }
}

describe('CoverFlowModal — lecture HLS', () => {
  beforeEach(() => {
    instances.length = 0
  })

  it('branche hls.js sur un clip .m3u8', () => {
    const item = makeClip('/api/v1/players/me/media/files/me/hls/clip/master.m3u8')
    renderWithProviders(
      <CoverFlowModal items={[item]} startIndex={0} onClose={vi.fn()} onToggleLike={vi.fn()} />,
    )
    expect(instances).toHaveLength(1)
    expect(instances[0].loadSource).toHaveBeenCalledWith(item.file_path)
    expect(instances[0].attachMedia).toHaveBeenCalled()
  })

  it('utilise hls.js même quand canPlayType renvoie "maybe" (quirk Chrome)', () => {
    // Régression incident 2026-06-14 : Chrome renvoie "maybe" pour HLS (truthy)
    // MAIS n'expose pas video.audioTracks. Prendre le natif en premier lisait la
    // vidéo sans jamais brancher hls.js → AUDIO_TRACKS_UPDATED jamais émis → aucun
    // sélecteur de pistes audio. hls.js doit primer dès que MSE est supporté.
    const spy = vi.spyOn(HTMLMediaElement.prototype, 'canPlayType').mockReturnValue('maybe')
    try {
      const item = makeClip('/api/v1/players/me/media/files/me/hls/clip/master.m3u8')
      renderWithProviders(
        <CoverFlowModal items={[item]} startIndex={0} onClose={vi.fn()} onToggleLike={vi.fn()} />,
      )
      expect(instances).toHaveLength(1) // hls.js branché malgré canPlayType="maybe"
      expect(instances[0].attachMedia).toHaveBeenCalled()
    } finally {
      spy.mockRestore()
    }
  })

  it('affiche le sélecteur par-piste (legacy) quand les renditions ne sont pas game/voices/full', () => {
    const item = makeClip('/x/master.m3u8')
    renderWithProviders(
      <CoverFlowModal items={[item]} startIndex={0} onClose={vi.fn()} onToggleLike={vi.fn()} />,
    )
    // Clip legacy : noms bruts "Game"/"Mic" → repli sur le sélecteur par-piste.
    act(() => {
      instances[0].handlers['audioTracksUpdated']?.(undefined, { audioTracks: instances[0].audioTracks })
    })
    expect(screen.getByText('Game')).toBeInTheDocument()
    expect(screen.getByText('Mic')).toBeInTheDocument()
  })

  it('lit un mp4 en direct sans instancier hls.js', () => {
    const item = makeClip('/media/clip.mp4')
    renderWithProviders(
      <CoverFlowModal items={[item]} startIndex={0} onClose={vi.fn()} onToggleLike={vi.fn()} />,
    )
    expect(instances).toHaveLength(0)
  })
})

describe('CoverFlowModal — interrupteurs Jeu/Voix (layout game/voices/full)', () => {
  beforeEach(() => {
    instances.length = 0
  })

  // Peuple les 3 renditions pré-mixées et retourne l'instance hls.js mockée.
  function setupToggleClip() {
    const item = makeClip('/x/master.m3u8')
    renderWithProviders(
      <CoverFlowModal items={[item]} startIndex={0} onClose={vi.fn()} onToggleLike={vi.fn()} />,
    )
    act(() => {
      instances[0].handlers['audioTracksUpdated']?.(undefined, {
        audioTracks: [{ name: 'game' }, { name: 'voices' }, { name: 'full' }],
      })
    })
    return instances[0]
  }

  it('affiche deux interrupteurs Jeu/Voix actifs par défaut → rendition full', () => {
    const hls = setupToggleClip()
    expect(screen.getByRole('button', { name: 'Jeu', pressed: true })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Voix', pressed: true })).toBeInTheDocument()
    expect(screen.queryByText('game')).toBeNull() // pas de noms bruts
    expect(hls.audioTrack).toBe(2) // index de 'full'
  })

  it('désactiver Voix bascule sur la rendition jeu seul', () => {
    const hls = setupToggleClip()
    act(() => {
      fireEvent.click(screen.getByRole('button', { name: 'Voix' }))
    })
    expect(screen.getByRole('button', { name: 'Voix', pressed: false })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Jeu', pressed: true })).toBeInTheDocument()
    expect(hls.audioTrack).toBe(0) // index de 'game'
  })

  it('désactiver Jeu seul bascule sur la rendition voix', () => {
    const hls = setupToggleClip()
    act(() => {
      fireEvent.click(screen.getByRole('button', { name: 'Jeu' }))
    })
    expect(hls.audioTrack).toBe(1) // index de 'voices'
  })

  it('désactiver les deux interrupteurs coupe le son (vidéo muette)', () => {
    setupToggleClip()
    act(() => {
      fireEvent.click(screen.getByRole('button', { name: 'Voix' }))
    })
    act(() => {
      fireEvent.click(screen.getByRole('button', { name: 'Jeu' }))
    })
    const video = document.querySelector('video') as HTMLVideoElement
    expect(video.muted).toBe(true)
  })
})

describe('CoverFlowModal — persistance du mute "deux OFF" au recentrage', () => {
  beforeEach(() => {
    instances.length = 0
  })

  // Régression : l'instance ClipPlayer persiste tant que le clip reste dans la
  // fenêtre ±2 (key portée par la div de slot). Scénario : deux toggles OFF →
  // navigation vers un voisin → retour. L'effet parent [currentItem] forçait
  // muted=false sur le clip recentré (et s'exécute APRÈS l'effet enfant du même
  // commit) → le son revenait alors que l'UI affichait Jeu OFF / Voix OFF. Le
  // marqueur data-audio-off, consulté par le parent, ferme la fenêtre.
  it('both-OFF → navigation ailleurs → retour : vidéo TOUJOURS muette et toggles OFF', () => {
    vi.useFakeTimers()
    try {
      const items = [
        makeClip('/x/A/master.m3u8'),
        makeClip('/x/B/master.m3u8'),
      ]
      const { container } = renderWithProviders(
        <CoverFlowModal items={items} startIndex={0} onClose={vi.fn()} onToggleLike={vi.fn()} />,
      )
      // A est centré → instances[0]. Peupler les 3 renditions → layout 2 toggles.
      act(() => {
        instances[0].handlers['audioTracksUpdated']?.(undefined, {
          audioTracks: [{ name: 'game' }, { name: 'voices' }, { name: 'full' }],
        })
      })
      // Couper Jeu ET Voix → A muet + marqueur data-audio-off posé.
      act(() => {
        fireEvent.click(screen.getByRole('button', { name: 'Voix' }))
      })
      act(() => {
        fireEvent.click(screen.getByRole('button', { name: 'Jeu' }))
      })
      const videoA = container.querySelector('video[controls]') as HTMLVideoElement
      expect(videoA.muted).toBe(true)
      expect(videoA.dataset.audioOff).toBe('1')

      // Naviguer vers B (A se décentre mais son instance persiste dans la ±2).
      act(() => {
        fireEvent.keyDown(window, { key: 'ArrowRight' })
      })
      act(() => {
        vi.advanceTimersByTime(500) // ANIM_MS : libère animatingRef
      })

      // Retour sur A : le parent NE doit PAS le démuter (data-audio-off respecté).
      act(() => {
        fireEvent.keyDown(window, { key: 'ArrowLeft' })
      })
      act(() => {
        vi.advanceTimersByTime(500)
      })

      const videoABack = container.querySelector('video[controls]') as HTMLVideoElement
      expect(videoABack.muted).toBe(true)
      expect(screen.getByRole('button', { name: 'Jeu', pressed: false })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Voix', pressed: false })).toBeInTheDocument()
    } finally {
      vi.useRealTimers()
    }
  })
})

/**
 * Garde-rail du correctif 2026-09-17 (« les voisins HLS ne lisent jamais »).
 *
 * hls.js expose stopLoad() sur TOUS ses networkControllers, PlaylistLoader en
 * tête : appeler stopLoad() sur une instance qui vient d'appeler loadSource()
 * avorte la requête du manifest encore en vol. Au recentrage, startLoad() ne
 * trouve aucun level et reste STOPPED sans jamais relancer MANIFEST_LOADING.
 * C'est exactement ce que faisait l'effet de centrage (stopLoad inconditionnel
 * au montage d'un voisin). Retirer le garde startedRef du composant doit faire
 * ROUGIR les deux premiers tests ci-dessous.
 */
describe('CoverFlowModal — chargement HLS réservé au clip centré', () => {
  beforeEach(() => {
    instances.length = 0
  })

  // Trois clips HLS, le clip du milieu centré : les 3 slots tiennent dans la
  // fenêtre de proximité (±2) → instances[0]=A (voisin gauche), [1]=B (centre),
  // [2]=C (voisin droit), dans l'ordre de montage.
  function renderTrio() {
    const items = [
      makeClip('/x/A/master.m3u8'),
      makeClip('/x/B/master.m3u8'),
      makeClip('/x/C/master.m3u8'),
    ]
    const { container } = renderWithProviders(
      <CoverFlowModal items={items} startIndex={1} onClose={vi.fn()} onToggleLike={vi.fn()} />,
    )
    return { items, container }
  }

  // Le bouton « suivant » n'a pas de libellé accessible : on le repère par le
  // chevron droit qu'il contient (même chemin SVG que dans le composant).
  function clickNext(container: HTMLElement) {
    const btn = Array.from(container.querySelectorAll('button')).find((b) =>
      b.querySelector('path[d="M9 5l7 7-7 7"]'),
    )
    expect(btn).toBeDefined()
    act(() => {
      fireEvent.click(btn as HTMLButtonElement)
    })
    act(() => {
      vi.advanceTimersByTime(500) // ANIM_MS : libère animatingRef
    })
  }

  it('un voisin monté charge son manifest et ne reçoit jamais stopLoad', () => {
    const { items } = renderTrio()
    // Chaque slot demande son manifest (loadSource), qui est lu et conservé. Les
    // pistes audio, elles, n'arrivent qu'au premier startLoad() (hls.js n'émet
    // AUDIO_TRACKS_UPDATED que depuis switchLevel, sur LEVEL_LOADING /
    // LEVEL_SWITCHING) : un voisin n'a donc pas encore son sélecteur, il l'obtient
    // au recentrage — seul moment où il est affiché.
    expect(instances).toHaveLength(3)
    expect(instances[0].loadSource).toHaveBeenCalledWith(items[0].file_path)
    expect(instances[1].loadSource).toHaveBeenCalledWith(items[1].file_path)
    expect(instances[2].loadSource).toHaveBeenCalledWith(items[2].file_path)

    // Seul le centre charge des segments.
    expect(instances[1].startLoad).toHaveBeenCalledTimes(1)
    expect(instances[0].startLoad).not.toHaveBeenCalled()
    expect(instances[2].startLoad).not.toHaveBeenCalled()

    // Et surtout : aucun stopLoad sur un voisin jamais démarré (sinon son
    // master.m3u8 est avorté et le clip ne lira plus jamais).
    expect(instances[0].stopLoad).not.toHaveBeenCalled()
    expect(instances[2].stopLoad).not.toHaveBeenCalled()
    expect(instances[1].stopLoad).not.toHaveBeenCalled()
  })

  it('passer au suivant démarre le voisin sans avoir coupé son manifest', () => {
    vi.useFakeTimers()
    try {
      const { container } = renderTrio()
      clickNext(container)

      // C (ex-voisin droit) devient le centre : il démarre, et son compteur
      // stopLoad est resté à 0 — son manifest n'a donc jamais été avorté.
      expect(instances[2].startLoad).toHaveBeenCalledTimes(1)
      expect(instances[2].stopLoad).not.toHaveBeenCalled()

      // B (ex-centre, démarré) quitte le centre : stopLoad exactement une fois,
      // seul cas où couper est utile (arrêt de ses téléchargements de segments).
      expect(instances[1].stopLoad).toHaveBeenCalledTimes(1)

      // A reste un voisin jamais démarré : toujours aucun stopLoad.
      expect(instances[0].startLoad).not.toHaveBeenCalled()
      expect(instances[0].stopLoad).not.toHaveBeenCalled()
    } finally {
      vi.useRealTimers()
    }
  })

  it('revenir sur un clip déjà lu le redémarre', () => {
    vi.useFakeTimers()
    try {
      renderTrio()
      act(() => {
        fireEvent.keyDown(window, { key: 'ArrowRight' })
      })
      act(() => {
        vi.advanceTimersByTime(500)
      })
      expect(instances[1].stopLoad).toHaveBeenCalledTimes(1)

      // Retour sur B : l'instance persiste (fenêtre ±2) et doit redémarrer.
      act(() => {
        fireEvent.keyDown(window, { key: 'ArrowLeft' })
      })
      act(() => {
        vi.advanceTimersByTime(500)
      })
      expect(instances[1].startLoad).toHaveBeenCalledTimes(2)
      expect(instances[2].stopLoad).toHaveBeenCalledTimes(1)
    } finally {
      vi.useRealTimers()
    }
  })
})
