/**
 * UsageEmptyNotice — L'ÉTAT VIDE D'UN BLOC D'USAGE, qui NOMME SA CAUSE (D8, 2026-09-21).
 *
 * UN BLOC D'UNE RANGÉE NE SE MASQUE PAS. Escamoté, il laisse la rangée bancale et se lit
 * comme un bug (constat du 2026-09-19, déjà appliqué à la carte de part du lobby). Il reste
 * donc affiché, avec son titre, et dit POURQUOI il est vide.
 *
 * Les quatre causes et leurs phrases vivent dans `usageAvailability.ts`, avec la porte qui
 * en rend deux : ce fichier n'est que le rendu.
 */
import { usageEmptyMessage, type UsageEmptyReason } from './usageAvailability'
import type { UsageText } from './usageI18n'

export function UsageEmptyNotice({ reason, t }: { reason: UsageEmptyReason; t: UsageText }) {
  return (
    <p className="text-sm text-muted-foreground" data-usage-empty={reason}>
      {usageEmptyMessage(reason, t)}
    </p>
  )
}
