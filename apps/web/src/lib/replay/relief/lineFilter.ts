/**
 * relief/lineFilter — LES TRAITS FILTRES PAR LA COUPE (conception §3.6, §6.3).
 *
 * Le fichier ne porte AUCUNE hauteur par trait : une chaîne n'a pas une hauteur (le long d'une même
 * chaîne la hauteur basse varie de plus de 0,5 u dans 39 % des cas). La coupe filtre donc les traits
 * par son masque `keep` (« garde », `slice.ts`), lu dans le repère de l'IMAGE, avant toute rotation.
 *
 * Chaque segment est découpé en tronçons d'au plus 0,5 u (un long segment peut traverser la bordure
 * de la garde) ; un tronçon est gardé si `keep` vaut 1 au pixel de son milieu. Pour une RUPTURE,
 * le masque est aussi interrogé à 1,5 pixel de part et d'autre du milieu, sur la normale : ce qui
 * tient SUR le sol de l'étage (pied d'un pilier, d'une caisse, d'un mur) y reste. Les tronçons gardés
 * contigus se rejoignent en une seule polyligne. Le liseré n'est jamais filtré.
 */
import { LINE_PIECE_U, LINE_SIDE_PX } from './constants'
import { LineSetBuilder } from './lineSet'
import type { LineSet } from './types'

/** La grille du masque. */
export interface KeepGrid {
  width: number
  height: number
  keep: Uint8Array
}

/** Vrai si le masque garde le pixel le plus proche de `(x, y)` (hors grille : non). */
function kept(grid: KeepGrid, x: number, y: number): boolean {
  const i = Math.round(x)
  const j = Math.round(y)
  return i >= 0 && j >= 0 && i < grid.width && j < grid.height && grid.keep[j * grid.width + i] === 1
}

/**
 * Les drapeaux « gardé » des `n` tronçons du segment `a -> b`. `sides` ajoute les deux pixels à
 * `LINE_SIDE_PX` du milieu sur la normale du segment.
 */
function pieceFlags(
  grid: KeepGrid,
  a: readonly [number, number],
  b: readonly [number, number],
  n: number,
  sides: boolean,
): Uint8Array {
  const flags = new Uint8Array(n)
  const dx = b[0] - a[0]
  const dy = b[1] - a[1]
  const len = Math.hypot(dx, dy) || 1
  const nx = (-dy / len) * LINE_SIDE_PX
  const ny = (dx / len) * LINE_SIDE_PX
  for (let t = 0; t < n; t++) {
    const mid = (t + 0.5) / n
    const mx = a[0] + dx * mid
    const my = a[1] + dy * mid
    flags[t] = kept(grid, mx, my) || (sides && (kept(grid, mx + nx, my + ny) || kept(grid, mx - nx, my - ny))) ? 1 : 0
  }
  return flags
}

/**
 * filterLinesByKeep rend les parties des polylignes que la coupe garde. `mpp` (unités par pixel)
 * fixe la longueur des tronçons ; `sides` est vrai pour les ruptures, faux pour les courbes de niveau.
 */
export function filterLinesByKeep(lines: LineSet, grid: KeepGrid, mpp: number, sides: boolean): LineSet {
  const { points, offsets } = lines
  const pieceLen = LINE_PIECE_U / mpp
  const out = new LineSetBuilder()
  for (let k = 0; k + 1 < offsets.length; k++) {
    let open = false
    for (let v = offsets[k]; v + 1 < offsets[k + 1]; v++) {
      const a: [number, number] = [points[2 * v], points[2 * v + 1]]
      const b: [number, number] = [points[2 * v + 2], points[2 * v + 3]]
      const n = Math.max(1, Math.ceil(Math.hypot(b[0] - a[0], b[1] - a[1]) / pieceLen))
      const flags = pieceFlags(grid, a, b, n, sides)
      const at = (t: number): [number, number] => [a[0] + ((b[0] - a[0]) * t) / n, a[1] + ((b[1] - a[1]) * t) / n]
      let t = 0
      while (t < n) {
        if (!flags[t]) {
          if (open) out.end()
          open = false
          t++
          continue
        }
        let u = t
        while (u < n && flags[u]) u++
        if (!open) out.add(...at(t))
        out.add(...at(u))
        open = u === n
        if (!open) out.end()
        t = u
      }
    }
    out.end()
  }
  return out.build()
}
