/**
 * killFocusLayer.ts — LA MISE EN VALEUR D'UNE ÉLIMINATION SUR LA CARTE : le voile, les équerres qui
 * se verrouillent sur le tueur et sur la victime, et leurs étiquettes de nom.
 *
 * DEUX PEINTRES, DEUX PLACES DANS L'ORDRE DE LA SCÈNE (`replayCompose`). Le voile et les équerres
 * se posent AU-DESSUS des pions et des événements ; les étiquettes viennent après, par-dessus tout :
 * un nom de tueur ne doit jamais passer sous une équerre ou sous le voile. Le nom normal du pion
 * est retiré par le calque des joueurs (`MarkerStyle.nameHiddenAtSlot`) pendant la mise en valeur :
 * l'étiquette REMPLACE le nom, elle n'en est pas un second affichage.
 *
 * LES PROPORTIONS SONT CELLES DE LA MAQUETTE VALIDÉE. Voile noir à 10 %, avec un trou rond de 34 px
 * autour de chaque joueur (ils restent à leur éclat normal) ; équerres de 22 px de demi-côté et de
 * 8 px de branche, trait de 3 px, à la couleur du camp du joueur ; étiquette 13 px en graisse 900, à
 * 27 px sous le centre du pion, coin bas droit coupé. Tueur : étiquette PLEINE à sa couleur de camp,
 * texte à l'encre sombre d'étiquette, icône de l'arme. Victime : étiquette EN CREUX (fond sombre,
 * liseré et texte à sa couleur de camp) avec une croix — jamais la couleur seule qui dit « victime ».
 *
 * RIEN N'EST DESSINÉ À UNE POSITION INVENTÉE. Un acteur absent (non résolu) n'a ni équerre ni
 * étiquette ; un acteur HORS CADRE non plus — son pion est alors une flèche à la marge, qu'une
 * équerre hors de la toile ne désignerait pas. Le temps écoulé arrive en argument : ces gestes
 * suivent le temps RÉEL (la lecture est en pause), pas l'image du rejeu.
 *
 * Tailles en pixels d'ÉCRAN, multipliées par la densité `k` comme tous les calques. Aucune couleur
 * écrite ici : les encres arrivent du hook (`canvasInk`, tokens d'équipe).
 */
import type { XY } from '../../../lib/replay/replayLogic'
import { edgeMarkFor, OFFSCREEN_MARGIN_PX } from '../model/edgeClamp'
import type { FocusActor, ResolvedKillFocus } from '../model/killFocusLogic'
import { fadeOpacity, lockPose, popPose, type MotionPose } from '../model/killFocusMotion'
import { projectTo, type CanvasView } from '../model/replayView'

/** Force du voile : le reste de la carte s'assombrit à peine. */
export const VEIL_ALPHA = 0.1
/** Rayon du trou du voile autour d'un joueur. */
export const VEIL_HOLE_RADIUS_PX = 34
/** Demi-côté des équerres et longueur de leurs branches. */
export const BRACKET_HALF_PX = 22
export const BRACKET_ARM_PX = 8
const BRACKET_WIDTH_PX = 3

const LABEL_FONT_PX = 13
const LABEL_WEIGHT = 900
const LABEL_FAMILY = 'ui-sans-serif, system-ui, sans-serif'
/** Distance entre le centre du pion et le haut de l'étiquette, et hauteur de l'étiquette. */
const LABEL_OFFSET_PX = 27
const LABEL_HEIGHT_PX = 22
const LABEL_PAD_LEFT_PX = 7
const LABEL_PAD_RIGHT_PX = 9
const LABEL_GAP_PX = 6
const LABEL_BORDER_PX = 1.5
/** Le coin bas droit coupé : où la découpe commence sur le côté droit et sur le bas (fractions). */
const CUT_RIGHT = 0.72
const CUT_BOTTOM = 0.9
const WEAPON_ICON_W_PX = 22
const WEAPON_ICON_H_PX = 10
const CROSS_PX = 10
const CROSS_WIDTH_PX = 2

/** Les encres du calque : le voile, et l'encre sombre des étiquettes (fond en creux, texte plein). */
export interface KillFocusInk {
  veil: string
  label: string
}

/** Ce que les deux peintres reçoivent, en plus de la mise en valeur et de la vue. */
export interface KillFocusPaint {
  /** Densité du canevas. */
  k: number
  /** Temps RÉEL écoulé depuis le clic ; `FOCUS_MOTION_MS` ou plus = poses d'arrivée. */
  elapsedMs: number
  ink: KillFocusInk
}

