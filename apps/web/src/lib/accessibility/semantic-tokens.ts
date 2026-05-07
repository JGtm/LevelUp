/**
 * semantic-tokens.ts — Contrat central de la couche accessibilité.
 *
 * Chaque SemanticToken est un rôle fonctionnel (ex: 'outcome-win').
 * Les palettes font correspondre chaque token à un hex. Les composants
 * et les scales n'utilisent jamais de hex directement — uniquement des tokens.
 *
 * CSS var associée : --ac-<token>  (préfixe "ac" pour "accessibility")
 */

export type SemanticToken =
  // ── Outcomes (4) ───────────────────────────────────────────────────────────
  | 'outcome-win'
  | 'outcome-loss'
  | 'outcome-draw'
  | 'outcome-dnf'

  // ── Perf / qualité — 5 tiers ordinaux (1=meilleur, 5=pire) ────────────────
  // Réutilisés par perfScale, accuracyScale, kdScale, progressScale
  | 'perf-tier-1'
  | 'perf-tier-2'
  | 'perf-tier-3'
  | 'perf-tier-4'
  | 'perf-tier-5'

  // ── Divergent — indicateurs signés (pos/neutre/neg) ────────────────────────
  | 'divergent-pos'
  | 'divergent-neutral'
  | 'divergent-neg'

  // ── Statuts UI (4) ─────────────────────────────────────────────────────────
  | 'success'
  | 'warning'
  | 'info'
  | 'destructive'

  // ── Comparaisons (2) ───────────────────────────────────────────────────────
  | 'compare-a'
  | 'compare-b'

  // ── Séries de charts (8 max — pile sur Okabe-Ito) ─────────────────────────
  | 'chart-series-1'
  | 'chart-series-2'
  | 'chart-series-3'
  | 'chart-series-4'
  | 'chart-series-5'
  | 'chart-series-6'
  | 'chart-series-7'
  | 'chart-series-8'

  // ── Badges narratifs — fond (5) ────────────────────────────────────────────
  | 'narrative-dominant'
  | 'narrative-humiliation'
  | 'narrative-remontada'
  | 'narrative-debacle'
  | 'narrative-contre-remontada'

  // ── Badges encounter (3) ───────────────────────────────────────────────────
  | 'narrative-encounter-ally-plus'
  | 'narrative-encounter-tough-enemy'
  | 'narrative-encounter-ordinal'

  // ── Badges narratifs — texte (5) ───────────────────────────────────────────
  // Texte sur fond coloré — calculé pour assurer le contraste WCAG AA
  | 'narrative-dominant-text'
  | 'narrative-humiliation-text'
  | 'narrative-remontada-text'
  | 'narrative-debacle-text'
  | 'narrative-contre-remontada-text'

  // ── Heatmaps (4) ──────────────────────────────────────────────────────────
  | 'heatmap-cold'
  | 'heatmap-hot'
  | 'heatmap-divergent-low'
  | 'heatmap-divergent-high'

  // ── Équipes (2) — couleurs configurables via les settings d'accessibilité ─
  // Correspondent aux couleurs d'outline choisies par l'utilisateur in-game.
  | 'team-ally'
  | 'team-enemy'

/** Nom CSS var pour un token : `--ac-outcome-win`, `--ac-perf-tier-1`, etc. */
export function tokenVar(token: SemanticToken): string {
  return `--ac-${token}`
}

/** Valeur CSS var à utiliser dans `style={{ color: ... }}` */
export function tokenCssVar(token: SemanticToken): string {
  return `var(--ac-${token})`
}

/** Une palette = hex pour chaque token. */
export type Palette = Record<SemanticToken, string>

/** Liste exhaustive de tous les tokens — utilisée pour les tests de couverture. */
export const ALL_TOKENS: readonly SemanticToken[] = [
  'outcome-win', 'outcome-loss', 'outcome-draw', 'outcome-dnf',
  'perf-tier-1', 'perf-tier-2', 'perf-tier-3', 'perf-tier-4', 'perf-tier-5',
  'divergent-pos', 'divergent-neutral', 'divergent-neg',
  'success', 'warning', 'info', 'destructive',
  'compare-a', 'compare-b',
  'chart-series-1', 'chart-series-2', 'chart-series-3', 'chart-series-4',
  'chart-series-5', 'chart-series-6', 'chart-series-7', 'chart-series-8',
  'narrative-dominant', 'narrative-humiliation', 'narrative-remontada',
  'narrative-debacle', 'narrative-contre-remontada',
  'narrative-dominant-text', 'narrative-humiliation-text', 'narrative-remontada-text',
  'narrative-debacle-text', 'narrative-contre-remontada-text',
  'narrative-encounter-ally-plus', 'narrative-encounter-tough-enemy', 'narrative-encounter-ordinal',
  'heatmap-cold', 'heatmap-hot', 'heatmap-divergent-low', 'heatmap-divergent-high',
  'team-ally', 'team-enemy',
] as const
