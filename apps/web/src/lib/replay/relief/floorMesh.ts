/**
 * relief/floorMesh — LES SOLS DU MAILLAGE : géométrie des faces, murs de la coupe, sol porteur.
 *
 * Les faces sont des polygones CONVEXES (`floors.faces`) ; elles se découpent en éventail depuis le
 * premier sommet. La même découpe sert partout (pile des sols, sol porteur) pour que deux lectures
 * du même point ne divergent pas.
 *
 * SOL PORTEUR (conception §8.4) : le plus haut des sols du maillage en `(x, y)` dont la hauteur est
 * au plus `z + 0,35`. Calcul direct sur les faces (boîte englobante, puis test point-dans-triangle,
 * hauteur interpolée) : 440 à 4 072 faces, quelques microsecondes, sur le fil principal, sans
 * attendre la pile du worker.
 *
 * MURS (§6.3) : les arêtes portées par UNE SEULE face ; leur hauteur est la moyenne de leurs deux
 * sommets. Ils se tracent sous une coupe, dans la bande de l'étage.
 */
import { FOLLOW_REACH_U } from './constants'
import { LineSetBuilder, worldToPixel } from './lineSet'
import type { FloorMesh, LineSet, ReliefCalibration } from './types'

/** Tolérance barycentrique : un point sur l'arête partagée de deux faces est dans les deux. */
const BARY_EPS = 1e-4

/** Le nombre de faces. */
export function faceCount(mesh: FloorMesh): number {
  return mesh.faceStart.length - 1
}

/**
 * triangleHeight rend la hauteur du plan du triangle `(a, b, c)` en `(px, py)`, ou `null` si le point
 * est hors du triangle. Les coordonnées sont dans un repère quelconque (monde ou pixels) : le test
 * est barycentrique, donc sans échelle.
 */
export function triangleHeight(
  a: readonly [number, number, number],
  b: readonly [number, number, number],
  c: readonly [number, number, number],
  px: number,
  py: number,
): number | null {
  const den = (b[1] - c[1]) * (a[0] - c[0]) + (c[0] - b[0]) * (a[1] - c[1])
  if (Math.abs(den) < 1e-12) return null
  const l1 = ((b[1] - c[1]) * (px - c[0]) + (c[0] - b[0]) * (py - c[1])) / den
  const l2 = ((c[1] - a[1]) * (px - c[0]) + (a[0] - c[0]) * (py - c[1])) / den
  const l3 = 1 - l1 - l2
  if (l1 < -BARY_EPS || l2 < -BARY_EPS || l3 < -BARY_EPS) return null
  return l1 * a[2] + l2 * b[2] + l3 * c[2]
}

/** Le sommet `v` du maillage. */
export function vertexAt(mesh: FloorMesh, v: number): [number, number, number] {
  const o = 3 * v
  return [mesh.vertices[o], mesh.vertices[o + 1], mesh.vertices[o + 2]]
}

/** floorHeightAt rend la hauteur de la face `f` en `(x, y)` monde, ou `null` hors de la face. */
export function floorHeightAt(mesh: FloorMesh, f: number, x: number, y: number): number | null {
  const from = mesh.faceStart[f]
  const to = mesh.faceStart[f + 1]
  const first = vertexAt(mesh, mesh.faceIndices[from])
  for (let t = from + 1; t + 1 < to; t++) {
    const h = triangleHeight(first, vertexAt(mesh, mesh.faceIndices[t]), vertexAt(mesh, mesh.faceIndices[t + 1]), x, y)
    if (h !== null) return h
  }
  return null
}

/**
 * bearingFloor rend le sol porteur en `(x, y, z)` : la plus haute hauteur de face en ce point qui ne
 * dépasse pas `z + reach` ; `null` hors maillage ou quand aucun sol n'est à portée.
 */
export function bearingFloor(
  mesh: FloorMesh,
  x: number,
  y: number,
  z: number,
  reach: number = FOLLOW_REACH_U,
): number | null {
  const limit = z + reach
  let best: number | null = null
  const n = faceCount(mesh)
  for (let f = 0; f < n; f++) {
    const o = 4 * f
    if (x < mesh.bounds[o] || x > mesh.bounds[o + 2] || y < mesh.bounds[o + 1] || y > mesh.bounds[o + 3]) continue
    const h = floorHeightAt(mesh, f, x, y)
    if (h !== null && h <= limit && (best === null || h > best)) best = h
  }
  return best
}

/** Les murs du maillage, en pixels du fond. */
export interface MeshWalls {
  /** `ax, ay, bx, by` à la suite, en pixels. */
  segments: Float32Array
  /** Hauteur moyenne des deux sommets de chaque mur, en unités. */
  heights: Float32Array
}

function vertexKey(mesh: FloorMesh, v: number): string {
  const o = 3 * v
  return `${mesh.vertices[o]},${mesh.vertices[o + 1]},${mesh.vertices[o + 2]}`
}

/**
 * meshWalls extrait les arêtes portées par une seule face. Deux sommets de mêmes coordonnées sont le
 * même point (le fichier peut dupliquer un sommet) : la clé de l'arête est celle des coordonnées.
 */
export function meshWalls(mesh: FloorMesh, calibration: ReliefCalibration): MeshWalls {
  const counts = new Map<string, { n: number; a: number; b: number }>()
  const n = faceCount(mesh)
  for (let f = 0; f < n; f++) {
    const from = mesh.faceStart[f]
    const to = mesh.faceStart[f + 1]
    for (let k = from; k < to; k++) {
      const a = mesh.faceIndices[k]
      const b = mesh.faceIndices[k + 1 < to ? k + 1 : from]
      const ka = vertexKey(mesh, a)
      const kb = vertexKey(mesh, b)
      const key = ka < kb ? `${ka}|${kb}` : `${kb}|${ka}`
      const known = counts.get(key)
      if (known) known.n++
      else counts.set(key, { n: 1, a, b })
    }
  }
  const segments: number[] = []
  const heights: number[] = []
  for (const e of counts.values()) {
    if (e.n !== 1) continue
    const [ax, ay, az] = vertexAt(mesh, e.a)
    const [bx, by, bz] = vertexAt(mesh, e.b)
    const pa = worldToPixel(calibration, ax, ay)
    const pb = worldToPixel(calibration, bx, by)
    segments.push(pa.x, pa.y, pb.x, pb.y)
    heights.push((az + bz) / 2)
  }
  return { segments: Float32Array.from(segments), heights: Float32Array.from(heights) }
}

/** wallsInBand rend les murs dont la hauteur tombe dans `[z − band ; z + band]`, en polylignes de deux sommets. */
export function wallsInBand(walls: MeshWalls, z: number, band: number): LineSet {
  const builder = new LineSetBuilder()
  for (let w = 0; w < walls.heights.length; w++) {
    const h = walls.heights[w]
    if (h < z - band || h > z + band) continue
    const o = 4 * w
    builder.add(walls.segments[o], walls.segments[o + 1])
    builder.add(walls.segments[o + 2], walls.segments[o + 3])
    builder.end()
  }
  return builder.build()
}
