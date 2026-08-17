/**
 * placementShapes.ts — LE SOCLE DU TRACÉ des poses d'équipement, et les FORMES qui tiennent
 * dans un CENTRE PROJETÉ.
 *
 * LA FRONTIÈRE EST GÉOMÉTRIQUE, PAS THÉMATIQUE, et c'est ce qui la rend tenable : ce fichier
 * porte le cadrage (`PlacementView`, `project`, `viewScale`), les primitives de trait, et les
 * formes qui n'ont besoin que d'un point déjà projeté (pour le disque, d'un rayon déjà converti
 * en pixels). Le MUR vit à part (`placementWall.ts`) parce qu'il se calcule en MONDE — il porte
 * une orientation, et une orientation n'a de sens qu'avant la projection.
 *
 * POURQUOI CE FICHIER EXISTE : le calque atteignait son seuil de taille (CLAUDE.md n°5) et ce
 * lot y ajoute trois familles. Plutôt qu'une exemption, la découpe — et elle tombe sur une
 * ligne qui se dit en une phrase.
 *
 * CE QUE CHAQUE FORME AFFIRME, ET CE QU'ELLE N'AFFIRME PAS. Aucune de ces formes ne porte
 * d'orientation : le film n'en publie aucune pour ces objets (le seul cap mesuré est celui du
 * REGARD du poseur, et il n'est exploité que par l'arc du mur). Le losange de la balise est
 * symétrique par construction — il ne pointe nulle part, et c'est délibéré.
 *
 * Pas de React, pas de couleur écrite : géométrie pure + un CanvasRenderingContext2D. L'encre
 * arrive du calque appelant, qui la tient des tokens du thème.
 */
import type { ReplayBounds } from '@/lib/api/types'

import { canvasScale, worldToCanvas, type XY } from './replayLogic'
import {
  REVEAL_FILL_ALPHA,
  REVEAL_RADIUS_PX,
  REVEAL_WIDTH,
  revealAlpha,
  SENSOR_PING_ALPHA,
  SENSOR_SWEEP_MS,
} from './threatSensor'

/** Cadrage du canvas (mêmes paramètres que worldToCanvas). */
export interface PlacementView {
  bounds: ReplayBounds
  width: number
  height: number
  pad: number
}

/** project — un point monde vers les pixels du canvas (la chaîne des tracks). */
export function project(p: XY, view: PlacementView): XY {
  return worldToCanvas(p, view.bounds, view.width, view.height, view.pad)
}

/** viewScale — le facteur pixels par mètre du cadrage courant (0 si le cadrage est dégénéré). */
export function viewScale(view: PlacementView): number {
  return canvasScale(view.bounds, view.width, view.height, view.pad)
}

/**
 * Ce que toute forme a besoin de savoir de l'ÉCRAN : la densité de pixels (les épaisseurs la
 * suivent, même règle que les marqueurs) et le respect du mouvement réduit.
 *
 * `PlacementTime` du calque en est un sur-ensemble : il se passe tel quel.
 */
export interface ShapeStyle {
  k: number
  reducedMotion: boolean
}

/**
 * LE POINTILLÉ DE L'INCERTITUDE — une grammaire, pas une décoration.
 *
 * Le calque l'emploie déjà pour le mur SANS CAP (« ici, un mur », sans dire dans quel sens) ;
 * le champ de réparation l'emploie pour sa BORNE (un rayon déclaré, pas mesuré). Dans les deux
 * cas le message est le même : cette limite n'est pas affirmée. Un trait plein, dans ce calque,
 * dit toujours une valeur qu'on tient d'une source (l'anneau du capteur : portée officielle).
 */
export const UNCERTAIN_DASH: readonly number[] = [3, 3]

/** strokePolyline trace une suite de points DÉJÀ projetés (arc ouvert, cercle, losange). */
export function strokePolyline(ctx: CanvasRenderingContext2D, pts: XY[], closed: boolean): void {
  if (pts.length === 0) return
  ctx.beginPath()
  ctx.moveTo(pts[0].x, pts[0].y)
  for (let i = 1; i < pts.length; i++) ctx.lineTo(pts[i].x, pts[i].y)
  if (closed) ctx.closePath()
  ctx.stroke()
}

// --- OBJET NON IDENTIFIÉ ------------------------------------------------------------------

