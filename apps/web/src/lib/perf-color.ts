/**
 * perf-color.ts — Wrappers de compatibilité vers la couche accessibilité.
 *
 * Ces fonctions existaient avant la migration. Les call-sites qui les utilisent
 * encore reçoivent la CSS var `var(--ac-perf-tier-N)` — réactive automatiquement
 * aux changements de palette via cascade CSS.
 *
 * Seuils canoniques : instances.ts (perfScale [80, 65, 50, 35])
 */
import { perfScale } from './accessibility/scales'
import { tokenCssVar } from './accessibility'

export interface PerfColorLevel {
  color: string
  label: string
}

const LABELS: Record<string, string> = {
  'perf-tier-1': 'Excellent',
  'perf-tier-2': 'Bon',
  'perf-tier-3': 'Correct',
  'perf-tier-4': 'Faible',
  'perf-tier-5': 'Mauvais',
}

/** Retourne la CSS var correspondant à un score de performance (0–100). */
export function getPerfColor(score: number): string {
  return tokenCssVar(perfScale(score))
}

/** Retourne la CSS var et le label correspondant à un score de performance. */
export function getPerfColorLevel(score: number): PerfColorLevel {
  const token = perfScale(score)
  return { color: tokenCssVar(token), label: LABELS[token] ?? 'Mauvais' }
}
