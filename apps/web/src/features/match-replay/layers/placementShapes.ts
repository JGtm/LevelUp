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

import { type XY } from '../model/replayLogic'
import {
  REVEAL_FILL_ALPHA,
  REVEAL_RADIUS_PX,
  REVEAL_WIDTH,
  revealAlpha,
  SENSOR_PING_ALPHA,
  SENSOR_SWEEP_MS,
} from '../model/threatSensor'
import { projectTo, scaleOf } from '../model/replayView'

/** Cadrage du canvas (mêmes paramètres que worldToCanvas). */
export interface PlacementView {
  bounds: ReplayBounds
  width: number
  height: number
  pad: number
}

/** project — un point monde vers les pixels du canvas (la chaîne des tracks). */
export function project(p: XY, view: PlacementView): XY {
  return projectTo(view, p)
}

/** viewScale — le facteur pixels par mètre du cadrage courant (0 si le cadrage est dégénéré). */
export function viewScale(view: PlacementView): number {
  return scaleOf(view)
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

// --- OBJET LÂCHÉ AU SOL -------------------------------------------------------------------

/** Rayon de l'anneau d'un objet lâché, en pixels d'écran. */
export const DROPPED_RADIUS_PX = 4.5

/**
 * L'ATTÉNUATION D'UN OBJET AU SOL. Elle est plus basse que celle du point non identifié (0,5) :
 * un lâcher est un fait certain mais SECONDAIRE, et il ne doit jamais prendre le pas sur le mur
 * ou le capteur déployés au même endroit.
 */
const DROPPED_ALPHA = 0.42

/** Épaisseur du liseré, à la densité de l'écran (même règle que les autres traits du calque). */
const DROPPED_LINE_WIDTH = 1.25

/**
 * drawDroppedObject — l'objet de puissance tombé au sol : un anneau POINTILLÉ, atténué, vide.
 *
 * TROIS CHOIX, ET CHACUN DIT LA MÊME CHOSE — l'objet est là, il n'agit pas.
 *  - POINTILLÉ : c'est la grammaire du calque (cf. UNCERTAIN_DASH), employée ici pour ce
 *    qu'elle sait déjà dire — la limite n'est pas affirmée, l'objet n'exerce aucune portée ;
 *  - VIDE (aucun remplissage) : les formes pleines de ce calque disent une ZONE d'effet
 *    (disque du capteur, champ de réparation). Un objet au sol n'en a aucune ;
 *  - UNE SEULE FORME pour toutes les familles : reprendre l'arc du mur ou le disque du capteur
 *    affirmerait un déploiement qui n'a pas eu lieu — la mesure dit l'inverse (un mur lâché
 *    porte l'identifiant de l'APPAREIL, jamais celui des panneaux).
 *
 * La famille, elle, se lit au survol : c'est l'infobulle qui nomme, jamais la forme.
 */
export function drawDroppedObject(
  ctx: CanvasRenderingContext2D,
  c: XY,
  style: ShapeStyle,
  color: string,
): void {
  const r = DROPPED_RADIUS_PX * style.k
  if (!(r > 0)) return
  ctx.save()
  ctx.globalAlpha = DROPPED_ALPHA
  ctx.strokeStyle = color
  ctx.lineWidth = DROPPED_LINE_WIDTH * style.k
  ctx.setLineDash(UNCERTAIN_DASH.map((d) => d * style.k))
  ctx.beginPath()
  ctx.arc(c.x, c.y, r, 0, Math.PI * 2)
  ctx.stroke()
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
 * LA CROIX DE PHARMACIE (V8, demande utilisateur du 2026-08-18 : « comme c'est un truc pour
 * la santé tu peux mettre une croix comme une croix de pharmacie qui pulse ? avec le cercle
 * autour évidemment »).
 *
 * CE QUE LA PULSATION DIT, ET CE QU'ELLE NE DIT PAS. Elle ne compte RIEN : le film ne porte
 * aucune cadence de soin, et prétendre le contraire serait inventer une mesure. Elle ne touche
 * donc QUE la croix — le disque et son anneau pointillé restent immobiles, et c'est eux qui
 * portent l'information (la zone, et la réserve sur sa portée). La croix respire parce qu'un
 * objet qui soigne est ACTIF, et une respiration lente est le seul signe d'activité qui ne
 * chiffre rien.
 *
 * SOUS MOUVEMENT RÉDUIT ELLE NE RESPIRE PAS, mais elle reste : la famille ne doit pas devenir
 * invisible pour qui demande moins de mouvement (même règle que l'onde du traqueur).
 */
const REPAIR_CROSS_PERIOD_MS = 1_800
/** Demi-longueur de la croix, en fraction du rayon du champ. */
const REPAIR_CROSS_REACH = 0.42
/** Épaisseur de la branche, même fraction : la croix garde ses proportions à toute échelle. */
const REPAIR_CROSS_ARM = 0.15
/** Croix la plus petite qui reste lisible, en pixels d'écran (BTB : 2,5 px/m). */
const REPAIR_CROSS_MIN_PX = 3.5
const REPAIR_CROSS_ALPHA_LOW = 0.55
const REPAIR_CROSS_ALPHA_HIGH = 0.95

/**
 * repairCrossAlpha — l'opacité de la croix à cet âge : une respiration lente entre deux bornes.
 *
 * FONCTION DU TEMPS, PAS ANIMATION À ÉTAT (même règle que `sensorPing` et `seekerImpulse`) : le
 * même âge redonne toujours la même image, donc un retour en arrière rejoue ce qu'on avait vu.
 * Sous mouvement réduit, la valeur haute, constante.
 */
export function repairCrossAlpha(ageMs: number, reducedMotion: boolean): number {
  if (reducedMotion) return REPAIR_CROSS_ALPHA_HIGH
  const phase = (1 - Math.cos((2 * Math.PI * ageMs) / REPAIR_CROSS_PERIOD_MS)) / 2
  return REPAIR_CROSS_ALPHA_LOW + (REPAIR_CROSS_ALPHA_HIGH - REPAIR_CROSS_ALPHA_LOW) * phase
}

/** Ce qu'il faut savoir d'un champ pour le tracer : où, jusqu'où, et depuis quand. */
export interface RepairFieldShape {
  c: XY
  /** Rayon DÉJÀ converti en pixels : la conversion appartient au calque, seul à voir le cadrage. */
  radiusPx: number
  /** Âge de la pose, en millisecondes de match — il donne sa phase à la respiration. */
  ageMs: number
}

/**
 * drawRepairField — le disque du champ, à l'encre de l'équipe du poseur, borné d'un pointillé,
 * et LA CROIX DE PHARMACIE qui respire en son centre (cf. REPAIR_CROSS_PERIOD_MS).
 *
 * LA ZONE NE BOUGE PAS, SEULE LA CROIX RESPIRE. C'est la même distinction que pour le capteur —
 * ce qui affirme une mesure est immobile, ce qui bouge ne chiffre rien. Le disque dit la portée
 * (déclarée, d'où le pointillé), la croix dit « ici, on se soigne ».
 */
export function drawRepairField(
  ctx: CanvasRenderingContext2D,
  field: RepairFieldShape,
  style: ShapeStyle,
  color: string,
): void {
  const { c, radiusPx } = field
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
  // LA CROIX : trait plein, à l'intérieur du cercle, jamais au-delà de sa borne.
  ctx.setLineDash([])
  ctx.globalAlpha = repairCrossAlpha(field.ageMs, style.reducedMotion)
  const reach = Math.max(radiusPx * REPAIR_CROSS_REACH, REPAIR_CROSS_MIN_PX * style.k)
  const arm = Math.max(radiusPx * REPAIR_CROSS_ARM, REPAIR_CROSS_MIN_PX * style.k * 0.36)
  ctx.beginPath()
  ctx.rect(c.x - reach, c.y - arm, reach * 2, arm * 2)
  ctx.rect(c.x - arm, c.y - reach, arm * 2, reach * 2)
  ctx.fill()
  ctx.restore()
}

// La FAILLE du translocateur et son arc de téléportation vivent dans `placementRift.ts`.

// --- ÉCRAN OCCULTANT (shroud screen) --------------------------------------------------------

/**
 * Rayon de l'écran occultant, en mètres monde. VALEUR DÉCLARÉE, PAS MESURÉE, PAS OFFICIELLE —
 * même statut que celui du champ de réparation, et il faut le dire aussi franchement.
 *
 * LES TROIS SOURCES SONT VIDES, ET C'EST LA RAISON D'ÊTRE DU BORD FLOU. Le film ne porte
 * aucune portée pour cet objet ; la source officielle qui chiffre le détecteur (4,25 m) ne
 * chiffre pas cet écran ; et rien n'a été mesuré dans le corpus. 6 m — 12 m de diamètre — est
 * un choix d'ÉCRAN, calibré sur ce que la bulle du jeu couvre en pratique (un couloir, un
 * passage étroit), et volontairement DEUX FOIS le champ de réparation pour que les trois
 * disques du calque (capteur 4,25 m, champ 3 m, écran 6 m) ne se confondent jamais.
 *
 * LE BORD FLOU PORTE CETTE RÉSERVE À L'ÉCRAN, comme le pointillé la porte pour le champ. Deux
 * conventions différentes pour la même réserve, et c'est voulu : le champ a une BORNE dont on
 * doute (un anneau, en pointillé), l'écran n'a pas de borne du tout (un dégradé, sans anneau).
 * Une limite nette dirait « au-delà de ce trait on se voit », ce que personne n'a établi.
 */
export const SHROUD_RADIUS_M = 6

/** Part du rayon occupée par le fondu du bord (verdict R2-6 : 22 %). */
const SHROUD_FADE_RATIO = 0.22
/** Opacité du cœur. Presque plein : le verdict du 2026-08-27 est « opaque ». */
const SHROUD_CORE_ALPHA = 0.94

/**
 * drawShroud — LA BULLE de l'écran occultant.
 *
 * OPAQUE, ET LES PIONS AU-DESSUS (verdict utilisateur du 2026-08-27, parmi trois propositions
 * mises côte à côte sur la planche). Ce que ce choix règle : l'opacité dit « ici, on ne se voit
 * pas » — c'est la fonction même de l'objet, et une bulle semi-transparente l'aurait affadie en
 * laissant transparaître le décor. Ce qu'il évite : perdre les joueurs sous la bulle, car un
 * rejeu dont on perd les pions ne se lit plus.
 *
 * LES PIONS PASSENT AU-DESSUS SANS UNE LIGNE DE CODE ICI, et c'est pour cela que la variante
 * était réalisable telle quelle : le calque des poses est tracé AVANT celui des pistes
 * (`drawEquipmentPlacementsLayer` puis `drawTracksLayer`). L'ordre du composant fait le travail
 * — les deux autres variantes, elles, auraient exigé de tracer l'écran APRÈS les pistes.
 *
 * PAS D'ANIMATION : rien dans le film ne bat au rythme de cet objet. Sa fenêtre d'activité est
 * portée par `isPlacementActive`, comme pour toutes les poses.
 */
export function drawShroud(
  ctx: CanvasRenderingContext2D,
  c: XY,
  radiusPx: number,
  color: string,
): void {
  if (!(radiusPx > 0)) return
  const plein = radiusPx * (1 - SHROUD_FADE_RATIO)
  ctx.save()
  ctx.fillStyle = color
  ctx.globalAlpha = SHROUD_CORE_ALPHA
  ctx.beginPath()
  ctx.arc(c.x, c.y, plein, 0, Math.PI * 2)
  ctx.fill()
  // Le fondu : du cœur plein jusqu'au néant, sans jamais poser de trait. C'est l'ABSENCE
  // d'anneau qui dit que la portée n'est pas connue.
  const g = ctx.createRadialGradient(c.x, c.y, plein, c.x, c.y, radiusPx)
  g.addColorStop(0, color)
  g.addColorStop(1, 'transparent')
  ctx.fillStyle = g
  ctx.beginPath()
  ctx.arc(c.x, c.y, radiusPx, 0, Math.PI * 2)
  ctx.fill()
  ctx.restore()
}