/** Rayon du point neutre des objets non identifiés, en pixels d'écran. */
export const UNNAMED_DOT_RADIUS_PX = 2.5

const UNNAMED_ALPHA = 0.5

/**
 * drawUnnamedDot — le point neutre d'un objet dont la nature n'est pas établie. Aucune forme
 * empruntée aux familles nommées : ce marqueur dit « un objet d'équipement est ici », et il n'a
 * pas le droit d'en dire plus.
 */
export function drawUnnamedDot(
  ctx: CanvasRenderingContext2D,
  c: XY,
  style: ShapeStyle,
  color: string,
): void {
  ctx.save()
  ctx.globalAlpha = UNNAMED_ALPHA
  ctx.fillStyle = color
  ctx.beginPath()
  ctx.arc(c.x, c.y, UNNAMED_DOT_RADIUS_PX * style.k, 0, Math.PI * 2)
  ctx.fill()
  ctx.restore()
}

// --- MARQUE « RÉVÉLÉ » --------------------------------------------------------------------

/**
 * drawRevealMark — la marque « révélé » d'UNE vie : un halo bas et son liseré, à la teinte de
 * l'équipe du POSEUR du capteur (c'est son camp qui voit, pas celui de la cible).
 *
 * Elle est peinte SOUS le marqueur du joueur — le calque des poses passe avant celui des vies —
 * donc elle l'entoure sans jamais le couvrir. Rayon en PIXELS et non en mètres : ce n'est pas
 * une portée, c'est un signe posé sur un point.
 */
export function drawRevealMark(
  ctx: CanvasRenderingContext2D,
  c: XY,
  sinceMs: number,
  style: ShapeStyle,
  color: string,
): void {
  const alpha = revealAlpha(sinceMs, style.reducedMotion)
  if (!(alpha > 0)) return
  const r = REVEAL_RADIUS_PX * style.k
  ctx.save()
  ctx.fillStyle = color
  ctx.strokeStyle = color
  ctx.globalAlpha = alpha * REVEAL_FILL_ALPHA
  ctx.beginPath()
  ctx.arc(c.x, c.y, r, 0, Math.PI * 2)
  ctx.fill()
  ctx.globalAlpha = alpha
  ctx.lineWidth = REVEAL_WIDTH * style.k
  ctx.beginPath()
  ctx.arc(c.x, c.y, r, 0, Math.PI * 2)
  ctx.stroke()
  ctx.restore()
}

// --- BALISE DU TRANSLOCATEUR --------------------------------------------------------------

/**
 * Demi-diagonale du losange de la balise, en pixels d'ÉCRAN.
 *
 * EN PIXELS, ET C'EST UNE MESURE ABSENTE QU'ON REFUSE D'INVENTER : le film ne porte aucune
 * dimension de cet objet, et la balise n'a de toute façon pas de portée — c'est un point de
 * retour. Un rayon en mètres laisserait croire à une zone d'effet. 5,5 px la posent entre le
 * point neutre (2,5 px) et la marque de révélation (12 px) : lisible, jamais dominante.
 */
export const BEACON_RADIUS_PX = 5.5

/** Le cœur du losange : il fixe le LIEU exact, que le contour ne fait qu'entourer. */
const BEACON_DOT_RADIUS_PX = 1.5
const BEACON_LINE_WIDTH = 1.5
const BEACON_ALPHA = 0.85

/**
 * beaconDiamond — les quatre sommets du losange, en pixels d'écran.
 *
 * UN LOSANGE PARCE QU'IL EST LE SEUL À NE RESSEMBLER À RIEN D'AUTRE DANS CE CALQUE : le cercle
 * est pris (capteur, objet non identifié, marque de révélation), l'arc aussi (mur). Il est
 * symétrique par ses deux axes, donc il ne suggère aucune direction — ce qui est exactement ce
 * que la mesure autorise à dire d'une balise.
 */
export function beaconDiamond(c: XY, radiusPx: number): XY[] {
  return [
    { x: c.x, y: c.y - radiusPx },
    { x: c.x + radiusPx, y: c.y },
    { x: c.x, y: c.y + radiusPx },
    { x: c.x - radiusPx, y: c.y },
  ]
}

/**
 * drawBeacon — le losange de la balise et son cœur, à demeure sur toute la fenêtre [t0, t1].
 *
 * AUCUNE PULSATION : rien n'est mesuré qui batte. Une balise de translocation attend d'être
 * rappelée ; elle ne balaie pas, ne recharge pas, n'émet pas. Un marqueur qui clignoterait
 * affirmerait une activité qu'aucune lecture ne soutient.
 */
