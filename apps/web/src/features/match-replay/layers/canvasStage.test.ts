/**
 * canvasStage.test.ts — la préparation de la toile avant chaque tracé : taille de dessin à la densité
 * du moment, effacement, contexte prêt. Extraite du canvas du rejeu sans changement de rendu.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'

import { recordingContext } from '../test/recordingContext'
import { prepareStage } from './canvasStage'

function fakeCanvas(width = 0, height = 0) {
  const { ctx, ops } = recordingContext()
  const canvas = { width, height, getContext: vi.fn(() => ctx) } as unknown as HTMLCanvasElement
  return { canvas, ops }
}

afterEach(() => vi.unstubAllGlobals())

describe('prepareStage', () => {
  it('rend null sans toile, sans largeur mesurée ou sans contexte', () => {
    expect(prepareStage(null, 400, 300)).toBeNull()
    expect(prepareStage(fakeCanvas().canvas, 0, 300)).toBeNull()
    const sansContexte = { width: 0, height: 0, getContext: () => null } as unknown as HTMLCanvasElement
    expect(prepareStage(sansContexte, 400, 300)).toBeNull()
  })

  it('dimensionne la toile à la densité du moment', () => {
    vi.stubGlobal('devicePixelRatio', 2)
    const { canvas } = fakeCanvas()
    const stage = prepareStage(canvas, 400, 300)
    expect(stage?.dpr).toBe(2)
    expect(canvas.width).toBe(800)
    expect(canvas.height).toBe(600)
  })

  it('ne retouche pas la taille d’une toile déjà à la bonne taille (redimensionner efface)', () => {
    vi.stubGlobal('devicePixelRatio', 1)
    const { canvas } = fakeCanvas(400, 300)
    const ecritures: number[] = []
    let largeur = 400
    Object.defineProperty(canvas, 'width', {
      get: () => largeur,
      set: (v: number) => {
        ecritures.push(v)
        largeur = v
      },
    })
    prepareStage(canvas, 400, 300)
    expect(ecritures).toEqual([])
  })

  it('pose le repère à la densité puis efface toute la zone de dessin', () => {
    vi.stubGlobal('devicePixelRatio', 2)
    const { canvas, ops } = fakeCanvas()
    prepareStage(canvas, 400, 300)
    expect(ops.map((o) => o.op)).toEqual(['setTransform', 'clearRect'])
    expect(ops[0].args).toEqual([2, 0, 0, 2, 0, 0])
    expect(ops[1].args).toEqual([0, 0, 400, 300])
  })
})
