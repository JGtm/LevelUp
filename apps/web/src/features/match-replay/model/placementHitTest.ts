/**
 * placementHitTest.ts — QUELLE POSE EST SOUS LE CURSEUR, et de quelle taille est sa cible.
 *
 * EXTRAIT DE `equipmentPlacementsLayer.ts` LE 2026-08-19 (lot des objets lâchés), qui approchait
 * son seuil de taille et gagnait une règle de rendu de plus. La découpe tombe sur la seule
 * question de ce fichier : le POINTEUR. Ce qui reste au calque, c'est la décision de rendu et
 * le tracé ; ce qui vient ici, c'est la zone sensible — et elle se raisonne sans dessiner.
 *
 * LE SURVOL SE REJOUE SUR LA DONNÉE, JAMAIS SUR LES PIXELS : une fois peinte, une forme n'est
 * plus qu'un tableau de couleurs. C'est pourquoi ce fichier refait le même parcours que le
 * tracé (`placementKind`, puis la fenêtre) au lieu de relire le canvas — et c'est ce qui
 * garantit qu'on n'inspecte jamais ce qui n'est pas à l'écran.
 *
 * LA DÉPENDANCE VA DANS UN SEUL SENS : ce module lit le calque, le calque ne le lit pas. Son
 * appelant est le hook de survol (`usePlacementHover`), qui l'importe directement.
 */
import type { ReplayEquipmentPlacement } from '@/lib/api/types'

import { RIFT_HALF_HEIGHT_PX } from '../layers/placementRift'
import {
  type PlacementHoverTime,
  type PlacementKind,
  placementKind,
} from '../layers/equipmentPlacementsLayer'
import {
  DROPPED_RADIUS_PX,
  type PlacementView,
  project,
  REPAIR_FIELD_RADIUS_M,
  SHROUD_RADIUS_M,
  SEEKER_IMPULSE_RADIUS_PX,
  UNNAMED_DOT_RADIUS_PX,
  viewScale,
} from '../layers/placementShapes'
import { wallRadiusM } from '../layers/placementWall'
import { isPlacementActive, placementShows } from './placementWindow'
import type { XY } from './replayLogic'
import { SENSOR_RADIUS_M } from './threatSensor'

/** Rayon minimal de la zone sensible au survol, en pixels : un point de 2,5 px ne se vise pas. */
const HOVER_MIN_RADIUS_PX = 9

/**
 * Rayon de la zone sensible au survol d'une pose, en pixels d'écran.
 *
 * Les familles dont la forme est déjà en PIXELS (balise, impulsion du traqueur, point neutre,
 * objet lâché) gardent leur taille, relevée au plancher de visée ; celles dont la forme est en
 * MÈTRES (capteur, champ de réparation, mur) suivent l'échelle de la carte — leur zone
 * sensible est celle qu'on voit.
 */
export function hoverRadiusPx(kind: PlacementKind, view: PlacementView): number {
  const scale = viewScale(view)
  if (kind === 'sensor') return SENSOR_RADIUS_M * scale
  if (kind === 'field') return REPAIR_FIELD_RADIUS_M * scale
  if (kind === 'wall') return Math.max(wallRadiusM(view) * scale, HOVER_MIN_RADIUS_PX)
  if (kind === 'seeker') return Math.max(SEEKER_IMPULSE_RADIUS_PX, HOVER_MIN_RADIUS_PX)
  if (kind === 'rift') return Math.max(RIFT_HALF_HEIGHT_PX, HOVER_MIN_RADIUS_PX)
  // L ECRAN : sa zone sensible est sa bulle entiere, fondu compris. Le survol suit ce que
  // l oeil voit — un disque de 6 m qu on ne pourrait designer qu en son centre mentirait.
  if (kind === 'shroud') return SHROUD_RADIUS_M * scale
  if (kind === 'dropped') return Math.max(DROPPED_RADIUS_PX, HOVER_MIN_RADIUS_PX)
  return Math.max(UNNAMED_DOT_RADIUS_PX, HOVER_MIN_RADIUS_PX)
}

/**
 * placementAt — la pose sous le curseur, à l'image courante, ou null.
 *
 * LA PLUS PETITE GAGNE, et c'est ce qui rend le survol utilisable : le disque du capteur
 * couvre 4,25 m de rayon, un mur posé dedans en couvre 1,6. Prendre la première trouvée
 * montrerait toujours le capteur, et le mur serait inatteignable au pointeur. C'est aussi ce
 * qui rend un objet LÂCHÉ atteignable sous la zone d'un capteur déployé au même endroit.
 *
 * `point` est en pixels CSS du canvas, le même repère que le dessin (le facteur de densité est
 * déjà absorbé par la transformation du contexte).
 */
export function placementAt(
  placements: readonly ReplayEquipmentPlacement[],
  view: PlacementView,
  time: PlacementHoverTime,
  point: XY,
): ReplayEquipmentPlacement | null {
  let best: ReplayEquipmentPlacement | null = null
  let bestR = Infinity
  for (const p of placements) {
    const kind = placementKind(p, time)
    if (!kind || !isPlacementActive(p, kind, time.frame, time)) continue
    if (!placementShows(p, kind, time)) continue
    const r = hoverRadiusPx(kind, view)
    if (r >= bestR) continue
    const c = project({ x: p.x, y: p.y }, view)
    const dx = point.x - c.x
    const dy = point.y - c.y
    if (dx * dx + dy * dy > r * r) continue
    best = p
    bestR = r
  }
  return best
}