export function drawBeacon(
  ctx: CanvasRenderingContext2D,
  c: XY,
  style: ShapeStyle,
  color: string,
): void {
  ctx.save()
  ctx.globalAlpha = BEACON_ALPHA
  ctx.strokeStyle = color
  ctx.fillStyle = color
  ctx.lineJoin = 'round'
  ctx.lineWidth = BEACON_LINE_WIDTH * style.k
  strokePolyline(ctx, beaconDiamond(c, BEACON_RADIUS_PX * style.k), true)
  ctx.beginPath()
  ctx.arc(c.x, c.y, BEACON_DOT_RADIUS_PX * style.k, 0, Math.PI * 2)
  ctx.fill()
  ctx.restore()
}

// --- TRAQUEUR DE MENACES ------------------------------------------------------------------

/**
 * Durée de l'UNIQUE impulsion du traqueur, en millisecondes.
 *
 * DÉRIVÉE DE L'ONDE DU CAPTEUR, et c'est délibéré : les deux objets émettent le même genre
 * d'événement — une impulsion qui doit se lire comme un coup. Écrire une seconde constante
 * laisserait deux rythmes divergents pour un seul geste (CLAUDE.md n°6).
 */
export const SEEKER_IMPULSE_MS = SENSOR_SWEEP_MS

/**
 * Rayon atteint par l'impulsion, en pixels d'ÉCRAN — PAS en mètres, et c'est le point.
 *
 * LE JEU NE PUBLIE AUCUNE PORTÉE POUR CET OBJET. La source officielle qui chiffre le détecteur
 * de menaces (portée 4,25 wu, ping 1,8 s, révélation 0,75 s — cf. threatSensor.ts) dit du
 * traqueur une seule chose : « un projectile à UNE impulsion qui rebondit ». Un rayon en mètres
 * serait donc une portée inventée, et le calque en dessinerait la zone. 14 px passent juste au
 * large de la marque de révélation (12 px) : l'événement se distingue de la marque qu'il
 * pourrait produire, sans prétendre couvrir un terrain.
 */
export const SEEKER_IMPULSE_RADIUS_PX = 14

/** Épaisseur de l'onde : franche, c'est un événement (même valeur que le ping du capteur). */
const SEEKER_IMPULSE_WIDTH = 2
const SEEKER_IMPULSE_ALPHA = SENSOR_PING_ALPHA

/** L'onde d'une impulsion : sa course (en fraction du rayon) et ce qu'il lui reste d'opacité. */
export interface ImpulseWave {
  reach: number
  alpha: number
}

/**
 * seekerImpulseActive — l'impulsion est-elle en cours à cet âge ?
 *
 * ELLE NE SE REJOUE JAMAIS : une seule impulsion par pose, et après elle il ne reste RIEN — ni
 * zone, ni anneau, ni point. C'est la différence de nature avec le détecteur, qui pinge toutes
 * les 1,8 s pendant toute sa durée. Le survol suit cette fenêtre (cf. `placementShows`) : on
 * n'inspecte pas ce qui n'est plus dessiné.
 */
export function seekerImpulseActive(ageMs: number): boolean {
  return ageMs >= 0 && ageMs < SEEKER_IMPULSE_MS
}

/**
 * seekerImpulse — l'onde de l'impulsion, ou null hors de sa fenêtre.
 *
 * FONCTION DU TEMPS, PAS ANIMATION À ÉTAT (même règle que `sensorPing` et `explosionFx`) : le
 * même âge rend toujours la même image, donc un retour en arrière rejoue ce qu'on avait vu.
 * L'ouverture suit un easeOutCubic — l'onde part vite puis se calme au bord, le patron d'une
 * impulsion.
 *
 * SOUS MOUVEMENT RÉDUIT, L'ONDE NE COURT PAS MAIS L'ÉVÉNEMENT RESTE : un anneau immobile au
 * rayon plein, pendant la même fenêtre. Supprimer l'onde sans rien mettre à sa place rendrait
 * la famille entièrement invisible pour qui demande moins de mouvement.
 */
