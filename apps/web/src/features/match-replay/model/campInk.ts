/**
 * campInk — L'ENCRE D'UN CAMP dans la colonne des équipes : la couleur d'allié ou d'adversaire du
 * joueur regardé (`team-ally` / `team-enemy`, que les réglages d'accessibilité peuvent surcharger),
 * ou la bordure quand l'allégeance du camp n'est pas connue. Pastille de l'en-tête de camp et pastille
 * des fiches compactes en dépendent : une seule écriture.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { Allegiance } from '@/lib/replay/filmAllegiance'

export function campInk(ally: Allegiance): string {
  return ally === null ? 'var(--border)' : tokenCssVar(ally ? 'team-ally' : 'team-enemy')
}
