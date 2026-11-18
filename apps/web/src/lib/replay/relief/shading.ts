/**
 * relief/shading — LE GRIS OMBRE : la clarté suit la hauteur, l'ombrage suit la pente.
 *
 * Formule de la maquette validée (conception §6.4, D-15) : une clarté `0,36 + 0,64 × t` (`t` = la
 * hauteur ramenée à la rampe `low`..`high` du JSON), modulée par un ombrage de Lambert calculé sur
 * les hauteurs LISSEES (moyenne 5 × 5 des seuls pixels connus). Le résultat est un gris 0..255 par
 * pixel ; la teinte est appliquée ensuite (`tint.ts`), une même image grise servant toutes les teintes.
 *
 * LUMIERE : toujours depuis le HAUT A GAUCHE DE L'ECRAN, carte tournée ou non. Une carte tournée de
 * 180° serait éclairée par le bas à droite et le relief s'inverserait à l'œil (les bosses se lisent
 * en creux). Comme l'ombrage se calcule dans le navigateur, la direction de la lumière est exprimée
 * DANS LE REPERE DE L'IMAGE pour `q` quarts de tour horaires de la carte :
 * `(−1 ; −1)`, `(−1 ; +1)`, `(+1 ; +1)`, `(+1 ; −1)` pour `q` = 0, 1, 2, 3 (x vers la droite, y vers
 * le bas). Un ombrage par orientation, mis en cache par l'appelant (`engine.ts`).
 *
 * La rotation de la carte à l'écran n'est PAS appliquée ici (`orientation.ts` la définit seul) : ce
 * module ne fait que choisir la direction de la lumière dans le repère de l'image.
 */
import { FLAG_KNOWN } from './constants'
import { boxMean } from './grid'
import type { QuarterTurns } from '../orientation'
import type { ReliefHeightMeta, ReliefHeights } from './types'

/** Rayon du lissage (carré 5 × 5). */
const SMOOTH_RADIUS = 2
/** Hauteur de la lumière au-dessus du plan, relative à son déplacement horizontal. */
const LIGHT_ELEVATION = 1.4
/** Clarté du bas de la rampe et part de la rampe dans la clarté. */
const GREY_BASE = 0.36
const GREY_RANGE = 0.64
/** Part de la clarté qui ne dépend pas de l'ombrage, et part qui en dépend. */
const SHADE_BASE = 0.62
const SHADE_RANGE = 0.38
/** Bornes de l'ombrage relatif au plat. */
const SHADE_MIN = 0.35
const SHADE_MAX = 1.3
/** Amplitude minimale de la rampe (unités) : une carte plate ne divise pas par zéro. */
const MIN_SPAN_U = 1

/** La direction de la lumière dans le repère de l'image (x vers la droite, y vers le bas). */
export interface LightDirection {
  x: number
  y: number
}

const LIGHT_BY_TURNS: Record<QuarterTurns, LightDirection> = {
  0: { x: -1, y: -1 },
  1: { x: -1, y: 1 },
  2: { x: 1, y: 1 },
  3: { x: 1, y: -1 },
}

/** lightForTurns rend la direction de la lumière dans le repère de l'image pour `turns` quarts de tour horaires. */
export function lightForTurns(turns: QuarterTurns): LightDirection {
  return LIGHT_BY_TURNS[turns]
}

/** La clarté d'une hauteur sur la rampe, 0..1 (sans ombrage) : celle d'un sol plat. */
export function heightGrey(z: number, meta: Pick<ReliefHeightMeta, 'low' | 'high'>): number {
  const span = Math.max(MIN_SPAN_U, meta.high - meta.low)
  const t = Math.min(1, Math.max(0, (z - meta.low) / span))
  return GREY_BASE + GREY_RANGE * t
}

/** Le gris (octet) d'une hauteur sur un sol plat, comme `shadeRelief` le poserait. */
export function heightGreyByte(z: number, meta: Pick<ReliefHeightMeta, 'low' | 'high'>): number {
  return Math.round(Math.min(1, heightGrey(z, meta)) * 255)
}

/**
 * shadeRelief calcule le gris ombré de chaque pixel CONNU (0 ailleurs) pour une orientation.
 * `mpp` est l'échelle du fond en unités par pixel : elle donne la pente en unités par unité.
 */
export function shadeRelief(
  image: ReliefHeights,
  meta: Pick<ReliefHeightMeta, 'low' | 'high'>,
  mpp: number,
  turns: QuarterTurns,
): Uint8Array {
  const { width, height, heights, flags } = image
  const known = (p: number): boolean => (flags[p] & FLAG_KNOWN) !== 0
  const smooth = boxMean(heights, known, width, height, SMOOTH_RADIUS)
  const light = lightForTurns(turns)
  const norm = Math.sqrt(light.x * light.x + light.y * light.y + LIGHT_ELEVATION * LIGHT_ELEVATION)
  const flat = LIGHT_ELEVATION / norm
  const grey = new Uint8Array(width * height)
  const k = FLAG_KNOWN
  for (let j = 0; j < height; j++) {
    for (let i = 0; i < width; i++) {
      const p = j * width + i
      if ((flags[p] & k) === 0) continue
      const here = smooth[p]
      const xl = i > 0 && (flags[p - 1] & k) !== 0 ? smooth[p - 1] : here
      const xr = i < width - 1 && (flags[p + 1] & k) !== 0 ? smooth[p + 1] : here
      const yu = j > 0 && (flags[p - width] & k) !== 0 ? smooth[p - width] : here
      const yd = j < height - 1 && (flags[p + width] & k) !== 0 ? smooth[p + width] : here
      const gx = (xr - xl) / (2 * mpp)
      const gy = (yd - yu) / (2 * mpp)
      const nn = Math.sqrt(gx * gx + gy * gy + 1)
      const lambert = Math.max(0, (-gx * light.x - gy * light.y + LIGHT_ELEVATION) / (nn * norm))
      const shade = Math.min(SHADE_MAX, Math.max(SHADE_MIN, lambert / flat))
      const value = Math.min(1, heightGrey(heights[p], meta) * (SHADE_BASE + SHADE_RANGE * shade))
      grey[p] = Math.round(value * 255)
    }
  }
  return grey
}
