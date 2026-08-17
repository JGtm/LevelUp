/**
 * equipmentPlacementsLayer.ts — LES POSES D'ÉQUIPEMENT sur la carte : le mur de protection,
 * le capteur de menaces, et les objets que l'archétype d'équipement porte sans qu'on les nomme.
 *
 * LA SOURCE est le document (schéma 9) : `equipmentPlacements`, une pose par vie d'objet, avec
 * sa fenêtre [t0, t1] sur l'axe des frames, sa position monde, la FAMILLE que le manifeste du
 * titre lui donne (`replay_labels.toml`), le SLOT de son poseur (`owner`, -1 si aucun bipède
 * contemporain à moins de 3 m) et le CAP de visée de ce poseur (`h`, absent une fois sur sept).
 *
 * LE RENDU EST UNE TABLE PAR FAMILLE (`PLACEMENT_RENDER`), et c'est la règle qui compte : une
 * famille absente de la table ne dessine RIEN. Le lot de nommage structurel en ajoutera
 * (translocateur, répulseur, propulseur…) ; chacune recevra SA règle de rendu ou restera muette
 * — jamais le dessin d'une voisine, même principe que les libellés et les sons.
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
 *
 * LE CAPTEUR PINGE, et tout ce qu'il affirme — portée, cadence, durée de révélation, camp
 * révélé — vit dans `threatSensor.ts` : les chiffres y sont OFFICIELS et sourcés, la logique y
 * est pure et testée. Ce fichier n'en garde que le TRACÉ (disque, anneau, onde, marques).
 *
 * Pas de React : géométrie pure + un CanvasRenderingContext2D (même règle que replayDraw).
 */
import type { ReplayBounds, ReplayEquipmentPlacement } from '@/lib/api/types'

import { canvasScale, worldToCanvas, type XY } from './replayLogic'
import type { ReplayTrackReady } from './replayNormalize'
import {
  REVEAL_FILL_ALPHA,
  REVEAL_RADIUS_PX,
  REVEAL_WIDTH,
  revealAlpha,
  SENSOR_DURATION_MS,
  SENSOR_FILL_ALPHA,
  SENSOR_PING_WIDTH,
  SENSOR_RADIUS_M,
  SENSOR_RING_ALPHA,
  SENSOR_RING_WIDTH,
  sensorPing,
  sensorReveals,
  type SensorReveal,
} from './threatSensor'

/** Cadrage du canvas (mêmes paramètres que worldToCanvas). */
export interface PlacementView {
  bounds: ReplayBounds
  width: number
  height: number
  pad: number
}

/**
 * Les trois façons de dessiner une pose, aujourd'hui. Ce ne sont PAS les familles : le jour
 * où deux familles partageront un rendu (deux murs de palettes différentes, par exemple), la
 * table les enverra toutes deux sur la même valeur sans qu'on duplique un tracé.
 */
export type PlacementKind = 'wall' | 'sensor' | 'unnamed'

/**
 * LA TABLE PAR FAMILLE — l'unique porte d'entrée du rendu. Les clés sont les identifiants
 * STABLES publiés par le document (`family`, posés par le manifeste du titre), jamais des
 * libellés. Une famille hors table = aucun dessin, jamais celui d'une voisine.
 */
export const PLACEMENT_RENDER: Readonly<Record<string, PlacementKind>> = {
  wall: 'wall',
  sensor: 'sensor',
  other: 'unnamed',
}

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

/** Rayon du point neutre des objets non identifiés, en pixels d'écran. */
const UNNAMED_DOT_RADIUS_PX = 2.5

/** Rayon minimal de la zone sensible au survol, en pixels : un point de 2,5 px ne se vise pas. */
const HOVER_MIN_RADIUS_PX = 9

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

const UNNAMED_ALPHA = 0.5

/** Motif du cercle pointillé — une pose sans cap ne montre aucune direction. */
const RING_DASH: readonly number[] = [3, 3]

