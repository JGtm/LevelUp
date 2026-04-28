/**
 * plotlyColorscale — helper série-colors pour ECharts/canvas.
 *
 * Usage dans les charts :
 *   color: getSeriesColors(n, ['perf-tier-1','perf-tier-2','perf-tier-3'])
 */
import type { SemanticToken } from './semantic-tokens'
import { resolveToken } from './resolveToken'

/** Retourne N couleurs en cyclant sur la liste de tokens. */
export function getSeriesColors(n: number, tokens: SemanticToken[]): string[] {
  return Array.from({ length: n }, (_, i) => resolveToken(tokens[i % tokens.length]))
}
