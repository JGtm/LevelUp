/**
 * tier.ts — score (0-100) → palier qualitatif + token couleur.
 *
 * 5 paliers alignés sur le système Python (get_score_label / get_score_color)
 * et sur les semantic tokens 'perf-tier-{1..5}' du module accessibility.
 */
import type { SemanticToken } from '@/lib/accessibility'
import { perfScale } from '@/lib/accessibility/scales'

export type TierKey = 'excellent' | 'good' | 'average' | 'poor' | 'bad'

export interface TierInfo {
  key: TierKey
  /** Token sémantique 'perf-tier-1' (meilleur) à 'perf-tier-5' (pire). */
  token: SemanticToken
}

// Clé qualitative par tier (index 1..5) — le mapping seuils→token vient de
// perfScale (source unique, 80/65/50/35). V7c : ce fichier NE recopie plus
// l'échelle, il n'ajoute que le libellé métier au-dessus du token.
const TIER_KEY_BY_TOKEN: Record<SemanticToken, TierKey> = {
  'perf-tier-1': 'excellent',
  'perf-tier-2': 'good',
  'perf-tier-3': 'average',
  'perf-tier-4': 'poor',
  'perf-tier-5': 'bad',
} as Record<SemanticToken, TierKey>

/**
 * Retourne le palier d'un score 0-100. Aligné avec scoreLabel() Go service via
 * perfScale (seuils uniques 80/65/50/35).
 */
export function getScoreTier(score: number): TierInfo {
  const token = perfScale(score)
  return { key: TIER_KEY_BY_TOKEN[token], token }
}