/**
 * Ce que le calque a besoin de savoir de l'instant courant. `frameMs` est la durée RÉELLE
 * d'une frame (cf. `frameToMs(1, doc)`) : c'est elle qui donne son âge à la pulsation, jamais
 * un compteur d'images — la cadence d'échantillonnage est choisie au build.
 */
export interface PlacementTime {
  frame: number
  frameMs: number
  /** Nombre total d'images du rejeu (`doc.frameCount`) : la borne des poses sans fin connue. */
  frames: number
  /** Densité de pixels : les épaisseurs d'écran la suivent (même règle que les marqueurs). */
  k: number
  reducedMotion: boolean
  /** Les objets non identifiés se dessinent-ils ? Bascule du tiroir, ÉTEINTE par défaut. */
  showUnnamed: boolean
}

/** Ce qu'il faut connaître du temps pour borner une pose — rien de plus (règle des 5 params). */
export type PlacementWindowTime = Pick<PlacementTime, 'frameMs' | 'frames'>

/**
 * Ce que le calque a besoin de LIRE : les poses, et — pour le seul capteur de menaces — les
 * vies et leur camp.
 *
 * POURQUOI LES TROIS VOYAGENT ENSEMBLE. Le capteur ne se dessine pas seul : son ping RÉVÈLE
 * des adversaires, et « adversaire » est une relation entre deux vies. Les grouper garde
 * `drawEquipmentPlacementsLayer` à cinq paramètres (seuil CLAUDE.md n°5) et dit ce qu'ils sont :
 * une SCÈNE, pas trois arguments indépendants.
 */
export interface PlacementScene {
  placements: readonly ReplayEquipmentPlacement[]
  /** Toutes les vies du film ; le balayage de révélation filtre lui-même sur leur fenêtre. */
  lives: readonly ReplayTrackReady[]
  /** Camp d'une vie (`team_side`) ; null = camp inconnu, donc jamais révélé ni révélateur. */
  sideOfSlot: (slot: number) => string | null
}

/** Les encres du calque : la couleur d'équipe du poseur, et le neutre quand il n'y en a pas. */
export interface PlacementInk {
  /** Couleur de la vie qui a posé l'objet ; null = vie inconnue, on tombe sur `neutral`. */
  colorOfSlot: (slot: number) => string | null
  /** Encre neutre du thème (`--muted-foreground`), servie aux poses sans poseur. */
  neutral: string
}

/**
 * placementEndFrame — LA DERNIÈRE IMAGE à laquelle une pose se dessine.
 *
 * `t1` N'EST PAS LA DISPARITION DE L'OBJET, et c'est mesuré, pas supposé (2026-08-18,
 * `filmdec/equipment_life_end_test.go`) : le décodeur ne suit que les records qui portent une
 * position, si bien que `t1` date l'instant où l'objet cesse de BOUGER. Un mur de protection y
 * atteint 0,7 s de médiane et un capteur 2,1 s, quand le recensement des keyframes montre
 * 101 de ces 295 objets encore présents plus d'une seconde après. Le film ne date AUCUNE
 * disparition d'équipement — le record de suppression et la queue de records sans position
 * sont l'un et l'autre du bruit au témoin. `t1` est donc une BORNE INFÉRIEURE.
 *
 * D'OÙ LA RÈGLE, en deux temps et sans rien inventer :
 *  - une famille dont la durée est PUBLIÉE par l'éditeur s'y tient — le capteur de menaces
 *    dure 15 s (cf. SENSOR_DURATION_MS), du même genre que sa portée et sa cadence, qui
 *    gouvernent déjà tout son tracé ;
 *  - les autres restent affichées jusqu'à la fin du rejeu. Effacer un mur à 0,7 s
 *    affirmerait une disparition que rien ne mesure ; le laisser en place n'affirme rien.
 *
 * Jamais AVANT `t1` : la borne mesurée l'emporte si elle dépasse la durée officielle.
 */
