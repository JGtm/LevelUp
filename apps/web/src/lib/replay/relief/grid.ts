/**
 * relief/grid — DEUX OPERATIONS DE GRILLE en temps constant par pixel : dilatation carrée et
 * moyenne sur un carré. Sommes glissantes, lignes puis colonnes : le coût ne dépend pas du rayon.
 */

/**
 * dilateBox rend 1 là où l'un des `(2r + 1)²` voisins vaut 1 (masque binaire, ligne par ligne).
 * `r <= 0` rend une copie.
 */
export function dilateBox(mask: Uint8Array, width: number, height: number, r: number): Uint8Array {
  if (r <= 0) return mask.slice()
  const rows = new Uint8Array(width * height)
  for (let j = 0; j < height; j++) {
    const row = j * width
    let c = 0
    for (let i = -r; i < width; i++) {
      const add = i + r
      const drop = i - r - 1
      if (add < width && mask[row + add]) c++
      if (drop >= 0 && mask[row + drop]) c--
      if (i >= 0) rows[row + i] = c > 0 ? 1 : 0
    }
  }
  const out = new Uint8Array(width * height)
  const cnt = new Int32Array(width)
  for (let j = -r; j < height; j++) {
    const add = j + r
    const drop = j - r - 1
    if (add < height) for (let i = 0, p = add * width; i < width; i++, p++) cnt[i] += rows[p]
    if (drop >= 0) for (let i = 0, p = drop * width; i < width; i++, p++) cnt[i] -= rows[p]
    if (j >= 0) for (let i = 0, p = j * width; i < width; i++, p++) out[p] = cnt[i] > 0 ? 1 : 0
  }
  return out
}

/**
 * boxMean rend la moyenne de `values` sur le carré `(2r + 1)²` autour de chaque pixel, calculée sur
 * les SEULS pixels où `valid` est vrai. Un pixel dont aucun voisin n'est valide garde sa valeur.
 */
export function boxMean(
  values: Float32Array,
  valid: (p: number) => boolean,
  width: number,
  height: number,
  r: number,
): Float32Array {
  const sums = new Float32Array(width * height)
  const counts = new Uint8Array(width * height)
  for (let j = 0; j < height; j++) {
    const row = j * width
    let s = 0
    let c = 0
    for (let i = -r; i < width; i++) {
      const add = i + r
      const drop = i - r - 1
      if (add < width && valid(row + add)) {
        s += values[row + add]
        c++
      }
      if (drop >= 0 && valid(row + drop)) {
        s -= values[row + drop]
        c--
      }
      if (i >= 0) {
        sums[row + i] = s
        counts[row + i] = c
      }
    }
  }
  const out = new Float32Array(width * height)
  for (let i = 0; i < width; i++) {
    let s = 0
    let c = 0
    for (let j = -r; j < height; j++) {
      const add = j + r
      const drop = j - r - 1
      if (add < height) {
        s += sums[add * width + i]
        c += counts[add * width + i]
      }
      if (drop >= 0) {
        s -= sums[drop * width + i]
        c -= counts[drop * width + i]
      }
      if (j >= 0) out[j * width + i] = c > 0 ? s / c : values[j * width + i]
    }
  }
  return out
}
