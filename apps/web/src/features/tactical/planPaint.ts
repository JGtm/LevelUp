/**
 * planPaint.ts — l'ORDRE des calques du plan de l'onglet Tactique, posés sur le fond de carte :
 *   0. le PLAN DES MURS de la carte (`lib/replay/wallPlanPaint`, le même trait que le rejeu 2D), tout
 *      en dessous : un trait de dessin sur le fond, sous les zones, la chaleur et les couches de morts ;
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
import type { WallPlanReady } from '@/lib/replay/wallPlan'
import { strokeWallPlan } from '@/lib/replay/wallPlanPaint'
import { flouDeLaChaleur, peindreLaChaleurLissee } from '@/lib/replay/chaleurLissee'
import type { TacticalGrid } from '@/lib/replay/heatPaint'

import { projectionDuPlan, vueDuPlan, type FenetreDuPlan, type RepereTactique } from './tacticalView.logic'

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
  /** Le plan des murs à tracer sous tout le reste ; `null`/absent = pas de plan, ou interrupteur éteint. */
  plan?: WallPlanReady | null
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
  const { zones, locale, encre, plan } = peinture
  strokeWallPlan(ctx, plan ?? null, projection, encre)
  drawCalloutsShapes(ctx, zones, projection, { bigColors: [encre], fineInk: encre, locale })
  peindreLaChaleurLissee(ctx, peinture.grid, vue, peinture.ramp, flouDeLaChaleur(repere.pasM, vue.scale))
  drawCalloutsLabels(ctx, zones, projection, locale)
}