export function placementEndFrame(
  p: ReplayEquipmentPlacement,
  kind: PlacementKind,
  time: PlacementWindowTime,
): number {
  const lastFrame = Math.max(time.frames - 1, p.t1)
  if (kind !== 'sensor' || !(time.frameMs > 0)) return lastFrame
  const official = p.t0 + Math.round(SENSOR_DURATION_MS / time.frameMs)
  return Math.min(Math.max(official, p.t1), lastFrame)
}

/** isPlacementActive — la pose se dessine-t-elle à cette image ? (cf. placementEndFrame). */
export function isPlacementActive(
  p: ReplayEquipmentPlacement,
  kind: PlacementKind,
  frame: number,
  time: PlacementWindowTime,
): boolean {
  return frame >= p.t0 && frame <= placementEndFrame(p, kind, time)
}

/**
 * placementKind rend la règle de rendu d'une pose, ou null quand il n'y en a pas : famille
 * inconnue du manifeste, ou objet non identifié alors que la bascule est éteinte.
 */
export function placementKind(
  p: ReplayEquipmentPlacement,
  showUnnamed: boolean,
): PlacementKind | null {
  const kind = PLACEMENT_RENDER[p.family]
  if (!kind) return null
  return kind === 'unnamed' && !showUnnamed ? null : kind
}

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
export function wallRingWorld(center: XY, radiusM: number, segments: number = WALL_RING_SEGMENTS): XY[] {
  return arcWorld(center, 0, Math.PI * 2, radiusM, segments)
}

/**
 * wallRadiusM — le rayon monde effectivement tracé : celui du plan, relevé au plancher de
 * lisibilité quand l'échelle de la carte le réduirait à une éraflure (cf. WALL_MIN_RADIUS_PX).
 */
export function wallRadiusM(view: PlacementView): number {
  const scale = canvasScale(view.bounds, view.width, view.height, view.pad)
  if (!(scale > 0)) return WALL_RADIUS_M
  return Math.max(WALL_RADIUS_M, WALL_MIN_RADIUS_PX / scale)
}

/** inkOf — la couleur d'équipe du poseur, ou le neutre quand la pose n'a pas de poseur connu. */
function inkOf(p: ReplayEquipmentPlacement, ink: PlacementInk): string {
  if (p.owner < 0) return ink.neutral
  return ink.colorOfSlot(p.owner) ?? ink.neutral
}

/** project — un point monde vers les pixels du canvas (la chaîne des tracks). */
function project(p: XY, view: PlacementView): XY {
  return worldToCanvas(p, view.bounds, view.width, view.height, view.pad)
}

/** strokePolyline trace une suite de points déjà projetés (le mur : arc ouvert ou cercle). */
function strokePolyline(ctx: CanvasRenderingContext2D, pts: XY[], closed: boolean): void {
  if (pts.length === 0) return
  ctx.beginPath()
  ctx.moveTo(pts[0].x, pts[0].y)
  for (let i = 1; i < pts.length; i++) ctx.lineTo(pts[i].x, pts[i].y)
  if (closed) ctx.closePath()
  ctx.stroke()
}

/**
 * drawWall — l'arc du mur (ou son cercle pointillé sans cap), halo puis trait franc.
 *
 * Le HALO est le même tracé, plus large et très transparent : sur un fond de carte chargé, un
 * trait de 2 px se perd dans le décor ; sur un fond clair, il se perd dans le blanc. Le halo
 * lui donne son assise sans rien ajouter à ce qu'il affirme.
 */
function drawWall(
  ctx: CanvasRenderingContext2D,
  p: ReplayEquipmentPlacement,
  view: PlacementView,
  time: PlacementTime,
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
  if (closed) ctx.setLineDash(RING_DASH.map((d) => d * time.k))
  ctx.globalAlpha = WALL_HALO_ALPHA
  ctx.lineWidth = WALL_HALO_WIDTH * time.k
  strokePolyline(ctx, pts, closed)
  ctx.globalAlpha = WALL_ALPHA
  ctx.lineWidth = WALL_LINE_WIDTH * time.k
  strokePolyline(ctx, pts, closed)
  ctx.restore()
}

