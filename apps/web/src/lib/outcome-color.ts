/**
 * outcome-color.ts — Wrappers de compatibilité vers la couche accessibilité.
 *
 * Seuils canoniques : instances.ts (outcomeScale)
 */
import { outcomeScale } from './accessibility/scales'
import { tokenCssVar } from './accessibility'

export const OUTCOME_COLORS = {
  win:  tokenCssVar('outcome-win'),
  loss: tokenCssVar('outcome-loss'),
  draw: tokenCssVar('outcome-draw'),
  dnf:  tokenCssVar('outcome-dnf'),
} as const

const OUTCOME_KEY: Record<number, 'win' | 'loss' | 'draw' | 'dnf'> = {
  2: 'win',
  1: 'draw',
  3: 'loss',
}

/** Retourne la CSS var correspondant à un outcome numérique Halo. */
export function getOutcomeColor(outcome: number): string {
  const key = OUTCOME_KEY[outcome] ?? 'dnf'
  return tokenCssVar(outcomeScale(key) ?? 'outcome-dnf')
}
