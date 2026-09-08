/**
 * withLowSampleNote — accole la réserve « échantillon faible » à un texte déjà formaté.
 *
 * Doctrine du dépôt : le drapeau `echantillon_faible` (couverture d'une mesure) INTERDIT
 * DE COMPARER la valeur, il ne la cache pas. La réserve s'affiche donc AVEC la valeur,
 * jamais à sa place. Source unique de cette forme depuis la revue de la vague 1
 * (2026-09-07, constat P1 : trois copies du motif dans `SquadEchangeKpi`,
 * `TacticalAnalysisView` et `SquadIsolementNuageCard`, avec deux séparateurs différents).
 * Garde-rail : `lowSampleNote.guard.test.ts`.
 *
 * @param base       le texte déjà formaté (sous-titre de tuile, ligne de tooltip…)
 * @param lowSample  le drapeau `echantillon_faible` de la couverture
 * @param note       la chaîne localisée « échantillon faible » (FR/EN, depuis le i18n appelant)
 * @param separator  ' — ' par défaut (texte) ; '<br/>' pour un tooltip HTML ECharts
 */
export function withLowSampleNote(
  base: string,
  lowSample: boolean,
  note: string,
  separator = ' — ',
): string {
  return lowSample ? `${base}${separator}${note}` : base
}
