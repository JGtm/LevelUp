/**
 * Tests ReplayMediaLightbox — lecture des clips. Même patron que le test HLS de la galerie
 * (`CoverFlowModal.hls.test.tsx`) : hls.js est mocké, jsdom n'ayant ni MediaSource ni décodeur.
 *
 * Ce qu'ils protègent :
 *  1. UN CLIP TRANSCODÉ EST DU HLS. Son `file_path` mue vers un `master.m3u8` qu'un
 *     `<video src>` nu ne lit pas sur Chrome/Firefox — c'est le piège que ce lot va chercher.
 *  2. UN MP4 RESTE EN LECTURE DIRECTE : pas d'instance hls.js pour rien.
 *  3. UN CLIP SANS DURÉE CONNUE RESTE UNE VIDÉO. La condition d'origine confondait « c'est un
 *     clip » et « on connaît sa durée », et rendait un `<img src=...mp4>` — un cadre vide.
 *  4. UN ÉCHEC SE DIT. Un cadre noir muet ne se distingue pas d'un clip qui charge.
 */
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { act, render, screen } from '@testing-library/react'

import { ReplayMediaLightbox } from './ReplayMediaLightbox'
import type { ReplayMediaItem } from '../model/replayTimelineTracksLogic'

interface MockHlsInstance {
  loadSource: ReturnType<typeof vi.fn>
  attachMedia: ReturnType<typeof vi.fn>
  destroy: ReturnType<typeof vi.fn>
  handlers: Record<string, ((evt?: unknown, data?: unknown) => void) | undefined>
  audioTrack: number
}

const { instances, control } = vi.hoisted(() => ({
  instances: [] as MockHlsInstance[],
  control: { supported: true },
}))

vi.mock('hls.js', () => {
  class Hls {
    static isSupported() {
      return control.supported
    }
    static Events = {
      MANIFEST_PARSED: 'manifestParsed',
      AUDIO_TRACKS_UPDATED: 'audioTracksUpdated',
      ERROR: 'error',
    }
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

function clip(over: Partial<ReplayMediaItem> = {}): ReplayMediaItem {
  return {
    id: 'media-1',
    kind: 'clip',
    replayMs: 30_000,
    durationMs: 20_000,
    thumbUrl: '/thumb.webp',
    url: '/api/v1/players/Hero/media/files/Hero/hls/clip/master.m3u8',
    label: 'Clip Streets',
    ...over,
  }
}

function open(item: ReplayMediaItem) {
  return render(<ReplayMediaLightbox item={item} locale="fr" onClose={vi.fn()} />)
}

describe('ReplayMediaLightbox — lecture HLS', () => {
  beforeEach(() => {
    instances.length = 0
    control.supported = true
  })

  it('branche hls.js sur un clip .m3u8, et NE POSE PAS d’attribut src', () => {
    const item = clip()
    const { container } = open(item)
    expect(instances).toHaveLength(1)
    expect(instances[0].loadSource).toHaveBeenCalledWith(item.url)
    expect(instances[0].attachMedia).toHaveBeenCalled()
    // `src` sur un manifest ferait tenter au navigateur une lecture directe, qui échoue.
    expect(container.querySelector('video')?.getAttribute('src')).toBeNull()
  })

  it('lit un mp4 en direct, sans instancier hls.js', () => {
    const { container } = open(clip({ url: '/api/v1/players/Hero/media/files/Hero/clip.mp4' }))
    expect(instances).toHaveLength(0)
    expect(container.querySelector('video')?.getAttribute('src')).toBe(
      '/api/v1/players/Hero/media/files/Hero/clip.mp4',
    )
  })

  it('UN CLIP SANS DURÉE CONNUE RESTE UNE VIDÉO (et perd seulement sa bande d’images)', () => {
    const { container } = open(clip({ durationMs: undefined, url: '/clip.mp4' }))
    expect(container.querySelector('video')).not.toBeNull()
    expect(container.querySelector('img')).toBeNull()
  })

  it('une capture reste une image, sans lecteur vidéo', () => {
    const { container } = open(clip({ kind: 'image', durationMs: undefined, url: '/shot.png' }))
    expect(container.querySelector('img')).not.toBeNull()
    expect(container.querySelector('video')).toBeNull()
    expect(instances).toHaveLength(0)
  })
})

describe('ReplayMediaLightbox — un échec de lecture se dit', () => {
  beforeEach(() => {
    instances.length = 0
    control.supported = true
  })

  it('erreur FATALE du flux : le message remplace le cadre muet', () => {
    open(clip())
    act(() => {
      instances[0].handlers['error']?.(undefined, { fatal: true, type: 'networkError', details: 'x' })
    })
    expect(screen.getByText('La lecture du clip a échoué.')).toBeTruthy()
  })

  it('erreur NON fatale : hls.js se rétablit seul, rien à dire au lecteur', () => {
    open(clip())
    act(() => {
      instances[0].handlers['error']?.(undefined, { fatal: false, type: 'networkError' })
    })
    expect(screen.queryByText('La lecture du clip a échoué.')).toBeNull()
  })

  it('navigateur sans MSE NI HLS natif : on le dit plutôt que d’afficher un cadre noir', () => {
    control.supported = false
    const spy = vi.spyOn(HTMLMediaElement.prototype, 'canPlayType').mockReturnValue('')
    try {
      open(clip())
      expect(instances).toHaveLength(0)
      expect(screen.getByText('Ce navigateur ne sait pas lire ce clip.')).toBeTruthy()
    } finally {
      spy.mockRestore()
    }
  })

  it('Safari/iOS (HLS natif, pas de MSE) : le manifest part en src direct, sans message', () => {
    control.supported = false
    const spy = vi.spyOn(HTMLMediaElement.prototype, 'canPlayType').mockReturnValue('maybe')
    try {
      const item = clip()
      const { container } = open(item)
      expect(instances).toHaveLength(0)
      expect(container.querySelector('video')?.src).toContain('master.m3u8')
      expect(screen.queryByText('Ce navigateur ne sait pas lire ce clip.')).toBeNull()
    } finally {
      spy.mockRestore()
    }
  })
})
