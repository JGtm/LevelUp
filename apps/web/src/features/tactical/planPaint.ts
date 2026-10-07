/**
 * planPaint.ts — l'ORDRE des calques du plan de l'onglet Tactique, posés sur le fond de carte :
 *   1. les CONTOURS des zones nommées (callouts du jeu), SOUS la chaleur ;
 *   2. la chaleur (`drawTacticalHeatmap`), LISSÉE par l'appelant (`chaleurLissee.ts`) et aux bords
 *      adoucis d'un léger flou : des zones de chaleur comme au rejeu, plus des carrés ;
 *   3. les NOMS des zones, AU-DESSUS, à leur point de référence.
 *
 * Les zones se peignent avec LE peintre du rejeu 2D (`lib/replay/calloutsPaint.ts`) : mêmes règles
 * (grandes zones à aplat léger et frontière, fines en pointillé, un nom par zone sur la plus haute,
 * blanc cerné de noir) et même taille de libellé. Seule l'encre change : celle du texte de l'app
 * (`text-foreground`, lue par l'appelant), que le peintre atténue — le plan n'a pas la série de
 * teintes du rejeu, ses couleurs sont celles de la chaleur.
 */
import type { Locale } from '@/lib/i18n/locale'
import { drawCalloutsLabels, drawCalloutsShapes, type CalloutZoneReady } from '@/lib/replay/calloutsPaint'
import { drawTacticalHeatmap, type TacticalGrid } from '@/lib/replay/heatPaint'

import { projectionDuPlan, vueDuPlan, type FenetreDuPlan, type RepereTactique } from './tacticalView.logic'

/**
 * Le flou des bords de la chaleur, en pas de la grille SERVIE : il lisse le contour en escalier des
 * sous-cellules sans déplacer les taches. GARDÉ FAIBLE À DESSEIN : un flou plus large éteignait la
 * tache d'une cellule isolée — souvent la plus chaude — au profit des amas tièdes.
 */
const FLOU_PAS = 0.15

/** Ce que le plan peint, une fois la lecture réindexée sur son repère. */
export interface PeintureDuPlan {
  /** La grille à peindre (déjà lissée), réindexée sur le repère. */
  grid: TacticalGrid
  ramp: readonly string[]
  /** Les zones nommées servies avec la lecture ([] quand le titre ou la carte n'en a pas). */
  zones: readonly CalloutZoneReady[]
  locale: Locale
  /** L'encre des contours, déjà résolue (jeton `foreground`). */
  encre: string
}

/**
 * peindreLePlan peint les trois calques dans leur ordre, projetés sur la FENÊTRE visible (le repère
 * entier à 1x) ; rien quand le repère est inexploitable.
 */
export function peindreLePlan(
  ctx: CanvasRenderingContext2D,
  repere: RepereTactique,
  canvasWidth: number,
  peinture: PeintureDuPlan,
  fenetre: FenetreDuPlan = repere,
): void {
  const vue = vueDuPlan(repere, canvasWidth, fenetre)
  const projection = projectionDuPlan(repere, canvasWidth, fenetre)
  if (!vue || !projection) return
  const { zones, locale, encre } = peinture
  drawCalloutsShapes(ctx, zones, projection, { bigColors: [encre], fineInk: encre, locale })
  peindreLaChaleur(ctx, peinture, vue, FLOU_PAS * repere.pasM * vue.scale)
  drawCalloutsLabels(ctx, zones, projection, locale)
}

/**
 * peindreLaChaleur — la chaleur, peinte sur une toile à part puis posée avec un flou de `flouPx`.
 * Sans filtre de canvas (navigateur ancien, environnement de test) : peinte directement, bords nets.
 */
function peindreLaChaleur(
  ctx: CanvasRenderingContext2D,
  peinture: PeintureDuPlan,
  vue: { topLeftWorld: { x: number; y: number }; scale: number },
  flouPx: number,
): void {
  const style = { ramp: peinture.ramp, k: 1 }
  const toile = 'filter' in ctx && flouPx > 0 ? document.createElement('canvas') : null
  const horsEcran = toile?.getContext('2d') ?? null
  if (!toile || !horsEcran) {
    drawTacticalHeatmap(ctx, peinture.grid, vue, style)
    return
  }
  toile.width = ctx.canvas.width
  toile.height = ctx.canvas.height
  drawTacticalHeatmap(horsEcran, peinture.grid, vue, style)
  ctx.save()
  ctx.filter = `blur(${Math.round(flouPx * 10) / 10}px)`
  ctx.drawImage(toile, 0, 0)
  ctx.restore()
}
