/**
 * campInk — L'ENCRE D'UN CAMP dans la colonne des équipes : la couleur d'allié ou d'adversaire du
 * joueur regardé (`team-ally` / `team-enemy`, que les réglages d'accessibilité peuvent surcharger),
 * ou la bordure quand l'allégeance du camp n'est pas connue. Pastille de l'en-tête de camp et pastille
 * des fiches compactes en dépendent : une seule écriture. `dominanceInk` y ajoute l'égalité, pour les bandes de
 * dominance des deux frises.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { Allegiance } from '@/lib/replay/filmAllegiance'

export function campInk(ally: Allegiance): string {
  return ally === null ? 'var(--border)' : tokenCssVar(ally ? 'team-ally' : 'team-enemy')
}

/**
 * L'encre d'une bande de dominance. TROIS CAS, ET TROIS ENCRES DISTINCTES :
 *  - ÉGALITÉ (`teamId === null`) : l'encre d'égalité DU DÉPÔT (`outcome-draw`, le bleu des
 *    matchs nuls, des tuiles neutres et des barres de bilan) — demande utilisateur du
 *    2026-08-28. Une quatrième couleur inventée ici ferait diverger le vocabulaire.
 *  - CAMP CONNU : allié ou adverse, les deux encres du rejeu.
 *  - CAMP NON RÉSOLU (`allyOf` rend `null`, joueur hors scoreboard) : la bordure neutre —
 *    jamais l'une des deux couleurs par défaut, qui désignerait un camp au hasard.
 */
export function dominanceInk(teamId: number | null, allyOf: (teamId: number) => boolean | null): string {
  if (teamId == null) return tokenCssVar('outcome-draw')
  return campInk(allyOf(teamId))
}
