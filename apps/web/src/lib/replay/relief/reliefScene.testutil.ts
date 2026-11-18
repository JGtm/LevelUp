/**
 * Banc de test du relief : de petites cartes synthétiques (images, sols, JSON) construites en code.
 *
 * Le repère est celui du fichier : le pixel `(i, j)` a son centre monde en
 * `originX + (i + 0,5) × mpp`, `originY − (j + 0,5) × mpp`. Ici `originX = 0` et
 * `originY = hauteur × mpp`, avec `mpp = 0,1` : toutes les arêtes tombent sur des centièmes entiers.
 * Aucune dépendance au DOM : utilisable depuis un test `node` comme `jsdom`.
 */
import { FLAG_KNOWN, FLAG_MESH_FLOOR, FLAG_ON_MAP } from './constants'
import { parseReliefData } from './decodeData'
import type { ReliefCalibration, ReliefData, ReliefHeights } from './types'

/** Un rectangle de sol, en pixels (bornes comprises) : `z` plat, ou quatre hauteurs de coin `[haut-gauche, haut-droit, bas-droit, bas-gauche]`. */
export interface FloorRect {
  i0: number
  j0: number
  i1: number
  j1: number
  z: number | [number, number, number, number]
}

/** Le calage d'une scène : `mpp` unités par pixel, origine monde en haut à gauche. */
export function sceneCalibration(width: number, height: number, mpp = 0.1): ReliefCalibration {
  return { metersPerPixel: mpp, originX: 0, originY: height * mpp, widthPx: width, heightPx: height }
}

const centi = (v: number): number => Math.round(v * 100)

/** Les sommets (centièmes) et la face d'un rectangle de sol. */
function rectGeometry(cal: ReliefCalibration, r: FloorRect, first: number): { vertices: number[]; face: number[] } {
  const mpp = cal.metersPerPixel
  const xs = [r.i0 * mpp, (r.i1 + 1) * mpp]
  const ys = [cal.originY - r.j0 * mpp, cal.originY - (r.j1 + 1) * mpp]
  const z = Array.isArray(r.z) ? r.z : [r.z, r.z, r.z, r.z]
  const corners: Array<[number, number, number]> = [
    [xs[0], ys[0], z[0]],
    [xs[1], ys[0], z[1]],
    [xs[1], ys[1], z[2]],
    [xs[0], ys[1], z[3]],
  ]
  return {
    vertices: corners.flatMap(([x, y, h]) => [centi(x), centi(y), centi(h)]),
    face: [first, first + 1, first + 2, first + 3],
  }
}

/** Les options d'un JSON de relief synthétique. */
export interface RawReliefOptions {
  width: number
  height: number
  mpp?: number
  rects?: FloorRect[]
  levels?: number[]
  z0?: number
  low?: number
  high?: number
  merge?: number
  tolerance?: number
  meshShare?: number
  steps?: number[][]
  contours?: number[][]
  outline?: number[][]
  degradations?: string[]
}

/** rawRelief écrit le JSON (déjà objet) d'un relief synthétique, au format du fichier. */
export function rawRelief(o: RawReliefOptions): Record<string, unknown> {
  const cal = sceneCalibration(o.width, o.height, o.mpp)
  const vertices: number[] = []
  const faces: number[][] = []
  for (const r of o.rects ?? []) {
    const g = rectGeometry(cal, r, vertices.length / 3)
    vertices.push(...g.vertices)
    faces.push(g.face)
  }
  return {
    schemaVersion: 1,
    calibration: cal,
    height: { z0: o.z0 ?? -7, step: 1 / 256, low: o.low ?? 0, high: o.high ?? 4 },
    floors: {
      levels: o.levels ?? [],
      min: o.levels?.[0],
      max: o.levels?.[o.levels.length - 1],
      merge: o.merge ?? 0.4,
      tolerance: o.tolerance ?? 0.3,
      meshShare: o.meshShare,
      vertices,
      faces,
    },
    lines: { steps: o.steps ?? [], contours: o.contours ?? [], outline: o.outline ?? [] },
    degradations: o.degradations ?? [],
  }
}

/** reliefData décode un relief synthétique par `parseReliefData` : le chemin de production. */
export function reliefData(o: RawReliefOptions): ReliefData {
  return parseReliefData(rawRelief(o))
}

/** Ce que contient un pixel d'une image synthétique ; `null` : hauteur inconnue. */
export type SceneCell = { z: number; onMap?: boolean; mesh?: boolean } | null

/** reliefImage construit les pixels décodés d'une scène : `cell(i, j)` dit ce que chaque pixel porte. */
export function reliefImage(width: number, height: number, cell: (i: number, j: number) => SceneCell): ReliefHeights {
  const heights = new Float32Array(width * height)
  const flags = new Uint8Array(width * height)
  for (let j = 0; j < height; j++) {
    for (let i = 0; i < width; i++) {
      const c = cell(i, j)
      const p = j * width + i
      if (c === null) {
        heights[p] = NaN
        continue
      }
      heights[p] = c.z
      flags[p] = FLAG_KNOWN | (c.onMap === false ? 0 : FLAG_ON_MAP) | (c.mesh ? FLAG_MESH_FLOOR : 0)
    }
  }
  return { width, height, heights, flags }
}

/**
 * La scène de quatre blocs de dix colonnes sur dix lignes (`mpp` 0,1), avec ses traits :
 * colonnes 0-9, un sol à 0 (dessus 0) ; 10-19, un sol à 2,2 (dessus 2,2) ; 20-29, un sol à 0 sous un
 * TOIT (dessus 2,5) ; 30-39, un sol à 2,2 que le fond a rendu PLUS BAS (dessus 1,0). Un trait de
 * rupture court sur toute la largeur à la ligne 5, une courbe de niveau verticale à la colonne 15.
 */
export function fourBlocksScene(options: Partial<RawReliefOptions> = {}): { data: ReliefData; image: ReliefHeights } {
  const tops = [0, 2.2, 2.5, 1.0]
  const block = (c0: number, c1: number, z: number): FloorRect => ({ i0: c0, j0: 0, i1: c1, j1: 9, z })
  const data = reliefData({
    width: 40,
    height: 10,
    levels: [0, 2.2],
    meshShare: 0.25,
    rects: [block(0, 9, 0), block(10, 19, 2.2), block(20, 29, 0), block(30, 39, 2.2)],
    steps: [[0, 50, 390, 0]],
    contours: [[150, 20, 0, 60]],
    outline: [[0, 0, 390, 0, 0, 90, -390, 0]],
    ...options,
  })
  return { data, image: reliefImage(40, 10, (i) => ({ z: tops[Math.floor(i / 10)], mesh: i >= 30 })) }
}
