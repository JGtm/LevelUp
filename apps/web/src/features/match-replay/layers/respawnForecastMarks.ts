/**
 * respawnForecastMarks.ts — LES REPÈRES de la probabilité de réapparition, au-dessus des pions :
 *
 *  - pendant le compte à rebours, un ANNEAU autour du lieu de la mort qui se vide jusqu'à la
 *    réapparition, avec sa puce « joueur · N s » ;
 *  - à la réapparition, le POINT RÉEL verrouillé par quatre équerres et son étiquette « X apparu en
 *    Zone » avec le rang de la zone et sa probabilité ;
 *  - les étiquettes des trois premières zones de la mort que la légende raconte (la plus récente), avec
 *    leur part ; au point réel, celles qui ne sont pas la zone réelle s'estompent.
 *
 * LA FORME DIT LA NATURE : l'anneau et les équerres sont des traits fins, sans remplissage, à la couleur
 * du joueur mort sous un liseré sombre ; une marque pleine se lirait comme quelqu'un qui est là. La
 * croix de mort est celle du calque « Morts » : l'anneau l'entoure, il ne la redessine pas.
 */
import type { XY } from '../../../lib/replay/replayLogic'
import { projectTo, type CanvasView } from '../model/replayView'
import { primaryMoment, type ForecastMoment } from '../model/respawnForecast.logic'
import { bracketsPath } from './killFocusLayer'
import { drawPill, hasShape, type ForecastPaint, type PillStyle } from './respawnForecastPaint'

const RING_RADIUS_PX = 12
const RING_WIDTH_PX = 2.4
const RING_TRACK_ALPHA = 0.22
const UNDERLAY_EXTRA_PX = 1.6
const UNDERLAY_ALPHA = 0.45
const BRACKET_HALF_PX = 15
const BRACKET_ARM_PX = 6
const BRACKET_WIDTH_PX = 2.6
const POINT_RADIUS_PX = 3
const CHIP_FONT_PX = 10.5
const CHIP_GAP_PX = RING_RADIUS_PX + 4
const TAG_FONT_PX = 11
const TAG_GAP_PX = BRACKET_HALF_PX + 5
const LABEL_FONT_PX = 10
/** Au point réel, une étiquette de zone qui n'est pas la zone réelle s'estompe sans disparaître. */
const DIMMED_LABEL_ALPHA = 0.35

/** Un trait posé deux fois : le liseré sombre, puis la couleur du joueur. */
function strokeTwice(ctx: CanvasRenderingContext2D, color: string, width: number, alpha: number, p: ForecastPaint): void {
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  ctx.strokeStyle = p.ink.underlay
  ctx.globalAlpha = alpha * UNDERLAY_ALPHA
  ctx.lineWidth = (width + UNDERLAY_EXTRA_PX) * p.k
  ctx.stroke()
  ctx.strokeStyle = color
  ctx.globalAlpha = alpha
  ctx.lineWidth = width * p.k
  ctx.stroke()
}

/** L'anneau du compte à rebours : une piste pâle entière, et l'arc de ce qui reste, depuis le haut. */
function drawCountdownRing(ctx: CanvasRenderingContext2D, m: ForecastMoment, c: XY, color: string, p: ForecastPaint): void {
  const r = RING_RADIUS_PX * p.k
  ctx.beginPath()
  ctx.arc(c.x, c.y, r, 0, Math.PI * 2)
  strokeTwice(ctx, color, RING_WIDTH_PX, RING_TRACK_ALPHA, p)
  if (m.fraction > 0) {
    ctx.beginPath()
    ctx.arc(c.x, c.y, r, -Math.PI / 2, -Math.PI / 2 + Math.PI * 2 * m.fraction)
    strokeTwice(ctx, color, RING_WIDTH_PX, 1, p)
  }
  ctx.globalAlpha = 1
}

