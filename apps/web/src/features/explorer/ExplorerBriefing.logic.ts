/**
 * ExplorerBriefing.logic — helpers purs du bandeau de briefing (mode Matchs).
 *
 * Aucune dépendance React/DOM : logique testable en isolation (formatage des
 * deltas signés, KDA agrégat, mapping de la frise). Les composants du bandeau
 * consomment ces helpers.
 */
import type { SemanticToken } from '@/lib/accessibility'

// formatSignedFixed (delta signé à N décimales, glyphe '−' U+2212) est centralisé
// dans `@/lib/formatters` — importé directement par les modules du briefing.

/**
 * Formate un delta de TAUX (ratio 0..1) en points de pourcentage signés
 * (ex. +0.30 → "+30 pts"). Unité « pts » pour distinguer d'un pourcentage absolu.
 */
export function formatSignedPoints(v: number | null | undefined): string {
  if (v == null || !Number.isFinite(v)) return ''
  const pts = Math.round(v * 100)
  if (pts === 0) return '±0 pts'
  return pts > 0 ? `+${pts} pts` : `−${Math.abs(pts)} pts`
}

/**
 * Vrai quand le scope affiché couvre TOUT l'historique (aucun filtre narrowing).
 *
 * Le scope est toujours un SOUS-ENSEMBLE de la baseline (un filtre ne peut que
 * rétrécir) : des cardinalités égales impliquent donc des ensembles identiques,
 * d'où des deltas « vs habituel » nuls par construction — à masquer (P-1). Faux
 * si la baseline est absente (`undefined !== number`) : sans baseline il n'y a de
 * toute façon aucun delta à afficher.
 */
export function isFullHistoryScope(
  scopeMatches: number | null | undefined,
  baselineMatches: number | null | undefined,
): boolean {
  return scopeMatches != null && baselineMatches === scopeMatches
}

/** Signe d'un nombre : -1 / 0 / 1 (0 pour nul, absent ou non fini). */
export function signOf(v: number | null | undefined): -1 | 0 | 1 {
  if (v == null || !Number.isFinite(v) || v === 0) return 0
  return v > 0 ? 1 : -1
}

/** Token de couleur d'un delta signé (positif = gagnant, négatif = perdant, nul = neutre). */
export function deltaToken(v: number | null | undefined): SemanticToken {
  const s = signOf(v)
  return s > 0 ? 'outcome-win' : s < 0 ? 'outcome-loss' : 'outcome-draw'
}

// Mapping code outcome → valeur de frise : centralisé dans `@/lib/outcome`
// (`outcomeCodeToTapeValue`). Consommé directement par ExplorerBriefingStrip.
