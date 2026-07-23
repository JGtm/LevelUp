/**
 * engagementGap — « Écart d'engagement cumulé », exprimé en ÉVÉNEMENTS
 * (excès/déficit vs l'attendu).
 *
 * Contribution d'un point = résidu d'engagement (événements/minute) × durée
 * (minutes) = `residualPerMinute × (durationSeconds / 60)`. En sommant sur les
 * matchs, l'unité devient un nombre d'événements en excès (positif) ou en déficit
 * (négatif) par rapport au rythme attendu.
 *
 * Le résidu par surface :
 *   - Timeseries (solo)  : `pace_joueur − pace_attendu` par match ;
 *   - Escouade (par joueur) : `pace_observed − team_expected` par match ;
 *   - Session : `engagement_score` de `match_series` (déjà un résidu évén./min).
 *
 * Report D5 : une contribution `null` (résidu OU durée absent/non-fini) ne fait
 * pas avancer le cumul — délégué au helper générique `cumulativeSigned`.
 */
import { finiteOrNull } from './cumulativeSeries'

/**
 * Contribution en événements d'un point : résidu (évén./min) × durée (min).
 * `null` si le résidu ou la durée est absent/non-fini (report D5).
 */
export function engagementGapEvents(
  residualPerMinute: number | null | undefined,
  durationSeconds: number | null | undefined,
): number | null {
  const residual = finiteOrNull(residualPerMinute ?? null)
  const duration = finiteOrNull(durationSeconds ?? null)
  if (residual == null || duration == null) return null
  return residual * (duration / 60)
}
