/**
 * relief/lineSet — LES POLYLIGNES DU RELIEF, À PLAT, et le passage pixels -> monde -> écran.
 *
 * Un `LineSet` (types.ts) range des milliers de polylignes dans deux tableaux typés : rien à
 * allouer par trait, et la structure traverse un `postMessage` par transfert. Les coordonnées sont
 * dans le repère des PIXELS du fond (le centre du pixel est l'entier, conception §1.2).
 *
 * Le dessin passe par `traceLineSet` : le même parcours trace un `Path2D` ou un contexte 2D, avec la
 * projection PROPRE de la vue (rejeu, Tactique, Occupation ; la rotation en quarts de tour est celle
 * de la projection, jamais réécrite ici — `orientation.guard.test.ts`). L'épaisseur est en pixels
 * d'écran, pas en unités : c'est l'appelant qui la pose.
 */
import type { LineSet, PointXY, ReliefCalibration } from './types'

/** Le monde d'un point en coordonnées de pixels : `originX + (px + 0,5) × mpp`, `originY − (py + 0,5) × mpp`. */
export function pixelToWorld(calibration: ReliefCalibration, px: number, py: number): PointXY {
  const mpp = calibration.metersPerPixel
  return { x: calibration.originX + (px + 0.5) * mpp, y: calibration.originY - (py + 0.5) * mpp }
}

/** Le pixel (fractionnaire) d'un point monde : l'inverse de `pixelToWorld`. */
export function worldToPixel(calibration: ReliefCalibration, x: number, y: number): PointXY {
  const mpp = calibration.metersPerPixel
  return { x: (x - calibration.originX) / mpp - 0.5, y: (calibration.originY - y) / mpp - 0.5 }
}

/** Un ensemble vide. */
export function emptyLineSet(): LineSet {
  return { points: new Float32Array(0), offsets: new Uint32Array(1) }
}

/** Le nombre de polylignes d'un ensemble. */
export function lineCount(lines: LineSet): number {
  return lines.offsets.length - 1
}

/** Construit un `LineSet` polyligne par polyligne, sans connaître le total d'avance. */
export class LineSetBuilder {
  private pts: number[] = []
  private offs: number[] = [0]
  private open = 0

  /** Ajoute un sommet à la polyligne en cours. */
  add(x: number, y: number): void {
    this.pts.push(x, y)
    this.open++
  }

  /** Clôt la polyligne en cours ; une polyligne vide ou d'un seul sommet est écartée. */
  end(): void {
    if (this.open >= 2) {
      this.offs.push(this.pts.length / 2)
    } else {
      this.pts.length -= this.open * 2
    }
    this.open = 0
  }

  build(): LineSet {
    this.end()
    return { points: Float32Array.from(this.pts), offsets: Uint32Array.from(this.offs) }
  }
}

/** Ce que `traceLineSet` sait tracer : `Path2D` comme `CanvasRenderingContext2D`. */
export interface PathTarget {
  moveTo(x: number, y: number): void
  lineTo(x: number, y: number): void
  closePath(): void
}

/**
 * traceLineSet ajoute chaque polyligne au chemin : `toScreen` reçoit un point en pixels du fond et
 * rend le point d'écran (projection de la vue). `closed` ferme chaque polyligne (liseré).
 * Les polylignes sont rendues dans un seul chemin : un `stroke` pour tout l'ensemble.
 */
export function traceLineSet(
  path: PathTarget,
  lines: LineSet,
  toScreen: (px: number, py: number) => PointXY,
  closed = false,
): void {
  const { points, offsets } = lines
  for (let k = 0; k + 1 < offsets.length; k++) {
    const from = offsets[k]
    const to = offsets[k + 1]
    for (let v = from; v < to; v++) {
      const s = toScreen(points[2 * v], points[2 * v + 1])
      if (v === from) path.moveTo(s.x, s.y)
      else path.lineTo(s.x, s.y)
    }
    if (closed) path.closePath()
  }
}

/** Les buffers d'un ensemble, pour un transfert (`postMessage(msg, transfer)`). */
export function lineSetBuffers(lines: LineSet): ArrayBuffer[] {
  return [lines.points.buffer as ArrayBuffer, lines.offsets.buffer as ArrayBuffer]
}
