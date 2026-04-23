/**
 * perf-color.ts — Palette de couleurs du score de performance.
 *
 * Échelle à 5 paliers cohérente avec le cockpit legacy v7 :
 *   ≥ 80 → vert    (#10B981)
 *   ≥ 65 → cyan    (#06B6D4)
 *   ≥ 50 → amber   (#F59E0B)
 *   ≥ 35 → orange  (#F97316)
 *   <  35 → rouge   (#EF4444)
 */

export interface PerfColorLevel {
  color: string
  label: string
}

const PERF_LEVELS: readonly { threshold: number; color: string; label: string }[] = [
  { threshold: 80, color: '#10B981', label: 'Excellent' },
  { threshold: 65, color: '#06B6D4', label: 'Bon' },
  { threshold: 50, color: '#F59E0B', label: 'Correct' },
  { threshold: 35, color: '#F97316', label: 'Faible' },
  { threshold: 0,  color: '#EF4444', label: 'Mauvais' },
]

/**
 * Retourne la couleur hex correspondant à un score de performance (0–100).
 */
export function getPerfColor(score: number): string {
  for (const level of PERF_LEVELS) {
    if (score >= level.threshold) return level.color
  }
  return '#EF4444'
}

/**
 * Retourne la couleur et le label correspondant à un score de performance.
 */
export function getPerfColorLevel(score: number): PerfColorLevel {
  for (const level of PERF_LEVELS) {
    if (score >= level.threshold) return { color: level.color, label: level.label }
  }
  return { color: '#EF4444', label: 'Mauvais' }
}
