/**
 * grappleLayer.ts — la LIGNE DE GRAPPIN : du joueur vers son point d'accroche.
 *
 * LA SOURCE est le document (schéma 8) : `grappleLines`, les tractions datées PAR VIE
 * (slot) sur l'axe des frames, avec l'ancre en coordonnées monde. La fenêtre [t0, t1] est
 * MESURÉE côté build (du tir à l'arrivée sur la trajectoire — plan PLAN_GRAPPIN_LIGNE,
 * gate 0) : l'effet dure exactement la traction, aucune rémanence inventée.
 *
 * LA POSITION DU JOUEUR EST CELLE DE LA TRACK à la frame courante (positionAt, la même
 * interpolation que le marqueur) : le joueur bouge pendant la traction, la ligne suit.
 * L'ancre, elle, est un point monde FIXE — c'est la mesure qui le dit (fixité 0,05-0,07 u
 * entre les deux lectures d'une paire).
 *
 * « BLANCHE » = ENCRE DU THÈME, jamais un hex : `--foreground`, le neutre le plus clair du
 * système en thème sombre (readInk, même règle que les encres de mise en page du canvas).
 * Épaisseur discrète, sous la densité des effets de tir. Une ligne statique par frame :
 * `prefers-reduced-motion` est respectée par construction — rien ne s'anime de soi-même.
 *
 * Pas de React : géométrie pure + un CanvasRenderingContext2D (même règle que replayDraw).
 */
import type { ReplayBounds, ReplayPoint } from '@/lib/api/types'

import { positionAt, worldToCanvas, type XY } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'

/** Cadrage du canvas (mêmes paramètres que worldToCanvas). */
interface CanvasView {
  bounds: ReplayBounds
  width: number
  height: number
  pad: number
}

/** Une traction prête à dessiner : la fenêtre, l'ancre, et les points de SA vie. */
export interface GrappleFxEntry {
  t0: number
  t1: number
  anchor: XY
  /** Les points de la track porteuse — la position du joueur s'y lit à chaque frame. */
  points: ReplayPoint[]
}

/**
 * buildGrappleFx précalcule les tractions dessinables : chaque ligne du document jointe
 * aux points de sa vie. Une ligne dont la vie n'est pas publiée ne se dessine pas — il n'y
 * aurait aucun joueur à relier (le build l'écarte déjà ; la jointure reste défensive).
 */
export function buildGrappleFx(doc: ReplayDocumentReady): GrappleFxEntry[] {
  if (doc.grappleLines.length === 0) return []
  const pointsBySlot = new Map(doc.tracks.map((t) => [t.slot, t.points]))
  const out: GrappleFxEntry[] = []
  for (const l of doc.grappleLines) {
    const points = pointsBySlot.get(l.slot)
    if (!points || points.length === 0 || l.t1 <= l.t0) continue
    out.push({ t0: l.t0, t1: l.t1, anchor: { x: l.ax, y: l.ay }, points })
  }
  return out
}

// Épaisseur et présence : un câble discret SOUS la densité des effets de tir (leurs traits
// montent à 2-3 px avec halo ; celui-ci reste à 1,25 px sans aucun halo). Le petit disque
// marque le point d'accroche — c'est lui que la ligne « vise ».
const GRAPPLE_LINE_WIDTH = 1.25
const GRAPPLE_ALPHA = 0.85
const GRAPPLE_ANCHOR_RADIUS = 2

/**
 * drawGrappleLayer trace les tractions actives à la frame courante : un segment de la
 * position COURANTE du joueur vers l'ancre, et le point d'accroche. Même chaîne de
 * projection que les tracks (worldToCanvas).
 */
export function drawGrappleLayer(
  ctx: CanvasRenderingContext2D,
  entries: GrappleFxEntry[],
  view: CanvasView,
  frame: number,
  ink: string,
): void {
  for (const e of entries) {
    if (frame < e.t0 || frame > e.t1) continue
    const pos = positionAt(e.points, frame)
    if (!pos) continue
    const p = worldToCanvas(pos, view.bounds, view.width, view.height, view.pad)
    const a = worldToCanvas(e.anchor, view.bounds, view.width, view.height, view.pad)
    ctx.save()
    ctx.globalAlpha = GRAPPLE_ALPHA
    ctx.strokeStyle = ink
    ctx.fillStyle = ink
    ctx.lineWidth = GRAPPLE_LINE_WIDTH
    ctx.beginPath()
    ctx.moveTo(p.x, p.y)
    ctx.lineTo(a.x, a.y)
    ctx.stroke()
    ctx.beginPath()
    ctx.arc(a.x, a.y, GRAPPLE_ANCHOR_RADIUS, 0, Math.PI * 2)
    ctx.fill()
    ctx.restore()
  }
}
