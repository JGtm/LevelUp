/**
 * planPaint.ts — l'ORDRE des calques du plan de l'onglet Tactique, posés sur le fond de carte :
 *   1. les CONTOURS des zones nommées (callouts du jeu), SOUS la chaleur ;
 *   2. la chaleur (`drawTacticalHeatmap`) ;
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

import { projectionDuPlan, vueDuPlan, type RepereTactique } from './tacticalView.logic'

/** Ce que le plan peint, une fois la lecture réindexée sur son repère. */
export interface PeintureDuPlan {
  grid: TacticalGrid
  ramp: readonly string[]
  /** Les zones nommées servies avec la lecture ([] quand le titre ou la carte n'en a pas). */
  zones: readonly CalloutZoneReady[]
  locale: Locale
  /** L'encre des contours, déjà résolue (jeton `foreground`). */
  encre: string
}

/** peindreLePlan peint les trois calques dans leur ordre ; rien quand le repère est inexploitable. */
export function peindreLePlan(
  ctx: CanvasRenderingContext2D,
  repere: RepereTactique,
  canvasWidth: number,
  peinture: PeintureDuPlan,
): void {
  const vue = vueDuPlan(repere, canvasWidth)
  const projection = projectionDuPlan(repere, canvasWidth)
  if (!vue || !projection) return
  const { zones, locale, encre } = peinture
  drawCalloutsShapes(ctx, zones, projection, { bigColors: [encre], fineInk: encre, locale })
  drawTacticalHeatmap(ctx, peinture.grid, vue, { ramp: peinture.ramp, k: 1 })
  drawCalloutsLabels(ctx, zones, projection, locale)
}
