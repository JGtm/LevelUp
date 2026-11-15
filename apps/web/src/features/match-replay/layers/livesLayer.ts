/**
 * livesLayer.ts — LE DESSIN DES TRACES DES VIES : la goutte creuse des apparitions, la croix des
 * morts, le triangle creux des tueurs et son trait vers la victime. Les marques et leur filtre
 * viennent de `model/livesMarks.ts` ; ce fichier ne fait que les peindre.
 *
 * RÈGLE DE FORME : CE QUI VIT EST PLEIN, CE QUI EST UNE TRACE EST CREUX ET FIN. Les pions des
 * joueurs (disque, losange d'ami, disque cerclé de celui qu'on regarde, cône de visée) sont
 * pleins ; ces trois calques disent ce qui a eu lieu et ne doivent jamais se lire comme quelqu'un
 * qui est là. D'où des traits d'une épaisseur de filet, sans remplissage, et des tailles
 * inférieures à celles du pion.
 *
 * LA COULEUR EST CELLE DE L'ÉQUIPE de celui qui a fait la marque (l'encre que le canvas donne à son
 * pion) ; un liseré sombre (`underlay`, l'encre de contour des noms) passe dessous pour que le
 * filet reste lisible sur un fond clair ou chargé. Les tailles sont en pixels d'écran, multipliées
 * par la densité `k` comme celles des autres calques.
 */
import { shiftHeadingDeg } from '@/lib/replay/orientation'

import { markAlpha, passesShown, passesWho, type DeathMark, type KillerMark, type LivesTime, type LivesWho, type SpawnMark } from '../model/livesMarks'
import { projectTo, turnsOfView, type CanvasView } from '../model/replayView'

/** La goutte reste plus petite que le pion (rayon 3,4 px) : anneau de 2,4 px, pointe à 5,2 px. */
const DROP_RING_RADIUS = 2.4
const DROP_TIP_LENGTH = 5.2
const DROP_WIDTH = 1.2
const CROSS_ARM = 3.4
const CROSS_WIDTH = 1.4
const TRIANGLE_RADIUS = 4
const TRIANGLE_WIDTH = 1.3
const LINK_WIDTH = 0.9
/** Le liseré sombre déborde le trait de part et d'autre : épaisseur ajoutée au trait de couleur. */
const UNDERLAY_EXTRA = 1.6
const UNDERLAY_ALPHA = 0.45

/** Ce que les trois peintres lisent : le filtre, le temps, l'encre du liseré et la densité. */
export interface LivesPaint {
  who: LivesWho
  time: LivesTime
  /** Encre du liseré sombre sous le trait de couleur. */
  underlay: string
  k: number
  /** « Seul » et « Masquer » du panneau de joueur : ce que porte une vie se dessine-t-il ? Absent = aucun filtre. */
  shownAtSlot?: (slot: number, frame: number) => boolean
}

/** Une marque se peint si elle est visible à cette image, du camp voulu et de ce que la carte montre. */
function shown(
  m: Pick<SpawnMark, 'slot' | 'slotFrame' | 'mine' | 'followed'>,
  alpha: number,
  p: LivesPaint,
): boolean {
  return alpha > 0 && passesWho(m, p.who) && passesShown(m, p.shownAtSlot)
}

/** Passe un trait deux fois : le liseré sombre, puis la couleur d'équipe. */
function strokeTwice(
  ctx: CanvasRenderingContext2D,
  color: string,
  width: number,
  alpha: number,
  p: LivesPaint,
): void {
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  ctx.strokeStyle = p.underlay
  ctx.globalAlpha = alpha * UNDERLAY_ALPHA
  ctx.lineWidth = (width + UNDERLAY_EXTRA) * p.k
  ctx.stroke()
  ctx.strokeStyle = color
  ctx.globalAlpha = alpha
  ctx.lineWidth = width * p.k
  ctx.stroke()
}

/**
 * Le tracé d'une goutte : un cercle de rayon `r` dont une pointe sort à `tip` dans la direction
 * `angle` (radians canvas). Les deux droites partent de la pointe, tangentes au cercle.
 */
