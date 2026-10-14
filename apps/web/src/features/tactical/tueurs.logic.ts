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
import { screenLengthPx } from '@/features/match-replay/model/screenSizes'
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

/** Au-delà de ce déplacement (px) entre l'appui et le relâcher, le geste est un glisser, pas un clic. */
export const SEUIL_CLIC_PX = 4

/**
 * LA TAILLE DE LA COUCHE SUIT LA CARTE ET LE ZOOM. Rayon du point, épaisseur du trait et rayon de
 * survol sont des longueurs MONDE (mètres), converties en px par le modèle de taille du rejeu 2D
 * (`screenLengthPx` : longueur × max(échelle de la carte, plancher d'échelle), plafond doux), avec
 * un plancher et un plafond en px pour rester lisibles. L'échelle est celle du plan à l'instant :
 * px par mètre de la fenêtre visible, donc le zoom y est compris.
 */
const RAYON_POINT_M = 0.42
const EPAISSEUR_TRAIT_M = 0.105
const RAYON_SURVOL_M = 1.5
/** Mise en avant (survol ou sélection) : facteur sur le rayon du point et l'épaisseur du trait. */
const FACTEUR_AVANT = 1.6
const RAYON_POINT_PX = { min: 1.7, max: 7.6 } as const
const EPAISSEUR_TRAIT_PX = { min: 1, max: 2.2 } as const
const RAYON_SURVOL_PX = { min: 9, max: 24 } as const

/** Les dimensions de la couche, en px, pour une échelle de plan donnée. */
export interface DimensionsDeLaCouche {
  rayonPoint: number
  rayonPointAvant: number
  epaisseurTrait: number
  epaisseurTraitAvant: number
  /** Distance maximale entre le curseur et un point pour le survoler. */
  rayonSurvol: number
}

function borne(valeur: number, { min, max }: { min: number; max: number }): number {
  return Math.min(Math.max(valeur, min), max)
}

/** dimensionsDeLaCouche — les tailles de la couche pour `scalePxPerM` px par mètre sur le plan. */
export function dimensionsDeLaCouche(scalePxPerM: number): DimensionsDeLaCouche {
  const px = (metres: number) => screenLengthPx(metres * 1000, scalePxPerM)
  const rayonPoint = borne(px(RAYON_POINT_M), RAYON_POINT_PX)
  const epaisseurTrait = borne(px(EPAISSEUR_TRAIT_M), EPAISSEUR_TRAIT_PX)
  return {
    rayonPoint,
    rayonPointAvant: rayonPoint * FACTEUR_AVANT,
    epaisseurTrait,
    epaisseurTraitAvant: epaisseurTrait * FACTEUR_AVANT,
    rayonSurvol: borne(px(RAYON_SURVOL_M), RAYON_SURVOL_PX),
  }
}

/** misEnAvant — la contribution en relief : celle qu'on survole, à défaut celle qui est choisie. */
export function misEnAvant(survol: number | null, choisie: number | null): number | null {
  return survol ?? choisie
}

/** basculerChoix — re-clic sur la contribution choisie = désélection, sinon elle devient la choisie. */
export function basculerChoix(choisie: number | null, index: number): number | null {
  return choisie === index ? null : index
}

/** formeDuRole — la forme qui distingue le rôle (jamais la couleur) : un tueur est un losange, une victime un rond. */
export function formeDuRole(role: RoleDuPoint): 'losange' | 'rond' {
  return role === 'tueur' ? 'losange' : 'rond'
}

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
  rayonPx: number,
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
