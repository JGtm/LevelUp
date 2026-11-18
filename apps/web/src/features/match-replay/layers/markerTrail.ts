/**
 * markerTrail.ts — LA TRAÎNÉE D'UNE VIE et L'ANNEAU D'APPARITION de son premier instant.
 *
 * Extrait de `replayMarkers.ts` : ces deux tracés ne lisent du style du calque que l'image
 * courante, les durées (`timing.trail`, `timing.spawn`) et la densité du canevas.
 */
import { trackWindow, trailAt, type XY } from '../../../lib/replay/replayLogic'
import type { ReplayTrackReady } from '../../../lib/replay/replayNormalize'
import { type CanvasView, projectTo } from '../model/replayView'

/** Ce que ces tracés lisent du style du calque. */
export interface TrailStyle {
  frame: number
  /** Densité du canevas : tout ce qui s'adresse à l'œil est multiplié par ce facteur. */
  k: number
  timing: { trail: number; spawn: number }
}

/**
 * LA TRAÎNÉE S'ÉCLAIRCIT VERS LE PASSÉ. Un seul trait à opacité constante disait « le joueur
 * est passé par là » sans dire DANS QUEL SENS ; l'alpha qui monte vers la tête donne la
 * direction du mouvement sans ajouter la moindre flèche. La polyligne se dessine donc segment
 * par segment, chacun avec son opacité.
 */
const TRAIL_WIDTH = 1.6
const TRAIL_ALPHA_TAIL = 0.08
const TRAIL_ALPHA_HEAD = 0.63
const SPAWN_RADIUS = 2
const SPAWN_GROWTH = 12
const SPAWN_WIDTH = 1.2
const SPAWN_ALPHA = 0.8

/**
 * drawTrail : la polyligne des 7 s écoulées, dessinée SEGMENT PAR SEGMENT.
 *
 * Un seul `stroke` à opacité constante coûterait moins cher, mais il ne dirait pas le SENS du
 * déplacement. L'opacité monte linéairement de la queue (0,08 — presque effacé) vers la tête
 * (0,63) : la trace la plus visible est toujours celle de l'instant.
 */
export function drawTrail(
  ctx: CanvasRenderingContext2D,
  track: ReplayTrackReady,
  view: CanvasView,
  style: TrailStyle,
  color: string,
): void {
  const trail = trailAt(track.points, style.frame, style.timing.trail)
  if (trail.length < 2) return
  const segments = trail.length - 1
  const span = TRAIL_ALPHA_HEAD - TRAIL_ALPHA_TAIL
  ctx.strokeStyle = color
  ctx.lineWidth = TRAIL_WIDTH * style.k
  let from = projectTo(view, trail[0])
  for (let i = 1; i < trail.length; i++) {
    const to = projectTo(view, trail[i])
    ctx.globalAlpha = TRAIL_ALPHA_TAIL + (span * i) / segments
    ctx.beginPath()
    ctx.moveTo(from.x, from.y)
    ctx.lineTo(to.x, to.y)
    ctx.stroke()
    from = to
  }
  ctx.globalAlpha = 1
}

/** drawSpawnRing : l'anneau qui s'ouvre au premier instant de la vie. */
export function drawSpawnRing(
  ctx: CanvasRenderingContext2D,
  track: ReplayTrackReady,
  c: XY,
  style: TrailStyle,
  color: string,
): void {
  const age = style.frame - trackWindow(track).start
  if (age < 0 || age > style.timing.spawn) return
  const f = 1 - age / style.timing.spawn
  ctx.beginPath()
  ctx.arc(c.x, c.y, (SPAWN_RADIUS + SPAWN_GROWTH * (1 - f)) * style.k, 0, Math.PI * 2)
  ctx.strokeStyle = color
  ctx.globalAlpha = SPAWN_ALPHA * f
  ctx.lineWidth = SPAWN_WIDTH * style.k
  ctx.stroke()
  ctx.globalAlpha = 1
}
