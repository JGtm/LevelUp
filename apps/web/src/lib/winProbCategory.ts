/**
 * winProbCategory — catégorise un match selon sa probabilité de victoire
 * pré-match (LUSR v2, Sprint 1.A/1.D) croisée avec le résultat réel.
 *
 * Source de la proba : `expected_win_prob` ∈ [0,1], calculée AVANT le match par
 * le modèle TrueSkill 2 (cf. ADR 0024). Permet de distinguer une défaite
 * "perdable" (match donné perdant) d'un vrai sous-régime, et de mettre en valeur
 * les victoires arrachées contre les pronostics.
 *
 * Pur, sans dépendance — testable seul (winProbCategory.test.ts).
 */

/** Catégorie de pronostic d'un match. */
export type WinProbCategory = 'expected' | 'upset' | 'strong-perf' | 'balanced'

/**
 * Demi-largeur de la zone "équilibrée" autour de 0.5. Un match dont la proba de
 * victoire ∈ [0.4, 0.6] est un toss-up : ni favori ni outsider net.
 */
export const BALANCED_MARGIN = 0.1

/**
 * categorizeWinProb classe un match :
 *  - 'balanced'    : pronostic ~50/50 (|prob − 0.5| ≤ BALANCED_MARGIN), quel que soit le résultat.
 *  - 'upset'       : victoire en outsider net (prob < 0.5 − marge ET won) — exploit.
 *  - 'strong-perf' : défaite en outsider net (prob < 0.5 − marge ET !won) — défaite "perdable", défendable.
 *  - 'expected'    : sinon (favori net ; résultat conforme au pronostic ou choke d'un favori).
 *
 * Retourne null si la proba est absente (match pré-v2 / non-LUSR) → l'UI n'affiche rien.
 */
export function categorizeWinProb(prob: number | null | undefined, won: boolean): WinProbCategory | null {
  if (prob == null || Number.isNaN(prob)) return null
  if (Math.abs(prob - 0.5) <= BALANCED_MARGIN) return 'balanced'
  const isUnderdog = prob < 0.5 - BALANCED_MARGIN
  if (isUnderdog) return won ? 'upset' : 'strong-perf'
  // Favori net (prob > 0.5 + marge) : victoire = attendue ; défaite = choke, rangé
  // ici faute de 5e catégorie produit — l'UI le distingue via le % brut + le résultat.
  return 'expected'
}

/** Formate la proba en pourcentage entier ("73%"). "" si absente. */
export function formatWinProb(prob: number | null | undefined): string {
  if (prob == null || Number.isNaN(prob)) return ''
  return `${Math.round(prob * 100)}%`
}
