/**
 * avgLife — résolution de la durée de vie moyenne d'un match.
 *
 * Le backend expose désormais `avg_life_seconds` (valeur RÉELLE de l'API, qui
 * tient compte des respawns effectifs). L'ancien proxy `temps joué / (morts + 1)`
 * n'est plus qu'un REPLI pour les matchs antérieurs à la colonne — il surestime
 * systématiquement les parties où le joueur ne meurt pas, et sous-estime les
 * parties courtes.
 *
 * Même ordre de préférence que le backend (`service.matchAvgLifeSeconds`) afin
 * que l'histogramme de distribution et la courbe temporelle racontent la même
 * histoire (règle « ≤ 2 copies » : ces deux implémentations sont les seules).
 */

/** Sous-ensemble structurel d'une ligne de match nécessaire au calcul. */
export interface AvgLifeSource {
  avg_life_seconds?: number | null
  time_played_seconds?: number | null
  deaths: number
}

/**
 * matchAvgLifeSeconds — durée de vie moyenne en secondes, ou `null` si aucune
 * source exploitable (le point est alors omis de la série, pas remplacé par 0).
 */
export function matchAvgLifeSeconds(row: AvgLifeSource): number | null {
  const real = row.avg_life_seconds
  if (real != null && real > 0) return real
  const played = row.time_played_seconds
  if (played != null && played > 0) return played / (row.deaths + 1)
  return null
}
