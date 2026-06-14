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
