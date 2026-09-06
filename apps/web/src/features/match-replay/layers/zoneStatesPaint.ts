/**
 * zoneStatesPaint.ts — COMMENT une zone vivante se peint : l'appartenance, la progression de
 * capture, et l'échelle d'opacités qui les hiérarchise.
 *
 * POURQUOI CE FICHIER EXISTE (2026-08-25, item D-R). `zoneStatesLayer.ts` décide CE QU'IL FAUT
 * peindre — il lit l'intervalle qui couvre la frame, la valeur de la jauge, la lettre de la
 * zone. Ce fichier-ci décide COMMENT : quelles opacités, dans quel ordre, avec quel découpage.
 * La frontière n'est pas cosmétique : le calque a grossi de 417 à 568 lignes en passant de l'arc
 * extérieur au remplissage de la forme, c'est-à-dire au-dessus du seuil du dépôt, et la règle
 * est d'EXTRAIRE plutôt que d'exempter. Le découpage suit la couture naturelle — lecture d'état
 * d'un côté, primitives de rendu de l'autre — et non un partage arbitraire pour tenir un compte
 * de lignes.
 *
 * LA GÉOMÉTRIE N'EST PAS ICI, ET NE DOIT JAMAIS Y ENTRER. Toute forme vient de `traceZonePath`
 * (`objectivesLayer.ts`), et l'emprise écran de `zoneCornersWorld` / `zoneCanvasRadius`, les
 * deux helpers que ce même fichier exporte pour ça. Une seconde formule de forme divergerait au
 * premier correctif, et l'écart serait invisible parce que crédible.
 *
 * LES ENCRES ARRIVENT RÉSOLUES (règle color-tokens) : ce fichier n'en choisit aucune, il ne
 * choisit que des OPACITÉS et des épaisseurs. Aucune valeur de couleur n'y est écrite.
 */
import {
  traceZonePath,
  zoneCanvasRadius,
  zoneCornersWorld,
  type ObjectiveElementReady,
} from './objectivesLayer'
import type { XY } from '../replayLogic'

/**
 * ZoneStateNow — ce qu'une zone montre à une frame donnée.
 *
 * `owner` vaut `null` quand PERSONNE ne la tient : c'est une mesure (la valeur neutre du canal
 * de propriété), pas une absence de donnée — d'où le champ, plutôt qu'un état omis. Le sommet
 * `progress` de l'intervalle n'y figure plus : le rendu ne le lit plus.
 *
 * IL VIT ICI, ET PAS DANS LE CALQUE, pour que la dépendance reste À SENS UNIQUE : le calque
 * importe la peinture, jamais l'inverse. Le type que les deux partagent appartient donc au
 * fichier importé.
 */
export interface ZoneStateNow {
  owner: number | null
  active: boolean
}

// Réglages du calque vivant. Plus francs que le calque statique (qui reste dessous) : c'est le
// contraste entre les deux qui fait lire la bascule, et les zones sans état courant gardent
// leur trait faible — elles paraissent estompées sans qu'on ait à les repeindre.
//
// LES DEUX REMPLISSAGES D'APPARTENANCE ONT ÉTÉ RENFORCÉS le 2026-08-25 (item D-R, retour
// utilisateur « la teinte est trop discrète ») : tenue 0,22 -> 0,30, active 0,30 -> 0,42.
// L'écart ENTRE LES DEUX se creuse en même temps qu'ils montent — sans quoi renforcer la zone
// tenue aurait effacé ce qui distingue la colline active, qui est le repère le plus utile de la
// carte. L'échelle complète et son invariant sont testés (`ZONE_ALPHA_ORDER`).
const ZONE_HELD_FILL_ALPHA = 0.3
const ZONE_HELD_STROKE_ALPHA = 0.95
const ZONE_HELD_STROKE_WIDTH = 2.5
const ZONE_ACTIVE_FILL_ALPHA = 0.42
const ZONE_ACTIVE_STROKE_WIDTH = 3.5
/**
 * ZONE NON PRISE : LE CONTOUR GRISÉ (demande utilisateur du 2026-08-25, « bases non prises :
 * contour grisé »).
 *
 * L'ENCRE ÉTAIT DÉJÀ LA BONNE — le neutre du thème — mais le TRAIT était celui d'une zone
 * tenue : même opacité (0,95), même épaisseur (2,5). Une base que personne ne tient
 * s'affirmait donc aussi fort qu'une base gagnée, et ne se distinguait que par sa teinte et
 * l'absence de remplissage. Elle est maintenant EN RETRAIT : plus fine, plus transparente.
 * « Personne » est un état faible, il se dessine faible.
 *
 * LE SEUIL EST `owner === null`, PAS `held` — et la nuance est du sens. `held` est faux dans
 * DEUX cas : personne ne tient la zone (une MESURE du film), ou bien quelqu'un la tient mais
 * la page ne sait pas situer son camp (aucune ligne « moi » au tableau de bord). Le second
 * n'est pas une zone libre : la griser dirait au lecteur qu'elle est à prendre alors qu'elle
 * est tenue. Elle garde donc le trait plein, à l'encre neutre.
 */
