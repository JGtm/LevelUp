/**
 * UsageEmptyNotice — L'ÉTAT VIDE D'UN BLOC D'USAGE, qui NOMME SA CAUSE (D8, 2026-09-21).
 *
 * UN BLOC D'UNE RANGÉE NE SE MASQUE PAS. Escamoté, il laisse la rangée bancale et se lit
 * comme un bug (constat du 2026-09-19, déjà appliqué à la carte de part du lobby). Il reste
 * donc affiché, avec son titre, et dit POURQUOI il est vide.
 *
 * IL SE DESSINE COMME TOUS LES AUTRES ÉTATS VIDES DE L'APP (2026-09-22) : `EmptyStateNotice`
 * (`components/ui/empty-state.tsx`, 48 fichiers), titre en gras puis description en gris
 * dans le cadre pointillé — le même gabarit que `TimeseriesRangeRolesCard`,
 * `SquadRangeRolesCard` ou `SquadUsagesPage` posent DÉJÀ à l'intérieur d'une carte. Ce
 * fichier rendait jusqu'ici un simple `<p>` gris : dans une rangée où la carte voisine
 * portait le cadre canonique, la même absence se lisait de deux façons (retour utilisateur
 * du 2026-09-22 sur « Contrôle des armes spéciales »). Aucun style local ici, donc : la
 * typographie de l'état vide se décide en UN endroit, `empty-state.tsx`.
 *
 * Les quatre causes, leurs titres et leurs phrases vivent dans `usageAvailability.ts`, avec
 * la porte qui en rend deux : ce fichier n'est que le rendu.
 */
import { EmptyStateNotice } from '@/components/ui/empty-state'

import { usageEmptyMessage, usageEmptyTitle, type UsageEmptyReason } from './usageAvailability'
import type { UsageText } from './usageI18n'

export function UsageEmptyNotice({ reason, t }: { reason: UsageEmptyReason; t: UsageText }) {
  // Le `div` ne porte AUCUNE classe : il n'existe que pour marquer la cause dans le DOM
  // (une capture d'écran doit pouvoir dire laquelle, cf. UsageA2Ajustements.test.tsx).
  return (
    <div data-usage-empty={reason}>
      <EmptyStateNotice
        title={usageEmptyTitle(reason, t)}
        description={usageEmptyMessage(reason, t)}
      />
    </div>
  )
}
