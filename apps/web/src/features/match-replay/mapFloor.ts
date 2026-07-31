/**
 * mapFloor.ts — LE SOL DE LA CARTE, RECONSTRUIT.
 *
 * CE QUE CE FICHIER REMPLACE, ET POURQUOI. Le fond de carte était un empilement de 10 223
 * rectangles translucides : chaque emprise du BSP peinte l'une sur l'autre. Le résultat n'est
 * pas une carte — c'est la superposition de toutes les boîtes du niveau, sols et poutres
 * confondus, où l'œil ne distingue ni un couloir ni une plateforme.
 *
 * Ce qu'on peint à la place est une GRANDEUR, pas une composition : par cellule de 25 cm,
 * l'altitude de la face supérieure la plus haute située sous le plafond jouable. C'est
 * exactement la surface sur laquelle un joueur se tient, et c'est celle que le témoin du
 * décodeur a validée — sur les positions à vitesse verticale quasi nulle, l'écart entre le
 * joueur et la surface sous lui a une médiane de 8 mm et tombe sous 5 cm dans 80,6 % des cas
 * (11,9 % attendus au hasard). Le fond de carte est donc la même donnée que celle qui porte
 * les trajectoires, pas un décor posé à côté.
 *
 * DEUX EXCLUSIONS, les mêmes qu'au contrôle du décodeur :
 *   - ce qui passe au-dessus de la tête des joueurs (toits, poutres) — sinon le toit masque
 *     la pièce qu'il couvre ;
 *   - les dalles englobantes de plus de 4 000 m², qui ne sont pas du sol mais la boîte du
 *     niveau entier.
 * Et un plancher : sous 1 m², c'est du mobilier ou un garde-corps, pas une surface de jeu.
 *
 * La partie PURE (construction de la trame) vit ici sans canvas, donc elle se teste. Le rendu
 * n'en est que la mise en couleur.
 */
import type { ReplayBounds } from '@/lib/api/types'

import type { ReplaySurfaceReady } from './replayNormalize'

/** Côté d'une cellule de la trame, en mètres. */
export const FLOOR_CELL_M = 0.25

/**
 * Marge au-dessus de l'altitude maximale ATTEINTE par un joueur : au-delà, une surface est
 * un plafond, pas un sol. Elle est relative aux positions décodées, jamais à une constante de
 * carte — une carte plus haute décale les deux ensemble.
 */
const CEILING_MARGIN_M = 1

/** Au-delà, l'emprise est la boîte du niveau, pas une surface de jeu. */
const SLAB_MAX_AREA_M2 = 4000

/** En deçà, c'est du mobilier ou un garde-corps. */
const MIN_AREA_M2 = 1

/**
 * Écart d'altitude au-delà duquel deux cellules voisines ne sont plus le même sol : une
 * marche, un rebord de plateforme, un mur. C'est ce seuil qui fait apparaître les arêtes.
 */
const FLOOR_EDGE_STEP_M = 0.45

/**
 * Centiles d'étalonnage de la teinte. POURQUOI PAS LES BORNES BRUTES : les paliers de la carte
 * tiennent dans une bande étroite (les sols de Cliffhanger vont de −2,5 à 0) alors que les
 * positions couvrent −4,2 à +7,1 — un joueur saute et tombe bien au-delà de ses sols.
 * Étalonner sur les bornes donnerait un aplat uniforme où aucun étage ne se distingue.
 */
const Z_PCT_LOW = 0.02
const Z_PCT_HIGH = 0.98

/**
 * FloorGrid est la trame d'altitudes reconstruite. `topZ` porte NaN là où aucune surface de
 * jeu n'a été trouvée : c'est le VIDE, et il doit le rester — le peindre reviendrait à
 * inventer du sol là où la carte n'en a pas.
 */
export interface FloorGrid {
  cell: number
  nx: number
  ny: number
  minX: number
  minY: number
  topZ: Float32Array
  /** Bornes de teinte (centiles des altitudes reconstruites), pas les bornes de la carte. */
  zLo: number
  zHi: number
  /** Nombre de cellules portant un sol — de quoi constater qu'une carte est exploitable. */
  filled: number
}

