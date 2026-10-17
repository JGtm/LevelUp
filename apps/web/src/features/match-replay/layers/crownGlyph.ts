/**
 * crownGlyph.ts — LA FORME DE LA COURONNE VIP, décrite UNE seule fois pour ses DEUX usages : le
 * glyphe de la carte (`vipCrownLayer`, canvas) et le filigrane de la fiche
 * (`ReplayObjectiveMark`, SVG). Même partage que `skullGlyph.ts` : une forme recopiée à deux
 * endroits finit par diverger.
 *
 * LA SILHOUETTE, en repère 16×16 : un BANDEAU séparé du corps par un jour, un corps à trois
 * pointes, et une PERLE au bout de chaque pointe. Le bandeau et les perles sont ce qui fait lire
 * « couronne » à petite taille ; trois pointes nues se lisaient comme un M ou une colline.
 */
import type { XY } from '../../../lib/replay/replayLogic'

/** Côté du repère de la forme. */
export const CROWN_BOX = 16

/** Le corps : base gauche, pointe gauche, creux, pointe centrale, creux, pointe droite, base droite. */
const BODY: readonly XY[] = [
  { x: 2.6, y: 11 },
  { x: 1.6, y: 4.8 },
  { x: 5.3, y: 7.9 },
  { x: 8, y: 3.4 },
  { x: 10.7, y: 7.9 },
  { x: 14.4, y: 4.8 },
  { x: 13.4, y: 11 },
]

/** Le bandeau, sous le corps, séparé de lui par un jour de 0,8. */
const BAND = { x0: 2.6, y0: 11.8, x1: 13.4, y1: 13.8 }

/** Les perles, au bout des trois pointes. */
const PEARLS: readonly { c: XY; r: number }[] = [
  { c: { x: 1.6, y: 4.3 }, r: 1.3 },
  { c: { x: 8, y: 2.6 }, r: 1.5 },
  { c: { x: 14.4, y: 4.3 }, r: 1.3 },
]

/** Bas de la forme dans son repère : l'appelant y aligne le haut du marqueur du joueur. */
export const CROWN_BOTTOM = BAND.y1

const n = (v: number) => Number(v.toFixed(2))

function pearlSvg({ c, r }: { c: XY; r: number }): string {
  return `M${n(c.x - r)} ${n(c.y)}a${r} ${r} 0 1 1 ${n(2 * r)} 0a${r} ${r} 0 1 1 ${n(-2 * r)} 0Z`
}

/** Le chemin SVG de la couronne, en repère 16×16 (filigrane de la fiche). */
export const CROWN_SVG_PATH = [
  `M${BODY.map((p) => `${n(p.x)} ${n(p.y)}`).join('L')}Z`,
  `M${BAND.x0} ${BAND.y0}H${BAND.x1}V${BAND.y1}H${BAND.x0}Z`,
  ...PEARLS.map(pearlSvg),
].join(' ')

/**
 * traceCrown pose le chemin de la couronne sur le contexte (sans le peindre), le repère 16×16
 * placé en `origin` et agrandi d'un facteur `scale`. L'appelant choisit fill et/ou stroke.
 */
export function traceCrown(ctx: CanvasRenderingContext2D, origin: XY, scale: number): void {
  const px = (p: XY): XY => ({ x: origin.x + p.x * scale, y: origin.y + p.y * scale })
  ctx.beginPath()
  BODY.forEach((p, i) => {
    const q = px(p)
    if (i === 0) ctx.moveTo(q.x, q.y)
    else ctx.lineTo(q.x, q.y)
  })
  ctx.closePath()
  const b0 = px({ x: BAND.x0, y: BAND.y0 })
  const b1 = px({ x: BAND.x1, y: BAND.y1 })
  ctx.rect(b0.x, b0.y, b1.x - b0.x, b1.y - b0.y)
  for (const { c, r } of PEARLS) {
    const q = px(c)
    ctx.moveTo(q.x + r * scale, q.y)
    ctx.arc(q.x, q.y, r * scale, 0, Math.PI * 2)
  }
}
