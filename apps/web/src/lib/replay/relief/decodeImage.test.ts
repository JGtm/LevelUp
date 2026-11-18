import { afterEach, describe, expect, it, vi } from 'vitest'

import { FLAG_KNOWN, FLAG_MESH_FLOOR, FLAG_ON_MAP } from './constants'
import {
  decodeReliefPixels,
  domLoaders,
  readReliefImage,
  type PixelSource,
  type RawPixels,
  type ReliefImageLoaders,
} from './decodeImage'
import { isReliefError, ReliefError } from './errors'
import { sceneCalibration } from './reliefScene.testutil'
import type { ReliefHeightMeta } from './types'

const META: ReliefHeightMeta = { z0: -7, step: 1 / 256, low: 0, high: 10 }
const CAL256 = sceneCalibration(256, 256)

/** Une image `w × h` où chaque pixel est `[r, g, b, a]` donné par `px(p)`. */
function pixels(w: number, h: number, px: (p: number) => [number, number, number, number]): RawPixels {
  const data = new Uint8ClampedArray(w * h * 4)
  for (let p = 0; p < w * h; p++) data.set(px(p), p * 4)
  return { width: w, height: h, data }
}

const known = (q: number, b = 0): [number, number, number, number] => [q & 255, q >> 8, b, 255]

function codeOf(fn: () => unknown): string | undefined {
  try {
    fn()
  } catch (err) {
    return isReliefError(err) ? err.code : `autre:${String(err)}`
  }
  return undefined
}

describe('decodeReliefPixels', () => {
  it('fait l’aller-retour de q sur toute la plage 0..65535, au bit près', () => {
    const image = pixels(256, 256, (p) => known(p))
    const out = decodeReliefPixels(image, CAL256, META)
    for (let q = 0; q < 65536; q++) {
      expect(out.heights[q]).toBe(META.z0 + q * META.step)
      if (Math.round((out.heights[q] - META.z0) / META.step) !== q) throw new Error(`q=${q} ne revient pas`)
    }
    expect(out.heights[65535]).toBe(-7 + 65535 / 256)
  })

  it('range R en octet faible et G en octet fort (q = R + 256 G)', () => {
    const out = decodeReliefPixels(pixels(256, 256, () => [3, 2, 0, 255]), CAL256, META)
    expect(out.heights[0]).toBe(-7 + (3 + 256 * 2) / 256)
  })

  it('décode les deux drapeaux et pose le bit interne « connu »', () => {
    const cell = (b: number) => pixels(256, 256, () => known(100, b))
    expect(decodeReliefPixels(cell(0), CAL256, META).flags[0]).toBe(FLAG_KNOWN)
    expect(decodeReliefPixels(cell(1), CAL256, META).flags[0]).toBe(FLAG_KNOWN | FLAG_ON_MAP)
    expect(decodeReliefPixels(cell(3), CAL256, META).flags[0]).toBe(FLAG_KNOWN | FLAG_ON_MAP | FLAG_MESH_FLOOR)
  })

  it('une hauteur inconnue (A = 0) rend NaN et aucun drapeau', () => {
    const out = decodeReliefPixels(pixels(256, 256, (p) => (p === 5 ? [0, 0, 0, 0] : known(p))), CAL256, META)
    expect(Number.isNaN(out.heights[5])).toBe(true)
    expect(out.flags[5]).toBe(0)
    expect(out.flags[6]).toBe(FLAG_KNOWN)
  })

  it('refuse toute alpha intermédiaire : jamais d’autre valeur que 0 et 255', () => {
    for (const a of [1, 2, 64, 127, 128, 200, 254]) {
      const img = pixels(256, 256, (p) => (p === 1000 ? [10, 20, 1, a] : known(p)))
      expect(codeOf(() => decodeReliefPixels(img, CAL256, META))).toBe('image-alpha')
    }
  })

  it('refuse A = 0 quand R, G ou B portent du contenu', () => {
    for (const bad of [[1, 0, 0, 0], [0, 1, 0, 0], [0, 0, 1, 0]] as const) {
      const img = pixels(256, 256, (p) => (p === 9 ? [bad[0], bad[1], bad[2], 0] : known(p)))
      expect(codeOf(() => decodeReliefPixels(img, CAL256, META))).toBe('image-flags')
    }
  })

  it('refuse un bit réservé de B (bits 2 à 7)', () => {
    for (const b of [4, 8, 16, 32, 64, 128, 255]) {
      expect(codeOf(() => decodeReliefPixels(pixels(256, 256, () => known(1, b)), CAL256, META))).toBe('image-flags')
    }
  })

  it('refuse des dimensions différentes du calage et un tableau de la mauvaise longueur', () => {
    expect(codeOf(() => decodeReliefPixels(pixels(128, 128, () => known(0)), CAL256, META))).toBe('image-size')
    const short = { width: 256, height: 256, data: new Uint8ClampedArray(10) }
    expect(codeOf(() => decodeReliefPixels(short, CAL256, META))).toBe('image-size')
  })
})

/** Un chargeur qui rend une image synthétique. */
function source(image: RawPixels, spies?: { released: () => void }): PixelSource {
  return {
    width: image.width,
    height: image.height,
    readPixels: () => image,
    release: () => spies?.released(),
  }
}

