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
 * LE CAPTEUR PULSE, ET SA PULSATION EST UNE FONCTION DU TEMPS (patron `explosionFx`) : la
 * lecture avance, recule, saute au curseur — une animation à état afficherait n'importe quoi
 * après un retour en arrière. Sous `prefers-reduced-motion`, la phase est figée.
 *
 * Pas de React : géométrie pure + un CanvasRenderingContext2D (même règle que replayDraw).
 */
import type { ReplayBounds, ReplayEquipmentPlacement } from '@/lib/api/types'

import { canvasScale, worldToCanvas, type XY } from './replayLogic'

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
 * Rayon du CAPTEUR, en mètres monde. VALEUR DÉCLARÉE, ET IL FAUT LE DIRE : le film ne porte
 * AUCUNE portée de détection — ni dans le record de création, ni dans la vie de l'objet. 8 m
 * est un choix d'écran : assez large pour se lire comme une ZONE (et non comme un second
 * marqueur ponctuel), assez sobre pour ne pas avaler une arène de 50 m. Le jour où une portée
 * serait mesurée, elle remplacerait cette constante — et la remplacerait seule.
 */
export const SENSOR_RADIUS_M = 8

/** Période de la pulsation du capteur, en millisecondes (lente, elle respire). */
export const SENSOR_PULSE_MS = 1_600

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

/** Remplissage du capteur : très bas — c'est une zone, elle ne doit rien masquer. */
const SENSOR_FILL_ALPHA = 0.1
const SENSOR_RING_ALPHA = 0.55
const SENSOR_RING_WIDTH = 1.25
/** Amplitude de la respiration : le rayon oscille de ±6 % autour du rayon déclaré. */
const SENSOR_PULSE_AMPLITUDE = 0.06

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
  /** Densité de pixels : les épaisseurs d'écran la suivent (même règle que les marqueurs). */
  k: number
  reducedMotion: boolean
  /** Les objets non identifiés se dessinent-ils ? Bascule du tiroir, ÉTEINTE par défaut. */
  showUnnamed: boolean
}

/** Les encres du calque : la couleur d'équipe du poseur, et le neutre quand il n'y en a pas. */
export interface PlacementInk {
  /** Couleur de la vie qui a posé l'objet ; null = vie inconnue, on tombe sur `neutral`. */
  colorOfSlot: (slot: number) => string | null
  /** Encre neutre du thème (`--muted-foreground`), servie aux poses sans poseur. */
  neutral: string
}

/** isPlacementActive — la pose existe-t-elle à cette image ? La fenêtre est EXACTE. */
export function isPlacementActive(p: ReplayEquipmentPlacement, frame: number): boolean {
  return frame >= p.t0 && frame <= p.t1
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

/**
 * sensorPulse — le facteur de rayon de la respiration du capteur, à l'âge donné.
 *
 * FONCTION DU TEMPS, PAS ANIMATION À ÉTAT : le même âge rend toujours la même valeur, donc un
 * retour en arrière rend exactement l'image qu'on avait vue. Sous mouvement réduit, la phase
 * est figée au rayon déclaré — le disque reste, il cesse de respirer.
 */
export function sensorPulse(ageMs: number, reducedMotion: boolean): number {
  if (reducedMotion) return 1
  const phase = (((ageMs % SENSOR_PULSE_MS) + SENSOR_PULSE_MS) % SENSOR_PULSE_MS) / SENSOR_PULSE_MS
  return 1 + SENSOR_PULSE_AMPLITUDE * Math.sin(phase * Math.PI * 2)
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
 * drawSensor — le disque pulsé : un remplissage à alpha très bas et un anneau qui le borde.
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
  const ageMs = (time.frame - p.t0) * time.frameMs
  const r = SENSOR_RADIUS_M * scale * sensorPulse(ageMs, time.reducedMotion)
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
 * drawEquipmentPlacementsLayer trace les poses ACTIVES à l'image courante.
 *
 * La fenêtre [t0, t1] est celle du document, sans rémanence : un mur qui expire disparaît à
 * l'image où sa vie s'arrête, exactement comme la ligne de grappin.
 */
export function drawEquipmentPlacementsLayer(
  ctx: CanvasRenderingContext2D,
  placements: readonly ReplayEquipmentPlacement[],
  view: PlacementView,
  time: PlacementTime,
  ink: PlacementInk,
): void {
  for (const p of placements) {
    if (!isPlacementActive(p, time.frame)) continue
    const kind = placementKind(p, time.showUnnamed)
    if (!kind) continue
    const color = inkOf(p, ink)
    if (kind === 'wall') drawWall(ctx, p, view, time, color)
    else if (kind === 'sensor') drawSensor(ctx, p, view, time, color)
    else drawUnnamed(ctx, p, view, time, color)
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
 * couvre 8 m, un mur posé dedans en couvre 1,6. Prendre la première trouvée montrerait
 * toujours le capteur, et le mur serait inatteignable au pointeur.
 *
 * `point` est en pixels CSS du canvas, le même repère que le dessin (le facteur de densité est
 * déjà absorbé par la transformation du contexte).
 */
export function placementAt(
  placements: readonly ReplayEquipmentPlacement[],
  view: PlacementView,
  time: Pick<PlacementTime, 'frame' | 'showUnnamed'>,
  point: XY,
): ReplayEquipmentPlacement | null {
  let best: ReplayEquipmentPlacement | null = null
  let bestR = Infinity
  for (const p of placements) {
    if (!isPlacementActive(p, time.frame)) continue
    const kind = placementKind(p, time.showUnnamed)
    if (!kind) continue
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