/**
 * drawSensor — la ZONE du capteur (disque à alpha très bas, borné par son anneau) et, quand
 * une onde est en cours, LE PING : un anneau parti du centre qui s'ouvre jusqu'au rayon en
 * s'effaçant.
 *
 * LA ZONE NE BOUGE PLUS, seule l'onde bouge : c'est ce qui distingue un capteur qui PINGE d'un
 * capteur qui respire. Le disque dit la portée officielle, à demeure ; l'onde dit l'instant.
 *
 * `ctx.arc` sur le centre PROJETÉ et le rayon à l'échelle : la projection du rejeu est une
 * similitude (même facteur en X et en Y, cf. worldToCanvas), donc un cercle monde reste un
 * cercle à l'écran. Ce qui se calcule en monde, c'est ce qui porte une ORIENTATION — le mur.
 */
function drawSensor(
  ctx: CanvasRenderingContext2D,
  p: ReplayEquipmentPlacement,
  view: PlacementView,
  time: PlacementTime,
  color: string,
): void {
  const c = project({ x: p.x, y: p.y }, view)
  const scale = canvasScale(view.bounds, view.width, view.height, view.pad)
  const r = SENSOR_RADIUS_M * scale
  if (!(r > 0)) return
  ctx.save()
  ctx.fillStyle = color
  ctx.strokeStyle = color
  ctx.globalAlpha = SENSOR_FILL_ALPHA
  ctx.beginPath()
  ctx.arc(c.x, c.y, r, 0, Math.PI * 2)
  ctx.fill()
  ctx.globalAlpha = SENSOR_RING_ALPHA
  ctx.lineWidth = SENSOR_RING_WIDTH * time.k
  ctx.beginPath()
  ctx.arc(c.x, c.y, r, 0, Math.PI * 2)
  ctx.stroke()
  const wave = sensorPing((time.frame - p.t0) * time.frameMs, time.reducedMotion)
  if (wave && wave.reach > 0) {
    ctx.globalAlpha = wave.alpha
    ctx.lineWidth = SENSOR_PING_WIDTH * time.k
    ctx.beginPath()
    ctx.arc(c.x, c.y, r * wave.reach, 0, Math.PI * 2)
    ctx.stroke()
  }
  ctx.restore()
}

/**
 * drawReveal — la marque « révélé » d'UNE vie : un halo bas et son liseré, à la teinte de
 * l'équipe du POSEUR (c'est son camp qui voit, pas celui de la cible).
 *
 * Elle est peinte SOUS le marqueur du joueur — le calque des poses passe avant celui des
 * vies — donc elle l'entoure sans jamais le couvrir. Rayon en PIXELS et non en mètres : ce
 * n'est pas une portée, c'est un signe posé sur un point.
 */
function drawReveal(
  ctx: CanvasRenderingContext2D,
  reveal: SensorReveal,
  view: PlacementView,
  time: PlacementTime,
  color: string,
): void {
  const alpha = revealAlpha(reveal.sinceMs, time.reducedMotion)
  if (!(alpha > 0)) return
  const c = project({ x: reveal.x, y: reveal.y }, view)
  const r = REVEAL_RADIUS_PX * time.k
  ctx.save()
  ctx.fillStyle = color
  ctx.strokeStyle = color
  ctx.globalAlpha = alpha * REVEAL_FILL_ALPHA
  ctx.beginPath()
  ctx.arc(c.x, c.y, r, 0, Math.PI * 2)
  ctx.fill()
  ctx.globalAlpha = alpha
  ctx.lineWidth = REVEAL_WIDTH * time.k
  ctx.beginPath()
  ctx.arc(c.x, c.y, r, 0, Math.PI * 2)
  ctx.stroke()
  ctx.restore()
}

/**
 * drawUnnamed — le point neutre d'un objet dont la nature n'est pas établie. Aucune forme
 * empruntée au mur ni au capteur : ce marqueur dit « un objet d'équipement est ici », et il
 * n'a pas le droit d'en dire plus.
 */
