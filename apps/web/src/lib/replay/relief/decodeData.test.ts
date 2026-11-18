import { describe, expect, it } from 'vitest'

import { decodePolylines, hasNoMesh, parseReliefData } from './decodeData'
import { isReliefError } from './errors'
import { lineCount, pixelToWorld, worldToPixel } from './lineSet'
import { rawRelief, reliefData } from './reliefScene.testutil'

function codeOf(fn: () => unknown): string | undefined {
  try {
    fn()
  } catch (err) {
    return isReliefError(err) ? err.code : `autre:${String(err)}`
  }
  return undefined
}

const mutate = (patch: (o: Record<string, unknown>) => void): Record<string, unknown> => {
  const o = rawRelief({ width: 20, height: 10, rects: [{ i0: 0, j0: 0, i1: 9, j1: 9, z: 1 }] })
  patch(o)
  return o
}

describe('traits delta-encodés au dixième de pixel', () => {
  it('le premier couple est absolu, les suivants sont des écarts', () => {
    const set = decodePolylines([[100, 200, 5, -5, 10, 0]], 'steps')
    expect(lineCount(set)).toBe(1)
    expect(Array.from(set.points)).toEqual([10, 20, 10.5, 19.5, 11.5, 19.5])
    expect(Array.from(set.offsets)).toEqual([0, 3])
  })

  it('range plusieurs polylignes à la suite avec leurs décalages', () => {
    const set = decodePolylines([[0, 0, 10, 0], [50, 50, 0, 10, 10, 0]], 'steps')
    expect(Array.from(set.offsets)).toEqual([0, 2, 5])
    expect(Array.from(set.points.slice(4, 6))).toEqual([5, 5])
    expect(Array.from(set.points.slice(8, 10))).toEqual([6, 6])
  })

  it('une polyligne d’un seul sommet est écartée', () => {
    const set = decodePolylines([[30, 40], [0, 0, 10, 10]], 'steps')
    expect(lineCount(set)).toBe(1)
  })

  it('refuse une longueur impaire et un dixième non entier', () => {
    expect(codeOf(() => decodePolylines([[1, 2, 3]], 'steps'))).toBe('data-invalid')
    expect(codeOf(() => decodePolylines([[1.5, 2]], 'steps'))).toBe('data-invalid')
    expect(codeOf(() => decodePolylines([['a', 2]], 'steps'))).toBe('data-invalid')
  })

  it('un point du fichier est au CENTRE du pixel : u = 105 est le bord entre les pixels 10 et 11', () => {
    const cal = { metersPerPixel: 0.029, originX: -10, originY: 28, widthPx: 100, heightPx: 100 }
    const set = decodePolylines([[105, 200, 0, 10]], 'steps')
    expect(set.points[0]).toBeCloseTo(10.5, 5)
    const w = pixelToWorld(cal, set.points[0], set.points[1])
    expect(w.x).toBeCloseTo(-10 + (10.5 + 0.5) * 0.029, 9)
    expect(w.y).toBeCloseTo(28 - (20 + 0.5) * 0.029, 9)
    const back = worldToPixel(cal, w.x, w.y)
    expect(back.x).toBeCloseTo(10.5, 9)
    expect(back.y).toBeCloseTo(20, 9)
  })
})

