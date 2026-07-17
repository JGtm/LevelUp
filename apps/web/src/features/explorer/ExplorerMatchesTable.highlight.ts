/**
 * ExplorerMatchesTable.highlight — logique pure de mise en valeur MVP/LVP
 * (meilleur ET pire par colonne clé) du tableau Matchs de l'Explorer.
 *
 * Réutilise cellState/cellStyle de MatchScoreboard.logic (mêmes tokens
 * sémantiques outcome-win / outcome-loss, teinte 28 % oklab, gras 600/500) — 3ᵉ
 * surface après le scoreboard et LeaderboardBlock.highlight ; le style est
 * IMPORTÉ, JAMAIS recopié (CLAUDE.md §6). Séparé du rendu pour test unitaire sans
 * jsdom (SRP).
 *
 * Extrêmes {min,max} par colonne calculés sur TOUT le scope chargé (les `rows` de
 * la table, pas la page visible), sur la valeur AFFICHÉE (=== exact vs extrêmes),
 * donc indépendants du tri et de la pagination. Clés = id de colonne TanStack.
 */
import { cellState, cellStyle, type Extremes } from '@/features/match-view/MatchScoreboard.logic'
import type { ExplorerMatchRow } from '@/lib/api/types'

export type ExplorerHighlightKey = 'kills' | 'deaths' | 'kda' | 'perf_score' | 'score_label'

/**
 * Direction par colonne : `true` = la PLUS PETITE valeur est la meilleure
 * (Morts) ; `false` = la plus grande (Frags, FDA, Perf, Score).
 */
export const EXPLORER_INVERTED: Record<ExplorerHighlightKey, boolean> = {
  kills: false,
  deaths: true,
  kda: false,
  perf_score: false,
  score_label: false,
}

/**
 * Score de l'équipe du joueur = 1er entier du libellé « A - B » (A = self, cf.
 * match_history_service_enrich.go: `"%d - %d"` MyTeamScore/EnemyTeamScore). null
 * si aucun chiffre (« - » = pas de score). NB : l'unité est mode-dépendante
 * (frags en Slayer, manches en objectif) — sur un scope multi-modes le highlight
 * Score compare des unités hétérogènes (réserve consignée Découverte-16).
 */
export function ownTeamScore(label: string | null | undefined): number | null {
  if (!label) return null
  const m = label.match(/\d+/)
  return m ? Number(m[0]) : null
}

/** Extracteurs = valeur AFFICHÉE dans la colonne (=== exact vs extrêmes). */
export const explorerHlExtract: Record<ExplorerHighlightKey, (r: ExplorerMatchRow) => number | null> = {
  kills: (r) => r.kills ?? null,
  deaths: (r) => r.deaths ?? null,
  kda: (r) => r.kda ?? null,
  // La cellule Perf n'affiche une valeur que si perf_score ET perf_tier sont
  // présents (sinon « - ») → aligner l'extracteur pour ne pas surligner un « - ».
  perf_score: (r) => (r.perf_score != null && r.perf_tier != null ? r.perf_score : null),
  score_label: (r) => ownTeamScore(r.score_label),
}

const HL_KEYS = Object.keys(explorerHlExtract) as ExplorerHighlightKey[]

export type ExplorerColumnExtremes = Record<ExplorerHighlightKey, Extremes>

/** Garde de type : `key` (id de colonne TanStack) est-il une colonne surlignée ? */
export function isExplorerHighlightKey(key: string): key is ExplorerHighlightKey {
  return key in EXPLORER_INVERTED
}

/**
 * Extrêmes {min,max} par colonne sur TOUT le scope. < 2 valeurs non nulles →
 * {null,null} (pas de highlight ; même garde ≥ 2 que getExtremes côté scoreboard,
 * évite un faux positif sur une colonne à une seule valeur).
 */
export function computeColumnExtremes(rows: ExplorerMatchRow[]): ExplorerColumnExtremes {
  const out = {} as ExplorerColumnExtremes
  for (const k of HL_KEYS) {
    const vals = rows.map(explorerHlExtract[k]).filter((v): v is number => v != null)
    out[k] =
      vals.length >= 2 ? { min: Math.min(...vals), max: Math.max(...vals) } : { min: null, max: null }
  }
  return out
}

/**
 * Style inline best/worst d'une cellule (vert outcome-win / rouge outcome-loss),
 * ou {} si neutre (null, colonne uniforme, < 2 valeurs). `value` DOIT être la
 * valeur affichée (=== exact vs extrêmes).
 */
export function columnHighlightStyle(
  key: ExplorerHighlightKey,
  value: number | null,
  extremes: ExplorerColumnExtremes,
) {
  return cellStyle(cellState(value, extremes[key], EXPLORER_INVERTED[key]))
}
