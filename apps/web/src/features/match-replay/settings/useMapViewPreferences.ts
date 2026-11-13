/**
 * useMapViewPreferences — LES RÉGLAGES DE LA CARTE QUE TOUTES LES VUES 2D PARTAGENT : l'orientation
 * retenue pour la carte et l'interrupteur du plan des murs.
 *
 * Une carte tournée dans le rejeu est tournée partout (onglet Tactique, « Occupation du terrain »),
 * et le plan des murs s'allume et s'éteint d'un seul geste : ces pages n'ont AUCUN réglage propre, elles
 * LISENT les deux préférences du rejeu (`replay-orientation.<mapId>` et `replay-show-wall-plan`) à
 * leur ouverture. Ce module est le seul point d'entrée des autres features : elles n'importent ni les
 * clés ni le stockage.
 *
 * LECTURE SEULE : la commande vit dans le panneau « Vue » du rejeu.
 */
import type { QuarterTurns } from '@/lib/replay/orientation'

import { useReplayOrientation } from './useReplayOrientation'
import { usePersistedFlag } from './usePersistedFlag'
import { SHOW_WALL_PLAN_DEFAULT, SHOW_WALL_PLAN_KEY } from './useReplayTerrainFlags'

export interface MapViewPreferences {
  /** L'orientation retenue pour cette carte, en quarts de tour horaires (0 sans carte connue). */
  turns: QuarterTurns
  /** Le plan des murs doit-il être tracé ? */
  showWallPlan: boolean
}

/** useMapViewPreferences lit l'orientation de `mapId` et l'interrupteur du plan des murs. */
export function useMapViewPreferences(mapId: string | null | undefined): MapViewPreferences {
  const { turns } = useReplayOrientation(mapId)
  const [showWallPlan] = usePersistedFlag(SHOW_WALL_PLAN_KEY, SHOW_WALL_PLAN_DEFAULT)
  return { turns, showWallPlan }
}
