/**
 * chatterLayer.ts — LE DESSIN DES RÉPLIQUES sur la carte : une bulle de texte sur le joueur qui parle, et une icône de voix.
 * Fonctions de dessin pures (aucun état) ; le choix des répliques vivantes et le placement sont dans
 * `model/chatterBubbles.ts`, le câblage dans `useReplayChatter.ts`.
 *
 * TROIS CAS, selon les deux interrupteurs d'affichage :
 *  - bulle seule : le texte, dans une bulle reliée au pion ;
 *  - bulle et icône : la même bulle, avec un haut-parleur devant le texte quand la réplique a une voix ;
 *  - icône seule : un petit disque à l'épaule du pion, avec le haut-parleur (rien du tout quand la réplique n'a pas de voix :
 *    l'icône ne promet pas un son qui n'existe pas).
 *
 * TAILLES EN PIXELS D'ÉCRAN, comme les pions (`k` est la densité de la toile) : le zoom de la carte déplace les bulles avec leur
 * joueur sans les grossir. La coupe par étage ne masque aucun joueur, donc aucune bulle. L'encre est celle du thème (la carte
 * et le texte du système de design) et le liseré est la couleur du joueur.
 */
import { CANVAS_FONT_FAMILY } from '@/lib/replay/canvasFont'
import { roundedRectPath } from '@/lib/replay/roundedRect'

import { placeBubbles, type BubbleSpec, type LiveChatter, type Rect } from '../model/chatterBubbles'
import { projectTo, type CanvasView } from '../model/replayView'
import type { XY } from '../../../lib/replay/replayLogic'

const FONT_PX = 12
const LINE_HEIGHT_PX = 15
const MAX_LINES = 3
const MAX_TEXT_PX = 170
const MIN_TEXT_PX = 100
/** La part de la largeur de la carte qu'une bulle ne dépasse pas. */
const MAX_WIDTH_RATIO = 0.34
const PAD_X_PX = 8
const PAD_Y_PX = 5
const ICON_SLOT_PX = 18
const ICON_PX = 14
const CORNER_PX = 5
const BORDER_PX = 1.5
const TAIL_PX = 1
const SOLO_ICON_RADIUS_PX = 10
const SOLO_ICON_DX_PX = 11
const SOLO_ICON_DY_PX = -13
const MARKER_BOX_PX = 24
const ELLIPSIS = '…'

/** Les encres : le fond de la bulle (la carte du système de design), son texte, et le contour des textes. */
export interface ChatterInk {
  fill: string
  text: string
}

/** Ce que le dessin lit : les répliques vivantes, où est chaque joueur, sa couleur, et les deux interrupteurs. */
export interface ChatterPaint {
  live: readonly LiveChatter[]
  /** La position MONDE du joueur à l'image, ou `null` quand le film ne la donne pas (rien n'est deviné). */
  posOf: (xuid: string, frame: number) => XY | null
  colorOf: (xuid: string) => string | null
  frame: number
  showBubble: boolean
  showIcon: boolean
  ink: ChatterInk
  /** Une couleur de repli pour un joueur dont l'équipe n'est pas lue. */
  fallbackColor: string
  /** La densité de la toile. */
  k: number
}

/** Le texte coupé en lignes d'au plus `maxW` pixels, trois au plus (la dernière finit par une ellipse). */
export function wrapText(measure: (s: string) => number, text: string, maxW: number): string[] {
  const words = text.split(' ').filter((w) => w !== '')
  const lines: string[] = []
  let line = ''
  for (const w of words) {
    const next = line ? `${line} ${w}` : w
    if (!line || measure(next) <= maxW) line = next
    else {
      lines.push(line)
      line = w
    }
  }
  if (line) lines.push(line)
  if (lines.length <= MAX_LINES) return lines
  const kept = lines.slice(0, MAX_LINES)
  kept[MAX_LINES - 1] = `${kept[MAX_LINES - 1].replace(/\s*\S*$/, '')}${ELLIPSIS}`
  return kept
}

/** Les deux ondes du haut-parleur (grille de 24) : des arcs de cercle centrés sur l'axe du corps. */
const SPEAKER_WAVES: ReadonlyArray<{ cx: number; r: number; half: number }> = [
  { cx: 12.65, r: 4.5, half: 0.73 },
  { cx: 12.69, r: 8, half: 0.76 },
]

/** Le haut-parleur : un corps plein et deux ondes, sur une grille de 24, centré en (x, y), de côté `size`. */
function drawSpeaker(ctx: CanvasRenderingContext2D, x: number, y: number, size: number, color: string): void {
  ctx.save()
  ctx.translate(x - size / 2, y - size / 2)
  ctx.scale(size / 24, size / 24)
  ctx.fillStyle = color
  ctx.strokeStyle = color
  ctx.lineWidth = 2
  ctx.lineCap = 'round'
  ctx.beginPath()
  ctx.moveTo(4, 9)
  ctx.lineTo(4, 15)
  ctx.lineTo(8, 15)
  ctx.lineTo(13, 19)
  ctx.lineTo(13, 5)
  ctx.lineTo(8, 9)
  ctx.closePath()
  ctx.fill()
  for (const wave of SPEAKER_WAVES) {
    ctx.beginPath()
    ctx.arc(wave.cx, 12, wave.r, -wave.half, wave.half)
    ctx.stroke()
  }
  ctx.restore()
}

/** Un joueur qui parle : sa réplique vivante, où il est sur la toile, et la couleur de son liseré. */
interface Speaker {
  chatter: LiveChatter
  at: XY
  color: string
}

