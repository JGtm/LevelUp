/**
 * bombGlyph.ts — LE GLYPHE DE LA BOMBE d'Assaut, tracé UNE seule fois pour ses DEUX états
 * (portée au-dessus du porteur, posée au sol), comme `skullGlyph.ts` dont il reprend le parti.
 *
 * POURQUOI VECTORIEL ET PAS LA VIGNETTE DU JEU, et c'est une décision, pas un oubli. La bombe a
 * bien un sprite extrait (`contour-34`, nommé « ball | bomb » dans l'index de l'atlas, tags weap
 * `3fee4fcf`/`523b4648`) — mais son URL n'est PAS cuite dans le document du rejeu, et le client
 * ne DEVINE jamais un index d'atlas : c'est le piège `killfeed-NN` documenté dans l'en-tête de
 * `skullGlyph.ts` (un index qui bouge d'une saison casse la jointure EN SILENCE). Le crâne a
 * tranché exactement ce cas le 2026-08-28 : glyphe canvas tenant le rôle tant qu'un lot Go ne
 * publie pas l'URL dans le document. Deux objets de mode du même écran suivent la même règle.
 *
 * SON HABILLAGE EST CELUI DU CRÂNE ET DU DRAPEAU : un LISERÉ à l'encre du FOND sous la
 * silhouette, qui rend un bord franc sur un fond de carte photographique, dans les deux thèmes
 * (encres RÉSOLUES par l'appelant — jamais une couleur en dur, règle color-tokens).
 *
 * LA FORME EST CELLE DU FILIGRANE DE FICHE (`ReplayObjectiveMark.tsx`, glyphe `bomb`) : un corps
 * rond, un collier, une mèche en biais — et c'est la MÈCHE, pas le corps, qui porte
 * l'identification : sans elle la bombe serait indiscernable du crâne à cette taille. Les cotes
 * pèsent comme le crâne (r=7) pour que les objets de mode aient le même poids visuel.
 */
import type { XY } from './replayLogic'

/** Rayon du corps, en pixels canvas. Le gabarit total (corps + collier + mèche) pèse ~r=7. */
export const BOMB_GLYPH_RADIUS = 6

/** Débord du liseré au-delà de la silhouette — MÊME COTE que le crâne et le drapeau. */
const BOMB_LISERE_PAD = 1.6
/** Largeur du trait de liseré (centré sur le chemin : seule la moitié extérieure déborde). */
const BOMB_LISERE_WIDTH = 2 * BOMB_LISERE_PAD

/** Le collier, posé sur le corps : demi-largeur et bornes verticales (repère centre du corps). */
const BOMB_NECK_HALF_W = 1.9
const BOMB_NECK_TOP = -8.4
const BOMB_NECK_BOTTOM = -4.6

/** La mèche : du sommet du collier vers le haut-droite, en léger arc. */
const BOMB_FUSE_X0 = 0.6
const BOMB_FUSE_Y0 = -8.2
const BOMB_FUSE_CTRL_X = 2.8
const BOMB_FUSE_CTRL_Y = -10.6
const BOMB_FUSE_X1 = 4.4
const BOMB_FUSE_Y1 = -10.2
const BOMB_FUSE_WIDTH = 1.5

/** Ce que le tracé d'une bombe a besoin de savoir (règle des 5 paramètres). */
export interface BombGlyphPaint {
  /** Encre de la silhouette : le neutre du thème, résolu par l'appelant. */
  ink: string
  /** Encre du FOND : le liseré posé SOUS la silhouette (`markInk.outline`). */
  outline: string
  alpha: number
}

/** Trace la silhouette (corps + collier) sur le chemin courant du contexte. */
function bombSilhouette(ctx: CanvasRenderingContext2D, c: XY): void {
  ctx.beginPath()
  ctx.arc(c.x, c.y, BOMB_GLYPH_RADIUS, 0, Math.PI * 2)
  ctx.rect(
    c.x - BOMB_NECK_HALF_W,
    c.y + BOMB_NECK_TOP,
    2 * BOMB_NECK_HALF_W,
    BOMB_NECK_BOTTOM - BOMB_NECK_TOP,
  )
}

/** Trace la mèche sur le chemin courant du contexte. */
function bombFuse(ctx: CanvasRenderingContext2D, c: XY): void {
  ctx.beginPath()
  ctx.moveTo(c.x + BOMB_FUSE_X0, c.y + BOMB_FUSE_Y0)
  ctx.quadraticCurveTo(
    c.x + BOMB_FUSE_CTRL_X,
    c.y + BOMB_FUSE_CTRL_Y,
    c.x + BOMB_FUSE_X1,
    c.y + BOMB_FUSE_Y1,
  )
}

/**
 * drawBombGlyph pose la bombe à `center` : liseré à l'encre du fond sous toute la silhouette
 * (mèche comprise), remplissage à l'encre neutre par-dessus — la technique du bord franc du
 * drapeau et du crâne.
 *
 * LE POINT SERVI EST LE CENTRE DU CORPS, jamais un point décalé : l'appelant applique son propre
 * décalage (au-dessus du marqueur du porteur, ou sur le dernier point du lâcheur au sol).
 */
export function drawBombGlyph(ctx: CanvasRenderingContext2D, center: XY, paint: BombGlyphPaint): void {
  ctx.globalAlpha = paint.alpha
  // LE LISERÉ D'ABORD : silhouette et mèche à l'encre du fond, en trait large.
  ctx.strokeStyle = paint.outline
  ctx.lineWidth = BOMB_LISERE_WIDTH
  ctx.lineCap = 'round'
  bombSilhouette(ctx, center)
  ctx.stroke()
  bombFuse(ctx, center)
  ctx.lineWidth = BOMB_FUSE_WIDTH + BOMB_LISERE_WIDTH
  ctx.stroke()
  // LA SILHOUETTE, à l'encre neutre : elle recouvre la moitié intérieure du liseré.
  ctx.fillStyle = paint.ink
  bombSilhouette(ctx, center)
  ctx.fill()
  // LA MÈCHE, à l'encre neutre sur son liseré.
  ctx.strokeStyle = paint.ink
  ctx.lineWidth = BOMB_FUSE_WIDTH
  bombFuse(ctx, center)
  ctx.stroke()
}
