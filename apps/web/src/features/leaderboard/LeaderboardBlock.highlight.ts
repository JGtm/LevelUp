/**
 * LeaderboardBlock.highlight — logique pure de mise en valeur best/worst par
 * colonne du classement (parité scoreboard).
 *
 * Réutilise cellState/cellStyle de MatchScoreboard.logic (mêmes tokens
 * sémantiques outcome-win / outcome-loss) ; calcule les extrêmes {min,max} par
 * colonne sur les entrées enrichies. Séparé du rendu pour test unitaire sans
 * jsdom (cf. CLAUDE.md SRP — un fichier = une responsabilité).
 */
import { cellState, cellStyle, type Extremes } from '@/features/match-view/MatchScoreboard.logic'
import type { LeaderboardEntry } from '@/lib/api/types'

/** Dégâts infligés par frag (k + a/3) depuis les compteurs bruts. null si dénominateur nul. */
export function dmgPerKill(e: LeaderboardEntry): number | null {
  const eff = (e.kills ?? 0) + (e.assists ?? 0) / 3
  return e.damage_dealt != null && eff > 0 ? e.damage_dealt / eff : null
}
/** Dégâts subis par mort. null si aucune mort. */
export function dmgPerDeath(e: LeaderboardEntry): number | null {
  return e.damage_taken != null && e.deaths ? e.damage_taken / e.deaths : null
}

export type HighlightKey =
  | 'csr'
  | 'fda'
  | 'kills'
  | 'deaths'
  | 'assists'
  | 'winRate'
  | 'accuracy'
  | 'dmgKill'
  | 'dmgDeath'

/**
 * Direction par colonne : `true` = la PLUS PETITE valeur est la meilleure
 * (Morts, Dégâts/frag) ; `false` = la plus grande (défaut).
 */
export const HL_INVERTED: Record<HighlightKey, boolean> = {
  csr: false,
  fda: false,
  kills: false,
  deaths: true,
  assists: false,
  winRate: false,
  accuracy: false,
  dmgKill: true,
  dmgDeath: false,
}

/** Valeur PAR MATCH (FDA/Précision) — même calcul que l'affichage → === exact. */
const perMatch = (e: LeaderboardEntry, v?: number | null): number | null =>
  v != null && e.match_count ? v / e.match_count : null

/** Extracteurs par colonne — identiques aux valeurs affichées dans la ligne. */
export const hlExtract: Record<HighlightKey, (e: LeaderboardEntry) => number | null> = {
  csr: (e) => e.csr_value,
  fda: (e) => perMatch(e, e.kda),
  kills: (e) => e.kills ?? null,
  deaths: (e) => e.deaths ?? null,
  assists: (e) => e.assists ?? null,
  winRate: (e) => e.win_rate ?? null,
  accuracy: (e) => perMatch(e, e.accuracy),
  dmgKill: dmgPerKill,
  dmgDeath: dmgPerDeath,
}

const HL_KEYS = Object.keys(hlExtract) as HighlightKey[]

export type ColumnExtremes = Record<HighlightKey, Extremes>

/**
 * Extrêmes {min,max} par colonne sur les entrées enrichies (match_count != null).
 * < 2 valeurs non nulles → {null,null} (pas de highlight ; même garde que
 * getExtremes côté scoreboard, évite un faux positif sur une seule entrée).
 */
export function computeColumnExtremes(entries: LeaderboardEntry[]): ColumnExtremes {
  const enriched = entries.filter((e) => e.match_count != null)
  const out = {} as ColumnExtremes
  for (const k of HL_KEYS) {
    const vals = enriched.map(hlExtract[k]).filter((v): v is number => v != null)
    out[k] = vals.length >= 2 ? { min: Math.min(...vals), max: Math.max(...vals) } : { min: null, max: null }
  }
  return out
}

/**
 * Style inline best/worst d'une cellule (vert outcome-win / rouge outcome-loss),
 * ou {} si neutre. `value` DOIT être la valeur affichée (=== exact vs extrêmes).
 */
export function columnHighlightStyle(key: HighlightKey, value: number | null, extremes: ColumnExtremes) {
  return cellStyle(cellState(value, extremes[key], HL_INVERTED[key]))
}