describe('readReliefImage', () => {
  afterEach(() => vi.restoreAllMocks())

  const good = pixels(256, 256, (p) => known(p))

  it('lit par createImageBitmap quand il marche, sans toucher au repli, et libère la source', async () => {
    const released = vi.fn()
    const loaders: ReliefImageLoaders = {
      loadBitmap: vi.fn(async () => source(good, { released })),
      loadElement: vi.fn(async () => source(good)),
    }
    const out = await readReliefImage(new Blob(['x']), CAL256, META, loaders)
    expect(out.heights[300]).toBe(META.z0 + 300 * META.step)
    expect(loaders.loadElement).not.toHaveBeenCalled()
    expect(released).toHaveBeenCalledTimes(1)
  })

  it('retombe sur l’élément image quand createImageBitmap refuse les options, avec le même contrôle', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const loaders: ReliefImageLoaders = {
      loadBitmap: vi.fn(async () => {
        throw new Error('options refusées')
      }),
      loadElement: vi.fn(async () => source(good)),
    }
    const out = await readReliefImage(new Blob(['x']), CAL256, META, loaders)
    expect(out.width).toBe(256)
    expect(loaders.loadElement).toHaveBeenCalledTimes(1)
    expect(warn).toHaveBeenCalledTimes(1)

    const converted = pixels(256, 256, (p) => (p === 7 ? [10, 20, 0, 128] : known(p)))
    const bad: ReliefImageLoaders = {
      loadBitmap: async () => {
        throw new Error('options refusées')
      },
      loadElement: async () => source(converted),
    }
    await expect(readReliefImage(new Blob(['x']), CAL256, META, bad)).rejects.toMatchObject({ code: 'image-alpha' })
  })

  it('une image corrompue (aucun chargeur ne la lit) rend une erreur typée : repli sur le Plan', async () => {
    vi.spyOn(console, 'warn').mockImplementation(() => {})
    const broken: ReliefImageLoaders = {
      loadBitmap: async () => {
        throw new Error('corrompue')
      },
      loadElement: async () => {
        throw new Error('corrompue aussi')
      },
    }
    const err = await readReliefImage(new Blob(['x']), CAL256, META, broken).catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ReliefError)
    expect((err as ReliefError).code).toBe('image-decode')
  })

  it('refuse une image de la mauvaise taille avant de relire ses pixels, et libère quand même', async () => {
    const released = vi.fn()
    const readPixels = vi.fn(() => good)
    const loaders: ReliefImageLoaders = {
      loadBitmap: async () => ({ width: 100, height: 100, readPixels, release: released }),
      loadElement: async () => source(good),
    }
    await expect(readReliefImage(new Blob(['x']), CAL256, META, loaders)).rejects.toMatchObject({ code: 'image-size' })
    expect(readPixels).not.toHaveBeenCalled()
    expect(released).toHaveBeenCalledTimes(1)
  })

  it('une relecture de pixels qui échoue rend image-decode', async () => {
    const loaders: ReliefImageLoaders = {
      loadBitmap: async () => ({
        width: 256,
        height: 256,
        readPixels: () => {
          throw new Error('canevas pollué')
        },
        release: () => {},
      }),
      loadElement: async () => source(good),
    }
    await expect(readReliefImage(new Blob(['x']), CAL256, META, loaders)).rejects.toMatchObject({ code: 'image-decode' })
  })
})

describe('domLoaders', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('appelle createImageBitmap SANS conversion de couleur ni prémultiplication', async () => {
    const close = vi.fn()
    const create = vi.fn(async () => ({ width: 2, height: 1, close }))
    vi.stubGlobal('createImageBitmap', create)
    const blob = new Blob(['x'])
    const src = await domLoaders.loadBitmap(blob)
    expect(create).toHaveBeenCalledWith(blob, { colorSpaceConversion: 'none', premultiplyAlpha: 'none' })
    expect(src.width).toBe(2)
    src.release()
    expect(close).toHaveBeenCalledTimes(1)
  })

  it('relit le bitmap à taille 1:1 sur un canevas sRGB sans lissage', async () => {
    const bitmap = { width: 2, height: 1, close: vi.fn() }
    vi.stubGlobal('createImageBitmap', vi.fn(async () => bitmap))
    const drawImage = vi.fn()
    const getImageData = vi.fn(() => ({ data: new Uint8ClampedArray(8) }))
    const ctx = { imageSmoothingEnabled: true, drawImage, getImageData }
    const getContext = vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(ctx as never)
    const src = await domLoaders.loadBitmap(new Blob(['x']))
    const raw = src.readPixels()
    expect(getContext).toHaveBeenCalledWith('2d', { willReadFrequently: true, colorSpace: 'srgb' })
    expect(ctx.imageSmoothingEnabled).toBe(false)
    expect(drawImage).toHaveBeenCalledWith(bitmap, 0, 0)
    expect(getImageData).toHaveBeenCalledWith(0, 0, 2, 1)
    expect(raw.data.length).toBe(8)
  })

  it('un contexte 2D indisponible est une erreur de relecture, pas un crash', async () => {
    vi.stubGlobal('createImageBitmap', vi.fn(async () => ({ width: 2, height: 1, close: vi.fn() })))
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null)
    const src = await domLoaders.loadBitmap(new Blob(['x']))
    expect(() => src.readPixels()).toThrow('contexte 2D indisponible')
  })
})
