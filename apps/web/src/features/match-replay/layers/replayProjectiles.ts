/**
 * replayProjectiles.ts — LES VOLS DE PROJECTILE en cours, une polyligne par vol.
 *
 * EXTRAIT DE `replayMarkers.ts` LE 2026-08-18 (lot R2-V), pour la même raison que le cône de
 * visée : un vol de grenade n'est pas un marqueur de joueur, il ne partage avec eux que le
 * cadrage. Le fichier d'origine dépassait le seuil de taille du dépôt.
 */
import { type XY } from '../../../lib/replay/replayLogic'
import type { ReplayProjectileReady } from '../../../lib/replay/replayNormalize'
import { type CanvasView, projectTo } from '../model/replayView'


const PROJECTILE_ALPHA = 0.5
const PROJECTILE_WIDTH = 1.2
/**
 * Le vol reste visible 700 ms après son dernier point répliqué, puis s'efface. Le tracé reçoit la
 * durée en images (`ProjectileWindow.tailFrames`), convertie avec le pas du document.
 */
export const PROJECTILE_TAIL_MS = 700

/** Fenêtre d'affichage : l'image courante et la durée de la traînée, en images. */
export interface ProjectileWindow {
  frame: number
  tailFrames: number
}

/**
 * drawProjectilesLayer dessine les vols de projectile en cours.
 *
 * LE DERNIER POINT N'EST PAS UN IMPACT : le film ne porte aucun événement de détonation. C'est
 * la dernière position RÉPLIQUÉE — pour une grenade, la réplication cesse ~1,4 s après le
 * lancer alors que la mèche court jusqu'à ~3 s. Le vol s'efface donc, il n'explose pas.
 */
export function drawProjectilesLayer(
  ctx: CanvasRenderingContext2D,
  projectiles: ReplayProjectileReady[],
  view: CanvasView,
  win: ProjectileWindow,
  color: string,
): void {
  const { frame, tailFrames } = win
  ctx.strokeStyle = color
  ctx.lineWidth = PROJECTILE_WIDTH
  for (const pr of projectiles) {
    const pts = pr.p
    if (pts.length < 2) continue
    const end = pr.t0 + pts[pts.length - 1][0]
    if (frame < pr.t0 || frame > end + tailFrames) continue
    const fade = frame > end ? 1 - (frame - end) / tailFrames : 1
    ctx.globalAlpha = PROJECTILE_ALPHA * fade
    ctx.beginPath()
    let started = false
    for (const [dt, x, y] of pts) {
      if (pr.t0 + dt > frame) break
      const c = project({ x, y }, view)
      if (!started) {
        ctx.moveTo(c.x, c.y)
        started = true
      } else ctx.lineTo(c.x, c.y)
    }
    if (started) ctx.stroke()
  }
  ctx.globalAlpha = 1
}

function project(p: XY, view: CanvasView): XY {
  return projectTo(view, p)
}