/** Les étiquettes portent en plus l'icône de l'arme du tueur. */
export interface KillFocusLabelPaint extends KillFocusPaint {
  /** Déjà teintée à l'encre d'étiquette ; `null` = pas d'icône (arme non identifiée, pas chargée). */
  weaponIcon: (CanvasImageSource & { width: number; height: number }) | null
}

type Corner = [XY, XY, XY]

/**
 * bracketsPath : les quatre angles ouverts d'un carré de demi-côté `half` centré en (x, y), chacun
 * une polyligne de trois points — branche, sommet, branche. Le milieu de chaque côté reste vide :
 * ce sont des ÉQUERRES, pas un cadre.
 */
export function bracketsPath(x: number, y: number, half: number, arm: number): Corner[] {
  return [
    [{ x: x - half, y: y - half + arm }, { x: x - half, y: y - half }, { x: x - half + arm, y: y - half }],
    [{ x: x + half - arm, y: y - half }, { x: x + half, y: y - half }, { x: x + half, y: y - half + arm }],
    [{ x: x + half, y: y + half - arm }, { x: x + half, y: y + half }, { x: x + half - arm, y: y + half }],
    [{ x: x - half + arm, y: y + half }, { x: x - half, y: y + half }, { x: x - half, y: y + half - arm }],
  ]
}

interface OnScreen {
  actor: FocusActor
  at: XY
}

/** Les acteurs résolus ET dans le cadre, projetés sur la toile : les seuls qu'on désigne. */
function onScreenActors(focus: ResolvedKillFocus, view: CanvasView, k: number): OnScreen[] {
  const out: OnScreen[] = []
  for (const actor of [focus.killer, focus.victim]) {
    if (!actor || edgeMarkFor(actor.pos, view, OFFSCREEN_MARGIN_PX, k)) continue
    out.push({ actor, at: projectTo(view, actor.pos) })
  }
  return out
}

/** Pose le repère d'un élément animé : centré en `c`, tourné et mis à l'échelle autour de lui. */
function applyPose(ctx: CanvasRenderingContext2D, c: XY, pose: MotionPose): void {
  ctx.translate(c.x, c.y)
  ctx.rotate((pose.rotationDeg * Math.PI) / 180)
  ctx.scale(pose.scale, pose.scale)
  ctx.translate(-c.x, -c.y)
  ctx.globalAlpha = pose.opacity
}

/**
 * drawKillFocusVeil : le voile (trous autour des joueurs) puis les équerres, à chacun sa couleur.
 * `focus` nul, ou aucun acteur dans le cadre : rien.
 */
export function drawKillFocusVeil(
  ctx: CanvasRenderingContext2D,
  focus: ResolvedKillFocus | null,
  view: CanvasView,
  o: KillFocusPaint,
): void {
  if (!focus) return
  const actors = onScreenActors(focus, view, o.k)
  if (actors.length === 0) return
  ctx.save()
  // LE VOILE : un rectangle, percé d'un rond par joueur (règle `evenodd`).
  ctx.beginPath()
  ctx.rect(0, 0, view.width, view.height)
  for (const { at } of actors) {
    ctx.moveTo(at.x + VEIL_HOLE_RADIUS_PX * o.k, at.y)
    ctx.arc(at.x, at.y, VEIL_HOLE_RADIUS_PX * o.k, 0, Math.PI * 2)
  }
  ctx.fillStyle = o.ink.veil
  ctx.globalAlpha = VEIL_ALPHA * fadeOpacity(o.elapsedMs)
  ctx.fill('evenodd')
  ctx.restore()
  // LES ÉQUERRES : un tracé par joueur, verrouillé par la pose du moment.
  const pose = lockPose(o.elapsedMs)
  for (const { actor, at } of actors) {
    ctx.save()
    applyPose(ctx, at, pose)
    ctx.lineWidth = BRACKET_WIDTH_PX * o.k
    ctx.lineJoin = 'miter'
    ctx.lineCap = 'butt'
    ctx.strokeStyle = actor.color
    ctx.beginPath()
    for (const [a, c, b] of bracketsPath(at.x, at.y, BRACKET_HALF_PX * o.k, BRACKET_ARM_PX * o.k)) {
      ctx.moveTo(a.x, a.y)
      ctx.lineTo(c.x, c.y)
      ctx.lineTo(b.x, b.y)
    }
    ctx.stroke()
    ctx.restore()
  }
}

/** La silhouette d'une étiquette : un rectangle dont le coin bas droit est coupé. */
function labelShape(ctx: CanvasRenderingContext2D, x: number, y: number, w: number, h: number): void {
  ctx.beginPath()
  ctx.moveTo(x, y)
  ctx.lineTo(x + w, y)
  ctx.lineTo(x + w, y + h * CUT_RIGHT)
  ctx.lineTo(x + w * CUT_BOTTOM, y + h)
  ctx.lineTo(x, y + h)
  ctx.closePath()
}

