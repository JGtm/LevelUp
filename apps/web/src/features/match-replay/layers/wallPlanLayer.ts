/**
 * wallPlanLayer — LE PLAN DES MURS D'UNE CARTE, EN TRAITS FINS PAR-DESSUS SON FOND.
 *
 * Le sidecar du fond de carte porte les arêtes des murs en mètres MONDE, dans le repère du
 * rejeu (`lib/replay/wallPlan.ts`). Ce calque les trace comme `drawCalloutsLayer` trace les zones
 * nommées : en vecteurs, projetés par le cadrage partagé (`projectTo`), donc exacts à tout zoom
 * et sous toute orientation de la carte. Le trait lui-même (épaisseur, opacité, un seul chemin) est
 * celui de `lib/replay/wallPlanPaint`, que l'onglet Tactique et « Occupation du terrain » emploient
 * aussi : ce fichier ne fait que brancher la projection du rejeu.
 *
 * CALQUE STATIQUE : il ne dépend ni de l'image courante ni de la lecture, seulement du cadrage
 * et de son encre. L'appelant le cuit hors écran (`useReplayStaticLayers`) et le recopie, comme
 * les zones nommées.
 *
 * Aucune couleur ici : l'encre vient de l'appelant (token du thème, `canvasInk`).
 */
import { strokeWallPlan } from '@/lib/replay/wallPlanPaint'
import type { WallPlanReady } from '@/lib/replay/wallPlan'

import { projectTo, type CanvasView } from '../model/replayView'

/** L'encre du plan : l'encre de mise en page la plus contrastée du thème, résolue par l'appelant. */
export interface WallPlanStyle {
  ink: string
}

/**
 * drawWallPlanLayer trace les arêtes du plan. Sans plan (ou sans arête), il ne touche pas au
 * contexte : pas de calque plutôt qu'un calque vide.
 */
export function drawWallPlanLayer(
  ctx: CanvasRenderingContext2D,
  plan: WallPlanReady | null,
  view: CanvasView,
  style: WallPlanStyle,
): void {
  strokeWallPlan(ctx, plan, (p) => projectTo(view, p), style.ink)
}
