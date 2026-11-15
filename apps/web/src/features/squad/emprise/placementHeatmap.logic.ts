/**
 * placementHeatmap.logic.ts — la grille de la carte de chaleur « Distance au coéquipier le plus
 * proche et frags, par vie » (Escouade › Emprise › Groupé ou isolé). Logique pure : le graphe ne
 * fait que peindre ce que ce fichier compte.
 *
 * Colonnes : 8 cases de 0,25 portée de radar, de 0 à 2 ; la dernière (1,75 et plus) reçoit aussi
 * les vies au-delà de 2 (le plafond d'affichage est celui de la spécification). Lignes : 0, 1, 2,
 * 3, 4 frags et « 5 et plus ». Une case porte sa part des vies de l'escouade (toutes les vies
 * mesurées de tous les joueurs du bloc) ; ses bornes, son nombre de vies et son intensité en
 * découlent. Aucune couleur ici.
 */
import { meanOfValid } from '@/lib/charts/stats'
import type { SquadEmprisePlacementPlayer } from '@/lib/api/types'

/** Axe X : 0 à 2 portées de radar, cases de 0,25. */
export const X_MAX = 2
export const X_STEP = 0.25
export const X_BINS = Math.round(X_MAX / X_STEP)
/** Axe Y : 0 à 5 frags, la dernière ligne vaut « 5 et plus ». */
export const KILLS_CAP = 5
export const Y_ROWS = KILLS_CAP + 1
/** Une valeur s'écrit dans sa case à partir de 1,5 % des vies de l'escouade. */
export const CELL_LABEL_MIN_SHARE = 0.015
/** Intensité plancher d'une case non vide (racine carrée de la part, ramenée entre ce plancher et 1). */
export const CELL_INTENSITY_MIN = 0.14

export interface HeatCell {
  /** Colonne 0..7 et ligne 0..5 (= frags dans la vie, plafonnés à 5). */
  col: number
  row: number
  /** Bornes de la colonne en portées de radar : [x0, x1[ ; `open` pour la dernière (au moins x0). */
  x0: number
  x1: number
  openRight: boolean
  /** La ligne 5 vaut « 5 et plus ». */
  openTop: boolean
  lives: number
  /** Part des vies de l'escouade, 0..1. */
  share: number
  /** Intensité d'affichage 0..1 : racine carrée de (part / part maximale), plancher `CELL_INTENSITY_MIN`. */
  intensity: number
}

export interface HeatGrid {
  /** Seulement les cases non vides. */
  cells: HeatCell[]
  /** Toutes les vies mesurées du bloc. */
  total: number
}

/** La colonne d'une vie : cases de 0,25, la dernière absorbe le plafond. */
export function columnOf(radarRatio: number): number {
  return Math.min(Math.max(Math.floor(radarRatio / X_STEP), 0), X_BINS - 1)
}

/** La ligne d'une vie : ses frags, plafonnés à 5. */
export function rowOf(kills: number): number {
  return Math.min(Math.max(Math.floor(kills), 0), KILLS_CAP)
}

/** Compte les vies de tous les joueurs du bloc dans la grille 8 x 6. */
export function binLives(players: readonly SquadEmprisePlacementPlayer[]): HeatGrid {
  const counts = new Map<number, number>()
  let total = 0
  for (const p of players) {
    for (const l of p.lives ?? []) {
      const key = rowOf(l.kills) * X_BINS + columnOf(l.radar_ratio)
      counts.set(key, (counts.get(key) ?? 0) + 1)
      total++
    }
  }
  if (total === 0) return { cells: [], total: 0 }
  const maxCount = Math.max(...counts.values())
  const cells: HeatCell[] = [...counts.entries()]
    .sort((a, b) => a[0] - b[0])
    .map(([key, lives]) => {
      const col = key % X_BINS
      const row = Math.floor(key / X_BINS)
      const share = lives / total
      return {
        col,
        row,
        x0: col * X_STEP,
        x1: (col + 1) * X_STEP,
        openRight: col === X_BINS - 1,
        openTop: row === KILLS_CAP,
        lives,
        share,
        intensity: CELL_INTENSITY_MIN + (1 - CELL_INTENSITY_MIN) * Math.sqrt(lives / maxCount),
      }
    })
  return { cells, total }
}

/** Une case s'écrit-elle ? (part d'au moins 1,5 %). */
export function cellShowsLabel(cell: HeatCell): boolean {
  return cell.share >= CELL_LABEL_MIN_SHARE
}

/**
 * Le gros point d'un joueur : sa distance médiane (portées de radar) et sa moyenne de frags par
 * vie, plafonnées à l'affichage comme les cases. `null` sans vie mesurée. La moyenne se calcule
 * sur les vies du bloc ; à défaut de liste de vies, la médiane du contrat tient lieu.
 */
export function playerSpot(p: SquadEmprisePlacementPlayer): { x: number; y: number; meanKills: number } | null {
  if (p.median_radar_ratio == null) return null
  const mean = meanOfValid((p.lives ?? []).map((l) => l.kills)) ?? p.median_kills
  if (mean == null) return null
  return { x: Math.min(p.median_radar_ratio, X_MAX), y: Math.min(mean, KILLS_CAP), meanKills: mean }
}
