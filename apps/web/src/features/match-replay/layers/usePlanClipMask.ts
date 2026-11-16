/**
 * usePlanClipMask — LE MASQUE QUI ROGNE UNE TEINTE À L'IMAGE PLAN de la carte (`clipToMapPlan`), stable tant que
 * l'image plan ne change pas. Sans vue plan (`plan` nul), pas de masque : les zones restent entières.
 */
import { useMemo } from 'react'

import type { CanvasView } from '../model/replayView'
import { clipToMapPlan, type MapBackgroundSource } from './mapBackgroundDraw'

export function usePlanClipMask(
  plan: MapBackgroundSource | null,
): ((ctx: CanvasRenderingContext2D, view: CanvasView) => void) | undefined {
  return useMemo(() => (plan ? (ctx, view) => clipToMapPlan(ctx, plan, view) : undefined), [plan])
}