export function seekerImpulse(ageMs: number, reducedMotion: boolean): ImpulseWave | null {
  if (!seekerImpulseActive(ageMs)) return null
  if (reducedMotion) return { reach: 1, alpha: SEEKER_IMPULSE_ALPHA }
  const pr = ageMs / SEEKER_IMPULSE_MS
  return { reach: 1 - Math.pow(1 - pr, 3), alpha: (1 - pr) * SEEKER_IMPULSE_ALPHA }
}

/** drawSeekerImpulse — l'anneau de l'unique impulsion ; rien du tout en dehors d'elle. */
export function drawSeekerImpulse(
  ctx: CanvasRenderingContext2D,
  c: XY,
  ageMs: number,
  style: ShapeStyle,
  color: string,
): void {
  const wave = seekerImpulse(ageMs, style.reducedMotion)
  if (!wave || !(wave.reach > 0)) return
  ctx.save()
  ctx.strokeStyle = color
  ctx.globalAlpha = wave.alpha
  ctx.lineWidth = SEEKER_IMPULSE_WIDTH * style.k
  ctx.beginPath()
  ctx.arc(c.x, c.y, SEEKER_IMPULSE_RADIUS_PX * style.k * wave.reach, 0, Math.PI * 2)
  ctx.stroke()
  ctx.restore()
}

// --- CHAMP DE RÉPARATION ------------------------------------------------------------------

/**
 * Rayon du champ de réparation, en mètres monde. VALEUR DÉCLARÉE, PAS MESURÉE, PAS OFFICIELLE.
 *
 * LES TROIS SOURCES POSSIBLES SONT VIDES, ET IL FAUT LE DIRE : le film ne porte aucune portée
 * (ni dans le record de création, ni dans la vie de l'objet — même constat que pour le
 * détecteur) ; la source officielle qui chiffre le détecteur ne chiffre pas ce champ ; et rien
 * n'a été mesuré dans le corpus. 3 m — 6 m de diamètre — est un choix d'ÉCRAN, calibré sur ce
 * que le dôme du jeu abrite (un joueur et un coéquipier), et il reste sous la portée officielle
 * du détecteur (4,25 m) pour que les deux disques ne se confondent pas.
 *
 * LE POINTILLÉ DE SA BORNE PORTE CETTE RÉSERVE À L'ÉCRAN (cf. UNCERTAIN_DASH) : l'anneau du
 * capteur est plein parce que sa portée est publiée, celui du champ est pointillé parce que la
 * sienne ne l'est pas. Le jour où une portée serait mesurée ou publiée, elle remplacerait cette
 * constante — et le trait deviendrait plein.
 */
export const REPAIR_FIELD_RADIUS_M = 3

/** Remplissage du disque : très bas — c'est une zone, elle ne doit rien masquer. */
const REPAIR_FIELD_FILL_ALPHA = 0.12
const REPAIR_FIELD_RING_ALPHA = 0.5
const REPAIR_FIELD_RING_WIDTH = 1.25

/**
 * drawRepairField — le disque du champ, à l'encre de l'équipe du poseur, borné d'un pointillé.
 *
 * AUCUNE ONDE, AUCUNE RESPIRATION : le champ soigne en continu, il n'émet pas d'événement. Ce
 * qui bouge dans ce calque signale une impulsion mesurée (le ping du capteur, l'impulsion du
 * traqueur) ; un champ qui pulserait inventerait une cadence.
 *
 * `radiusPx` arrive DÉJÀ converti : la conversion mètres -> pixels appartient au calque, qui
 * seul connaît le cadrage.
 */
export function drawRepairField(
  ctx: CanvasRenderingContext2D,
  c: XY,
  radiusPx: number,
  style: ShapeStyle,
  color: string,
): void {
  if (!(radiusPx > 0)) return
  ctx.save()
  ctx.fillStyle = color
  ctx.strokeStyle = color
  ctx.globalAlpha = REPAIR_FIELD_FILL_ALPHA
  ctx.beginPath()
  ctx.arc(c.x, c.y, radiusPx, 0, Math.PI * 2)
  ctx.fill()
  ctx.globalAlpha = REPAIR_FIELD_RING_ALPHA
  ctx.lineWidth = REPAIR_FIELD_RING_WIDTH * style.k
  ctx.setLineDash(UNCERTAIN_DASH.map((d) => d * style.k))
  ctx.beginPath()
  ctx.arc(c.x, c.y, radiusPx, 0, Math.PI * 2)
  ctx.stroke()
  ctx.restore()
}
