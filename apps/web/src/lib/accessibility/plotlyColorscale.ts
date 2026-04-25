/**
 * plotlyColorscale — helpers pour intégrer la couche accessibilité dans Plotly.
 *
 * Plotly attend des colorscales sous forme [[0,'#hex'],[0.5,'#hex'],[1,'#hex']]
 * avec des couleurs résolues au moment de la construction du layout.
 * Ces helpers lisent les CSS vars via resolveToken() (synchrone, DOM-based).
 *
 * Usage dans les charts :
 *   colorscale: buildDivergentColorscale('divergent-neg', 'divergent-neutral', 'divergent-pos')
 *   colorscale: buildOrdinalColorscale(['perf-tier-5','perf-tier-4','perf-tier-3','perf-tier-2','perf-tier-1'])
 *   color: getSeriesColors(n, ['perf-tier-1','perf-tier-2','perf-tier-3'])
 */
import type { SemanticToken } from './semantic-tokens'
import { resolveToken } from './resolveToken'

/** Colorscale divergente 3 stops : négatif → neutre → positif. */
export function buildDivergentColorscale(
  negToken: SemanticToken,
  neutralToken: SemanticToken,
  posToken: SemanticToken,
): [number, string][] {
  return [
    [0, resolveToken(negToken)],
    [0.5, resolveToken(neutralToken)],
    [1, resolveToken(posToken)],
  ]
}

/**
 * Colorscale ordinale N stops équidistants.
 * Les tokens sont dans l'ordre croissant de la valeur (index 0 = min, N-1 = max).
 */
export function buildOrdinalColorscale(tokens: SemanticToken[]): [number, string][] {
  if (tokens.length === 1) return [[0, resolveToken(tokens[0])], [1, resolveToken(tokens[0])]]
  return tokens.map((t, i) => [i / (tokens.length - 1), resolveToken(t)] as [number, string])
}

/** Retourne N couleurs en cyclant sur la liste de tokens. */
export function getSeriesColors(n: number, tokens: SemanticToken[]): string[] {
  return Array.from({ length: n }, (_, i) => resolveToken(tokens[i % tokens.length]))
}
