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

  // ── Comparaisons (3) ───────────────────────────────────────────────────────
  | 'compare-a'
  | 'compare-b'
  | 'compare-c'

  // ── Joueurs d'escouade (4) — identité d'un joueur dans TOUTE l'app ─────────
  // 1 = joueur principal, 2..4 = coéquipiers dans l'ordre de sélection.
  // Famille dédiée (et non des tokens empruntés à d'autres rôles) : ces quatre
  // couleurs sont verrouillées par un garde-rail dédié (contraste ≥ 3:1 sur les
  // deux surfaces produit + écart perceptuel toutes-paires, y compris sous
  // simulation daltonisme) — cf. squadPlayerTokens.test.ts.
  // Source unique d'affectation : `features/squad/colors.ts`.
  | 'squad-player-1'
  | 'squad-player-2'
  | 'squad-player-3'
  | 'squad-player-4'

  // ── Séries de charts (8 max — pile sur Okabe-Ito) ─────────────────────────
  | 'chart-series-1'
  | 'chart-series-2'
  | 'chart-series-3'
  | 'chart-series-4'
  | 'chart-series-5'
  | 'chart-series-6'
  | 'chart-series-7'
  | 'chart-series-8'

  // ── Bonus (1) — segment "assistances" empilé dans les charts squad/timeseries ─
  // Couleur dédiée et distincte des 8 couleurs verrouillées de l'escouade
  // (4 joueurs + leurs opposés colorimétriques) — cf. squadPerformanceLineCharts.
  | 'bonus'

  // ── Rareté (1) — accent "légendaire", réutilisable hors Battlepass ─────────
  // PAS un doublon de `rarity.ts` (qui reste la SEULE source des 5 teintes de
  // rareté GameCMS, exception tolérée règle couleurs) : `legendary` est un accent
  // générique pour tout état "rare/précieux" qui n'est PAS une rareté de reward
  // (ex. encadré du surbouclier au rejeu 2D — un sur-bouclier est un état de jeu
  // rare et précieux, pas un item du Battlepass). Un seul token, pas cinq : les
  // autres paliers de rareté n'ont pas d'usage hors Battlepass à ce jour.
  | 'legendary'

  // ── Extrême rare (1) — le sommet d'une rampe d'intensité ──────────────────
  // Accent VIOLET pour ce qui sort de l'ordinaire par le HAUT, au-delà de la zone
  // « chaude » : le dernier palier d'une carte de chaleur, là où une poignée de
  // cellules concentre ce que le reste de la carte n'a pas. Même statut générique
  // que `legendary` (un rôle, pas un composant) : `legendary` dit « rare et
  // précieux », `extreme` dit « rare et intense » — un pic, pas un trésor.
  // PAS un token de la famille `heatmap-*` : ces six-là forment des rampes fermées
  // à deux bouts (cold/hot, freq-low/high, divergent-low/high) et sont remappées
  // ensemble ; celui-ci est un troisième point, réutilisable hors heatmap.
  // Composition de la rampe qui l'emploie : `heatmapRampTokens('intensity')`.
  | 'extreme'

  // ── Badges narratifs — fond (5) ────────────────────────────────────────────
  | 'narrative-dominant'
  | 'narrative-humiliation'
  | 'narrative-remontada'
  | 'narrative-debacle'
  | 'narrative-contre-remontada'

  // ── Badges encounter (4 + 5 solid hub Relations) ───────────────────────────
  | 'narrative-encounter-ally-plus'
  | 'narrative-encounter-tough-enemy'
  | 'narrative-encounter-coriace'
  | 'narrative-encounter-ordinal'
  | 'narrative-encounter-duo-gagnant'
  | 'narrative-encounter-cameleon'
  | 'narrative-encounter-de-longue-date'
  | 'narrative-encounter-recrue'
  | 'narrative-encounter-proie-favorite'
  | 'narrative-encounter-cross-game'

  // ── Badges narratifs — texte (5) ───────────────────────────────────────────
  // Texte sur fond coloré — calculé pour assurer le contraste WCAG AA
  | 'narrative-dominant-text'
  | 'narrative-humiliation-text'
  | 'narrative-remontada-text'
  | 'narrative-debacle-text'
  | 'narrative-contre-remontada-text'

  // ── Heatmaps (6) ──────────────────────────────────────────────────────────
  // cold/hot + divergent : rampes À CONNOTATION (win-rate, K/D → bien/mal).
  // freq-low/high : rampe NEUTRE mono-teinte pour les heatmaps de FRÉQUENCE
  // (intensité d'activité, sans jugement bien/mal). Cf. RelationsMomentsHeatmap.
  | 'heatmap-cold'
  | 'heatmap-hot'
  | 'heatmap-divergent-low'
  | 'heatmap-divergent-high'
  | 'heatmap-freq-low'
  | 'heatmap-freq-high'

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
  'compare-a', 'compare-b', 'compare-c',
  'squad-player-1', 'squad-player-2', 'squad-player-3', 'squad-player-4',
  'chart-series-1', 'chart-series-2', 'chart-series-3', 'chart-series-4',
  'chart-series-5', 'chart-series-6', 'chart-series-7', 'chart-series-8',
  'bonus',
  'legendary',
  'extreme',
  'narrative-dominant', 'narrative-humiliation', 'narrative-remontada',
  'narrative-debacle', 'narrative-contre-remontada',
  'narrative-dominant-text', 'narrative-humiliation-text', 'narrative-remontada-text',
  'narrative-debacle-text', 'narrative-contre-remontada-text',
  'narrative-encounter-ally-plus', 'narrative-encounter-tough-enemy',
  'narrative-encounter-coriace', 'narrative-encounter-ordinal',
  'narrative-encounter-duo-gagnant', 'narrative-encounter-cameleon',
  'narrative-encounter-de-longue-date', 'narrative-encounter-recrue',
  'narrative-encounter-proie-favorite', 'narrative-encounter-cross-game',
  'heatmap-cold', 'heatmap-hot', 'heatmap-divergent-low', 'heatmap-divergent-high',
  'heatmap-freq-low', 'heatmap-freq-high',
  'team-ally', 'team-enemy',
] as const
