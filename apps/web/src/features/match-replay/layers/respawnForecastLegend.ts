/**
 * respawnForecastLegend.ts — LA LÉGENDE de la probabilité de réapparition, posée sur le plan en bas à
 * gauche : le titre, qui est mort et dans combien de secondes il réapparaît (puis où il est apparu et
 * à quel rang), l'échelle de la teinte de 0 % à la part de la première zone, et la fréquence de mise à
 * jour. Elle raconte UNE mort, la plus récente (`primaryMoment`), et disparaît avec elle.
 *
 * ELLE SE PEINT DANS LA TOILE, pas en surimpression : la légende est lue avec la carte, et une
 * capture ou un export la porte avec elle. Voile et texte sur voile sont les jetons `scrim` et
 * `on-scrim` ; la teinte de l'échelle est celle du joueur, aux mêmes opacités que les zones.
 */
import { TINT_AMPLITUDE_ALPHA, TINT_BASE_ALPHA, type ForecastMoment } from '../model/respawnForecast.logic'
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
/** L'échelle est un dégradé en bandes : assez fines pour se lire continues. */
const BAR_STEPS = 24
const PANEL_ALPHA = 0.82
const PANEL_RADIUS_PX = 8

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

/** L'échelle de la teinte : des bandes de la couleur du joueur, du fond léger à la teinte pleine. */
function drawScale(
  ctx: CanvasRenderingContext2D,
  at: { x: number; y: number; w: number },
  color: string,
  k: number,
  opacity: number,
): void {
  const { x, y, w } = at
  const step = w / BAR_STEPS
  ctx.fillStyle = color
  for (let i = 0; i < BAR_STEPS; i++) {
    ctx.globalAlpha = (TINT_BASE_ALPHA + TINT_AMPLITUDE_ALPHA * (i / (BAR_STEPS - 1))) * opacity
    ctx.fillRect(x + i * step, y, step + 0.5, BAR_HEIGHT_PX * k)
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
  const lineH = (px: number) => (px * 1.3 + LINE_GAP_PX) * k
  const h = pad * 2 + lineH(TITLE_PX) + lineH(BODY_PX) + (BAR_HEIGHT_PX + 2) * k + lineH(SMALL_PX) * 3
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
  cursor += lineH(TITLE_PX)

  ctx.fillStyle = color
  ctx.beginPath()
  ctx.arc(x + pad + DOT_RADIUS_PX * k, cursor + (BODY_PX * 0.65) * k, DOT_RADIUS_PX * k, 0, Math.PI * 2)
  ctx.fill()
  ctx.fillStyle = p.ink.onPanel
  ctx.font = forecastFont(BODY_PX, 600, k)
  const indent = (DOT_RADIUS_PX * 2 + 5) * k
  ctx.fillText(legendWho(m, p), x + pad + indent, cursor, w - pad * 2 - indent)
  cursor += lineH(BODY_PX)

  drawScale(ctx, { x: x + pad, y: cursor, w: w - pad * 2 }, color, k, m.opacity)
  ctx.globalAlpha = m.opacity
  cursor += (BAR_HEIGHT_PX + 2) * k
  ctx.fillStyle = p.ink.onPanel
  ctx.font = forecastFont(SMALL_PX, 500, k)
  ctx.fillText(p.wording.share(0), x + pad, cursor)
  ctx.textAlign = 'right'
  ctx.fillText(p.wording.share(m.peak), x + w - pad, cursor)
  ctx.textAlign = 'left'
  cursor += lineH(SMALL_PX)
  ctx.fillText(p.wording.legendScale, x + pad, cursor)
  cursor += lineH(SMALL_PX)
  ctx.fillText(p.wording.legendFooter, x + pad, cursor)
  ctx.restore()
}
