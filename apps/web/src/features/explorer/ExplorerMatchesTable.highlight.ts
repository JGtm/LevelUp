/**
 * ExplorerMatchesTable.highlight — logique pure de mise en valeur MVP/LVP du
 * tableau Matchs de l'Explorer, en BANDE DE DÉCILE (top 10 % / pire 10 %).
 *
 * Réutilise cellStyle de MatchScoreboard.logic (mêmes tokens sémantiques
 * outcome-win / outcome-loss, gras 600/500) — le style est IMPORTÉ, JAMAIS
 * recopié (CLAUDE.md §6). L'état best/worst, lui, n'est PAS `cellState` (dont
 * l'égalité `===` cible un extrême UNIQUE) mais `decileCellState` local : une
 * valeur dans le décile haut (≥ p90) est « meilleure », dans le décile bas
 * (≤ p10) « pire » (sens inversé pour Morts). Séparé du rendu pour test
 * unitaire sans jsdom (SRP).
 *
 * Seuils p10/p90 par colonne calculés sur TOUT le scope chargé (les `rows` de
 * la table, pas la page visible), sur la valeur AFFICHÉE, donc indépendants du
 * tri et de la pagination. Clés = id de colonne TanStack.
 */
import { cellStyle, type CellState } from '@/features/match-view/MatchScoreboard.logic'
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

/** Extracteurs = valeur AFFICHÉE dans la colonne. */
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

/**
 * Taille minimale d'échantillon (valeurs non nulles) pour qu'une colonne soit
 * éligible au surlignage par décile. En deçà, les déciles p10/p90 n'ont pas de
 * sens statistique → aucune mise en valeur. Nommé + ajustable en revue visuelle.
 */
export const MIN_DECILE_SAMPLE = 10

/**
 * Intensité (%) de la teinte de fond des cellules surlignées, passée à
 * `cellStyle`. Volontairement plus DOUCE que le défaut scoreboard/leaderboard
 * (28 %) car une bande de décile colore davantage de cellules qu'un extrême
 * unique. Nommé + ajustable en revue visuelle.
 */
export const DECILE_TINT_PCT = 16

/** Seuils de décile d'une colonne (null si échantillon insuffisant). */
export type Deciles = { p10: number | null; p90: number | null }

export type ExplorerColumnDeciles = Record<ExplorerHighlightKey, Deciles>

/** Garde de type : `key` (id de colonne TanStack) est-il une colonne surlignée ? */
export function isExplorerHighlightKey(key: string): key is ExplorerHighlightKey {
  return key in EXPLORER_INVERTED
}

/**
 * Percentile « nearest-rank » sur un tableau trié ASC non vide : rang ordinal
 * n = ceil(p/100 × N), valeur = sorted[n − 1] (borné dans [0, N−1]).
 */
function percentileNearestRank(sortedAsc: number[], p: number): number {
  const n = sortedAsc.length
  const rank = Math.ceil((p / 100) * n)
  const idx = Math.min(Math.max(rank - 1, 0), n - 1)
  return sortedAsc[idx]
}

/**
 * Seuils p10/p90 par colonne sur TOUT le scope. < MIN_DECILE_SAMPLE valeurs non
 * nulles → {null,null} (pas de highlight — un décile n'a pas de sens sous ce
 * seuil).
 */
export function computeColumnDeciles(rows: ExplorerMatchRow[]): ExplorerColumnDeciles {
  const out = {} as ExplorerColumnDeciles
  for (const k of HL_KEYS) {
    const vals = rows
      .map(explorerHlExtract[k])
      .filter((v): v is number => v != null)
      .sort((a, b) => a - b)
    out[k] =
      vals.length >= MIN_DECILE_SAMPLE
        ? { p10: percentileNearestRank(vals, 10), p90: percentileNearestRank(vals, 90) }
        : { p10: null, p90: null }
  }
  return out
}

/**
 * État best/worst/neutral d'une cellule selon sa position dans la bande de
 * décile de sa colonne. `p10 === p90` (colonne quasi-uniforme) → neutre. Non
 * inversé : ≥ p90 = meilleur, ≤ p10 = pire. Inversé (Morts) : symétrique.
 */
export function decileCellState(value: number | null, d: Deciles, inverted: boolean): CellState {
  if (value == null || d.p10 == null || d.p90 == null || d.p10 === d.p90) return 'neutral'
  const inTop = value >= d.p90
  const inBottom = value <= d.p10
  if (inverted) {
    if (inBottom) return 'best'
    if (inTop) return 'worst'
  } else {
    if (inTop) return 'best'
    if (inBottom) return 'worst'
  }
  return 'neutral'
}

/**
 * Style inline best/worst d'une cellule (vert outcome-win / rouge outcome-loss)
 * en teinte DOUCE (DECILE_TINT_PCT), ou {} si neutre (null, colonne uniforme,
 * échantillon insuffisant, hors bande). `value` DOIT être la valeur affichée.
 */
export function columnHighlightStyle(
  key: ExplorerHighlightKey,
  value: number | null,
  deciles: ExplorerColumnDeciles,
) {
  return cellStyle(decileCellState(value, deciles[key], EXPLORER_INVERTED[key]), DECILE_TINT_PCT)
}
