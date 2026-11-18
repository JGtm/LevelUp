/**
 * relief/slice — LA COUPE PAR ETAGE : fonction pure, à la hauteur `z`.
 *
 * Une coupe dit, pour chaque pixel, quelle part de lui est « dans l'étage » : le dessus de la carte
 * s'y montre en plein, le dessous reste en pâle, le dessus qui domine l'étage est presque effacé, et
 * un sol de l'étage CACHE (sous un toit, ou que le fond a rendu plus bas) est révélé à plat. « Tout »
 * (`cut = null`) est l'absence de coupe.
 *
 * REGLE (conception §6.3, avec la correction du §4.4 : un sol de la coupe se montre toujours) :
 *
 *   poids(v) = 1 dans [z − BANDE ; z + BANDE], décroît linéairement jusqu'à 0 sur FONDU au-delà
 *   pour chaque pixel : meilleur = le sol de la pile de plus grand poids (aucun si tous à 0)
 *   solDansCoupe = poids(meilleur) > 0,5 ; pres = solDansCoupe dilaté de PRES
 *   pour chaque pixel de dessus d :
 *     si meilleur existe et (d > meilleur + TOIT  ou  d < meilleur − TOLERANCE) :
 *         dedans = poids(meilleur) ; le gris se mélange vers le gris de la hauteur de meilleur ; garde = 0
 *     sinon :
 *         dessusEstUnSol = un sol de la pile est à moins de TOLERANCE de d
 *         dedans = (dessusEstUnSol ou non pres ou d < z) ? poids(d) : 1 ; garde = dedans > 0,5
 *     alpha = dedans + (1 − dedans) × (d > z ? OPACITE_DESSUS : OPACITE_DESSOUS)
 *
 * SORTIES, un octet par pixel :
 *   - `alpha` : l'opacité de la coupe sur TOUT le cadre — le masque du fond Rendu (D-2) ; un pixel sans
 *     hauteur (A = 0) prend l'opacité du dessous dès qu'une coupe est active, sinon il resterait seul
 *     en clair. Le fond Relief la lit sur les seuls pixels de la carte (`tint.ts`) ;
 *   - `grey` : le gris, modifié là où un sol caché est révélé ;
 *   - `keep` (« garde ») : le masque qui filtre les traits (§3.6) — pas de hauteur par trait ;
 *   - `planAlpha` (à la demande) : le masque du fond Plan (D-2). L'image Plan montre le sol le plus HAUT
 *     de chaque pixel : son masque se calcule sur ce sol, pas sur le dessus du relief. Sans sol, l'alpha.
 *
 * Sans pile (carte sans maillage) il n'y a pas de coupe : le résultat est celui de « Tout ».
 */
import {
  CUT_ALPHA_ABOVE,
  CUT_ALPHA_BELOW,
  CUT_BAND_U,
  CUT_FADE_U,
  CUT_MIN_WEIGHT,
  CUT_NEAR_U,
  CUT_ROOF_U,
  FLAG_KNOWN,
  FLAG_ON_MAP,
} from './constants'
import { stackHeight, stackTop, type FloorStack } from './floorStack'
import { dilateBox } from './grid'
import { heightGrey } from './shading'
import type { ReliefHeightMeta, ReliefHeights } from './types'

/** Ce que la coupe lit. */
export interface SliceInput {
  image: ReliefHeights
  /** Le gris ombré de l'orientation courante (`shadeRelief`). */
  grey: Uint8Array
  /** La pile des sols ; `null` sans maillage. */
  stack: FloorStack | null
  /** Unités du jeu par pixel. */
  mpp: number
  /** Bornes de la rampe de clarté. */
  ramp: Pick<ReliefHeightMeta, 'low' | 'high'>
  /** Ecart toléré entre un dessus et son sol (JSON). */
  tolerance: number
}

/** Le résultat d'une coupe : un octet par pixel dans chaque tableau. */
export interface ReliefSlice {
  /** La hauteur de la coupe, `null` pour « Tout ». */
  cut: number | null
  grey: Uint8Array
  alpha: Uint8Array
  keep: Uint8Array
  planAlpha: Uint8Array | null
}

/** Options d'une coupe. */
export interface SliceOptions {
  /** Calculer aussi le masque du fond Plan. */
  plan?: boolean
}

/** Le poids d'une hauteur dans la coupe `z` : 1 dans la bande, un fondu linéaire au-delà, 0 ensuite. */
export function cutWeight(v: number, z: number): number {
  const lo = z - CUT_BAND_U
  const hi = z + CUT_BAND_U
  if (v < lo) return Math.max(0, 1 - (lo - v) / CUT_FADE_U)
  if (v > hi) return Math.max(0, 1 - (v - hi) / CUT_FADE_U)
  return 1
}

