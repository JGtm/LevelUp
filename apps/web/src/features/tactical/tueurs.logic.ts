/**
 * tueurs.logic — la logique PURE de la couche « autre joueur » du plan : pour chaque contribution
 * de la zone sélectionnée qui porte la position de l'événement ET celle de l'autre joueur, un trait
 * de l'une à l'autre et un point à l'autre joueur.
 *
 * LE RÔLE DU POINT SE DÉCIDE PAR LA FACE DE LA CONTRIBUTION, jamais par la lecture demandée : une
 * face `mort` pose le TUEUR, une face `frag` la VICTIME. Les autres faces (entrée, réapparition)
 * n'ont pas d'autre joueur et ne posent rien ; une contribution dont un des deux points manque non
 * plus (jamais de position de repli).
 */
import type { TacticalContribution } from '@/lib/api/types'

/** Le rôle de l'autre joueur : le tueur d'une mort, la victime d'un frag. */
export type RoleDuPoint = 'tueur' | 'victime'

/** Un point du plan, en mètres monde. */
interface PointMonde {
  x: number
  y: number
}

/** Un trait et son point : ce que la couche peint pour une contribution. */
export interface PointDeLaCouche {
  /** L'index de la contribution dans la liste servie (la clé du survol croisé tuile ↔ plan). */
  index: number
  role: RoleDuPoint
  /** La position de l'événement (le trait en part). */
  de: PointMonde
  /** La position de l'autre joueur (le point et le bout du trait). */
  vers: PointMonde
  /** Distance entre les deux, au mètre ; `null` sans les deux points. */
  distanceM: number | null
}

/** Ordre fixe des rôles, pour la légende. */
const ORDRE_DES_ROLES: readonly RoleDuPoint[] = ['tueur', 'victime']

/** Au-delà de cette distance (px écran) du curseur, aucun point n'est survolé. */
export const RAYON_DE_SURVOL_PX = 9

/** roleDeLaFace — le rôle de l'autre joueur d'une face, `null` pour une face sans autre joueur. */
export function roleDeLaFace(face: string | undefined): RoleDuPoint | null {
  if (face === 'mort') return 'tueur'
  if (face === 'frag') return 'victime'
  return null
}

function fini(p: PointMonde | undefined): p is PointMonde {
  return p !== undefined && Number.isFinite(p.x) && Number.isFinite(p.y)
}

/** distanceEnMetres — la distance 2D entre l'événement et l'autre joueur, au mètre ; `null` si un point manque. */
export function distanceEnMetres(c: Pick<TacticalContribution, 'point' | 'autre_point'>): number | null {
  if (!fini(c.point) || !fini(c.autre_point)) return null
  return Math.round(Math.hypot(c.autre_point.x - c.point.x, c.autre_point.y - c.point.y))
}

/** pointsDeLaCouche — les traits et points à peindre pour les contributions de la zone. */
export function pointsDeLaCouche(contributions: readonly TacticalContribution[]): PointDeLaCouche[] {
  const out: PointDeLaCouche[] = []
  contributions.forEach((c, index) => {
    const role = roleDeLaFace(c.face)
    if (!role || !fini(c.point) || !fini(c.autre_point)) return
    out.push({
      index,
      role,
      de: { x: c.point.x, y: c.point.y },
      vers: { x: c.autre_point.x, y: c.autre_point.y },
      distanceM: distanceEnMetres(c),
    })
  })
  return out
}

/** rolesPresents — les rôles que la couche porte, dans l'ordre de la légende (tueurs, victimes). */
export function rolesPresents(points: readonly PointDeLaCouche[]): RoleDuPoint[] {
  return ORDRE_DES_ROLES.filter((r) => points.some((p) => p.role === r))
}

/**
 * pointLePlusProche — l'index de contribution du point le plus proche du curseur (px écran), à
 * `rayonPx` près au plus ; `null` sinon. `projeter` est la projection monde → écran du plan.
 */
export function pointLePlusProche(
  points: readonly PointDeLaCouche[],
  curseur: { x: number; y: number },
  projeter: (p: PointMonde) => { x: number; y: number },
  rayonPx: number = RAYON_DE_SURVOL_PX,
): number | null {
  let meilleur: number | null = null
  let ecart = rayonPx
  for (const p of points) {
    const s = projeter(p.vers)
    const d = Math.hypot(s.x - curseur.x, s.y - curseur.y)
    if (d <= ecart) {
      ecart = d
      meilleur = p.index
    }
  }
  return meilleur
}
