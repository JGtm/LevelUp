/**
 * useMapViewPreferences — LES RÉGLAGES DE LA CARTE QUE TOUTES LES VUES 2D PARTAGENT : l'orientation
 * de la carte, l'interrupteur du plan des murs et le style du fond (plan, rendu ou relief).
 *
 * Une carte tournée dans le rejeu est tournée partout (onglet Tactique, « Occupation du terrain »),
 * et le plan des murs s'allume et s'éteint d'un seul geste : ces pages n'ont AUCUN réglage propre, elles
 * LISENT les trois préférences du rejeu (`replay-orientation.<mapId>`, `replay-show-wall-plan` et `replay-background-style`) à
 * leur ouverture. Ce module est le seul point d'entrée des autres features : elles n'importent ni les
 * clés ni le stockage.
 *
 * Orientation et plan des murs : LECTURE SEULE, leur commande vit dans le panneau « Vue » du rejeu. Le style du fond se
 * lit ET s'écrit (pilule « Fond » de Tactique) : la clé est la même, les deux côtés bougent ensemble.
 *
 * L'ORIENTATION SUIT LA RÈGLE DU REJEU : un choix retenu pour la carte prime toujours ; sans choix (Auto), une
 * vue qui donne SA zone d'affichage (`zone`) applique la règle Auto du rejeu (`useAutoTurned`) à la carte et à
 * cette zone, et une vue qui n'en donne pas garde le plan droit (elle ne lit que le style du fond ou l'interrupteur).
 */
import type { MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'
import type { Dimensions, QuarterTurns } from '@/lib/replay/orientation'
import { useAutoTurned } from '@/lib/replay/useAutoTurned'

import { useBackgroundStyle } from './useBackgroundStyle'
import { useReplayOrientation } from './useReplayOrientation'
import { usePersistedFlag } from './usePersistedFlag'
import { SHOW_WALL_PLAN_DEFAULT, SHOW_WALL_PLAN_KEY } from './useReplayTerrainFlags'

/**
 * LA ZONE D'AFFICHAGE d'une vue : la carte posée droite (en unités monde ; `null` = inconnue) et la boîte que
 * la vue lui offre (px CSS ; une dimension nulle = pas encore mesurée). C'est sur elle, et non sur la toile du
 * rejeu, que la règle Auto juge si la carte gagne à être tournée.
 */
export interface MapViewZone {
  carte: Dimensions | null
  boite: Dimensions
}

export interface MapViewPreferences {
  /**
   * L'orientation de cette carte, en quarts de tour horaires : le choix retenu s'il y en a un ; sinon, l'Auto
   * sur la zone de la vue (`zone`), ou 0 sans zone et sans carte connue.
   */
  turns: QuarterTurns
  /** Le plan des murs doit-il être tracé ? */
  showWallPlan: boolean
  /**
   * Le style de fond voulu : `plan`, `rendu` ou `relief`. C'est un VOULOIR : il ne vaut que pour un fond qui a
   * l'image demandée (`resolveBackgroundVariant`) ; le plan des murs, lui, est déjà dans l'image plan. Les trois vues 2D
   * (rejeu, Tactique, « Occupation du terrain ») le lisent avec ce que leur fond offre : sans relief lisible, un `relief`
   * voulu se lit `plan`, puis `rendu`.
   */
  backgroundStyle: MapBackgroundVariant
  /** Change et retient le style du fond : la MÊME clé que le panneau « Vue » du rejeu (une seule source, lue et écrite des deux côtés). */
  setBackgroundStyle: (style: MapBackgroundVariant) => void
}

/**
 * useMapViewPreferences lit l'orientation de `mapId`, l'interrupteur du plan des murs et le style du fond.
 * `zone` : la zone d'affichage de la vue, pour que l'Auto s'y applique ; l'état de l'Auto est rangé par carte.
 */
export function useMapViewPreferences(mapId: string | null | undefined, zone?: MapViewZone): MapViewPreferences {
  const { retained } = useReplayOrientation(mapId)
  const autoTurned = useAutoTurned({
    scopeKey: mapId ?? '',
    choice: retained,
    carte: zone?.carte ?? null,
    boite: zone?.boite ?? { largeur: 0, hauteur: 0 },
  })
  const [showWallPlan] = usePersistedFlag(SHOW_WALL_PLAN_KEY, SHOW_WALL_PLAN_DEFAULT)
  const [backgroundStyle, setBackgroundStyle] = useBackgroundStyle()
  return { turns: retained ?? (autoTurned ? 1 : 0), showWallPlan, backgroundStyle, setBackgroundStyle }
}
