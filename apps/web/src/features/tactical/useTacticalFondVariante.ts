/**
 * useTacticalFondVariante — LE FOND que le plan de l'onglet Tactique pose : son URL et la variante
 * RÉELLEMENT chargée. La variante demandée est `plan` quand le réglage « Fond » du rejeu 2D le veut ET
 * que la carte en porte une, `rendu` sinon ; si l'image plan ne se charge pas, le rendu est posé en repli
 * et la variante dite est `rendu` (les murs réapparaissent). Le fond (`TacticalPlanFond`) et la carte
 * (`TacticalPlanCard`, qui retire le plan des murs déjà dessiné dans l'image plan) lisent la MÊME
 * décision ici — les deux appels partagent les mêmes requêtes en cache.
 *
 * Avec le réglage plan, rien n'est demandé avant que le sidecar du fond ait dit si la carte a une image
 * plan (sinon le rendu serait téléchargé pour rien).
 */
import { useMapViewPreferences } from '@/features/match-replay/settings/useMapViewPreferences'
import { resolveBackgroundVariant, type MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'

import { useTacticalMapBackgroundImage, useTacticalMapPlanImage } from './queries'

export function useTacticalFondVariante(
  playerSlug: string,
  mapId: string,
): { url: string | null; variante: MapBackgroundVariant } {
  const { backgroundStyle } = useMapViewPreferences(mapId)
  const { planImage, pending } = useTacticalMapPlanImage(playerSlug, mapId)
  const demandee = resolveBackgroundVariant(backgroundStyle, planImage)
  const pret = backgroundStyle === 'rendu' || !pending
  return useTacticalMapBackgroundImage(playerSlug, mapId, pret, demandee)
}
