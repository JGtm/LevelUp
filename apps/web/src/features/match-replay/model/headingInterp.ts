/**
 * headingInterp — LE CAP ENTRE DEUX ÉCHANTILLONS, par le plus court chemin angulaire.
 *
 * POURQUOI (contrôles v8, 2026-10-21, témoin 7b0d89c4 : « rotations saccadées des véhicules »). Le
 * film publie un cap par image du document (10 Hz) et le rendu tourne à l'écran bien plus vite : la
 * POSITION d'un véhicule s'interpole entre deux échantillons (`positionAt`), son CAP était tenu au
 * dernier échantillon — le sprite glissait en continu puis pivotait par saccades de 100 ms. Un cap
 * s'interpole sur le cercle : de 350 à 10 degrés, le chemin est de +20, pas de -340 (d'où le plus
 * court chemin, et non une moyenne arithmétique).
 *
 * CE QUE CETTE INTERPOLATION NE FAIT PAS, ET C'EST VOLONTAIRE :
 *  - elle ne traverse jamais un SILENCE de réplication (`g > 0`) : comme la
 *    position, le cap est tenu au point qui précède — un véhicule ne tourne pas pendant que le film
 *    se tait ;
 *  - elle ne comble pas un trou plus long que `HEADING_INTERP_MAX_SPAN_FRAMES` : au-delà, les deux
 *    lectures n'appartiennent plus à un même mouvement et le cap reste tenu (un virage lissé sur
 *    dix secondes serait une invention).
 *
 * Pas de React, pas de canevas : de l'arithmétique datée, testable seule.
 */

/**
 * Écart maximal, en images, entre deux lectures de cap pour qu'on tourne de l'une à l'autre. Une
 * seconde (10 images) : le même ordre que le maintien d'une lecture de visée
 * (`VEHICLE_AIM_HOLD_FRAMES`), pour la même raison — au-delà, ce n'est plus un flux continu.
 */
export const HEADING_INTERP_MAX_SPAN_FRAMES = 10

/** Un point qui porte (ou non) un cap : un échantillon de véhicule ou une lecture de visée. */
export interface HeadingPoint {
  t: number
  h?: number
  /** Silence de réplication qui PRÉCÈDE ce point (ms) : on ne tourne pas au travers. */
  g?: number
}

/**
 * lerpHeadingDeg — le cap à la fraction `f` (0..1) de `from` vers `to`, par le PLUS COURT chemin
 * (écart ramené dans [-180, 180)). Le résultat est ramené dans [0, 360).
 */
export function lerpHeadingDeg(from: number, to: number, f: number): number {
  const delta = ((((to - from) % 360) + 540) % 360) - 180
  return (((from + delta * f) % 360) + 360) % 360
}

/**
 * headingTowardNext — le cap à `frame`, sachant que `points[prevIdx]` est la dernière lecture de
 * cap au plus tard à cet instant. S'il existe une lecture suivante de cap, atteignable sans
 * traverser de silence et à moins de `maxSpanFrames`, le cap tourne vers elle ; sinon il reste celui
 * de `points[prevIdx]`. `prevIdx` doit désigner un point qui porte un cap.
 */
export function headingTowardNext(
  points: readonly HeadingPoint[],
  prevIdx: number,
  frame: number,
  maxSpanFrames: number = HEADING_INTERP_MAX_SPAN_FRAMES,
): number {
  const prev = points[prevIdx]
  const held = prev.h as number
  if (frame <= prev.t) return held
  for (let j = prevIdx + 1; j < points.length; j++) {
    const next = points[j]
    if ((next.g ?? 0) > 0) return held
    if (next.t - prev.t > maxSpanFrames) return held
    if (next.h === undefined || next.t <= frame) continue
    return lerpHeadingDeg(held, next.h, (frame - prev.t) / (next.t - prev.t))
  }
  return held
}
