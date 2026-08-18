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
 * pose ; l'arc est donc orienté par le geste, pas par l'objet.
 *
 * TROIS SOURCES DE CAP, DANS CET ORDRE, ET AUCUNE N'EST INVENTÉE (V3, retour utilisateur du
 * 2026-08-18 : « sans cap je préférerais qu'on tente de corréler la visée ou la trajectoire du
 * joueur, un mur portatif rond serait trop troublant »). Mesure sur le corpus en cache — 32
 * films, 62 panneaux de mur :
 *
 *   1. `placement` — le cap `h` de la POSE elle-même : 54 panneaux sur 62 (87,1 %) ;
 *   2. `trajectory` — la DERNIÈRE DIRECTION DE DÉPLACEMENT du poseur avant la pose, prise sur
 *      le dernier segment d'au moins 0,5 m : les 8 panneaux restants (12,9 %), TOUS résolus
 *      (déplacement maximal mesuré de 0,50 à 0,74 m — le seuil tombe juste, et c'est pour cela
 *      qu'il est bas) ;
 *   3. `aim` — la visée `h` de la dernière image de la vie du poseur qui en porte une : 0 cas
 *      sur 62 en pratique, mais disponible dans les 8 (âge 2 à 10 images). C'est le repli du
 *      repli, et il tient parce que les deux directions concordent souvent — écart médian de
 *      19,8° sur ces 8 cas, 5 sous 20°, 3 au-delà (43,5°, 83,8°, 103,8°).
 *
 * LE CERCLE POINTILLÉ NE DISPARAÎT PAS DU CODE, MAIS IL DISPARAÎT DE L'ÉCRAN : il ne reste que
 * pour une pose dont le POSEUR N'A AUCUNE PISTE — 0 panneau sur 62 dans le corpus, alors que
 * 154 poses sur 2 563 (6,0 %, toutes familles confondues) ont bien un poseur sans piste. Ce
 * n'est donc pas une branche morte, c'est le cas qu'aucun film n'a encore produit POUR UN MUR.
 *
 * UN CAP DÉDUIT SE VOIT : l'arc est POINTILLÉ quand il vient de la trajectoire ou de la visée,
 * franc quand il vient de la pose. Même grammaire que partout dans ce calque (UNCERTAIN_DASH) —
 * un trait plein dit une valeur qu'on tient de sa source.
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
import type { ReplayTrackReady } from './replayNormalize'

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

/**
 * DÉPLACEMENT MINIMAL, en mètres monde, pour qu'un segment de trajectoire dise une DIRECTION.
 *
 * 0,5 m N'EST PAS UN CHOIX DE CONFORT : les trajectoires sont échantillonnées à 100 ms, et un
 * joueur qui vise sans bouger produit des points séparés de quelques centimètres — du bruit de
 * position, dont l'angle tourne à chaque image. Sous ce seuil, la direction ne serait pas
 * mesurée, elle serait tirée au sort. Les 8 murs sans cap du corpus atteignent 0,50 à 0,74 m
 * de déplacement maximal avant la pose : le seuil est exactement au bord, et le relever les
 * perdrait tous.
 */
export const WALL_TRAJECTORY_MIN_M = 0.5

/** D'où vient le cap retenu pour l'arc — l'ordre de la chaîne est celui de cette union. */
export type WallHeadingSource = 'placement' | 'trajectory' | 'aim'

/** Le cap retenu, et SA PROVENANCE : c'est elle qui décide du trait (franc ou pointillé). */
export interface WallHeading {
  deg: number
  source: WallHeadingSource
}

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
 * poserPointsBefore — les points de la vie du poseur ANTÉRIEURS à la pose, dans l'ordre.
 *
 * PAR SLOT ET PAR FENÊTRE : un slot de bipède est réattribué à chaque réapparition, donc
 * plusieurs vies peuvent porter le même. Celle qui compte est celle qui CONTIENT l'instant de
 * la pose ; à défaut, la plus longue qui la précède — c'est ce que fait la comparaison sur le
 * nombre de points retenus.
 */
function poserPointsBefore(
  lives: readonly ReplayTrackReady[],
  owner: number,
  t0: number,
): ReplayTrackReady['points'] {
  let best: ReplayTrackReady['points'] = []
  for (const life of lives) {
    if (life.slot !== owner) continue
    const pts = life.points.filter((p) => p.t <= t0)
    if (pts.length > best.length) best = pts
  }
  return best
}

/**
 * trajectoryHeadingDeg — la DERNIÈRE direction de déplacement, ou null.
 *
 * On remonte depuis le dernier point connu jusqu'à trouver un point distant d'au moins
 * `WALL_TRAJECTORY_MIN_M` : le vecteur qui les joint est la direction dans laquelle le joueur
 * ARRIVAIT. Remonter — plutôt que prendre les deux derniers points — est ce qui rend la mesure
 * stable : deux points consécutifs d'un joueur à l'arrêt ne disent rien.
 */
function trajectoryHeadingDeg(points: ReplayTrackReady['points']): number | null {
  const head = points[points.length - 1]
  if (!head) return null
  for (let i = points.length - 2; i >= 0; i--) {
    const dx = head.x - points[i].x
    const dy = head.y - points[i].y
    if (Math.hypot(dx, dy) < WALL_TRAJECTORY_MIN_M) continue
    return ((Math.atan2(dy, dx) * 180) / Math.PI + 360) % 360
  }
  return null
}

/** aimHeadingDeg — la visée `h` de la dernière image qui en porte une, ou null. */
function aimHeadingDeg(points: ReplayTrackReady['points']): number | null {
  for (let i = points.length - 1; i >= 0; i--) {
    const h = points[i].h
    if (h !== undefined && h !== null) return h
  }
  return null
}

/**
 * wallHeading — LE CAP DE L'ARC et sa provenance, ou null si aucune source n'existe.
 *
 * `h` de la pose est un cap MESURÉ : nul est une valeur (un poseur peut regarder vers 0°),
 * seule son ABSENCE ouvre la chaîne — d'où le test sur `undefined`/`null` et non sur la
 * véracité.
 */
export function wallHeading(
  p: ReplayEquipmentPlacement,
  lives: readonly ReplayTrackReady[],
): WallHeading | null {
  if (p.h !== undefined && p.h !== null) return { deg: p.h, source: 'placement' }
  const points = poserPointsBefore(lives, p.owner, p.t0)
  const traj = trajectoryHeadingDeg(points)
  if (traj !== null) return { deg: traj, source: 'trajectory' }
  const aim = aimHeadingDeg(points)
  if (aim !== null) return { deg: aim, source: 'aim' }
  return null
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
 * drawWall — l'arc du mur, halo puis trait franc. Cercle pointillé SEULEMENT quand aucune
 * source de cap n'existe (cf. la chaîne en tête de fichier : 0 panneau sur 62 mesurés).
 *
 * Le HALO est le même tracé, plus large et très transparent : sur un fond de carte chargé, un
 * trait de 2 px se perd dans le décor ; sur un fond clair, il se perd dans le blanc. Le halo
 * lui donne son assise sans rien ajouter à ce qu'il affirme.
 *
 * LE POINTILLÉ DIT LA RÉSERVE, PAS LA FORME : un arc déduit de la trajectoire ou de la visée
 * reste un arc — il est simplement tracé en pointillé, parce que sa direction n'est pas celle
 * que le record de la pose affirme.
 */
export function drawWall(
  ctx: CanvasRenderingContext2D,
  wall: { p: ReplayEquipmentPlacement; heading: WallHeading | null },
  view: PlacementView,
  style: ShapeStyle,
  color: string,
): void {
  const { p, heading } = wall
  const center = { x: p.x, y: p.y }
  const radiusM = wallRadiusM(view)
  const world =
    heading === null
      ? wallRingWorld(center, radiusM)
      : wallArcWorld(center, heading.deg, radiusM)
  const closed = heading === null
  const pts = world.map((w) => project(w, view))
  ctx.save()
  ctx.strokeStyle = color
  ctx.lineCap = 'round'
  if (closed || heading?.source !== 'placement') {
    ctx.setLineDash(UNCERTAIN_DASH.map((d) => d * style.k))
  }
  ctx.globalAlpha = WALL_HALO_ALPHA
  ctx.lineWidth = WALL_HALO_WIDTH * style.k
  strokePolyline(ctx, pts, closed)
  ctx.globalAlpha = WALL_ALPHA
  ctx.lineWidth = WALL_LINE_WIDTH * style.k
  strokePolyline(ctx, pts, closed)
  ctx.restore()
}
