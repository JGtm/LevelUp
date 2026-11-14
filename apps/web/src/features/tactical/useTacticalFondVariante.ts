/**
 * useTacticalFondVariante — L'IMAGE DU FOND que le plan de l'onglet Tactique doit poser : `plan` quand
 * le réglage « Fond » du rejeu 2D le demande ET que la carte en porte une, `rendu` sinon (repli
 * silencieux). Le fond (`TacticalPlanFond`) et la carte (`TacticalPlanCard`, qui retire le plan des
 * murs déjà dessiné dans l'image plan) lisent la MÊME décision ici.
 *
 * `pret` = l'image peut se demander : avec le réglage plan, pas avant que le sidecar du fond ait dit
 * si la carte a une image plan (sinon le rendu serait téléchargé pour rien).
 */
import { useMapViewPreferences } from '@/features/match-replay/settings/useMapViewPreferences'
import { resolveBackgroundVariant, type MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'

import { useTacticalMapPlanImage } from './queries'

export function useTacticalFondVariante(
  playerSlug: string,
  mapId: string,
): { variante: MapBackgroundVariant; pret: boolean } {
  const { backgroundStyle } = useMapViewPreferences(mapId)
  const { planImage, pending } = useTacticalMapPlanImage(playerSlug, mapId)
  return {
    variante: resolveBackgroundVariant(backgroundStyle, planImage),
    pret: backgroundStyle === 'rendu' || !pending,
  }
}
