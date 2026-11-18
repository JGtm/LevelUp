/**
 * reliefBackground.test.ts — les images du fond que produit le relief, au contexte enregistreur : le relief
 * teinté se pose tel quel (RGBA de la frame), le fond Plan ou Rendu est copié puis son opacité est multipliée par le
 * masque de la coupe (`destination-in` d'un canvas dont l'alpha est le masque), les canvas se réutilisent d'une
 * frame à l'autre, et un navigateur sans contexte 2D rend `null` (le fond de base reste posé).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { recordingContext, type RecordedOp } from './canvasRecorder.testutil'
import type { ReliefFrame } from './engine'
import { paintMaskedImage, paintReliefImage, ReliefCanvases } from './reliefBackground'

/** ImageData minimal : jsdom n'en fournit pas, et seul le contenu transmis à putImageData compte ici. */
class FakeImageData {
  readonly data: Uint8ClampedArray
  readonly width: number
  readonly height: number

  constructor(data: Uint8ClampedArray, width: number, height: number) {
    this.data = data
    this.width = width
    this.height = height
  }
}

let journal: Array<{ canvas: HTMLCanvasElement; ops: RecordedOp[] }> = []
let refuse = false

beforeEach(() => {
  journal = []
  refuse = false
  vi.stubGlobal('ImageData', FakeImageData)
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(function (this: HTMLCanvasElement) {
    if (refuse) return null
    const rec = recordingContext()
    journal.push({ canvas: this, ops: rec.ops })
    return rec.ctx as never
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

function frame(over: Partial<ReliefFrame> = {}): ReliefFrame {
  return {
    width: 3,
    height: 2,
    cut: null,
    image: new Uint8ClampedArray(3 * 2 * 4).fill(7),
    alpha: null,
    planAlpha: null,
    lines: null,
    ...over,
  }
}

const IMAGE = {} as CanvasImageSource

describe('paintReliefImage', () => {
  it('pose le RGBA teinté de la frame dans un canvas de la taille du calage', () => {
    const canvas = paintReliefImage(null, frame())
    expect(canvas?.width).toBe(3)
    expect(canvas?.height).toBe(2)
    const put = journal.flatMap((j) => j.ops).find((o) => o.op === 'putImageData')
    const data = put?.args[0] as FakeImageData
    expect(data.width).toBe(3)
    expect(data.height).toBe(2)
    expect(data.data.every((v) => v === 7)).toBe(true)
  })

  it('réutilise le canvas d’une frame à l’autre quand la taille ne change pas', () => {
    const first = paintReliefImage(null, frame())
    const second = paintReliefImage(first, frame())
    expect(second).toBe(first)
  })

  it('rend null sans image teintée (fond Relief non demandé) ou sans contexte 2D', () => {
    expect(paintReliefImage(null, frame({ image: null }))).toBeNull()
    refuse = true
    expect(paintReliefImage(null, frame())).toBeNull()
  })
})

describe('paintMaskedImage', () => {
  it('copie l’image puis compose le masque en destination-in : l’opacité est multipliée', () => {
    const mask = Uint8Array.of(0, 128, 255, 255, 64, 0)
    const canvas = paintMaskedImage(null, IMAGE, mask, 3, 2)
    expect(canvas).not.toBeNull()
    const target = journal.find((j) => j.canvas === canvas)
    const ops = target?.ops.map((o) => (o.op === 'set globalCompositeOperation' ? `${o.op}:${String(o.args[0])}` : o.op)) ?? []
    // La copie de l'image d'abord (source-over), le masque ensuite (destination-in).
    const copy = ops.indexOf('drawImage')
    const compose = ops.indexOf('set globalCompositeOperation:destination-in')
    expect(copy).toBeGreaterThanOrEqual(0)
    expect(compose).toBeGreaterThan(copy)
    expect(ops.lastIndexOf('drawImage')).toBeGreaterThan(compose)
  })

  it('le canvas du masque porte le masque dans son alpha, un octet par pixel', () => {
    const mask = Uint8Array.of(0, 128, 255, 255, 64, 0)
    paintMaskedImage(null, IMAGE, mask, 3, 2)
    const put = journal.flatMap((j) => j.ops).find((o) => o.op === 'putImageData')
    const rgba = (put?.args[0] as FakeImageData).data
    expect(rgba.length).toBe(3 * 2 * 4)
    expect(Array.from({ length: 6 }, (_, p) => rgba[p * 4 + 3])).toEqual(Array.from(mask))
    expect(Array.from({ length: 6 }, (_, p) => rgba[p * 4])).toEqual([0, 0, 0, 0, 0, 0])
  })

  it('rend null sans contexte 2D : l’appelant garde l’image d’origine', () => {
    refuse = true
    expect(paintMaskedImage(null, IMAGE, Uint8Array.of(1), 1, 1)).toBeNull()
  })
})

describe('ReliefCanvases', () => {
  it('le fond Plan prend le masque du plan, le Rendu celui du rendu ; sans masque, rien', () => {
    const canvases = new ReliefCanvases()
    const f = frame({ alpha: Uint8Array.of(1, 1, 1, 1, 1, 1), planAlpha: Uint8Array.of(2, 2, 2, 2, 2, 2) })
    canvases.maskedImage(f, IMAGE, true)
    const planPut = journal.flatMap((j) => j.ops).filter((o) => o.op === 'putImageData').pop()
    expect((planPut?.args[0] as FakeImageData).data[3]).toBe(2)
    journal = []
    canvases.maskedImage(f, IMAGE, false)
    const renduPut = journal.flatMap((j) => j.ops).filter((o) => o.op === 'putImageData').pop()
    expect((renduPut?.args[0] as FakeImageData).data[3]).toBe(1)
    expect(canvases.maskedImage(frame(), IMAGE, true)).toBeNull()
  })

  it('réutilise ses deux canvas d’une frame à l’autre', () => {
    const canvases = new ReliefCanvases()
    expect(canvases.reliefImage(frame())).toBe(canvases.reliefImage(frame()))
    const f = frame({ alpha: Uint8Array.of(1, 1, 1, 1, 1, 1) })
    expect(canvases.maskedImage(f, IMAGE, false)).toBe(canvases.maskedImage(f, IMAGE, false))
  })
})
