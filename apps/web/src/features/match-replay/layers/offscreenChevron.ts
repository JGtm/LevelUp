/**
 * offscreenChevron.ts — LA FLÈCHE HORS CADRE (plan escouade hors cadre, chantier B, décision
 * D2, gabarit fourni par l'utilisateur : `Pictures/Screenpresso/2026-09-09_14h26_04.png`).
 *
 * CE QUE CE FICHIER DESSINE, ET RIEN DE PLUS : le glyphe posé par `edgeMarkFor` (`model/
 * edgeClamp.ts`) — une flèche PLEINE À BASE CONCAVE, pivotée vers la position réelle, et
 * l'étiquette « nom · distance » qui l'accompagne. La géométrie du bornage (où, sous quel
 * angle, à quelle distance) vit dans `edgeClamp.ts` ; ce module ne fait QUE peindre ce qu'elle
 * a calculé — même partage que `edgeClamp.ts` / les calques de tracé partout ailleurs dans le
 * rejeu.
 *
 * LE GABARIT EST NORMALISÉ, POINTANT +X : `ctx.rotate` fait le reste. Quatre sommets ferment le
 * tracé — pointe, arrière-gauche, ENCOCHE (le creux qui rend la base concave, pas un triangle
 * plein), arrière-droite — exactement les coordonnées du gabarit validé par l'utilisateur.
 * `ctx.scale` porte le gabarit à sa taille d'ÉCRAN : `CHEVRON_SIZE_PX * k`, jamais un facteur du
 * canevas — même convention que tout ce qui s'adresse à l'œil dans cette feature (cf. l'en-tête
 * de `replayMarkers.ts`).
 *
 * L'ÉTIQUETTE SE POSE DU CÔTÉ INTÉRIEUR DE LA FLÈCHE, à l'opposé de `angle` : la flèche pointe
 * vers une position qui est HORS TOILE, poser le texte dans son prolongement le sortirait du
 * cadre visible. Elle emprunte la même technique de lisibilité que `replayLabels.ts` (un
 * contour puis un remplissage), à une ENCRE résolue par l'appelant — jamais un littéral, jamais
 * `readInk` appelé d'ici (cf. `canvasInk.ts`, la règle color-tokens).
 */
import type { XY } from '../../../lib/replay/replayLogic'

/**
 * Le gabarit normalisé, pointant +X — quatre sommets, dans l'ORDRE où `ctx.lineTo` les trace :
 * pointe, arrière-gauche, encoche (le creux central, base CONCAVE), arrière-droite. Ce sont les
 * coordonnées exactes du gabarit fourni par l'utilisateur (décision D2), à l'échelle de 1.
 */
const TIP: XY = { x: 1, y: 0 }
const BACK_LEFT: XY = { x: -0.75, y: -0.7 }
const NOTCH: XY = { x: -0.35, y: 0 }
const BACK_RIGHT: XY = { x: -0.75, y: 0.7 }

/** Taille du gabarit, en pixels d'ÉCRAN DE RÉFÉRENCE (avant mise à l'échelle par `k`). */
export const CHEVRON_SIZE_PX = 12

/**
 * drawOffscreenChevron peint le gabarit à `at`, pivoté de `angle` (radians, espace canvas),
 * mis à l'échelle de l'écran par `k`, rempli de `color`.
 *
 * `save`/`restore` encadrent le geste : `translate`/`rotate`/`scale` ne doivent fuiter vers
 * aucun calque suivant (même règle que `drawRotatedSprite`, `replayDraw.ts`).
 */
export function drawOffscreenChevron(
  ctx: CanvasRenderingContext2D,
  at: XY,
  angle: number,
  k: number,
  color: string,
): void {
  ctx.save()
  ctx.translate(at.x, at.y)
  ctx.rotate(angle)
  ctx.scale(CHEVRON_SIZE_PX * k, CHEVRON_SIZE_PX * k)
  ctx.beginPath()
  ctx.moveTo(TIP.x, TIP.y)
  ctx.lineTo(BACK_LEFT.x, BACK_LEFT.y)
  ctx.lineTo(NOTCH.x, NOTCH.y)
  ctx.lineTo(BACK_RIGHT.x, BACK_RIGHT.y)
  ctx.closePath()
  ctx.fillStyle = color
  ctx.fill()
  ctx.restore()
}

/** Écart entre le repère et l'ancre de l'étiquette, en pixels d'ÉCRAN DE RÉFÉRENCE. */
const LABEL_GAP_PX = 14
/** Corps de la police, en pixels d'ÉCRAN — même corps que `replayLabels.ts`. */
const LABEL_FONT_PX = 8.5
const LABEL_WEIGHT = 600
const LABEL_STROKE_PX = 2.6

/**
 * offscreenLabelAnchor rend le point où poser l'étiquette : à `LABEL_GAP_PX * k` de `at`, dans
 * la direction OPPOSÉE à `angle` — le côté INTÉRIEUR de la flèche, celui qui reste sur la toile.
 * Fonction PURE, testée à part de son tracé.
 */
export function offscreenLabelAnchor(at: XY, angle: number, k: number): XY {
  const gap = LABEL_GAP_PX * k
  return { x: at.x - Math.cos(angle) * gap, y: at.y - Math.sin(angle) * gap }
}

/** Ce que le tracé de l'étiquette emprunte à l'appelant : la densité de l'écran, et le contour. */
export interface OffscreenLabelStyle {
  /** Densité du canevas : tout ce qui s'adresse à l'œil est multiplié par ce facteur. */
  k: number
  /** Encre du contour de lisibilité. Vide = le thème ne porte pas la variable : pas de contour. */
  labelStroke: string
}

/**
 * drawOffscreenLabel écrit `text` (déjà composé par l'appelant — « nom · distance », ou tout
 * autre libellé) centré sur `offscreenLabelAnchor`, au double trait (contour puis
 * remplissage) — même technique que `replayLabels.ts`, seconde copie assumée (règle des 3
 * copies, CLAUDE.md n° 6) : la position diffère (ancrée sur un point directionnel, pas sous un
 * marqueur), une troisième occurrence centraliserait les deux.
 */
export function drawOffscreenLabel(
  ctx: CanvasRenderingContext2D,
  at: XY,
  angle: number,
  text: string,
  style: OffscreenLabelStyle,
  color: string,
): void {
  const k = style.k
  const anchor = offscreenLabelAnchor(at, angle, k)
  ctx.font = `${LABEL_WEIGHT} ${LABEL_FONT_PX * k}px ui-sans-serif, system-ui, sans-serif`
  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'
  ctx.globalAlpha = 1
  if (style.labelStroke) {
    ctx.lineJoin = 'round'
    ctx.lineWidth = LABEL_STROKE_PX * k
    ctx.strokeStyle = style.labelStroke
    ctx.strokeText(text, anchor.x, anchor.y)
  }
  ctx.fillStyle = color
  ctx.fillText(text, anchor.x, anchor.y)
}
