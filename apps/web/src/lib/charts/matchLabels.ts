/**
 * matchLabels — helpers partagés pour étiquetage X catégoriel "match" sur
 * les charts par-match (timeseries, squad, engagement…).
 *
 * Pure helpers — pas de dépendance à un type feature-specific.
 */

/**
 * Tronque un nom de carte à `max` caractères, avec coupure intelligente sur
 * le 1er ' ' ou '-' rencontré pour éviter les troncatures au milieu d'un mot
 * quand c'est possible.
 */
export function truncateMap(s: string, max = 9): string {
  if (s.length <= max) return s
  const sepIdx = Math.min(
    ...[' ', '-'].map((c) => {
      const i = s.indexOf(c)
      return i > 0 ? i : Infinity
    }),
  )
  if (sepIdx <= max) return `${s.slice(0, sepIdx)}…`
  return `${s.slice(0, max - 1)}…`
}
