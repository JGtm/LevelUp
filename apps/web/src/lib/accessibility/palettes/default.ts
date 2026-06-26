/**
 * palettes/default.ts — Palette "baseline" de l'app.
 *
 * Extraction exacte des hex utilisés avant la migration accessibilité.
 * Sources : perf-color.ts, outcome-color.ts, match-card-presentation.ts,
 *           match-card.tsx, delta-card.tsx, HomePage.tsx, timeseries-heatmap.tsx.
 *
 * ⚠️  Ce fichier est la SEULE exception à la règle "zéro magic hex" —
 *     c'est sa raison d'être. Ne jamais importer ce fichier depuis un composant.
 */
import type { Palette } from '../semantic-tokens'

export const defaultPalette: Palette = {
  // ── Perf tiers (source : perf-color.ts) ────────────────────────────────────
  'perf-tier-1': '#10B981', // ≥80 — Excellent (emerald-500)
  'perf-tier-2': '#06B6D4', // ≥65 — Bon      (cyan-500)
  'perf-tier-3': '#F59E0B', // ≥50 — Correct  (amber-500)
  'perf-tier-4': '#F97316', // ≥35 — Faible   (orange-500)
  'perf-tier-5': '#EF4444', // <35 — Mauvais  (red-500)

  // ── Outcomes (source : outcome-color.ts) ───────────────────────────────────
  'outcome-win':  '#10B981', // emerald-500
  'outcome-loss': '#EF4444', // red-500
  'outcome-draw': '#3B82F6', // blue-500
  'outcome-dnf':  '#8B5CF6', // violet-500

  // ── Divergent (source : match-card.tsx, delta-card.tsx) ────────────────────
  'divergent-pos':     '#22C55E', // green-500
  'divergent-neutral': '#60A5FA', // blue-400 (zone non significative)
  'divergent-neg':     '#EF4444', // red-500

  // ── Statuts UI ─────────────────────────────────────────────────────────────
  'success':     '#10B981',
  'warning':     '#F59E0B',
  'info':        '#3B82F6',
  'destructive': '#EF4444',

  // ── Comparaisons (approximations des OKLCh de globals.css) ─────────────────
  'compare-a': '#818CF8', // oklch(0.65 0.15 250) ≈ indigo-400
  'compare-b': '#D97706', // oklch(0.70 0.15 55)  ≈ amber-600
  'compare-c': '#10B981', // oklch(0.72 0.15 160) ≈ emerald-500

  // ── Chart series — palette bleue/indigo actuelle ────────────────────────────
  'chart-series-1': '#93C5FD', // oklch(0.81 0.10 252) ≈ blue-300
  'chart-series-2': '#6366F1', // oklch(0.62 0.19 260) ≈ indigo-500
  'chart-series-3': '#4F46E5', // oklch(0.55 0.22 263) ≈ indigo-600
  'chart-series-4': '#4338CA', // oklch(0.49 0.22 264) ≈ indigo-700
  'chart-series-5': '#3730A3', // oklch(0.42 0.18 266) ≈ indigo-800
  'chart-series-6': '#10B981', // emerald — 6e série
  'chart-series-7': '#F59E0B', // amber   — 7e série
  'chart-series-8': '#EC4899', // pink    — 8e série

  // ── Bonus (assistances) — violet, distinct des 8 couleurs squad verrouillées ─
  // (joueurs + opposés). N'utilise PAS chart-series-7 (== perf-tier-3 ambre = joueur 3).
  'bonus': '#A855F7', // violet-500 (teinte 271°)

  // ── Badges narratifs (source : match-card-presentation.ts) ─────────────────
  // Couleurs ajustées pour atteindre WCAG AA (≥ 4.5:1) — cf. wcagContrast.test.ts
  'narrative-dominant':             '#00DC82',
  'narrative-dominant-text':        '#052E16',
  'narrative-humiliation':          '#7C3AED', // violet-600 (foncé) au lieu de violet-500 — contraste blanc 5.6
  'narrative-humiliation-text':     '#F8FAFC',
  'narrative-remontada':            '#0072B2',
  'narrative-remontada-text':       '#F8FAFC',
  'narrative-debacle':              '#D55E00',
  'narrative-debacle-text':         '#000000', // noir sur orange (5.4) — crème ne passait pas AA (3.64)
  'narrative-contre-remontada':     '#33D6FF',
  'narrative-contre-remontada-text':'#082F49',

  // ── Badges encounter (source : narrative/encounter.go ColorToken) ──────────
  'narrative-encounter-ally-plus':    '#10B981', // emerald-500 — allié positif
  'narrative-encounter-tough-enemy':  '#EF4444', // red-500     — K/D contre nous mauvais (Dur à cuire)
  'narrative-encounter-coriace':      '#F59E0B', // amber-500   — winrate vs lui mauvais (Coriace)
  'narrative-encounter-ordinal':      '#3B82F6', // blue-500    — compteur rencontres
  // Badges « solid » du hub Relations (Phase 1) : vert = positif, bleu = neutre.
  'narrative-encounter-duo-gagnant':    '#10B981', // emerald-500 — binôme gagnant
  'narrative-encounter-cameleon':       '#3B82F6', // blue-500    — autant allié qu'ennemi
  'narrative-encounter-de-longue-date': '#3B82F6', // blue-500    — relation ancienne
  'narrative-encounter-recrue':         '#3B82F6', // blue-500    — relation récente
  'narrative-encounter-proie-favorite': '#10B981', // emerald-500 — domination en duel

  // ── Heatmaps (source : timeseries-heatmap.tsx, heatmapChart.ts) ────────────
  'heatmap-cold':           '#EF4444', // mauvais — rouge
  'heatmap-hot':            '#10B981', // bon     — vert
  'heatmap-divergent-low':  '#FF4B4B', // K/D bas
  'heatmap-divergent-high': '#00DC82', // K/D haut

  // ── Équipes — défauts "Red vs Blue" classique Halo ─────────────────────────
  'team-ally':  '#3B82F6', // bleu  — overridable via settings accessibilité
  'team-enemy': '#EF4444', // rouge — overridable via settings accessibilité
}
