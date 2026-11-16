/**
 * mapBackgroundLayerLogic — LE FOND DE CARTE ASSEMBLÉ pour le canvas, hors de la page : ce que
 * posent le calage, l'image affichée et l'image plan qui rogne les zones nommées.
 *
 * LA RÈGLE DU ROGNAGE AU PLAN. Les zones nommées et la probabilité de réapparition se rognent à
 * l'image plan (`lib/replay/planMask`), QUEL QUE SOIT le fond affiché : sur le rendu comme sur le plan.
 * Le masque n'existe que si l'image PLAN est réellement chargée (`planVariante === 'plan'`) : un plan
 * en échec est remplacé par le rendu (repli silencieux de `useReplayMapImage`), et rogner à une image
 * rendu donnerait un masque faux. Sans image plan, sans calage ou en échec : zones entières.
 */
import type { ReplayMapBackground } from '@/lib/api/types'
import type { MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'
import { normalizeWallPlan } from '@/lib/replay/wallPlan'

import type { ReplayMapBackgroundLayer } from './useReplayView'

export interface MapBackgroundInputs {
  /** Le calage servi ; absent = la carte n'a pas d'image figée. */
  background: ReplayMapBackground | undefined
  /** L'image posée (celle de la variante réellement chargée), et sa variante. */
  image: HTMLImageElement | null
  variante: MapBackgroundVariant
  /** L'image plan demandée pour le masque, et la variante réellement chargée à sa place. */
  planImage: HTMLImageElement | null
  planVariante: MapBackgroundVariant
}

/** Le masque des zones nommées : l'image plan et son calage, `null` quand le plan n'est pas réellement chargé. */
export function zonesClipOf(
  background: ReplayMapBackground,
  planImage: HTMLImageElement | null,
  planVariante: MapBackgroundVariant,
): ReplayMapBackgroundLayer['zonesClip'] {
  return planImage && planVariante === 'plan' ? { calibration: background.calibration, image: planImage } : null
}

/** Le fond de carte du canvas ; `null` tant qu'il manque le calage ou l'image (le rejeu garde son sol reconstruit). */
export function buildMapBackgroundLayer(inputs: MapBackgroundInputs): ReplayMapBackgroundLayer | null {
  const { background, image, variante, planImage, planVariante } = inputs
  if (!background || !image) return null
  return {
    calibration: background.calibration,
    image,
    plan: normalizeWallPlan(background.plan),
    variante,
    planAvailable: !!background.planImage,
    zonesClip: zonesClipOf(background, planImage, planVariante),
  }
}
