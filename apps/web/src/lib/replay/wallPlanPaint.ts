/**
 * wallPlanPaint — LE TRAIT DU PLAN DES MURS, partagé par toutes les cartes 2D de l'application.
 *
 * Le plan des murs d'une carte (`lib/replay/wallPlan.ts`) est un nuage d'arêtes en mètres MONDE.
 * Chaque vue 2D (rejeu, onglet Tactique, Occupation du terrain) les trace de la même façon, avec sa
 * PROPRE projection monde -> toile : ce module ne connaît que cette fonction de projection, pas le
 * cadrage de la vue. Une seule définition du trait (épaisseur, opacité, un seul chemin) : c'est ce
 * qui garde les trois vues identiques.
 *
 * UN SEUL TRAIT, UNE SEULE FOIS : toutes les arêtes forment un seul chemin tracé d'un coup. Une
 * carte en porte plusieurs dizaines de milliers ; un `stroke` par arête serait la cuisson la plus
 * lente du rejeu pour un résultat identique.
 *
 * Aucune couleur ici : l'encre vient de l'appelant (jeton du thème).
 */
import type { XY } from './replayLogic'
import type { WallPlanReady } from './wallPlan'

/** Épaisseur du trait, en pixels CSS : le contexte est déjà à la densité de la toile. */
export const WALL_PLAN_LINE_WIDTH = 1

/**
 * Opacité du trait : assez pour se lire sur une image claire comme sombre, assez peu pour que le
 * plan ne couvre pas les contours que l'image dessine déjà.
 */
export const WALL_PLAN_ALPHA = 0.6

/** strokeWallPlan trace les arêtes projetées par `project`. Sans plan ni arête, il ne touche pas au contexte. */
export function strokeWallPlan(
  ctx: CanvasRenderingContext2D,
  plan: WallPlanReady | null,
  project: (p: XY) => XY,
  ink: string,
): void {
  if (!plan || plan.edges.length === 0) return
  ctx.save()
  ctx.globalAlpha = WALL_PLAN_ALPHA
  ctx.strokeStyle = ink
  ctx.lineWidth = WALL_PLAN_LINE_WIDTH
  ctx.lineCap = 'butt'
  ctx.beginPath()
  for (const [x1, y1, x2, y2] of plan.edges) {
    const a = project({ x: x1, y: y1 })
    const b = project({ x: x2, y: y2 })
    ctx.moveTo(a.x, a.y)
    ctx.lineTo(b.x, b.y)
  }
  ctx.stroke()
  ctx.restore()
}
