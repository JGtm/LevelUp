/**
 * objectivesChart.ts — LA PROJECTION DES DEUX VUES DE LA SECTION « OBJECTIFS ».
 *
 * LE TABLEAU EST DEVENU UN GRAPHE (2026-09-03, retours utilisateur sur la page match). Deux vues
 * empilées le remplacent :
 *   1. « Actions d'objectif par joueur » — la grille partagée `components/charts/ValueGrid` :
 *      lignes = joueurs camp par camp, colonnes = grandeurs du mode, chacune avec SON échelle ;
 *   2. « Total d'objectif par équipe » — un face-à-face : une ligne par grandeur, le zéro au
 *      CENTRE, la longueur d'un côté disant l'AVANTAGE et non la valeur absolue.
 *
 * RIEN N'EST ÉCRIT EN DUR SUR LE MODE, ET C'EST LE POINT CRITIQUE. Les colonnes viennent de
 * `objectiveColsFor(mode)` et l'agrégat d'équipe de `objectiveTeamTotal` — qui fait `sum` OU
 * `max` selon la colonne, parce qu'un « meilleur temps » ne s'additionne pas. Les deux vues
 * encaissent donc les six modes à objectif sans une ligne de code par mode : drapeau (4
 * colonnes), zones — Bastion ET King of the Hill (3), crâne (3), réserve (4), extraction (4) et
 * VIP (5).
 *
 * `null` N'EST PAS ZÉRO. Une grandeur absente du bloc d'objectif d'un joueur est NON MESURÉE :
 * sa barre reste vide et sa cellule écrit le repli, jamais un zéro qui se lirait comme une
 * mesure. Un zéro présent, lui, est une mesure — d'où le repli « 0:00 » des durées, et non le
 * tiret que `formatDurationMMSS` sert par défaut à une durée de MATCH nulle.
 *
 * L'ENCRE D'UN CAMP EST CELLE DES JETONS `team-ally` / `team-enemy` (via `teamTokenCssVar`), donc
 * la palette d'accessibilité réglée par l'utilisateur — et non la cascade d'IDENTITÉ de
 * `teamColor.ts` que la section employait tant qu'elle était un tableau : un graphe suit le
 * réglage de l'utilisateur, un bandeau d'équipe suit la couleur officielle du jeu (la frontière
 * est écrite en tête de `teamSeriesColor.ts`).
 *
 * Pur : aucun React, aucun hex, aucune langue — les libellés et les encres arrivent par
 * l'appelant.
 */
import type { ValueGridModel, ValueGridRow } from '@/components/charts/valueGridModel'
import { buildValueGrid } from '@/components/charts/valueGridModel'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { formatDurationMMSS } from '@/lib/formatters/duration'
import { displayPlayerName } from '@/lib/players/displayName'

import { objectiveTeamTotal, type ObjectiveColSpec } from './MatchScoreboard.logic'

/**
 * Le texte d'une valeur d'objectif. Une DURÉE mesurée à zéro s'écrit « 0:00 » : la colonne
 * n'existe que parce que le mode la mesure, et un tiret s'y lirait « non mesuré ».
 */
export function objectiveValueText(value: number, col: ObjectiveColSpec): string {
  return col.duration ? formatDurationMMSS(value, '0:00') : String(value)
}

/** La valeur d'une colonne pour une ligne de scoreboard. `null` = non mesurée. */
export function objectiveValue(row: MatchScoreboardRow, col: ObjectiveColSpec): number | null {
  const raw = row.objective ? (row.objective[col.key] as number | null | undefined) : null
  return raw ?? null
}

/** Ce que l'appelant fournit pour habiller un camp. */
export interface ObjectiveTeamVisual {
  teamLabel: (side: string) => string
  teamColor: (side: string) => string
}

/** Les entrées de la vue 1. */
export interface ObjectiveGridInput extends ObjectiveTeamVisual {
  /** Les lignes à montrer, DÉJÀ groupées par camp dans l'ordre d'affichage. */
  rows: MatchScoreboardRow[]
  cols: ObjectiveColSpec[]
  colLabel: (col: ObjectiveColSpec) => string
  tipFmt: (player: string, metric: string, value: string) => string
}

/**
 * buildObjectiveGrid — la vue 1. Les barres prennent l'encre du CAMP du joueur (et non celle de
 * la grandeur) : sur cette vue, ce qu'on cherche est de quel côté penche chaque colonne.
 */
