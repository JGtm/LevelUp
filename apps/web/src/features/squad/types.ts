/**
 * Types partagés du feature Squad (V2).
 *
 * Miroir TypeScript des DTOs Go définis dans
 * `apps/go-api/internal/domain/squad_v2.go`. À régénérer via
 * `npm run generate-types` quand on aura l'OpenAPI.
 */

/** Comparaison du score d'un joueur vs la moyenne squad. */
export type PlayerScoreComparison = 'above' | 'below' | 'near'

/** Label qualitatif d'un score (couleur via tokens CSS). */
export type PlayerScoreLabel = 'excellent' | 'good' | 'average' | 'poor' | 'bad'
