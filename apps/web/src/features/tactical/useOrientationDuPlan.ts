/**
 * useOrientationDuPlan — L'ORIENTATION D'UN PLAN DE L'ONGLET TACTIQUE, sur SA zone d'affichage.
 *
 * Les deux plans (placement et réapparitions) lisent les préférences de carte du rejeu
 * (`useMapViewPreferences`) : le choix retenu pour la carte prime toujours ; sans choix (Auto), la règle du rejeu
 * s'applique à la carte et à la zone que le plan occupe. Cette zone est la largeur utile du conteneur du cadre
 * (`zoneRef` : sa largeur moins la marge de la légende) par la hauteur que la fenêtre lui laisse
 * (`useHauteurDuPlan`, la règle de la toile du rejeu) : l'une et l'autre ne dépendent pas de l'orientation, donc
 * la règle ne court pas après sa propre sortie.
 *
 * Rend aussi `hauteurMax` : la même mesure sert à poser le cadre (`boiteDuPlan`).
 */
import type { RefObject } from 'react'

import { useMapViewPreferences, type MapViewPreferences } from '@/features/match-replay/settings/useMapViewPreferences'
import type { Dimensions } from '@/lib/replay/orientation'
import { useContentWidth } from '@/lib/replay/useContentWidth'

import { useHauteurDuPlan } from './useHauteurDuPlan'

export interface OrientationDuPlan extends Pick<MapViewPreferences, 'turns' | 'showWallPlan'> {
  /** La hauteur que la fenêtre laisse au cadre du plan, en px. */
  hauteurMax: number
  /** À poser sur le conteneur du cadre : c'est sa largeur utile qu'on mesure. */
  zoneRef: (el: HTMLElement | null) => void
}

/**
 * @param mapId  la carte (clé du choix retenu) ; vide = aucune carte connue.
 * @param carte  la carte posée droite, en unités monde (`dimensionsDuPlan`) ; `null` = inconnue.
 * @param carteRef  l'élément qui porte le plan ET son chrome (`useHauteurDuPlan`).
 * @param cadreRef  le cadre du fond.
 */
export function useOrientationDuPlan(
  mapId: string,
  carte: Dimensions | null,
  carteRef: RefObject<HTMLElement | null>,
  cadreRef: RefObject<HTMLElement | null>,
): OrientationDuPlan {
  const hauteurMax = useHauteurDuPlan(carteRef, cadreRef)
  const { width, ref } = useContentWidth()
  const { turns, showWallPlan } = useMapViewPreferences(mapId, { carte, boite: { largeur: width, hauteur: hauteurMax } })
  return { turns, showWallPlan, hauteurMax, zoneRef: ref }
}
