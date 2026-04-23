/**
 * outcome-color.ts — Couleurs par type d'outcome (Halo Infinite).
 *
 * Outcomes numériques : 0=Unknown/DNF, 1=Draw, 2=Win, 3=Loss
 */

export const OUTCOME_COLORS = {
  win:  '#10B981', // emerald-500
  loss: '#EF4444', // red-500
  draw: '#3B82F6', // blue-500
  dnf:  '#8B5CF6', // violet-500
} as const

/** Retourne la couleur hex correspondant à un outcome numérique Halo. */
export function getOutcomeColor(outcome: number): string {
  switch (outcome) {
    case 2: return OUTCOME_COLORS.win
    case 3: return OUTCOME_COLORS.loss
    case 1: return OUTCOME_COLORS.draw
    default: return OUTCOME_COLORS.dnf
  }
}