function drawUnnamed(
  ctx: CanvasRenderingContext2D,
  p: ReplayEquipmentPlacement,
  view: PlacementView,
  time: PlacementTime,
  color: string,
): void {
  const c = project({ x: p.x, y: p.y }, view)
  ctx.save()
  ctx.globalAlpha = UNNAMED_ALPHA
  ctx.fillStyle = color
  ctx.beginPath()
  ctx.arc(c.x, c.y, UNNAMED_DOT_RADIUS_PX * time.k, 0, Math.PI * 2)
  ctx.fill()
  ctx.restore()
}

/**
 * drawEquipmentPlacementsLayer trace les poses ACTIVES à l'image courante, puis les marques de
 * révélation que les pings de capteur produisent à cet instant.
 *
 * La fenêtre dessinée n'est PAS [t0, t1] : `t1` date la mise au repos de l'objet, pas sa
 * disparition, que le film ne porte pas (cf. placementEndFrame). Le capteur se tient à sa durée
 * officielle, les autres poses restent jusqu'à la fin du rejeu.
 *
 * LES MARQUES PASSENT APRÈS TOUTES LES POSES, et jamais dans la boucle : une marque est un
 * signe posé sur un JOUEUR, elle doit se lire par-dessus les zones — y compris celle d'un
 * second capteur qui recouvrirait le premier.
 */
export function drawEquipmentPlacementsLayer(
  ctx: CanvasRenderingContext2D,
  scene: PlacementScene,
  view: PlacementView,
  time: PlacementTime,
  ink: PlacementInk,
): void {
  const sensors: ReplayEquipmentPlacement[] = []
  for (const p of scene.placements) {
    const kind = placementKind(p, time.showUnnamed)
    if (!kind || !isPlacementActive(p, kind, time.frame, time)) continue
    const color = inkOf(p, ink)
    if (kind === 'wall') drawWall(ctx, p, view, time, color)
    else if (kind === 'sensor') {
      drawSensor(ctx, p, view, time, color)
      sensors.push(p)
    } else drawUnnamed(ctx, p, view, time, color)
  }
  if (sensors.length === 0) return
  for (const reveal of sensorReveals(sensors, scene, time)) {
    drawReveal(ctx, reveal, view, time, ink.colorOfSlot(reveal.owner) ?? ink.neutral)
  }
}

/** Rayon de la zone sensible au survol d'une pose, en pixels d'écran. */
function hoverRadiusPx(kind: PlacementKind, view: PlacementView): number {
  const scale = canvasScale(view.bounds, view.width, view.height, view.pad)
  if (kind === 'sensor') return SENSOR_RADIUS_M * scale
  const r = kind === 'wall' ? wallRadiusM(view) * scale : UNNAMED_DOT_RADIUS_PX
  return Math.max(r, HOVER_MIN_RADIUS_PX)
}

/**
 * placementAt — la pose sous le curseur, à l'image courante, ou null.
 *
 * LA PLUS PETITE GAGNE, et c'est ce qui rend le survol utilisable : le disque du capteur
 * couvre 4,25 m de rayon, un mur posé dedans en couvre 1,6. Prendre la première trouvée
 * montrerait toujours le capteur, et le mur serait inatteignable au pointeur.
 *
 * `point` est en pixels CSS du canvas, le même repère que le dessin (le facteur de densité est
 * déjà absorbé par la transformation du contexte).
 */
export function placementAt(
  placements: readonly ReplayEquipmentPlacement[],
  view: PlacementView,
  time: Pick<PlacementTime, 'frame' | 'showUnnamed' | 'frameMs' | 'frames'>,
  point: XY,
): ReplayEquipmentPlacement | null {
  let best: ReplayEquipmentPlacement | null = null
  let bestR = Infinity
  for (const p of placements) {
    const kind = placementKind(p, time.showUnnamed)
    if (!kind || !isPlacementActive(p, kind, time.frame, time)) continue
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
