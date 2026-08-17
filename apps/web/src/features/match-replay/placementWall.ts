/**
 * placementWall.ts — LE MUR DE PROTECTION : sa géométrie, son tracé, et SUR QUELLE POSE il se
 * dessine.
 *
 * DEUX RÈGLES DISTINCTES, ET C'EST POUR LES TENIR ENSEMBLE QUE LE MUR A SON FICHIER. La
 * première est de forme : un arc, orienté par le regard du poseur. La seconde est d'IDENTITÉ :
 * un mur déployé produit DEUX poses, et l'arc n'en concerne qu'une (cf. WALL_PANEL_IDS).
 *
 * CE QUE LE MUR AFFIRME, ET CE QU'IL N'AFFIRME PAS. Le record de création ne porte AUCUNE
 * orientation d'objet (mesure du 18/08). `h` est le cap où le POSEUR REGARDAIT au moment de la
 * pose, et c'est tout ce dont on dispose ; l'arc est donc orienté par le geste, pas par l'objet.
 * Sans cap, aucune orientation n'est inventée : la pose devient un cercle pointillé du même
 * rayon, qui dit « ici, un mur » sans dire « dans ce sens ».
 *
 * POURQUOI UN ARC ET PAS UN RECTANGLE (décision utilisateur du 18/08) : « cet équipement laisse
 * passer les dégâts dans un sens et pas dans l'autre ». La concavité regarde le poseur — ses
 * tirs sortent par l'intérieur, ceux d'en face butent sur la face convexe. Le milieu de l'arc
 * est donc DEVANT la position, dans la direction du cap (on pose le mur devant soi), et l'arc
 * s'ouvre vers l'arrière.
 */
import type { ReplayEquipmentPlacement } from '@/lib/api/types'

import {
  project,
  type PlacementView,
  type ShapeStyle,
  strokePolyline,
  UNCERTAIN_DASH,
  viewScale,
} from './placementShapes'
import type { XY } from './replayLogic'

/**
 * LES PANNEAUX DU MUR — les deux identifiants sur lesquels l'arc se dessine.
 *
 * PROVENANCE : le manifeste du titre (`config/titles/halo_infinite/mappings/replay_labels.toml`,
 * `[[equipment_objects]]`) leur donne `kind = "deployed"` et la provenance `sofa_parent` — ils
 * sont ENGENDRÉS par l'appareil, ils ne sont pas portés. Le web ne reçoit que `family` et `id`,
 * d'où cette table côté client ; le garde-rail `placementPanels.guard.test.ts` la rejoue contre
 * le TOML, dans les deux sens, pour qu'un troisième panneau au manifeste devienne rouge ici.
 *
 * POURQUOI L'ARC N'EST PAS SUR L'APPAREIL. Un mur déployé produit DEUX poses `deployed` :
 * l'appareil qui vole ET ses panneaux. Les dessiner toutes deux ferait deux arcs pour un seul
 * mur. Et la mesure désigne les panneaux sans ambiguïté — 97,9 % et 97,7 % de déploiements,
 * contre 29,4 % et 13,0 % pour les deux appareils : 97,7 % d'un côté, 29,4 % de l'autre, et
 * rien entre les deux. Un appareil de mur à la position d'un joueur est, 87 fois sur 100,
 * l'objet qu'il tenait en mourant.
 */
export const WALL_PANEL_IDS: readonly string[] = ['0x528fce46', '0x686b40c9']

/**
 * Rayon du MUR, en mètres monde. 1,6 m avec une ouverture de 110° donne une corde de 2,6 m —
 * l'ordre de grandeur du mur du jeu. C'est un choix d'ÉCRAN calibré sur la carte, pas une
 * mesure : le film ne porte ni les dimensions de l'objet ni son orientation.
 */
export const WALL_RADIUS_M = 1.6

/** Ouverture de l'arc, en radians (110°). */
export const WALL_OPENING_RAD = (110 * Math.PI) / 180

/**
 * PLANCHER DE LISIBILITÉ, en pixels d'écran. Un mur de 1,6 m sur une carte de Big Team Battle
 * (150 à 200 m de large pour 432 px utiles, soit ~2,5 px/m) tomberait à 4 px de rayon : une
 * éraflure. Le rayon monde effectif est donc relevé jusqu'à ce que l'arc atteigne ce plancher —
 * la pose reste à sa place exacte, seule sa TAILLE cesse de suivre l'échelle sous ce seuil.
 */
export const WALL_MIN_RADIUS_PX = 6

/** Nombre de segments du polygone qui approche l'arc du mur (110° -> ~7° par segment). */
const WALL_ARC_SEGMENTS = 16

/** Nombre de segments du cercle pointillé (pose sans cap) — un tour complet. */
const WALL_RING_SEGMENTS = 48