/**
 * buildFloorGrid rasterise les emprises du BSP en altitudes de sol sur l'étendue `bounds`.
 *
 * L'ÉTENDUE EST CELLE DES JOUEURS, pas celle de la structure. La structure d'une carte couvre
 * ±250 m (skybox, décor lointain) quand la zone parcourue en fait 50 : rasteriser la seconde
 * sur la première donnerait une trame 25 fois trop grande pour peindre les mêmes pixels.
 */
export function buildFloorGrid(surfaces: ReplaySurfaceReady[], bounds: ReplayBounds): FloorGrid {
  const cell = FLOOR_CELL_M
  const nx = Math.ceil((bounds.maxX - bounds.minX) / cell) + 1
  const ny = Math.ceil((bounds.maxY - bounds.minY) / cell) + 1
  const topZ = new Float32Array(nx * ny).fill(NaN)
  const ceiling = (bounds.maxZ ?? 0) + CEILING_MARGIN_M

  for (const s of surfaces) {
    if (s.z > ceiling) continue
    const area = (s.x1 - s.x0) * (s.y1 - s.y0)
    if (area > SLAB_MAX_AREA_M2 || area < MIN_AREA_M2) continue
    rasterize(s, topZ, { cell, nx, ny, minX: bounds.minX, minY: bounds.minY })
  }

  const { lo, hi, filled } = tintRange(topZ, bounds)
  return { cell, nx, ny, minX: bounds.minX, minY: bounds.minY, topZ, zLo: lo, zHi: hi, filled }
}

/** Repère d'une trame, sans ses valeurs — évite de passer 5 paramètres séparés. */
interface GridFrame {
  cell: number
  nx: number
  ny: number
  minX: number
  minY: number
}

/**
 * rasterize inscrit une emprise dans la trame : chaque cellule garde l'altitude LA PLUS HAUTE,
 * puisque c'est celle sur laquelle un joueur se tiendrait.
 *
 * L'emprise ORIENTÉE (`poly`) est employée quand elle existe : la boîte alignée d'une pièce
 * tournée déborde largement de la pièce. Aucun fichier de structure figé n'en porte à ce jour
 * (0 sur 10 223 pour `ridgeline`) — le code la traite quand même, sans quoi le jour où
 * `mapstruct-build` en produira, la trame les ignorerait en silence.
 */
function rasterize(s: ReplaySurfaceReady, topZ: Float32Array, g: GridFrame): void {
  const poly = s.poly && s.poly.length > 2 ? s.poly : null
  const box = poly ? polyBox(poly) : { x0: s.x0, y0: s.y0, x1: s.x1, y1: s.y1 }
  const cx0 = cellIndex(box.x0, g.minX, g.nx, g.cell)
  const cx1 = cellIndex(box.x1, g.minX, g.nx, g.cell)
  const cy0 = cellIndex(box.y0, g.minY, g.ny, g.cell)
  const cy1 = cellIndex(box.y1, g.minY, g.ny, g.cell)
  for (let y = cy0; y <= cy1; y++) {
    for (let x = cx0; x <= cx1; x++) {
      if (poly) {
        // Le CENTRE de la cellule, jamais son coin : un test au coin décalerait toute
        // l'emprise d'une demi-cellule, soit 12,5 cm sur l'ensemble de la carte.
        const px = g.minX + (x + 0.5) * g.cell
        const py = g.minY + (y + 0.5) * g.cell
        if (!pointInPolygon(px, py, poly)) continue
      }
      const k = y * g.nx + x
      if (Number.isNaN(topZ[k]) || s.z > topZ[k]) topZ[k] = s.z
    }
  }
}

/** cellIndex borne l'index de cellule à la trame (une emprise déborde de l'étendue jouable). */
function cellIndex(v: number, lo: number, n: number, cell: number): number {
  const i = Math.floor((v - lo) / cell)
  return i < 0 ? 0 : i > n - 1 ? n - 1 : i
}

