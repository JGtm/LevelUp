/**
 * TacticalTab — l'onglet Tactique tel que la route le monte.
 *
 * LA PORTE DE TITRE EST ICI, PAS DANS LE COMPOSANT DE PAGE. `RouteCapabilityGate` rend
 * l'état « indisponible pour ce titre » À LA PLACE de la page : la page n'est donc pas
 * montée, ses hooks de lecture ne tournent pas, et aucune requête n'est émise pour un titre
 * qui n'a pas de rejeu. C'est la seconde des deux portes du lot C — la première étant
 * l'onglet lui-même, masqué par `FeatureGate` dans `AscensionLayout`.
 *
 * `replay` (title-level) est la bonne clé : sans décodage de film, ce titre ne produit
 * aucune position mesurée, donc aucune lecture de placement. La granularité fine
 * (`film.kill_positions` / `match.events.spatial`) est jugée CÔTÉ GO, par le service, qui
 * dégrade en 503 propre — la relire ici en ferait deux sources de vérité (doctrine
 * `dataCapabilities.ts`).
 *
 * Ce fichier existe séparément de la route pour la même raison que
 * `AscensionCoachingTab` : une route file-based reste un câblage de trois lignes, et le
 * composant qu'elle monte se teste sans routeur.
 */
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'

import { TacticalPage } from './TacticalPage'

export function TacticalTab() {
  return (
    <RouteCapabilityGate capability="replay">
      <TacticalPage />
    </RouteCapabilityGate>
  )
}
