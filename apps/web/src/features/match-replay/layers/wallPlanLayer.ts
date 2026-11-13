/**
 * wallPlanLayer — LE PLAN DES MURS D'UNE CARTE, EN TRAITS FINS PAR-DESSUS SON FOND.
 *
 * Le sidecar du fond de carte porte les arêtes des murs en mètres MONDE, dans le repère du
 * rejeu (`model/wallPlan.ts`). Ce calque les trace comme `drawCalloutsLayer` trace les zones
 * nommées : en vecteurs, projetés par le cadrage partagé (`projectTo`), donc exacts à tout zoom
 * et sous toute orientation de la carte. Il se pose sur l'image (qui en dessine déjà une
 * lecture, mais en pixels figés) et sous tout ce qui bouge.
 *
 * CALQUE STATIQUE : il ne dépend ni de l'image courante ni de la lecture, seulement du cadrage
 * et de son encre. L'appelant le cuit hors écran (`useReplayStaticLayers`) et le recopie, comme
 * les zones nommées.
 *
 * UN SEUL TRAIT, UNE SEULE FOIS : toutes les arêtes forment un seul chemin tracé d'un coup. Une
 * carte en porte plusieurs dizaines de milliers ; un `stroke` par arête serait la cuisson la
 * plus lente du rejeu pour un résultat identique.
 *
 * Aucune couleur ici : l'encre vient de l'appelant (token du thème, `canvasInk`).
 */
import { projectTo, type CanvasView } from '../model/replayView'
import type { WallPlanReady } from '../model/wallPlan'

/** Épaisseur du trait, en pixels CSS : le contexte cuit est déjà à la densité de la toile. */
export const WALL_PLAN_LINE_WIDTH = 1

/**
 * Opacité du trait : assez pour se lire sur une image claire comme sombre, assez peu pour que
 * le plan ne couvre pas les contours que l'image dessine déjà.
 */
export const WALL_PLAN_ALPHA = 0.6

/** L'encre du plan : l'encre de mise en page la plus contrastée du thème, résolue par l'appelant. */
export interface WallPlanStyle {
  ink: string
}

/**
 * drawWallPlanLayer trace les arêtes du plan. Sans plan (ou sans arête), il ne touche pas au
 * contexte : pas de calque plutôt qu'un calque vide.
 */
export function drawWallPlanLayer(
  ctx: CanvasRenderingContext2D,
  plan: WallPlanReady | null,
  view: CanvasView,
  style: WallPlanStyle,
): void {
  if (!plan || plan.edges.length === 0) return
  ctx.save()
  ctx.globalAlpha = WALL_PLAN_ALPHA
  ctx.strokeStyle = style.ink
  ctx.lineWidth = WALL_PLAN_LINE_WIDTH
  ctx.lineCap = 'butt'
  ctx.beginPath()
  for (const [x1, y1, x2, y2] of plan.edges) {
    const a = projectTo(view, { x: x1, y: y1 })
    const b = projectTo(view, { x: x2, y: y2 })
    ctx.moveTo(a.x, a.y)
    ctx.lineTo(b.x, b.y)
  }
  ctx.stroke()
  ctx.restore()
}