function polyBox(poly: [number, number][]): { x0: number; y0: number; x1: number; y1: number } {
  let x0 = Infinity
  let y0 = Infinity
  let x1 = -Infinity
  let y1 = -Infinity
  for (const [x, y] of poly) {
    if (x < x0) x0 = x
    if (x > x1) x1 = x
    if (y < y0) y0 = y
    if (y > y1) y1 = y
  }
  return { x0, y0, x1, y1 }
}

/** pointInPolygon — lancer de rayon, règle pair-impair. */
export function pointInPolygon(px: number, py: number, poly: [number, number][]): boolean {
  let inside = false
  for (let a = 0, b = poly.length - 1; a < poly.length; b = a++) {
    const [ax, ay] = poly[a]
    const [bx, by] = poly[b]
    if (ay > py !== by > py && px < ((bx - ax) * (py - ay)) / (by - ay) + ax) inside = !inside
  }
  return inside
}

/**
 * tintRange étalonne la teinte sur les centiles des altitudes RECONSTRUITES.
 * Une trame vide retombe sur les bornes de la carte — un fond uniforme, ce qui est la lecture
 * juste : sans sol reconstruit, il n'y a pas d'étage à distinguer.
 */
function tintRange(
  topZ: Float32Array,
  bounds: ReplayBounds,
): { lo: number; hi: number; filled: number } {
  const zs: number[] = []
  for (let i = 0; i < topZ.length; i++) {
    if (!Number.isNaN(topZ[i])) zs.push(topZ[i])
  }
  if (zs.length === 0) {
    return { lo: bounds.minZ ?? 0, hi: bounds.maxZ ?? 1, filled: 0 }
  }
  zs.sort((a, b) => a - b)
  return {
    lo: zs[Math.floor(zs.length * Z_PCT_LOW)],
    hi: zs[Math.floor(zs.length * Z_PCT_HIGH)],
    filled: zs.length,
  }
}

/** altitudeTint normalise une altitude de sol dans [0,1] sur les bornes d'étalonnage. */
export function altitudeTint(z: number, grid: FloorGrid): number {
  const span = Math.max(grid.zHi - grid.zLo, 0.001)
  const r = (z - grid.zLo) / span
  return r < 0 ? 0 : r > 1 ? 1 : r
}

/**
 * floorRun rend la longueur de la plage horizontale de MÊME altitude commençant en (x, y),
 * ou 0 si la cellule est vide.
 *
 * POURQUOI DES PLAGES ET PAS DES CELLULES : deux rectangles voisins arrondis au pixel se
 * chevauchent d'un pixel, et ce chevauchement dessine un quadrillage parasite là où il n'y a
 * qu'un sol continu. Peindre la plage d'un seul trait supprime la cause.
 */
export function floorRun(grid: FloorGrid, x: number, y: number): number {
  const z = grid.topZ[y * grid.nx + x]
  if (Number.isNaN(z)) return 0
  let end = x
  while (end + 1 < grid.nx && grid.topZ[y * grid.nx + end + 1] === z) end++
  return end - x + 1
}

/**
 * hasEdge dit si la cellule (x, y) porte une arête sur le côté demandé : une marche au-delà du
 * seuil, ou un bord de vide. Ce sont les murs, rebords et marches de la carte — ils viennent
 * des altitudes reconstruites, pas d'un contour dessiné à la main.
 */
export function hasEdge(grid: FloorGrid, x: number, y: number, side: 'right' | 'up'): boolean {
  const z = grid.topZ[y * grid.nx + x]
  if (Number.isNaN(z)) return false
  const nx = side === 'right' ? x + 1 : x
  const ny = side === 'right' ? y : y + 1
  if (nx >= grid.nx || ny >= grid.ny) return true
  const nz = grid.topZ[ny * grid.nx + nx]
  return Number.isNaN(nz) || Math.abs(nz - z) > FLOOR_EDGE_STEP_M
}