const ZONE_FREE_STROKE_ALPHA = 0.5
const ZONE_FREE_STROKE_WIDTH = 1.6

/**
 * LA CAPTURE EN COURS : la progression de l'attaquant, et l'effacement du propriétaire.
 *
 * `ZONE_CAPTURE_FILL_ALPHA` est le plus fort remplissage du calque, et c'est voulu : pendant
 * une capture, ce qui compte à l'écran est CE QUI EST EN TRAIN DE CHANGER, pas ce qui est
 * acquis. `ZONE_UNDER_CAPTURE_FILL_ALPHA` est le plus faible, sous celui d'une zone tenue
 * ordinaire : le propriétaire recule pendant qu'on lui prend sa base. Les deux se lisent
 * ensemble — la part encore à lui, la part déjà prise —, ce qu'un arc extérieur ne pouvait pas
 * dire.
 *
 * UNE ZONE LIBRE EN COURS DE CAPTURE N'A PAS DE TEINTE DE PROPRIÉTAIRE À AFFAIBLIR : la
 * progression y est le SEUL remplissage, et son encre reste NEUTRE (le film ne dit pas qui
 * pousse la jauge, cf. `ZoneStateStyle.colorOfCapturer`).
 */
const ZONE_CAPTURE_FILL_ALPHA = 0.55
const ZONE_UNDER_CAPTURE_FILL_ALPHA = 0.16

/**
 * ZONE_ALPHA_ORDER — L'ÉCHELLE D'APPARTENANCE, du plus faible au plus fort, EXPORTÉE POUR ÊTRE
 * TESTÉE.
 *
 * POURQUOI CETTE CONSTANTE EXISTE. Ces cinq opacités ne valent que les unes par rapport aux
 * autres : c'est leur ORDRE qui porte la lecture (libre < en cours de perte < tenue < active <
 * progression), pas leurs valeurs absolues, et le gate visuel de l'utilisateur peut toutes les
 * bouger. Un réglage remonté seul casserait la lecture sans casser aucun test qui vise une
 * valeur — d'où un invariant d'ORDRE plutôt que cinq attentes numériques.
 *
 * La zone LIBRE y entre à zéro : elle n'a aucun remplissage, et c'est le bas de l'échelle.
 */
export const ZONE_ALPHA_ORDER = [
  0,
  ZONE_UNDER_CAPTURE_FILL_ALPHA,
  ZONE_HELD_FILL_ALPHA,
  ZONE_ACTIVE_FILL_ALPHA,
  ZONE_CAPTURE_FILL_ALPHA,
] as const

/** Ce que le tracé d'une zone vivante a besoin de savoir (règle des 5 paramètres). */
export interface ZonePaint {
  px: (p: XY) => XY
  scale: number
  ink: string
  /**
   * `false` = pas de camp À TEINTER : LISERÉ SEUL, aucun remplissage d'appartenance. Deux
   * situations distinctes le produisent — personne ne tient la zone, ou son camp n'est pas
   * situable (aucune ligne « moi ») — et seule la PREMIÈRE grise le contour (cf.
   * ZONE_FREE_STROKE_*, qui se décide sur `now.owner`, pas sur ce drapeau).
   */
  held: boolean
  /**
   * L'état de la zone à cette frame, ou `null` quand AUCUN intervalle ne la couvre. Le second
   * cas n'arrive que porté par une `capture` : une rampe de jauge qui précède la première
   * émission du canal de propriété. Rien n'y est peint que la progression — ni teinte ni
   * contour, faute d'appartenance à affirmer.
   */
  now: ZoneStateNow | null
  /** La valeur de la jauge à l'image, dans ]0, 1], ou `null` : aucune capture en cours. */
  capture: number | null
  /** L'encre de la progression : le camp d'en face, ou le neutre (cf. `colorOfCapturer`). */
  captureInk: string
}

/**
 * Zone tenue : remplissage + liseré à l'encre du camp. Zone active : les deux, renforcés.
 * Zone NON PRISE : contour GRISÉ, en retrait, sans remplissage (cf. ZONE_FREE_STROKE_*).
 * Zone EN COURS DE CAPTURE : la teinte du propriétaire s'efface et la progression de
 * l'attaquant remplit la forme par le bas (cf. `paintCaptureFill`).
 *
 * L'ORDRE DE PEINTURE EST LE SENS DE LECTURE, et il n'est pas interchangeable : appartenance,
 * puis progression, puis contour. La progression passe SUR la teinte parce que c'est elle
 * l'information neuve ; le contour passe en DERNIER parce qu'un liseré à demi recouvert par un
 * remplissage clippé à la même forme se lirait comme un trait plus fin — c'est-à-dire comme une
 * zone libre, exactement le contresens que le lot A venait de corriger.
 */
