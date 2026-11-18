/**
 * reliefLinesLayer — LES TRAITS DU RELIEF, EN VECTEURS PAR-DESSUS LE FOND.
 *
 * Quatre familles, toutes en pixels du fond (`lib/replay/relief`) : les courbes de niveau sur les
 * pentes, les murs du maillage dans la bande de la coupe (seulement quand une coupe est active), les
 * ruptures de hauteur et le liseré de la zone jouable (anneaux fermés). Chaque famille est UN chemin
 * et un seul `stroke` : une carte en porte des dizaines de milliers de segments.
 *
 * LA PROJECTION EST CELLE DE LA VUE, passée par l'appelant : pixel du fond -> monde (`pixelToWorld`, le
 * pixel a son centre en `(i + 0,5)`) -> toile (`project`, la projection monde -> toile de la vue, rotation par
 * quarts de tour comprise). Ce module ne connaît aucune vue : le rejeu, la Tactique et « Occupation du terrain »
 * lui donnent chacun la leur, comme au plan des murs. Aucune rotation n'est réécrite ici, et l'épaisseur est en
 * pixels d'écran : les traits restent nets à tout zoom.
 *
 * CALQUE STATIQUE, comme le plan des murs : il ne dépend ni de l'image courante ni de la lecture, mais du
 * cadrage, de la coupe et des encres. Le rejeu le cuit hors écran (`useReplayStaticLayers`) et le recopie.
 *
 * Aucune couleur ici : les encres viennent de l'appelant (jeton `map-relief-base` et ses mélanges, filet
 * du cerne des pions pour le liseré).
 */
import { lineCount, pixelToWorld, traceLineSet } from './lineSet'
import type { LineSet, PointXY, ReliefCalibration } from './types'

/** Les quatre encres des traits, en couleurs CSS. */
export interface ReliefLinesInk {
  edge: string
  contour: string
  wall: string
  outline: string
}

/** Ce que le calque dessine : les traits d'une coupe (ou de « Tout ») et leur calage. */
export interface ReliefLinesInput {
  calibration: ReliefCalibration
  steps: LineSet
  contours: LineSet
  /** Les murs de la coupe ; vide sans coupe. */
  walls: LineSet
  /** Le liseré : jamais filtré par la coupe. */
  outline: LineSet
  ink: ReliefLinesInk
}

/** Le style d'une famille de traits : opacité, épaisseur (px d'écran), encre, et fermeture des polylignes. */
interface FamilyStyle {
  alpha: number
  width: number
  ink: string
  closed: boolean
}

/** Opacité et épaisseur de chaque famille : celles de la maquette validée du 2026-11-18. */
const CONTOUR_STYLE = { alpha: 0.35, width: 0.75, closed: false }
const WALL_STYLE = { alpha: 0.85, width: 1.1, closed: false }
const EDGE_STYLE = { alpha: 0.7, width: 1, closed: false }
const OUTLINE_STYLE = { alpha: 1, width: 1.6, closed: true }

/** Trace une famille en un seul chemin ; ne touche pas au contexte quand elle est vide. */
function strokeFamily(
  ctx: CanvasRenderingContext2D,
  lines: LineSet,
  toScreen: (px: number, py: number) => PointXY,
  style: FamilyStyle,
): void {
  if (lineCount(lines) === 0) return
  ctx.globalAlpha = style.alpha
  ctx.strokeStyle = style.ink
  ctx.lineWidth = style.width
  ctx.beginPath()
  traceLineSet(ctx, lines, toScreen, style.closed)
  ctx.stroke()
}

/**
 * drawReliefLinesLayer trace les traits du relief avec la projection monde -> toile `project`, du plus
 * discret au plus net : courbes, murs de la coupe, ruptures, liseré. Le contexte retrouve son état d'avant.
 */
export function drawReliefLinesLayer(
  ctx: CanvasRenderingContext2D,
  input: ReliefLinesInput,
  project: (world: PointXY) => PointXY,
): void {
  const { calibration, ink } = input
  const toScreen = (px: number, py: number) => project(pixelToWorld(calibration, px, py))
  ctx.save()
  ctx.lineCap = 'butt'
  ctx.lineJoin = 'round'
  strokeFamily(ctx, input.contours, toScreen, { ...CONTOUR_STYLE, ink: ink.contour })
  strokeFamily(ctx, input.walls, toScreen, { ...WALL_STYLE, ink: ink.wall })
  strokeFamily(ctx, input.steps, toScreen, { ...EDGE_STYLE, ink: ink.edge })
  strokeFamily(ctx, input.outline, toScreen, { ...OUTLINE_STYLE, ink: ink.outline })
  ctx.restore()
}
