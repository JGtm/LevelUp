/**
 * canvasStage.ts — LA PRÉPARATION DE LA TOILE avant chaque tracé du rejeu : la taille de dessin à la
 * densité du moment, le repère, l'effacement.
 *
 * EXTRAIT DE `ReplayCanvas.tsx` (lot 2 du plan « maquettes du rejeu »), sans changement de rendu : le
 * composant est au plafond de taille, et ce geste-là n'a rien à voir avec le choix des calques.
 *
 * LA DENSITÉ EST CELLE DU FORMAT PENDANT UN EXPORT, celle de l'écran sinon (`canvasPixelRatio`). Le
 * backing store n'est retouché que s'il a changé : le redimensionner EFFACE la toile et coûte une
 * réallocation, à chaque image.
 */
import { canvasPixelRatio } from '../export/exportLayoutStore'

/** Ce qu'un tracé reçoit : le contexte prêt, et la densité à passer aux calques. */
export interface Stage {
  ctx: CanvasRenderingContext2D
  dpr: number
}

/**
 * prepareStage dimensionne, pose le repère à la densité et efface. `null` quand il n'y a rien à
 * peindre : pas de toile montée, largeur pas encore mesurée, ou contexte indisponible.
 */
export function prepareStage(
  canvas: HTMLCanvasElement | null,
  renderWidth: number,
  renderHeight: number,
): Stage | null {
  if (!canvas || renderWidth === 0) return null
  const ctx = canvas.getContext('2d')
  if (!ctx) return null
  const dpr = canvasPixelRatio()
  const pw = Math.round(renderWidth * dpr)
  const ph = Math.round(renderHeight * dpr)
  if (canvas.width !== pw || canvas.height !== ph) {
    canvas.width = pw
    canvas.height = ph
  }
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, renderWidth, renderHeight)
  return { ctx, dpr }
}