export function paintZoneState(
  ctx: CanvasRenderingContext2D,
  e: ObjectiveElementReady,
  p: ZonePaint,
): void {
  if (p.capture !== null) paintCaptureFill(ctx, e, p)
  else paintOwnerFill(ctx, e, p, false)
  if (p.now === null) return
  // « LIBRE » EST UNE MESURE, pas une absence : `owner === null` est la valeur neutre du canal
  // de propriété (cf. ZoneSpan.Owner côté Go). Une zone dont AUCUN intervalle ne couvre la
  // frame n'affiche que sa progression (ci-dessus) : « on ne sait pas » ne doit pas se lire
  // « personne ne la tient ».
  const free = p.now.owner === null && !p.now.active
  ctx.globalAlpha = free ? ZONE_FREE_STROKE_ALPHA : ZONE_HELD_STROKE_ALPHA
  ctx.strokeStyle = p.ink
  ctx.lineWidth = p.now.active
    ? ZONE_ACTIVE_STROKE_WIDTH
    : free
      ? ZONE_FREE_STROKE_WIDTH
      : ZONE_HELD_STROKE_WIDTH
  traceZonePath(ctx, e, p.px, p.scale)
  ctx.stroke()
}

/**
 * paintCaptureFill peint une zone EN COURS DE CAPTURE : la teinte du propriétaire AFFAIBLIE
 * d'abord, la progression de l'attaquant FRANCHE par-dessus.
 *
 * LES DEUX SONT ICI PARCE QU'ILS SE RÈGLENT L'UN PAR RAPPORT À L'AUTRE : l'effacement du
 * propriétaire n'a de sens qu'accompagné de ce qui le remplace. Le cas sans capture, lui, ne
 * pose que la teinte pleine — `paintOwnerFill` seul.
 */
function paintCaptureFill(
  ctx: CanvasRenderingContext2D,
  e: ObjectiveElementReady,
  p: ZonePaint,
): void {
  paintOwnerFill(ctx, e, p, true)
  const box = zoneCanvasBox(e, p.px, p.scale)
  const h = (box.bottom - box.top) * Math.min(Math.max(p.capture ?? 0, 0), 1)
  if (h <= 0) return
  // LE DÉCOUPAGE PAR LA FORME ELLE-MÊME : un rectangle en espace ÉCRAN, clippé par le contour
  // de la zone. C'est ce qui rend le remplissage agnostique de la famille — une boîte orientée
  // et un cylindre passent par le même `traceZonePath`, sans un seul cas particulier ici.
  ctx.save()
  traceZonePath(ctx, e, p.px, p.scale)
  ctx.clip()
  ctx.globalAlpha = ZONE_CAPTURE_FILL_ALPHA
  ctx.fillStyle = p.captureInk
  ctx.fillRect(box.left, box.bottom - h, box.right - box.left, h)
  ctx.restore()
}

/**
 * paintOwnerFill pose la teinte d'appartenance — AFFAIBLIE quand une capture est en cours.
 * Aucune teinte sur une zone libre ni sur un camp non situable : `held` le dit.
 */
function paintOwnerFill(
  ctx: CanvasRenderingContext2D,
  e: ObjectiveElementReady,
  p: ZonePaint,
  underCapture: boolean,
): void {
  if (p.now === null || !(p.held || p.now.active)) return
  traceZonePath(ctx, e, p.px, p.scale)
  ctx.globalAlpha = underCapture
    ? ZONE_UNDER_CAPTURE_FILL_ALPHA
    : p.now.active
      ? ZONE_ACTIVE_FILL_ALPHA
      : ZONE_HELD_FILL_ALPHA
  ctx.fillStyle = p.ink
  ctx.fill()
}

/** L'emprise ÉCRAN d'une zone : ce que le remplissage de capture balaie du bas vers le haut. */
interface ZoneCanvasBox {
  left: number
  right: number
  top: number
  bottom: number
}

/**
 * zoneCanvasBox rend l'emprise écran d'une zone, à partir de LA MÊME géométrie que le tracé —
 * `zoneCornersWorld` pour une boîte, `zoneCanvasRadius` pour un cylindre, tous deux exportés
 * par `objectivesLayer` exactement pour ça. Aucune formule de forme n'est recopiée ici.
 */
function zoneCanvasBox(
  e: ObjectiveElementReady,
  px: (p: XY) => XY,
  scale: number,
): ZoneCanvasBox {
  if (e.family === 'cylinder') {
    const c = px(e)
    const r = zoneCanvasRadius(e, scale)
    return { left: c.x - r, right: c.x + r, top: c.y - r, bottom: c.y + r }
  }
  const pts = zoneCornersWorld(e).map(px)
  const xs = pts.map((q) => q.x)
  const ys = pts.map((q) => q.y)
  return {
    left: Math.min(...xs),
    right: Math.max(...xs),
    top: Math.min(...ys),
    bottom: Math.max(...ys),
  }
}
