/**
 * useColor.ts — Hooks React pour consommer la couche accessibilité.
 *
 * Retourne des CSS var strings (`var(--ac-token)`) — pas des hex résolus —
 * afin que les composants réagissent automatiquement aux changements de palette
 * via la cascade CSS, sans re-render React.
 */
import { tokenCssVar, type SemanticToken } from './semantic-tokens'

/**
 * Retourne la CSS var string pour un token.
 * Usage : `style={{ color: useColor('outcome-win') }}`
 */
export function useColor(token: SemanticToken): string {
  return tokenCssVar(token)
}

/**
 * Applique une scale à une valeur et retourne la CSS var correspondante.
 * Usage : `style={{ color: useScaleColor(perfScale, score) }}`
 */
export function useScaleColor<V>(
  scale: (value: V) => SemanticToken,
  value: V,
): string {
  return tokenCssVar(scale(value))
}
