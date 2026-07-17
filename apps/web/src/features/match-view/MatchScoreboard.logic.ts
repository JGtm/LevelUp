/**
 * MatchScoreboard.logic — logique pure du scoreboard, séparée du rendu.
 *
 * Extraite de MatchScoreboard.tsx pour respecter la limite de 500 lignes par
 * module (CLAUDE.md §14) et faciliter le test unitaire (cf. CLAUDE.md §17 :
 * SRP — un fichier = une responsabilité).
 *
 * Contient uniquement des fonctions stateless :
 *   - getExtremes        : min/max d'une colonne sur le lobby
 *   - cellState          : best/worst/neutral d'une cellule (gère `inverted`)
 *   - cellStyle          : style CSS d'une cellule selon son état
 *   - getMvpLvp          : sélection MVP/LVP au LOBBY (≥2 best / ≥2 worst)
 *   - formatRank/Score   : formatters communs au scoreboard
 *
 * Tests : MatchScoreboard.test.ts (tests unitaires sans rendu).
 */
import type { MatchScoreboardRow } from '@/lib/api/types'

export interface ColDef {
  key: keyof MatchScoreboardRow
  label: string
  inverted: boolean
  fmt?: (v: number) => string
}

export type Extremes = { min: number | null; max: number | null }

export function getExtremes(rows: MatchScoreboardRow[], key: keyof MatchScoreboardRow): Extremes {
  const vals = rows.map((r) => r[key] as number | null).filter((v): v is number => v != null)
  if (vals.length < 2) return { min: null, max: null }
  return { min: Math.min(...vals), max: Math.max(...vals) }
}

export type CellState = 'best' | 'worst' | 'neutral'

export function cellState(value: number | null, ex: Extremes, inverted: boolean): CellState {
  if (value == null || ex.min == null || ex.max == null || ex.min === ex.max) return 'neutral'
  const isBest = inverted ? value === ex.min : value === ex.max
  if (isBest) return 'best'
  const isWorst = inverted ? value === ex.max : value === ex.min
  if (isWorst) return 'worst'
  return 'neutral'
}

/**
 * Style inline pour une cellule selon son état best/worst.
 *
 * Tokens `outcome-win` / `outcome-loss` accessibility-aware, overridables via
 * les palettes utilisateur (cf. lib/accessibility/palettes/). Cohérent avec
 * la border MVP/LVP sur la cellule gamertag (même rôle visuel : meilleur
 * lobby / pire lobby — voir thought_log 2026-05-07).
 *
 * Background tinté via `color-mix(in oklab, ...)` pour rester lisible avec le
 * texte pleine couleur du même token. `intensityPct` (défaut 28) pilote la
 * densité de la teinte : les callers scoreboard/leaderboard gardent 28 %,
 * l'Explorer passe une valeur plus douce pour ses bandes de décile
 * (ExplorerMatchesTable.highlight, DECILE_TINT_PCT). Pas de hex hardcodé.
 */
export function cellStyle(state: CellState, intensityPct: number = 28): React.CSSProperties {
  if (state === 'neutral') return {}
  const tokenVar = state === 'best' ? 'var(--ac-outcome-win)' : 'var(--ac-outcome-loss)'
  return {
    backgroundColor: `color-mix(in oklab, ${tokenVar} ${intensityPct}%, transparent)`,
    color: tokenVar,
    fontWeight: state === 'best' ? 600 : 500,
  }
}

/**
 * MVP/LVP basé sur le nombre de cellules best/worst par joueur sur les
 * colonnes comparables (mock 4b). Seuil ≥2 best / ≥2 worst pour éviter les
 * faux positifs sur de petits scoreboards. Tie → premier du tri stable.
 *
 * Calculé sur l'ensemble du LOBBY (toutes équipes confondues, hors bots) —
 * le caller filtre les bots avant de passer `rows`. Un seul MVP + un seul
 * LVP par lobby (pas par équipe).
 */
export function getMvpLvp(
  rows: MatchScoreboardRow[],
  cols: ColDef[],
  extremesByKey: Record<string, Extremes>,
): { mvp: string | null; lvp: string | null } {
  if (rows.length < 2) return { mvp: null, lvp: null }
  let mvpXuid: string | null = null
  let mvpBest = 1
  let lvpXuid: string | null = null
  let lvpWorst = 1
  for (const r of rows) {
    let best = 0
    let worst = 0
    for (const c of cols) {
      const ex = extremesByKey[String(c.key)]
      if (!ex) continue
      const state = cellState(r[c.key] as number | null, ex, c.inverted)
      if (state === 'best') best += 1
      else if (state === 'worst') worst += 1
    }
    if (best >= 2 && best > mvpBest) {
      mvpBest = best
      mvpXuid = r.xuid
    }
    if (worst >= 2 && worst > lvpWorst) {
      lvpWorst = worst
      lvpXuid = r.xuid
    }
  }
  return { mvp: mvpXuid, lvp: lvpXuid }
}

export function formatRank(rank: number | null): string {
  if (rank == null) return '—'
  return String(rank)
}
