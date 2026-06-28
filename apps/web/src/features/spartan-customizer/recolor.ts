/**
 * Recoloration des masques Spartan (Halo 5) — modèle ADDITIF.
 *
 * Les masques extraits du jeu encodent les zones coloriables dans les canaux RGBA :
 *   R = zone primaire, G = zone secondaire, B = zone tertiaire ; hors-zone = noir.
 * Rendu (par canal) : `out = primary*R + secondary*G + tertiary*B`, `alpha = max(R,G,B)`
 * — identique au shader du jeu et à la pré-validation offline. La forme (silhouette)
 * vient de l'union des canaux : un pixel noir partout devient transparent.
 *
 * Fonction PURE (zéro DOM) → testable sans canvas. Le wrapper canvas applique ça sur
 * un `ImageData` (getImageData → recolorMaskData → putImageData).
 */

export type Rgb = readonly [number, number, number]

export function hexToRgb(hex: string): Rgb {
  const h = hex.replace('#', '')
  return [
    parseInt(h.slice(0, 2), 16),
    parseInt(h.slice(2, 4), 16),
    parseInt(h.slice(4, 6), 16),
  ]
}

export interface RecolorColors {
  primary: string
  secondary: string
  tertiary: string
}

/**
 * Recolorise un buffer RGBA (ordre R,G,B,A ; longueur multiple de 4) selon le modèle
 * additif. Retourne un nouveau `Uint8ClampedArray` (le clamp [0,255] est implicite).
 */
export function recolorMaskData(
  src: Uint8ClampedArray,
  colors: RecolorColors,
): Uint8ClampedArray {
  const [pr, pg, pb] = hexToRgb(colors.primary)
  const [sr, sg, sb] = hexToRgb(colors.secondary)
  const [tr, tg, tb] = hexToRgb(colors.tertiary)
  const out = new Uint8ClampedArray(src.length)
  for (let i = 0; i < src.length; i += 4) {
    const fr = src[i] / 255
    const fg = src[i + 1] / 255
    const fb = src[i + 2] / 255
    out[i] = pr * fr + sr * fg + tr * fb
    out[i + 1] = pg * fr + sg * fg + tg * fb
    out[i + 2] = pb * fr + sb * fg + tb * fb
    out[i + 3] = Math.max(src[i], src[i + 1], src[i + 2])
  }
  return out
}
