/**
 * replayLabels.ts — LE NOM SOUS LE POINT.
 *
 * DEMANDE UTILISATEUR DU 2026-08-16, mot pour mot : « ajouter le pseudo en dessous des points
 * (avec une bordure sur les lettres pour faciliter la lecture sur les fonds clairs et
 * foncés) ». La bordure n'est pas un ornement : le fond de carte cuit va du gris très clair
 * (aire jouable) au presque noir (pourtour), et un nom d'une seule couleur y disparaît d'un
 * côté ou de l'autre. C'est la technique du POC — `strokeText` puis `fillText`, contour rond
 * pour que les jonctions de lettres ne fassent pas de pointes.
 *
 * LE CONTOUR EST SOMBRE DANS LES DEUX THÈMES (`--replay-label-stroke`, cf. globals.css) : il
 * cerne des lettres qui portent, elles, la couleur d'ÉQUIPE du joueur. Un contour clair en
 * thème clair rendrait le nom illisible sur l'aire jouable, qui est justement claire.
 *
 * POURQUOI UN FICHIER À PART : `replayMarkers.ts` frôlait le seuil de taille du dépôt, et
 * l'étiquette est une préoccupation distincte du marqueur (typographie contre géométrie).
 */
import { CANVAS_FONT_FAMILY } from '@/lib/replay/canvasFont'
import type { XY } from '../../../lib/replay/replayLogic'
import { drawTitleRun, titleRunWidth, type LabelTitle } from './antagonistMarks'

/** Corps de la police, en pixels d'ÉCRAN (comme tout ce qui s'adresse à l'œil ici). */
const LABEL_FONT_PX = 8.5
/** Graisse : semi-gras — un nom de 8,5 px en graisse normale se dissout sous son contour. */
const LABEL_WEIGHT = 600
/** Épaisseur du contour de lisibilité, en pixels d'écran. */
const LABEL_STROKE_PX = 2.6
/** Écart entre le bord externe du marqueur et le haut du texte. */
const LABEL_GAP_PX = 2.5
/** Où tombe le centre optique d'une ligne de texte posée par son haut, en fraction du corps. */
const LABEL_MID_RATIO = 0.6

/**
 * Ce que l'étiquette emprunte au style du calque. Structurel à dessein : `MarkerStyle` le
 * satisfait sans que ce fichier ait à importer le marqueur (l'inverse serait un cycle).
 */
export interface LabelStyle {
  /** Densité du canevas. */
  k: number
  /** Encre du contour des lettres. Vide = le thème ne porte pas la variable : pas de contour. */
  labelStroke: string
  /**
   * Les titres d'antagoniste du joueur (icône et compte), écrits devant le nom ; l'ensemble est centré sous
   * le marqueur. Absents ou vides = le nom seul, centré, comme toujours.
   */
  titles?: readonly LabelTitle[]
}

/**
 * drawNameLabel écrit `name` centré sous le marqueur.
 *
 * `markerEdge` est la distance du centre au bord EXTERNE de TOUT ce que le marqueur dessine —
 * son liseré, son anneau d'étage (posé à un rayon fixe, plus large que le disque) ou son
 * anneau d'identité, le plus grand des trois — en pixels d'écran : elle appartient à la
 * géométrie du marqueur, qui la calcule et la passe. Le texte s'aligne par son HAUT
 * (`textBaseline top`), donc il descend sous ce bord, jamais il ne remonte sur le point.
 */
export function drawNameLabel(
  ctx: CanvasRenderingContext2D,
  c: XY,
  name: string,
  style: LabelStyle,
  color: string,
  markerEdge: number,
): void {
  const k = style.k
  const titles = style.titles?.length ? style.titles : null
  ctx.font = `${LABEL_WEIGHT} ${LABEL_FONT_PX * k}px ${CANVAS_FONT_FAMILY}`
  if (!titles) ctx.textAlign = 'center'
  ctx.textBaseline = 'top'
  const y = c.y + markerEdge + LABEL_GAP_PX * k
  ctx.globalAlpha = 1
  if (!titles) {
    strokeAndFill(ctx, name, { x: c.x, y }, style, color)
    return
  }
  // LES TITRES D'ABORD, puis le nom, l'ensemble centré : la suite « icône, compte » se dessine depuis la
  // gauche, le nom prend l'abscisse où elle s'arrête.
  const left = c.x - (titleRunWidth(ctx, titles, k) + ctx.measureText(name).width) / 2
  const x = drawTitleRun(ctx, titles, { x: left, y: y + LABEL_MID_RATIO * LABEL_FONT_PX * k }, style, color)
  ctx.textAlign = 'left'
  ctx.textBaseline = 'top'
  strokeAndFill(ctx, name, { x, y }, style, color)
}

/** Le texte au double trait : le contour s'il y a une encre, puis le remplissage. */
function strokeAndFill(
  ctx: CanvasRenderingContext2D,
  text: string,
  at: XY,
  style: LabelStyle,
  color: string,
): void {
  if (style.labelStroke) {
    ctx.lineJoin = 'round'
    ctx.lineWidth = LABEL_STROKE_PX * style.k
    ctx.strokeStyle = style.labelStroke
    ctx.strokeText(text, at.x, at.y)
  }
  ctx.fillStyle = color
  ctx.fillText(text, at.x, at.y)
}