const toByte = (a: number): number => Math.round(Math.min(1, Math.max(0, a)) * 255)

/** L'opacité d'un pixel dont une part `inside` est dans la coupe ; le reste est pâle (sous) ou presque effacé (sur). */
function sliceAlpha(inside: number, top: number, z: number): number {
  return toByte(inside + (1 - inside) * (top > z ? CUT_ALPHA_ABOVE : CUT_ALPHA_BELOW))
}

/**
 * Le résultat de « Tout » : rien n'est masqué, la garde est la carte. Le gris est celui de l'ombrage,
 * PARTAGE (pas de copie) : l'appelant qui le transfère le copie d'abord.
 */
function wholeSlice(input: SliceInput): ReliefSlice {
  const n = input.image.flags.length
  const keep = new Uint8Array(n)
  for (let p = 0; p < n; p++) keep[p] = input.image.flags[p] & FLAG_ON_MAP
  return { cut: null, grey: input.grey, alpha: new Uint8Array(n).fill(255), keep, planAlpha: null }
}

/** Pour chaque pixel : l'indice du sol de plus grand poids (−1 si tous sont à 0) et si ce sol est dans la coupe. */
function bestFloors(stack: FloorStack, z: number): { bestK: Int8Array; inCut: Uint8Array } {
  const n = stack.count.length
  const bestK = new Int8Array(n).fill(-1)
  const inCut = new Uint8Array(n)
  for (let p = 0; p < n; p++) {
    let bw = 0
    for (let k = 0; k < stack.count[p]; k++) {
      const w = cutWeight(stackHeight(stack, p, k), z)
      if (w > bw) {
        bw = w
        bestK[p] = k
      }
    }
    inCut[p] = bw > CUT_MIN_WEIGHT ? 1 : 0
  }
  return { bestK, inCut }
}

/** Vrai si un sol de la pile du pixel est à moins de `tolerance` du dessus : le dessus est alors un sol. */
function topIsFloor(stack: FloorStack, p: number, top: number, tolerance: number): boolean {
  for (let k = 0; k < stack.count[p]; k++) {
    if (Math.abs(stackHeight(stack, p, k) - top) < tolerance) return true
  }
  return false
}

/** computeSlice calcule la coupe de hauteur `cut` (`null` : « Tout »). Fonction pure : aucune entrée n'est modifiée. */
export function computeSlice(input: SliceInput, cut: number | null, options: SliceOptions = {}): ReliefSlice {
  const { stack } = input
  if (cut === null || stack === null) return wholeSlice(input)
  const { image, mpp, tolerance } = input
  const { heights, flags } = image
  const n = flags.length
  const { bestK, inCut } = bestFloors(stack, cut)
  const near = dilateBox(inCut, image.width, image.height, Math.max(1, Math.round(CUT_NEAR_U / mpp)))
  const grey = input.grey.slice()
  const alpha = new Uint8Array(n).fill(toByte(CUT_ALPHA_BELOW))
  const keep = new Uint8Array(n)
  for (let p = 0; p < n; p++) {
    if ((flags[p] & FLAG_KNOWN) === 0) continue
    const top = heights[p]
    const k = bestK[p]
    const floorZ = k >= 0 ? stackHeight(stack, p, k) : NaN
    let inside: number
    if (k >= 0 && (top > floorZ + CUT_ROOF_U || top < floorZ - tolerance)) {
      // Un sol de la coupe que le dessus recouvre ou que le fond a rendu plus bas : montré à plat.
      inside = cutWeight(floorZ, cut)
      grey[p] = Math.round(grey[p] + (toByte(heightGrey(floorZ, input.ramp)) - grey[p]) * inside)
    } else {
      const flat = topIsFloor(stack, p, top, tolerance) || !near[p] || top < cut
      inside = flat ? cutWeight(top, cut) : 1
      if (inside > CUT_MIN_WEIGHT) keep[p] = flags[p] & FLAG_ON_MAP
    }
    alpha[p] = sliceAlpha(inside, top, cut)
  }
  const planAlpha = options.plan ? planMask(stack, alpha, cut) : null
  return { cut, grey, alpha, keep, planAlpha }
}

/** Le masque du fond Plan : le poids du sol le plus haut de la pile ; sans sol, l'alpha du relief. */
function planMask(stack: FloorStack, alpha: Uint8Array, z: number): Uint8Array {
  const plan = alpha.slice()
  for (let p = 0; p < plan.length; p++) {
    const top = stackTop(stack, p)
    if (top !== null) plan[p] = sliceAlpha(cutWeight(top, z), top, z)
  }
  return plan
}
