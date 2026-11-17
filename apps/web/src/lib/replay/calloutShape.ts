import type { XY } from './replayLogic'

/** Un contour fermé a au moins trois sommets : en dessous, ce n'est pas une surface. */
export const MIN_RING_VERTICES = 3

/** Un anneau de sommets est-il un contour exploitable ? */
export function isRing(points: readonly unknown[]): boolean {
  return points.length >= MIN_RING_VERTICES
}

/** Les trois sortes de contours d'une zone nommée (provenance « découpe » : parties et trous). */
export interface ZoneOutline {
  polygon: readonly XY[]
  parts: readonly (readonly XY[])[]
  holes: readonly (readonly XY[])[]
}

/**
 * zoneRings énumère les contours exploitables d'une zone, dans l'ordre contour principal, parties,
 * trous : ceux qu'un remplissage pair-impair ou un test d'appartenance doivent traverser.
 */
export function zoneRings(zone: ZoneOutline): (readonly XY[])[] {
  return [zone.polygon, ...zone.parts, ...zone.holes].filter(isRing)
}

/**
 * zoneHasShape est LA règle « la zone a une forme » : un contour principal OU une partie d'au moins
 * trois sommets. C'est celle du jeu, qui définit une zone nommée par l'un ou par l'autre ; le dessin,
 * le nommage d'un point, la probabilité de réapparition et la Tactique la partagent, aucun ne retient
 * une règle maison. Des trous seuls ne font pas une forme.
 */
export function zoneHasShape(zone: Pick<ZoneOutline, 'polygon' | 'parts'>): boolean {
  return isRing(zone.polygon) || zone.parts.some(isRing)
}