/** Épaisseur du trait du mur, en pixels d'écran (décision du plan : 2 px). */
const WALL_LINE_WIDTH = 2
/** Le halo : le même trait, plus large et transparent, posé sous le trait franc. */
const WALL_HALO_WIDTH = 5
const WALL_HALO_ALPHA = 0.22
const WALL_ALPHA = 0.9

/**
 * arcWorld — les points MONDE d'un arc de cercle centré sur `center`, de `fromRad` à `toRad`.
 *
 * En MONDE et non en pixels : c'est là que l'orientation a un sens (l'axe Y du canvas est
 * inversé, celui du monde ne l'est pas), et c'est donc là que le test peut dire « le milieu de
 * l'arc est bien devant le poseur ». La projection vient après, point par point.
 */
function arcWorld(
  center: XY,
  fromRad: number,
  toRad: number,
  radiusM: number,
  segments: number,
): XY[] {
  const out: XY[] = []
  for (let i = 0; i <= segments; i++) {
    const a = fromRad + ((toRad - fromRad) * i) / segments
    out.push({ x: center.x + radiusM * Math.cos(a), y: center.y + radiusM * Math.sin(a) })
  }
  return out
}

/**
 * wallArcWorld — l'arc du mur, en coordonnées monde.
 *
 * `headingDeg` est le cap de VISÉE du poseur, dans la convention de `Point.h` : 0° pointe vers
 * les X croissants, l'angle tourne dans le sens direct du repère monde (+Y vers le haut). Le
 * MILIEU de l'arc est posé à ce cap, à `radiusM` de la position — donc devant le poseur — et
 * l'arc s'ouvre de part et d'autre : sa concavité regarde la position, c'est-à-dire le poseur.
 */
export function wallArcWorld(
  center: XY,
  headingDeg: number,
  radiusM: number,
  segments: number = WALL_ARC_SEGMENTS,
): XY[] {
  const mid = (headingDeg * Math.PI) / 180
  return arcWorld(center, mid - WALL_OPENING_RAD / 2, mid + WALL_OPENING_RAD / 2, radiusM, segments)
}

/**
 * wallRingWorld — le cercle complet servi à une pose SANS cap. Aucune orientation n'est
 * inventée : le tracé est fermé et pointillé, il dit le lieu et le rayon, rien d'autre.
 */
export function wallRingWorld(
  center: XY,
  radiusM: number,
  segments: number = WALL_RING_SEGMENTS,
): XY[] {
  return arcWorld(center, 0, Math.PI * 2, radiusM, segments)
}

/**
 * wallRadiusM — le rayon monde effectivement tracé : celui du plan, relevé au plancher de
 * lisibilité quand l'échelle de la carte le réduirait à une éraflure (cf. WALL_MIN_RADIUS_PX).
 */
export function wallRadiusM(view: PlacementView): number {
  const scale = viewScale(view)
  if (!(scale > 0)) return WALL_RADIUS_M
  return Math.max(WALL_RADIUS_M, WALL_MIN_RADIUS_PX / scale)
}

/**
 * drawWall — l'arc du mur (ou son cercle pointillé sans cap), halo puis trait franc.
 *
 * Le HALO est le même tracé, plus large et très transparent : sur un fond de carte chargé, un
 * trait de 2 px se perd dans le décor ; sur un fond clair, il se perd dans le blanc. Le halo
 * lui donne son assise sans rien ajouter à ce qu'il affirme.
 */
export function drawWall(
  ctx: CanvasRenderingContext2D,
  p: ReplayEquipmentPlacement,
  view: PlacementView,
  style: ShapeStyle,
  color: string,
): void {
  const center = { x: p.x, y: p.y }
  const radiusM = wallRadiusM(view)
  // `h` est un cap MESURÉ : nul est une valeur (un poseur peut regarder vers 0°), seule son
  // absence rend la pose non orientée — d'où le test sur `undefined`/`null` et non sur la
  // véracité. Le cercle est FERMÉ et pointillé, l'arc est ouvert et franc.
  const heading = p.h === undefined || p.h === null ? null : p.h
  const world =
    heading === null ? wallRingWorld(center, radiusM) : wallArcWorld(center, heading, radiusM)
  const closed = heading === null
  const pts = world.map((w) => project(w, view))
  ctx.save()
  ctx.strokeStyle = color
  ctx.lineCap = 'round'
  if (closed) ctx.setLineDash(UNCERTAIN_DASH.map((d) => d * style.k))
  ctx.globalAlpha = WALL_HALO_ALPHA
  ctx.lineWidth = WALL_HALO_WIDTH * style.k
  strokePolyline(ctx, pts, closed)
  ctx.globalAlpha = WALL_ALPHA
  ctx.lineWidth = WALL_LINE_WIDTH * style.k
  strokePolyline(ctx, pts, closed)
  ctx.restore()
}