/** La croix du nom de la victime, centrée en (cx, cy). */
function drawCross(ctx: CanvasRenderingContext2D, cx: number, cy: number, size: number): void {
  const r = size / 2 - 1
  ctx.beginPath()
  ctx.moveTo(cx - r, cy - r)
  ctx.lineTo(cx + r, cy + r)
  ctx.moveTo(cx + r, cy - r)
  ctx.lineTo(cx - r, cy + r)
  ctx.stroke()
}

/**
 * drawLabel : l'étiquette d'un acteur. `isKiller` = pleine, avec l'arme ; sinon en creux, avec la
 * croix. Le texte est mesuré, la boîte s'ajuste, l'ensemble apparaît par la pose `pop`.
 */
function drawLabel(
  ctx: CanvasRenderingContext2D,
  { actor, at }: OnScreen,
  isKiller: boolean,
  o: KillFocusLabelPaint,
): void {
  if (!actor.name) return
  const k = o.k
  ctx.font = `${LABEL_WEIGHT} ${LABEL_FONT_PX * k}px ${LABEL_FAMILY}`
  const textW = ctx.measureText(actor.name).width
  const glyphW = (isKiller ? (o.weaponIcon ? WEAPON_ICON_W_PX : 0) : CROSS_PX) * k
  const gap = glyphW > 0 ? LABEL_GAP_PX * k : 0
  const w = LABEL_PAD_LEFT_PX * k + glyphW + gap + textW + LABEL_PAD_RIGHT_PX * k
  const h = LABEL_HEIGHT_PX * k
  const x = at.x - w / 2
  const y = at.y + LABEL_OFFSET_PX * k
  const mid = y + h / 2
  const textColor = isKiller ? o.ink.label : actor.color

  ctx.save()
  applyPose(ctx, { x: at.x, y: mid }, popPose(o.elapsedMs))
  ctx.fillStyle = isKiller ? actor.color : o.ink.label
  labelShape(ctx, x, y, w, h)
  ctx.fill()
  if (!isKiller) {
    // Le liseré EN CREUX : un cadre posé sur le bord et rogné par la silhouette, donc la moitié
    // intérieure seule se voit — et pas le long de la découpe du coin.
    ctx.save()
    labelShape(ctx, x, y, w, h)
    ctx.clip()
    ctx.strokeStyle = actor.color
    ctx.lineWidth = LABEL_BORDER_PX * 2 * k
    ctx.strokeRect(x, y, w, h)
    ctx.restore()
  }
  let cursor = x + LABEL_PAD_LEFT_PX * k
  if (isKiller && o.weaponIcon) {
    drawWeaponIcon(ctx, o.weaponIcon, cursor, mid, k)
  } else if (!isKiller) {
    ctx.strokeStyle = actor.color
    ctx.lineWidth = CROSS_WIDTH_PX * k
    drawCross(ctx, cursor + glyphW / 2, mid, glyphW)
  }
  cursor += glyphW + gap
  ctx.fillStyle = textColor
  ctx.textAlign = 'left'
  ctx.textBaseline = 'middle'
  ctx.fillText(actor.name, cursor, mid)
  ctx.restore()
}

/** L'icône de l'arme, inscrite sans déformation dans son gabarit 22 x 10, centrée sur `midY`. */
function drawWeaponIcon(
  ctx: CanvasRenderingContext2D,
  icon: NonNullable<KillFocusLabelPaint['weaponIcon']>,
  x: number,
  midY: number,
  k: number,
): void {
  const boxW = WEAPON_ICON_W_PX * k
  const boxH = WEAPON_ICON_H_PX * k
  const ratio = icon.width > 0 && icon.height > 0 ? icon.width / icon.height : boxW / boxH
  const w = Math.min(boxW, boxH * ratio)
  const h = w / ratio
  ctx.drawImage(icon, x + (boxW - w) / 2, midY - h / 2, w, h)
}

/** drawKillFocusLabels : les étiquettes du tueur (pleine, avec l'arme) et de la victime (en creux). */
export function drawKillFocusLabels(
  ctx: CanvasRenderingContext2D,
  focus: ResolvedKillFocus | null,
  view: CanvasView,
  o: KillFocusLabelPaint,
): void {
  if (!focus) return
  for (const entry of onScreenActors(focus, view, o.k)) {
    drawLabel(ctx, entry, entry.actor === focus.killer, o)
  }
}
