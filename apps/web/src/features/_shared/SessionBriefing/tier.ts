/**
 * tier.ts — score (0-100) → palier qualitatif + token couleur.
 *
 * 5 paliers alignés sur le système Python (get_score_label / get_score_color)
 * et sur les semantic tokens 'perf-tier-{1..5}' du module accessibility.
 */
import type { SemanticToken } from '@/lib/accessibility'

export type TierKey = 'excellent' | 'good' | 'average' | 'poor' | 'bad'

export interface TierInfo {
  key: TierKey
  /** Token sémantique 'perf-tier-1' (meilleur) à 'perf-tier-5' (pire). */
  token: SemanticToken
}

/**
 * Retourne le palier d'un score 0-100. Aligné avec scoreLabel() Go service.
 */
export function getScoreTier(score: number): TierInfo {
  if (score >= 80) return { key: 'excellent', token: 'perf-tier-1' }
  if (score >= 65) return { key: 'good', token: 'perf-tier-2' }
  if (score >= 50) return { key: 'average', token: 'perf-tier-3' }
  if (score >= 35) return { key: 'poor', token: 'perf-tier-4' }
  return { key: 'bad', token: 'perf-tier-5' }
}
