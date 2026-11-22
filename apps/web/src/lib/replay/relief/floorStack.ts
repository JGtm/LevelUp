/**
 * relief/floorStack — LA PILE DES SOLS : les sols du maillage, empilés par pixel.
 *
 * Les faces de `floors` sont rastérisées dans la grille du fond (centre du pixel = centre de la
 * cellule, hauteur interpolée sur la face) ; un pixel porte jusqu'à `STACK_LAYERS` sols, rangés du
 * plus bas au plus haut. Deux sols à moins de `merge` l'un de l'autre n'en font qu'un (le premier
 * rencontré) ; un cinquième sol est perdu.
 *
 * POURQUOI UNE PILE (conception §2) : 8 à 12 % de la carte porte deux sols ou plus l'un sur l'autre ;
 * un bit « toit » par pixel ne dirait pas de quel sol il est le toit. La coupe compare donc le dessus
 * à chaque sol de la pile (`slice.ts`).
 *
 * MEMOIRE : hauteurs en `Uint16` au pas du fichier, `STACK_LAYERS` couches (23 Mo pour 2,84 millions
 * de pixels), jamais en `Float32` (45 Mo). Le zéro du code est `z0 − STACK_HEADROOM_BELOW_U` : un sol
 * peut passer sous le dessus le plus bas (un tunnel sous un terrain). La pile se construit à la
 * PREMIERE coupe seulement ; « Tout » n'en a pas besoin.
 */
import { STACK_HEADROOM_BELOW_U, STACK_LAYERS } from './constants'
import { faceCount, triangleHeight, vertexAt } from './floorMesh'
import { worldToPixel } from './lineSet'
import type { ReliefData } from './types'

/** Les sols empilés de chaque pixel. */
export interface FloorStack {
  width: number
  height: number
  /** Hauteur du code 0, en unités. */
  ref: number
  /** Hauteur d'un pas de code, en unités. */
  step: number
  /** `STACK_LAYERS` codes par pixel, du plus bas au plus haut ; seuls les `count[p]` premiers comptent. */
  codes: Uint16Array
  count: Uint8Array
}

/** stackHeight rend la hauteur du sol `k` (du bas) du pixel `p`, en unités. */
export function stackHeight(stack: FloorStack, p: number, k: number): number {
  return stack.ref + stack.codes[p * STACK_LAYERS + k] * stack.step
}

/** stackTop rend la hauteur du plus haut sol du pixel `p`, ou `null` sans sol. */
export function stackTop(stack: FloorStack, p: number): number | null {
  const c = stack.count[p]
  return c === 0 ? null : stackHeight(stack, p, c - 1)
}

/** Ajoute le sol `z` au pixel `p` : fusionné s'il est à moins de `merge` d'un sol déjà posé. */
function putFloor(stack: FloorStack, p: number, z: number, merge: number): void {
  const base = p * STACK_LAYERS
  const have = stack.count[p]
  for (let k = 0; k < have; k++) {
    if (Math.abs(stack.ref + stack.codes[base + k] * stack.step - z) < merge) return
  }
  if (have >= STACK_LAYERS) return
  const code = Math.round((z - stack.ref) / stack.step)
  stack.codes[base + have] = Math.min(65535, Math.max(0, code))
  stack.count[p] = have + 1
}

/** Range les sols d'un pixel du plus bas au plus haut (au plus quatre éléments : tri par insertion). */
function sortColumn(codes: Uint16Array, base: number, count: number): void {
  for (let i = 1; i < count; i++) {
    const v = codes[base + i]
    let j = i - 1
    while (j >= 0 && codes[base + j] > v) {
      codes[base + j + 1] = codes[base + j]
      j--
    }
    codes[base + j + 1] = v
  }
}

/** Rastérise une face convexe, en éventail, dans la pile. */
function rasterFace(stack: FloorStack, px: Float64Array, py: Float64Array, pz: Float64Array, merge: number): void {
  const { width, height } = stack
  const a: [number, number, number] = [px[0], py[0], pz[0]]
  for (let t = 1; t + 1 < px.length; t++) {
    const b: [number, number, number] = [px[t], py[t], pz[t]]
    const c: [number, number, number] = [px[t + 1], py[t + 1], pz[t + 1]]
    const i0 = Math.max(0, Math.floor(Math.min(a[0], b[0], c[0])))
    const i1 = Math.min(width - 1, Math.ceil(Math.max(a[0], b[0], c[0])))
    const j0 = Math.max(0, Math.floor(Math.min(a[1], b[1], c[1])))
    const j1 = Math.min(height - 1, Math.ceil(Math.max(a[1], b[1], c[1])))
    for (let j = j0; j <= j1; j++) {
      for (let i = i0; i <= i1; i++) {
        const z = triangleHeight(a, b, c, i, j)
        if (z !== null) putFloor(stack, j * width + i, z, merge)
      }
    }
  }
}

/**
 * buildFloorStack empile les sols du maillage dans la grille du fond. Les seuils (`merge`) et le
 * calage viennent du JSON : le web reconstruit la pile avec ceux de la cuisson.
 */
export function buildFloorStack(data: ReliefData): FloorStack {
  const { calibration, height, floors } = data
  const width = calibration.widthPx
  const rows = calibration.heightPx
  const n = width * rows
  const stack: FloorStack = {
    width,
    height: rows,
    ref: height.z0 - STACK_HEADROOM_BELOW_U,
    step: height.step,
    codes: new Uint16Array(n * STACK_LAYERS),
    count: new Uint8Array(n),
  }
  const mesh = floors.mesh
  for (let f = 0; f < faceCount(mesh); f++) {
    const from = mesh.faceStart[f]
    const size = mesh.faceStart[f + 1] - from
    const px = new Float64Array(size)
    const py = new Float64Array(size)
    const pz = new Float64Array(size)
    for (let k = 0; k < size; k++) {
      const [x, y, z] = vertexAt(mesh, mesh.faceIndices[from + k])
      const p = worldToPixel(calibration, x, y)
      px[k] = p.x
      py[k] = p.y
      pz[k] = z
    }
    rasterFace(stack, px, py, pz, floors.merge)
  }
  for (let p = 0; p < n; p++) {
    if (stack.count[p] > 1) sortColumn(stack.codes, p * STACK_LAYERS, stack.count[p])
  }
  return stack
}
