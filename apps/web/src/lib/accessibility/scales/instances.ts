/**
 * instances.ts — ★ Source de vérité unique de tous les seuils métier.
 *
 * Toute modification de seuil se fait ICI exclusivement.
 * Le test instances.test.ts snapshot ces valeurs : toute modification
 * non-intentionnelle fait échouer la CI.
 *
 * Décisions arbitrées (2026-04-25, cf. .ai/PLAN_OKABE_ITO_ACCESSIBILITY.md §9) :
 * - K/D : 3 tiers [1.0, 0.0] — ≥1 / [0,1[ / <0
 * - Bande neutre divergente : strict [0,0] par défaut
 * - mmrDeltaScale : ±10 justifié (bruit MMR < 10 non significatif)
 */
import { makeOrdinalScale } from './makeOrdinalScale'
import { makeDivergentScale } from './makeDivergentScale'
import { makeCategoricalScale } from './makeCategoricalScale'

// ── Ordinales ──────────────────────────────────────────────────────────────────

/** Score de performance global (0–100). Seuils : 80 / 65 / 50 / 35. */
export const perfScale = makeOrdinalScale({
  tiers: ['perf-tier-1', 'perf-tier-2', 'perf-tier-3', 'perf-tier-4', 'perf-tier-5'],
  thresholds: [80, 65, 50, 35],
})

/** Précision / accuracy (%). Seuils : 55 / 40. */
export const accuracyScale = makeOrdinalScale({
  tiers: ['perf-tier-1', 'perf-tier-3', 'perf-tier-5'],
  thresholds: [55, 40],
})

/** K/D ratio. 3 tiers : ≥1 (bon) / [0,1[ (moyen) / <0 (mauvais). */
export const kdScale = makeOrdinalScale({
  tiers: ['perf-tier-1', 'perf-tier-3', 'perf-tier-5'],
  thresholds: [1.0, 0.0], // décision §9.7
})

/** Progression gauge / barres de progression (0–100 %). Seuils : 75 / 50 / 25. */
export const progressScale = makeOrdinalScale({
  tiers: ['perf-tier-1', 'perf-tier-2', 'perf-tier-4', 'perf-tier-5'],
  thresholds: [75, 50, 25],
})

// ── Divergentes ────────────────────────────────────────────────────────────────

/**
 * Delta MMR — bande neutre ±10 (bruit MMR < 10 non significatif).
 * Décision §9.6.
 */
export const mmrDeltaScale = makeDivergentScale({
  positive: 'divergent-pos',
  neutral: 'divergent-neutral',
  negative: 'divergent-neg',
  neutralBand: [-10, 10],
})

/**
 * Delta CSR / LUSR / skill — strict zéro.
 * Aussi utilisé par DeltaCard en mode générique.
 */
export const skillDeltaScale = makeDivergentScale({
  positive: 'divergent-pos',
  neutral: 'divergent-neutral',
  negative: 'divergent-neg',
  neutralBand: [0, 0],
})

// ── Catégorielles ──────────────────────────────────────────────────────────────

/** Outcome numérique Halo : 2=win, 1=draw, 3=loss, 0/default=dnf. */
export const outcomeScale = makeCategoricalScale({
  win:  'outcome-win',
  loss: 'outcome-loss',
  draw: 'outcome-draw',
  dnf:  'outcome-dnf',
})

/** Badges narratifs de match. */
export const narrativeScale = makeCategoricalScale({
  dominant:          'narrative-dominant',
  humiliation:       'narrative-humiliation',
  remontada:         'narrative-remontada',
  debacle:           'narrative-debacle',
  contre_remontada:  'narrative-contre-remontada',
})