function drawCountdown(ctx: CanvasRenderingContext2D, m: ForecastMoment, view: CanvasView, p: ForecastPaint): void {
  const c = projectTo(view, m.death.death)
  const color = p.colorOf(m.death.xuid)
  drawCountdownRing(ctx, m, c, color, p)
  const seconds = Math.ceil(m.remainingMs / 1000)
  drawPill(ctx, c, [p.wording.chip(p.nameOf(m.death.xuid), seconds)], {
    k: p.k, ink: p.ink, border: color, alpha: m.opacity, anchor: 'above', fontPx: CHIP_FONT_PX, gapPx: CHIP_GAP_PX,
  })
}

/** Le point réel : un rond fin, quatre équerres qui se verrouillent dessus, et l'étiquette d'apparition. */
function drawReveal(ctx: CanvasRenderingContext2D, m: ForecastMoment, view: CanvasView, p: ForecastPaint): void {
  const c = projectTo(view, m.death.actual)
  const color = p.colorOf(m.death.xuid)
  ctx.beginPath()
  ctx.arc(c.x, c.y, POINT_RADIUS_PX * p.k, 0, Math.PI * 2)
  strokeTwice(ctx, color, RING_WIDTH_PX, m.opacity, p)
  ctx.beginPath()
  for (const [a, corner, b] of bracketsPath(c.x, c.y, BRACKET_HALF_PX * p.k, BRACKET_ARM_PX * p.k)) {
    ctx.moveTo(a.x, a.y)
    ctx.lineTo(corner.x, corner.y)
    ctx.lineTo(b.x, b.y)
  }
  strokeTwice(ctx, color, BRACKET_WIDTH_PX, m.opacity, p)
  ctx.globalAlpha = 1
  drawPill(ctx, c, revealLines(m, p), {
    k: p.k, ink: p.ink, border: color, alpha: m.opacity, anchor: 'below', fontPx: TAG_FONT_PX, gapPx: TAG_GAP_PX,
  })
}

/** Les lignes de l'étiquette d'apparition : où, puis le rang et la probabilité de la zone (ou son absence des zones probables). */
export function revealLines(m: ForecastMoment, p: Pick<ForecastPaint, 'zones' | 'wording' | 'nameOf'>): string[] {
  const name = p.nameOf(m.death.xuid)
  const zone = m.death.actualZone === null ? undefined : p.zones.get(m.death.actualZone)
  if (!zone) return [p.wording.appearedOutside(name)]
  const second =
    m.rank === null ? p.wording.appearedUnlikely : p.wording.appearedRank(p.wording.rank(m.rank), p.wording.share(m.actualP))
  return [p.wording.appearedIn(name, zone.name), second]
}

/** Les étiquettes des trois premières zones d'une mort : rang, nom, part. */
function drawZoneLabels(ctx: CanvasRenderingContext2D, m: ForecastMoment, view: CanvasView, p: ForecastPaint): void {
  const color = p.colorOf(m.death.xuid)
  m.top.forEach((z, i) => {
    const zone = p.zones.get(z.zone)
    if (!zone || !hasShape(zone.shape)) return
    const at = projectTo(view, { x: zone.shape.labelX, y: zone.shape.labelY })
    const dimmed = m.phase === 'reveal' && z.zone !== m.death.actualZone
    const style: PillStyle = {
      k: p.k, ink: p.ink, border: color, anchor: 'center', fontPx: LABEL_FONT_PX, gapPx: 0,
      alpha: (dimmed ? DIMMED_LABEL_ALPHA : 1) * m.opacity,
    }
    drawPill(ctx, at, [`${i + 1}  ${zone.name} ${p.wording.share(z.p)}`], style)
  })
}

/**
 * drawForecastMarks peint les repères des morts actives, puis les étiquettes de zone de la mort que la
 * légende raconte. Sans mort active, rien n'est émis.
 */
export function drawForecastMarks(
  ctx: CanvasRenderingContext2D,
  moments: readonly ForecastMoment[],
  view: CanvasView,
  p: ForecastPaint,
): void {
  if (moments.length === 0) return
  const primary = primaryMoment(moments)
  ctx.save()
  for (const m of moments) {
    if (m.phase === 'countdown') drawCountdown(ctx, m, view, p)
    else drawReveal(ctx, m, view, p)
  }
  if (primary) drawZoneLabels(ctx, primary, view, p)
  ctx.restore()
}