function traceDrop(ctx: CanvasRenderingContext2D, c: { x: number; y: number }, angle: number, r: number, tip: number): void {
  const { x: cx, y: cy } = c
  const spread = Math.acos(Math.min(1, r / tip))
  const tx = cx + Math.cos(angle) * tip
  const ty = cy + Math.sin(angle) * tip
  ctx.beginPath()
  ctx.moveTo(tx, ty)
  ctx.lineTo(cx + Math.cos(angle + spread) * r, cy + Math.sin(angle + spread) * r)
  ctx.arc(cx, cy, r, angle + spread, angle - spread + Math.PI * 2)
  ctx.closePath()
}

/** drawSpawnMarks — une goutte creuse à l'apparition de chaque vie ; un anneau seul sans cap lu. */
export function drawSpawnMarks(
  ctx: CanvasRenderingContext2D,
  marks: readonly SpawnMark[],
  view: CanvasView,
  p: LivesPaint,
): void {
  const turns = turnsOfView(view)
  for (const m of marks) {
    const alpha = markAlpha(m.frame, p.time, m.endFrame)
    if (!shown(m, alpha, p)) continue
    const c = projectTo(view, m)
    const r = DROP_RING_RADIUS * p.k
    if (m.headingDeg === null) {
      ctx.beginPath()
      ctx.arc(c.x, c.y, r, 0, Math.PI * 2)
    } else {
      // Monde -> canevas : l'axe Y est inversé, donc l'angle l'est aussi, et le cap suit la rotation de la carte.
      traceDrop(ctx, c, (-shiftHeadingDeg(m.headingDeg, turns) * Math.PI) / 180, r, DROP_TIP_LENGTH * p.k)
    }
    strokeTwice(ctx, m.color, DROP_WIDTH, alpha, p)
  }
  ctx.globalAlpha = 1
}

/** drawDeathMarks — une croix fine au lieu de chaque mort, à la couleur de la victime. */
export function drawDeathMarks(
  ctx: CanvasRenderingContext2D,
  marks: readonly DeathMark[],
  view: CanvasView,
  p: LivesPaint,
): void {
  for (const m of marks) {
    const alpha = markAlpha(m.frame, p.time)
    if (!shown(m, alpha, p)) continue
    const c = projectTo(view, m)
    const a = CROSS_ARM * p.k
    ctx.beginPath()
    ctx.moveTo(c.x - a, c.y - a)
    ctx.lineTo(c.x + a, c.y + a)
    ctx.moveTo(c.x + a, c.y - a)
    ctx.lineTo(c.x - a, c.y + a)
    strokeTwice(ctx, m.color, CROSS_WIDTH, alpha, p)
  }
  ctx.globalAlpha = 1
}

/** Le trait fin du tueur à la victime, quand les deux sont positionnés et ne se confondent pas. */
function traceLink(ctx: CanvasRenderingContext2D, from: { x: number; y: number }, to: { x: number; y: number }): boolean {
  if (Math.hypot(to.x - from.x, to.y - from.y) < 1.5) return false
  ctx.beginPath()
  ctx.moveTo(from.x, from.y)
  ctx.lineTo(to.x, to.y)
  return true
}

/** Le triangle creux du tueur : pointe vers la victime quand elle est connue, vers le haut sinon. */
function traceTriangle(ctx: CanvasRenderingContext2D, c: { x: number; y: number }, angle: number, r: number): void {
  ctx.beginPath()
  for (let i = 0; i < 3; i++) {
    const a = angle + (i * 2 * Math.PI) / 3
    const x = c.x + Math.cos(a) * r
    const y = c.y + Math.sin(a) * r
    if (i === 0) ctx.moveTo(x, y)
    else ctx.lineTo(x, y)
  }
  ctx.closePath()
}

/** drawKillerMarks — un triangle creux au tueur, et un trait fin vers la victime. */
export function drawKillerMarks(
  ctx: CanvasRenderingContext2D,
  marks: readonly KillerMark[],
  view: CanvasView,
  p: LivesPaint,
): void {
  for (const m of marks) {
    const alpha = markAlpha(m.frame, p.time)
    if (!shown(m, alpha, p)) continue
    const c = projectTo(view, m)
    const v = m.victim ? projectTo(view, m.victim) : null
    if (v && traceLink(ctx, c, v)) strokeTwice(ctx, m.color, LINK_WIDTH, alpha, p)
    const toward = v ? Math.atan2(v.y - c.y, v.x - c.x) : -Math.PI / 2
    traceTriangle(ctx, c, toward, TRIANGLE_RADIUS * p.k)
    strokeTwice(ctx, m.color, TRIANGLE_WIDTH, alpha, p)
  }
  ctx.globalAlpha = 1
}
