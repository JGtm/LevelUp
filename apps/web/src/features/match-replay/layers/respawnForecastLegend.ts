/**
 * respawnForecastLegend.ts — LA LÉGENDE de la probabilité de réapparition, posée sur le plan en bas à
 * gauche : le titre, qui est mort et dans combien de secondes il réapparaît (puis où il est apparu et
 * à quel rang), l'échelle de ce que le dessin montre — disques (probabilité d'un point) et/ou teinte
 * (probabilité d'une zone), selon le choix « Dessin » — et la fréquence de mise à jour. Elle raconte UNE
 * mort, la plus récente (`primaryMoment`), et disparaît avec elle.
 *
 * ELLE SE PEINT DANS LA TOILE, pas en surimpression : la légende est lue avec la carte, et une
 * capture ou un export la porte avec elle. Voile et texte sur voile sont les jetons `scrim` et
 * `on-scrim` ; teinte et disques sont à la couleur du joueur, aux mêmes opacités que le dessin.
 */
import {
  DISC_SCALE_P,
  discRadiusPx,
  TINT_AMPLITUDE_ALPHA,
  TINT_AMPLITUDE_ALPHA_UNDER_POINTS,
  TINT_BASE_ALPHA,
  TINT_BASE_ALPHA_UNDER_POINTS,
  type ForecastDraw,
  type ForecastMoment,
} from '../model/respawnForecast.logic'
import type { CanvasView } from '../model/replayView'
import { forecastFont, type ForecastPaint } from './respawnForecastPaint'
import { revealLines } from './respawnForecastMarks'

const MARGIN_PX = 10
const WIDTH_PX = 236
const PAD_PX = 9
const TITLE_PX = 12
const BODY_PX = 11
const SMALL_PX = 10
const LINE_GAP_PX = 6
const DOT_RADIUS_PX = 4
const BAR_HEIGHT_PX = 8
/** L'échelle de teinte est un dégradé en bandes : assez fines pour se lire continues. */
const BAR_STEPS = 24
const PANEL_ALPHA = 0.82
const PANEL_RADIUS_PX = 8
/** Les disques de l'échelle sont dessinés à la moitié de leur taille sur la carte : la légende reste compacte. */
const SCALE_DISC_FACTOR = 0.5
const SCALE_DISC_GAP_PX = 14
const SCALE_DISC_FILL_ALPHA = 0.32

/** Les lignes de texte de la légende : ce qu'elle dit de la mort racontée. */
export function legendWho(m: ForecastMoment, p: Pick<ForecastPaint, 'zones' | 'wording' | 'nameOf'>): string {
  if (m.phase === 'countdown') {
    return p.wording.legendCountdown(p.nameOf(m.death.xuid), Math.ceil(m.remainingMs / 1000))
  }
  return revealLines(m, p).join(', ')
}

function panelPath(ctx: CanvasRenderingContext2D, x: number, y: number, w: number, h: number, r: number): void {
  ctx.beginPath()
  ctx.moveTo(x + r, y)
  ctx.arcTo(x + w, y, x + w, y + h, r)
  ctx.arcTo(x + w, y + h, x, y + h, r)
  ctx.arcTo(x, y + h, x, y, r)
  ctx.arcTo(x, y, x + w, y, r)
  ctx.closePath()
}

const hasPoints = (draw: ForecastDraw) => draw !== 'zones'
const hasZones = (draw: ForecastDraw) => draw !== 'points'

/** Le pas vertical d'une ligne de texte de `px` pixels, en pixels d'écran. */
const lineOf = (px: number) => px * 1.3 + LINE_GAP_PX

/** La hauteur de l'échelle des disques (rangée de disques et ses pourcentages, puis la légende), en pixels d'écran. */
function pointsBlockHeight(): number {
  const tallest = 2 * discRadiusPx(DISC_SCALE_P[DISC_SCALE_P.length - 1]) * SCALE_DISC_FACTOR
  return tallest + lineOf(SMALL_PX) * 2
}

/** La hauteur de l'échelle de teinte : seule, avec ses bornes ; sous les points, la barre et sa légende. */
function zonesBlockHeight(draw: ForecastDraw): number {
  return BAR_HEIGHT_PX + 2 + lineOf(SMALL_PX) * (draw === 'zones' ? 2 : 1)
}

/** L'échelle de la teinte : des bandes de la couleur du joueur, du fond léger à la teinte pleine. */
function drawTintBar(
  ctx: CanvasRenderingContext2D,
  at: { x: number; y: number; w: number },
  color: string,
  draw: ForecastDraw,
  opacity: number,
  k: number,
): void {
  const under = draw === 'both'
  const base = under ? TINT_BASE_ALPHA_UNDER_POINTS : TINT_BASE_ALPHA
  const amplitude = under ? TINT_AMPLITUDE_ALPHA_UNDER_POINTS : TINT_AMPLITUDE_ALPHA
  const step = at.w / BAR_STEPS
  ctx.fillStyle = color
  for (let i = 0; i < BAR_STEPS; i++) {
    ctx.globalAlpha = (base + amplitude * (i / (BAR_STEPS - 1))) * opacity
    ctx.fillRect(at.x + i * step, at.y, step + 0.5, BAR_HEIGHT_PX * k)
  }
  ctx.globalAlpha = opacity
}