describe('parseReliefData', () => {
  it('décode le calage, la rampe, les étages, les sols et les degradations', () => {
    const data = reliefData({
      width: 20,
      height: 10,
      levels: [0, 2.2],
      meshShare: 0.46,
      rects: [{ i0: 0, j0: 0, i1: 9, j1: 9, z: 2.2 }],
      steps: [[10, 20, 5, 5]],
      degradations: ['pile-tronquee'],
    })
    expect(data.calibration).toEqual({ metersPerPixel: 0.1, originX: 0, originY: 1, widthPx: 20, heightPx: 10 })
    expect(data.height).toEqual({ z0: -7, step: 1 / 256, low: 0, high: 4 })
    expect(data.floors.levels).toEqual([0, 2.2])
    expect(data.floors.meshShare).toBe(0.46)
    expect(data.floors.merge).toBe(0.4)
    expect(data.degradations).toEqual(['pile-tronquee'])
    expect(data.lines.steps.points.length).toBe(4)
    expect(data.lines.contours.points.length).toBe(0)
  })

  it('range les sommets en unités monde (centièmes / 100) et les faces en indices à plat', () => {
    const data = reliefData({ width: 20, height: 10, rects: [{ i0: 0, j0: 0, i1: 9, j1: 9, z: 2.2 }] })
    const { mesh } = data.floors
    expect(Array.from(mesh.faceStart)).toEqual([0, 4])
    expect(Array.from(mesh.faceIndices)).toEqual([0, 1, 2, 3])
    expect(mesh.vertices[2]).toBeCloseTo(2.2, 9)
    expect(Array.from(mesh.bounds)).toEqual([0, 0, 1, 1])
  })

  it('un fichier plus ancien sans merge ni tolerance prend les défauts', () => {
    const raw = mutate((o) => {
      const floors = o.floors as Record<string, unknown>
      delete floors.merge
      delete floors.tolerance
    })
    const data = parseReliefData(raw)
    expect(data.floors.merge).toBe(0.4)
    expect(data.floors.tolerance).toBe(0.3)
  })

  it('ignore les clés inconnues et les codes de dégradation inconnus', () => {
    const raw = mutate((o) => {
      o.image = 'x.webp'
      o.degradations = ['sans-maillage', 'code-du-futur']
    })
    expect(parseReliefData(raw).degradations).toEqual(['sans-maillage'])
  })

  it('refuse une version de schéma inconnue avec un code distinct', () => {
    expect(codeOf(() => parseReliefData(mutate((o) => (o.schemaVersion = 2))))).toBe('data-version')
    expect(codeOf(() => parseReliefData(mutate((o) => delete o.schemaVersion)))).toBe('data-version')
  })

  it('refuse ce qui n’est pas un relief', () => {
    expect(codeOf(() => parseReliefData(null))).toBe('data-invalid')
    expect(codeOf(() => parseReliefData([]))).toBe('data-invalid')
    expect(codeOf(() => parseReliefData('{}'))).toBe('data-invalid')
    expect(codeOf(() => parseReliefData(mutate((o) => delete o.calibration)))).toBe('data-invalid')
    expect(codeOf(() => parseReliefData(mutate((o) => delete o.height)))).toBe('data-invalid')
  })

  it('refuse un calage ou une rampe incohérents', () => {
    const cal = (patch: Record<string, unknown>) =>
      mutate((o) => (o.calibration = { ...(o.calibration as object), ...patch }))
    expect(codeOf(() => parseReliefData(cal({ metersPerPixel: 0 })))).toBe('data-invalid')
    expect(codeOf(() => parseReliefData(cal({ widthPx: 10.5 })))).toBe('data-invalid')
    expect(codeOf(() => parseReliefData(cal({ heightPx: -3 })))).toBe('data-invalid')
    expect(codeOf(() => parseReliefData(mutate((o) => (o.height = { ...(o.height as object), step: 0 }))))).toBe('data-invalid')
  })

  it('refuse des sommets ou des faces mal formés', () => {
    const floors = (patch: Record<string, unknown>) =>
      mutate((o) => (o.floors = { ...(o.floors as object), ...patch }))
    expect(codeOf(() => parseReliefData(floors({ vertices: [1, 2] })))).toBe('data-invalid')
    expect(codeOf(() => parseReliefData(floors({ vertices: [1.5, 2, 3, 4, 5, 6, 7, 8, 9] })))).toBe('data-invalid')
    expect(codeOf(() => parseReliefData(floors({ vertices: [0, 0, 0, 1, 0, 0, 1, 1, 0], faces: [[0, 1]] })))).toBe('data-invalid')
    expect(codeOf(() => parseReliefData(floors({ vertices: [0, 0, 0, 1, 0, 0, 1, 1, 0], faces: [[0, 1, 3]] })))).toBe('data-invalid')
    expect(codeOf(() => parseReliefData(floors({ meshShare: 1.5 })))).toBe('data-invalid')
  })

  it('un relief sans maillage : pas de coupe, drapeau de la dégradation ou maillage vide', () => {
    const degraded = reliefData({ width: 20, height: 10, degradations: ['sans-maillage'] })
    expect(hasNoMesh(degraded)).toBe(true)
    expect(hasNoMesh(reliefData({ width: 20, height: 10 }))).toBe(true)
    const withMesh = reliefData({ width: 20, height: 10, rects: [{ i0: 0, j0: 0, i1: 9, j1: 9, z: 1 }] })
    expect(hasNoMesh(withMesh)).toBe(false)
  })
})