function speakersOf(p: ChatterPaint, view: CanvasView): Speaker[] {
  const out: Speaker[] = []
  for (const c of p.live) {
    const world = p.posOf(c.line.xuid, p.frame)
    if (!world) continue
    out.push({ chatter: c, at: projectTo(view, world), color: p.colorOf(c.line.xuid) ?? p.fallbackColor })
  }
  return out
}

/** La mesure d'une bulle : le texte coupé en lignes, et sa taille. */
interface Measured {
  lines: string[]
  spec: BubbleSpec
}

function measureBubble(ctx: CanvasRenderingContext2D, s: Speaker, view: CanvasView, p: ChatterPaint): Measured {
  const k = p.k
  ctx.font = `${FONT_PX * k}px ${CANVAS_FONT_FAMILY}`
  const maxText = Math.max(MIN_TEXT_PX * k, Math.min(MAX_TEXT_PX * k, view.width * MAX_WIDTH_RATIO))
  const lines = wrapText((t) => ctx.measureText(t).width, s.chatter.line.text, maxText)
  const textW = Math.max(0, ...lines.map((l) => ctx.measureText(l).width))
  const hasIcon = p.showIcon && s.chatter.line.voice !== null
  const w = Math.ceil(textW) + 2 * PAD_X_PX * k + (hasIcon ? ICON_SLOT_PX * k : 0)
  const h = lines.length * LINE_HEIGHT_PX * k + 2 * PAD_Y_PX * k
  return { lines, spec: { w, h, anchorX: s.at.x, anchorY: s.at.y } }
}

function drawBubble(ctx: CanvasRenderingContext2D, s: Speaker, m: Measured, r: Rect, p: ChatterPaint): void {
  const k = p.k
  ctx.globalAlpha = s.chatter.alpha
  // Le trait qui relie la bulle au pion : du pion au point de la bulle le plus proche.
  const cx = Math.min(Math.max(s.at.x, r.x), r.x + r.w)
  const cy = Math.min(Math.max(s.at.y, r.y), r.y + r.h)
  ctx.strokeStyle = s.color
  ctx.lineWidth = TAIL_PX * k
  ctx.beginPath()
  ctx.moveTo(s.at.x, s.at.y)
  ctx.lineTo(cx, cy)
  ctx.stroke()
  roundedRectPath(ctx, r, CORNER_PX * k)
  ctx.fillStyle = p.ink.fill
  ctx.fill()
  ctx.lineWidth = BORDER_PX * k
  ctx.stroke()
  const hasIcon = p.showIcon && s.chatter.line.voice !== null
  const textX = r.x + PAD_X_PX * k + (hasIcon ? ICON_SLOT_PX * k : 0)
  if (hasIcon) drawSpeaker(ctx, r.x + PAD_X_PX * k + (ICON_PX * k) / 2, r.y + r.h / 2, ICON_PX * k, s.color)
  ctx.fillStyle = p.ink.text
  ctx.font = `${FONT_PX * k}px ${CANVAS_FONT_FAMILY}`
  ctx.textAlign = 'left'
  ctx.textBaseline = 'top'
  m.lines.forEach((line, i) => ctx.fillText(line, textX, r.y + PAD_Y_PX * k + i * LINE_HEIGHT_PX * k))
  ctx.globalAlpha = 1
}

/** L'icône seule : un disque à l'épaule du pion portant le haut-parleur. Rien quand la réplique n'a pas de voix. */
function drawSoloIcon(ctx: CanvasRenderingContext2D, s: Speaker, p: ChatterPaint): void {
  if (s.chatter.line.voice === null) return
  const k = p.k
  const x = s.at.x + SOLO_ICON_DX_PX * k
  const y = s.at.y + SOLO_ICON_DY_PX * k
  ctx.globalAlpha = s.chatter.alpha
  ctx.beginPath()
  ctx.arc(x, y, SOLO_ICON_RADIUS_PX * k, 0, Math.PI * 2)
  ctx.fillStyle = p.ink.fill
  ctx.fill()
  ctx.strokeStyle = s.color
  ctx.lineWidth = BORDER_PX * k
  ctx.stroke()
  drawSpeaker(ctx, x, y, ICON_PX * k, s.color)
  ctx.globalAlpha = 1
}

/**
 * drawChatter pose les répliques vivantes : les bulles d'abord (placées pour ne pas se recouvrir), puis, quand seule l'icône
 * est demandée, les icônes. Un joueur dont le film ne donne pas la position à cette image n'a ni bulle ni icône.
 */
export function drawChatter(ctx: CanvasRenderingContext2D, view: CanvasView, p: ChatterPaint): void {
  if (!p.showBubble && !p.showIcon) return
  const speakers = speakersOf(p, view)
  if (speakers.length === 0) return
  ctx.save()
  if (p.showBubble) {
    const measured = speakers.map((s) => measureBubble(ctx, s, view, p))
    const markers: Rect[] = speakers.map((s) => ({
      x: s.at.x - (MARKER_BOX_PX * p.k) / 2,
      y: s.at.y - (MARKER_BOX_PX * p.k) / 2,
      w: MARKER_BOX_PX * p.k,
      h: MARKER_BOX_PX * p.k,
    }))
    const rects = placeBubbles(measured.map((m) => m.spec), markers, { width: view.width, height: view.height })
    speakers.forEach((s, i) => drawBubble(ctx, s, measured[i], rects[i], p))
  } else {
    for (const s of speakers) drawSoloIcon(ctx, s, p)
  }
  ctx.restore()
}