/** Les trois disques de repère (5, 20, 45 %) avec leur pourcentage, puis la légende « disque : point · étiquette : zone ». */
function drawDiscScale(ctx: CanvasRenderingContext2D, at: { x: number; y: number }, color: string, m: ForecastMoment, p: ForecastPaint): void {
  const { k } = p
  const tallest = discRadiusPx(DISC_SCALE_P[DISC_SCALE_P.length - 1]) * SCALE_DISC_FACTOR
  let cursor = at.x
  ctx.textAlign = 'center'
  ctx.textBaseline = 'top'
  ctx.font = forecastFont(SMALL_PX, 500, k)
  for (const q of DISC_SCALE_P) {
    const r = discRadiusPx(q) * SCALE_DISC_FACTOR * k
    const label = p.wording.share(q)
    const w = Math.max(2 * r, ctx.measureText(label).width)
    const cx = cursor + w / 2
    ctx.beginPath()
    ctx.arc(cx, at.y + 2 * tallest * k - r, r, 0, Math.PI * 2)
    ctx.fillStyle = color
    ctx.globalAlpha = SCALE_DISC_FILL_ALPHA * m.opacity
    ctx.fill()
    ctx.globalAlpha = m.opacity
    ctx.strokeStyle = color
    ctx.lineWidth = 1.2 * k
    ctx.stroke()
    ctx.fillStyle = p.ink.onPanel
    ctx.fillText(label, cx, at.y + 2 * tallest * k + 2 * k)
    cursor += w + SCALE_DISC_GAP_PX * k
  }
  ctx.textAlign = 'left'
  ctx.fillStyle = p.ink.onPanel
  ctx.fillText(p.wording.legendPoints, at.x, at.y + 2 * tallest * k + lineOf(SMALL_PX) * k)
}

/** L'échelle de teinte avec sa légende : bornes 0 % et part de la première zone quand elle est seule. */
function drawZonesScale(ctx: CanvasRenderingContext2D, at: { x: number; y: number; w: number }, color: string, m: ForecastMoment, p: ForecastPaint): void {
  const { k } = p
  drawTintBar(ctx, at, color, p.draw, m.opacity, k)
  let cursor = at.y + (BAR_HEIGHT_PX + 2) * k
  ctx.fillStyle = p.ink.onPanel
  ctx.font = forecastFont(SMALL_PX, 500, k)
  ctx.textBaseline = 'top'
  if (p.draw === 'zones') {
    ctx.textAlign = 'left'
    ctx.fillText(p.wording.share(0), at.x, cursor)
    ctx.textAlign = 'right'
    ctx.fillText(p.wording.share(m.peak), at.x + at.w, cursor)
    ctx.textAlign = 'left'
    cursor += lineOf(SMALL_PX) * k
    ctx.fillText(p.wording.legendScale, at.x, cursor)
  } else {
    ctx.textAlign = 'left'
    ctx.fillText(p.wording.legendTintUnderPoints, at.x, cursor)
  }
}

/**
 * drawForecastLegend pose la légende de la mort racontée. `null` : rien n'est émis.
 */
export function drawForecastLegend(
  ctx: CanvasRenderingContext2D,
  m: ForecastMoment | null,
  view: CanvasView,
  p: ForecastPaint,
): void {
  if (!m) return
  const { k } = p
  const x = MARGIN_PX * k
  const w = WIDTH_PX * k
  const pad = PAD_PX * k
  const blocks = (hasPoints(p.draw) ? pointsBlockHeight() + LINE_GAP_PX : 0) + (hasZones(p.draw) ? zonesBlockHeight(p.draw) + LINE_GAP_PX : 0)
  const h = pad * 2 + (lineOf(TITLE_PX) + lineOf(BODY_PX) + blocks + lineOf(SMALL_PX)) * k
  const y = view.height - MARGIN_PX * k - h
  const color = p.colorOf(m.death.xuid)

  ctx.save()
  ctx.globalAlpha = PANEL_ALPHA * m.opacity
  ctx.fillStyle = p.ink.panel
  panelPath(ctx, x, y, w, h, PANEL_RADIUS_PX * k)
  ctx.fill()
  ctx.globalAlpha = m.opacity
  ctx.textBaseline = 'top'
  ctx.textAlign = 'left'
  ctx.fillStyle = p.ink.onPanel

  let cursor = y + pad
  ctx.font = forecastFont(TITLE_PX, 700, k)
  ctx.fillText(p.wording.title, x + pad, cursor)
  cursor += lineOf(TITLE_PX) * k

  ctx.fillStyle = color
  ctx.beginPath()
  ctx.arc(x + pad + DOT_RADIUS_PX * k, cursor + BODY_PX * 0.65 * k, DOT_RADIUS_PX * k, 0, Math.PI * 2)
  ctx.fill()
  ctx.fillStyle = p.ink.onPanel
  ctx.font = forecastFont(BODY_PX, 600, k)
  const indent = (DOT_RADIUS_PX * 2 + 5) * k
  ctx.fillText(legendWho(m, p), x + pad + indent, cursor, w - pad * 2 - indent)
  cursor += lineOf(BODY_PX) * k

  if (hasPoints(p.draw)) {
    drawDiscScale(ctx, { x: x + pad, y: cursor }, color, m, p)
    cursor += (pointsBlockHeight() + LINE_GAP_PX) * k
  }
  if (hasZones(p.draw)) {
    drawZonesScale(ctx, { x: x + pad, y: cursor, w: w - pad * 2 }, color, m, p)
    cursor += (zonesBlockHeight(p.draw) + LINE_GAP_PX) * k
  }
  ctx.globalAlpha = m.opacity
  ctx.fillStyle = p.ink.onPanel
  ctx.font = forecastFont(SMALL_PX, 500, k)
  ctx.textAlign = 'left'
  ctx.textBaseline = 'top'
  ctx.fillText(p.wording.legendFooter, x + pad, cursor)
  ctx.restore()
}