export function buildObjectiveGrid(input: ObjectiveGridInput): ValueGridModel {
  const { rows, cols } = input
  const gridRows: ValueGridRow[] = rows.map((r, i) => {
    const side = r.team_side ?? ''
    return {
      // Clé composite : le xuid peut être vide (joueurs H5 sans xuid) et plusieurs lignes se
      // percuteraient sur une clé vide.
      key: `${r.xuid}||${r.gamertag}||${i}`,
      label: displayPlayerName(r.gamertag, r.xuid),
      group: side,
      accent: input.teamColor(side),
      emphasis: r.is_me === true,
      hint: `${displayPlayerName(r.gamertag, r.xuid)} — ${input.teamLabel(side)}`,
    }
  })
  return buildValueGrid({
    rows: gridRows,
    columns: cols.map((c) => ({
      key: String(c.key),
      label: input.colLabel(c),
      duration: c.duration,
      // Une somme de « meilleurs temps » n'a pas de sens : la colonne `max` n'affiche pas de
      // total (même règle que `objectiveTeamTotal`).
      showTotal: c.agg === 'sum',
    })),
    value: (r, c) => objectiveValue(rows[r], cols[c]),
    format: (v, c) => objectiveValueText(v, cols[c]),
    color: (r) => input.teamColor(rows[r].team_side ?? ''),
    tooltip: (r, c, text) => input.tipFmt(gridRows[r].label, input.colLabel(cols[c]), text),
  })
}

/** Un côté d'un face-à-face : un camp sur une grandeur. */
export interface ObjectiveDuelSide {
  side: string
  label: string
  value: number | null
  text: string
  /** Part de la plus grande des deux valeurs de LA LIGNE, dans [0, 1]. */
  fraction: number
  color: string
  tooltip: string
}

/** Une ligne du face-à-face : une grandeur, ses deux camps. */
export interface ObjectiveDuelRow {
  key: string
  label: string
  left: ObjectiveDuelSide
  right: ObjectiveDuelSide
}

/** Les entrées de la vue 2. */
export interface ObjectiveDuelInput extends ObjectiveTeamVisual {
  rows: MatchScoreboardRow[]
  /** EXACTEMENT deux camps : un face-à-face à trois côtés n'existe pas. */
  teams: readonly [string, string]
  cols: ObjectiveColSpec[]
  colLabel: (col: ObjectiveColSpec) => string
  tipFmt: (team: string, metric: string, value: string) => string
}

/**
 * buildObjectiveDuel — la vue 2.
 *
 * CHAQUE LIGNE A SA PROPRE ÉCHELLE, et le zéro est au CENTRE : la longueur d'un côté dit
 * l'avantage sur cette grandeur, pas sa valeur absolue. Une échelle commune à toutes les lignes
 * écraserait « zones sécurisées » (une poignée) contre « temps en zone » (des minutes).
 *
 * L'AGRÉGAT VIENT DE `objectiveTeamTotal` : cumul, ou MAXIMUM pour un « meilleur temps ». Le
 * recalculer ici donnerait deux totaux d'équipe pour la même colonne.
 */
export function buildObjectiveDuel(input: ObjectiveDuelInput): ObjectiveDuelRow[] {
  const [leftSide, rightSide] = input.teams
  const rowsOf = (side: string) => input.rows.filter((r) => (r.team_side ?? '') === side)
  const left = rowsOf(leftSide)
  const right = rowsOf(rightSide)
  return input.cols.map((col) => {
    const a = objectiveTeamTotal(left, col)
    const b = objectiveTeamTotal(right, col)
    const scale = Math.max(a ?? 0, b ?? 0) || 1
    const label = input.colLabel(col)
    const sideOf = (side: string, value: number | null): ObjectiveDuelSide => {
      const text = value == null ? '—' : objectiveValueText(value, col)
      return {
        side,
        label: input.teamLabel(side),
        value,
        text,
        fraction: value == null ? 0 : Math.max(0, Math.min(1, value / scale)),
        color: input.teamColor(side),
        tooltip: input.tipFmt(input.teamLabel(side), label, text),
      }
    }
    return {
      key: String(col.key),
      label,
      left: sideOf(leftSide, a),
      right: sideOf(rightSide, b),
    }
  })
}
