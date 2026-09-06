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
import type { ReplayPoint } from '@/lib/api/types'

import { positionAt, type XY } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { type CanvasView, projectTo } from '../model/replayView'
import { buildLivesBySlot, lifeOfSlotAt } from '../model/livesPosition'

/** Cadrage du canvas (mêmes paramètres que worldToCanvas). */
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
 * aux points de LA VIE QUI COUVRE SON DÉPART. Une ligne dont aucune vie du slot ne couvre
 * l'instant ne se dessine pas — il n'y aurait aucun joueur à relier.
 *
 * UNE TRACE = UNE VIE, ET LE SLOT DE BIPED EST RÉATTRIBUÉ À CHAQUE RÉAPPARITION. C'est
 * l'invariant du dossier (`shotFx.ts`, `fireMark.ts`, `riftStations.ts`, `replayMarkers.ts`,
 * `thrusterDashFx.ts`) : indexer `slot -> points` ne garderait que la DERNIÈRE piste du slot,
 * et une accroche d'une vie antérieure irait chercher les points d'une autre vie — au mieux
 * elle disparaîtrait (l'instant précède les points retenus, `positionAt` rend `null`), au pire
 * le câble se peindrait à la position d'UN AUTRE JOUEUR, tendu vers une ancre qui n'est pas la
 * sienne. On groupe donc par slot, puis on retient la vie qui COUVRE le départ de la traction —
 * le patron exact de `buildShotFx`, `buildFireMarks` et `buildThrusterDashFx`.
 *
 * LE DÉPART (`t0`) EST L'INSTANT DE RÉFÉRENCE, pas `t1` : c'est le tir qui appartient à une
 * vie. Une traction peut se terminer après la mort du porteur — `positionAt` fige alors la
 * dernière position, ce qui est exactement ce qu'on veut voir.
 */
export function buildGrappleFx(doc: ReplayDocumentReady): GrappleFxEntry[] {
  if (doc.grappleLines.length === 0) return []
  const bySlot = buildLivesBySlot(doc.tracks)
  const out: GrappleFxEntry[] = []
  for (const l of doc.grappleLines) {
    if (l.t1 <= l.t0) continue
    const track = lifeOfSlotAt(bySlot, l.slot, l.t0)
    if (!track || track.points.length === 0) continue
    out.push({ t0: l.t0, t1: l.t1, anchor: { x: l.ax, y: l.ay }, points: track.points })
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
    const p = projectTo(view, pos)
    const a = projectTo(view, e.anchor)
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
