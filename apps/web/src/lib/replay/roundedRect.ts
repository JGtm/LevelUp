/** Un rectangle posé sur un canvas : coin haut gauche, largeur et hauteur, en pixels. */
export interface PixelRect {
  x: number
  y: number
  w: number
  h: number
}

/**
 * roundedRectPath pose le tracé (sans le remplir) d'un rectangle à coins arrondis, un chemin neuf
 * (`beginPath`) fermé. C'est LE seul endroit du front qui trace un rectangle arrondi : `ctx.roundRect`
 * quand le moteur l'offre ; sinon quatre `arcTo`, au rayon borné à la demi-largeur et à la demi-hauteur
 * (comme le fait `roundRect`) — jamais un rectangle net ni un tracé vide.
 */
export function roundedRectPath(ctx: CanvasRenderingContext2D, rect: PixelRect, radius: number): void {
  const { x, y, w, h } = rect
  ctx.beginPath()
  if (typeof ctx.roundRect === 'function') {
    ctx.roundRect(x, y, w, h, radius)
    return
  }
  const r = Math.max(0, Math.min(radius, w / 2, h / 2))
  ctx.moveTo(x + r, y)
  ctx.arcTo(x + w, y, x + w, y + h, r)
  ctx.arcTo(x + w, y + h, x, y + h, r)
  ctx.arcTo(x, y + h, x, y, r)
  ctx.arcTo(x, y, x + w, y, r)
  ctx.closePath()
}
