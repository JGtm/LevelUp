/**
 * exportFormats.test.ts — le catalogue des formats et la geometrie qu'il impose.
 *
 * Ce qui se verrouille : des dimensions que H.264 accepte (paires), un 16:9 EXACT, une densite
 * qui retombe au pixel pres sur le format (la toile est dimensionnee par `Math.round`), et un
 * identifiant inconnu qui ramene au defaut plutot qu'a un export sans format.
 */
import { describe, expect, it } from 'vitest'

import {
  DEFAULT_EXPORT_FORMAT_ID,
  DEFAULT_EXPORT_FRAMING,
  EXPORT_FORMAT_IDS,
  EXPORT_FORMATS,
  EXPORT_LAYOUT,
  exportFormatOf,
  exportLayoutFor,
} from './exportFormats'

describe('exportFormats — le catalogue', () => {
  it('propose exactement 1080p (defaut) puis 720p', () => {
    expect(EXPORT_FORMAT_IDS).toEqual(['1080p', '720p'])
    expect(DEFAULT_EXPORT_FORMAT_ID).toBe('1080p')
    expect(exportFormatOf('1080p')).toMatchObject({ width: 1920, height: 1080 })
    expect(exportFormatOf('720p')).toMatchObject({ width: 1280, height: 720 })
  })

  it('ne sort que des dimensions paires, en 16:9 exact', () => {
    for (const f of EXPORT_FORMATS) {
      expect(f.width % 2).toBe(0)
      expect(f.height % 2).toBe(0)
      expect(f.width * 9).toBe(f.height * 16)
    }
    expect(EXPORT_LAYOUT.width * 9).toBe(EXPORT_LAYOUT.height * 16)
  })

  it('un identifiant inconnu, absent ou vide ramene au defaut', () => {
    for (const id of ['4k', '', null, undefined, '1080P']) {
      expect(exportFormatOf(id).id).toBe(DEFAULT_EXPORT_FORMAT_ID)
    }
  })
})

describe('exportLayoutFor — la geometrie de rendu', () => {
  it('le cadre logique rendu a sa densite retombe AU PIXEL sur le format', () => {
    for (const f of EXPORT_FORMATS) {
      const l = exportLayoutFor(f)
      // Meme arrondi que le dimensionnement de la toile (`ReplayCanvas.draw`).
      expect(Math.round(l.width * l.pixelRatio)).toBe(f.width)
      expect(Math.round(l.height * l.pixelRatio)).toBe(f.height)
    }
  })

  it('les deux formats partagent la meme mise en page : seule la densite change', () => {
    const [a, b] = EXPORT_FORMATS.map((f) => exportLayoutFor(f))
    expect({ w: a.width, h: a.height }).toEqual({ w: b.width, h: b.height })
    expect(a.pixelRatio).toBe(2)
    expect(b.pixelRatio).toBeCloseTo(4 / 3, 12)
  })

  it('cadrage : carte entiere par defaut, le cadrage demande sinon', () => {
    expect(DEFAULT_EXPORT_FRAMING).toBe('whole')
    expect(exportLayoutFor(exportFormatOf('1080p')).framing).toBe('whole')
    expect(exportLayoutFor(exportFormatOf('720p'), 'current').framing).toBe('current')
  })

  it('ne depend pas de la densite de l’ecran', () => {
    const avant = exportLayoutFor(exportFormatOf('1080p'))
    const dpr = window.devicePixelRatio
    Object.defineProperty(window, 'devicePixelRatio', { value: 3, configurable: true })
    try {
      expect(exportLayoutFor(exportFormatOf('1080p'))).toEqual(avant)
    } finally {
      Object.defineProperty(window, 'devicePixelRatio', { value: dpr, configurable: true })
    }
  })
})
