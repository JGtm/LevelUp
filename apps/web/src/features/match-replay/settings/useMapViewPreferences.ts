/**
 * useMapViewPreferences — LES RÉGLAGES DE LA CARTE QUE TOUTES LES VUES 2D PARTAGENT : l'orientation
 * retenue pour la carte, l'interrupteur du plan des murs et le style du fond (rendu ou plan).
 *
 * Une carte tournée dans le rejeu est tournée partout (onglet Tactique, « Occupation du terrain »),
 * et le plan des murs s'allume et s'éteint d'un seul geste : ces pages n'ont AUCUN réglage propre, elles
 * LISENT les trois préférences du rejeu (`replay-orientation.<mapId>`, `replay-show-wall-plan` et `replay-background-style`) à
 * leur ouverture. Ce module est le seul point d'entrée des autres features : elles n'importent ni les
 * clés ni le stockage.
 *
 * LECTURE SEULE : la commande vit dans le panneau « Vue » du rejeu.
 */
import type { MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'
import type { QuarterTurns } from '@/lib/replay/orientation'

import { useBackgroundStyle } from './useBackgroundStyle'
import { useReplayOrientation } from './useReplayOrientation'
import { usePersistedFlag } from './usePersistedFlag'
import { SHOW_WALL_PLAN_DEFAULT, SHOW_WALL_PLAN_KEY } from './useReplayTerrainFlags'

export interface MapViewPreferences {
  /** L'orientation retenue pour cette carte, en quarts de tour horaires (0 sans carte connue). */
  turns: QuarterTurns
  /** Le plan des murs doit-il être tracé ? */
  showWallPlan: boolean
  /**
   * Le style de fond voulu : `rendu` ou `plan`. C'est un VOULOIR : il ne vaut que pour un fond qui a
   * une image plan (`resolveBackgroundVariant`) ; le plan des murs, lui, est déjà dans cette image.
   */
  backgroundStyle: MapBackgroundVariant
}

/** useMapViewPreferences lit l'orientation de `mapId`, l'interrupteur du plan des murs et le style du fond. */
export function useMapViewPreferences(mapId: string | null | undefined): MapViewPreferences {
  const { turns } = useReplayOrientation(mapId)
  const [showWallPlan] = usePersistedFlag(SHOW_WALL_PLAN_KEY, SHOW_WALL_PLAN_DEFAULT)
  const [backgroundStyle] = useBackgroundStyle()
  return { turns, showWallPlan, backgroundStyle }
}
