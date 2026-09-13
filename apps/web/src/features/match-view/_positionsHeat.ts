/**
 * _positionsHeat.ts — LA GRILLE DE CHALEUR DES POSITIONS D'UN MATCH, calée sur le fond de carte.
 *
 * POURQUOI CE MODULE EXISTE. Le bloc « Carte de chaleur des positions » binnait ces positions
 * en une grille 20×20 posée sur les bornes du nuage de points, SANS fond de carte : un damier
 * de nombres dont l'utilisateur a dit le 2026-09-13 « je ne sais pas ce que ça rend, à quoi ça
 * sert ou quel est le narratif ». La donnée, elle, est bonne — ce qui manquait est le PLAN. Le
 * match porte déjà son fond (`…/replay/background`, le même que le rejeu 2D et l'onglet
 * Tactique) et son CALAGE monde↔pixels : la grille se pose donc sur la carte, à l'échelle du
 * jeu, et la question devient lisible — où ça se joue.
 *
 * LE PAS EST LE RAYON D'ENGAGEMENT DU REJEU (`HEAT_SIGMA_M`, 2 m — cf. `POSITIONS_CELL_M`), et
 * c'est un PLANCHER : sur une très grande carte il grandit pour que la grille reste sous son
 * plafond de cellules, exactement comme `cellSizeFor` côté rejeu. Une cellule jamais atteinte
 * reste VIDE — pas « froide » : personne n'y est passé, l'écran ne doit pas peindre un zéro
 * mesuré.
 *
 * L'ÉCHELLE EST QUANTILE (p50 → p95) comme partout ailleurs dans le dépôt : un seul point
 * extrême n'écrase pas la carte. Échelle dégénérée (toutes les cellules à la même valeur — le
 * cas d'un match peu échantillonné) : on retombe sur [0, max], la règle de `scaleOf`.
 *
 * Pur : aucun React, aucun canvas, aucune langue.
 */
import type { MatchPlayerPosition, ReplayMapBackgroundCalibration } from '@/lib/api/types'
import {
  buildTacticalGrid,
  HEAT_SIGMA_M,
  mapFrame as mapFrameDuNoyau,
  type MapFrame,
  type TacticalCell,
  type TacticalGrid,
} from '@/lib/replay/heatPaint'

export type { MapFrame }

/**
 * Pas de grille visé, en mètres monde : le RAYON D'ENGAGEMENT du rejeu (`HEAT_SIGMA_M`).
 *
 * POURQUOI PAS LES 0,5 m DE LA CARTE DU REJEU. Là-bas, la cellule de 0,5 m n'est lisible que
 * parce qu'un lissage gaussien d'écart-type 2 m étale ensuite chaque dépôt sur son voisinage.
 * Ici l'entrée n'est pas une trajectoire continue mais des positions AUX IMAGES-CLÉS — 367 sur
 * le match témoin, pour une carte de 63 x 55 m : à 0,5 m sans lissage, la carte rend 367 confettis
 * isolés (constaté sur capture le 2026-09-13), où l'œil ne lit aucune zone. La cellule porte donc
 * elle-même le rayon d'engagement, ce que le lissage faisait là-bas : même échelle de lecture,
 * sans inventer une présence entre deux mesures.
 */
export const POSITIONS_CELL_M = HEAT_SIGMA_M

/** Plafond de cellules : au-delà, le pas grandit (mémoire et temps de tracé bornés). */
const MAX_CELLS = 200_000

/** Quantiles d'étalonnage (bas / haut) — la convention du dépôt. */
const Q_LOW = 0.5
const Q_HIGH = 0.95

/** `team = -1` → camp non attribué par le film (regroupement spatial best-effort). */
export const TEAM_UNKNOWN = -1

/**
 * mapFrame traduit le calage du fond en cadre monde exploitable.
 *
 * LE TYPE ET LE CALCUL VIVENT DANS LE NOYAU PARTAGÉ (`lib/replay/heatPaint.ts`) depuis le
 * 2026-09-13 : le plan de l'onglet Tactique s'y est ajouté comme troisième lecteur, et à la
 * troisième copie la définition remonte. Ce wrapper garde la signature typée du contrat.
 */
export function mapFrame(cal: ReplayMapBackgroundCalibration): MapFrame {
  return mapFrameDuNoyau(cal)
}

/** Le pas retenu : le plancher, agrandi si la carte dépasse le plafond de cellules. */
export function positionsCellSize(frame: MapFrame): number {
  const w = Math.max(frame.widthM, 1)
  const h = Math.max(frame.heightM, 1)
  return Math.max(POSITIONS_CELL_M, Math.sqrt((w * h) / MAX_CELLS))
}

function quantile(sorted: number[], q: number): number {
  return sorted[Math.min(sorted.length - 1, Math.floor(sorted.length * q))]
}

/**
 * buildPositionsGrid — les positions déposées dans la grille du fond de carte.
 *
 * La ligne 0 est EN HAUT (le sens de `drawTacticalHeatmap`, celui d'une image) : une position
 * de Y monde élevé tombe dans une ligne basse d'index. Hors du cadre du fond : la position est
 * IGNORÉE, jamais rabattue sur un bord — un point hors carte n'a rien à dire d'un bord.
 *
 * Rend `null` quand rien n'est déposé : il n'y a alors pas de carte à peindre.
 */
export function buildPositionsGrid(
  positions: readonly MatchPlayerPosition[],
  frame: MapFrame,
): TacticalGrid | null {
  const cell = positionsCellSize(frame)
  const nx = Math.max(1, Math.ceil(frame.widthM / cell))
  const ny = Math.max(1, Math.ceil(frame.heightM / cell))
  const counts = new Map<string, number>()
  for (const p of positions) {
    const col = Math.floor((p.x - frame.originX) / cell)
    const row = Math.floor((frame.originY - p.y) / cell)
    if (col < 0 || col >= nx || row < 0 || row >= ny) continue
    const key = `${col},${row}`
    counts.set(key, (counts.get(key) ?? 0) + 1)
  }
  if (counts.size === 0) return null

  const cells: TacticalCell[] = []
  const values: number[] = []
  for (const [key, value] of counts) {
    const [col, row] = key.split(',').map(Number)
    cells.push({ col, row, value })
    values.push(value)
  }
  values.sort((a, b) => a - b)
  const lo = quantile(values, Q_LOW)
  const hi = quantile(values, Q_HIGH)
  const scale = hi > lo ? { lo, hi } : { lo: 0, hi: values[values.length - 1] }
  return buildTacticalGrid(
    cells,
    { cell, nx, ny, minX: frame.originX, minY: frame.originY - ny * cell },
    scale,
    cells.length,
  )
}

/** hasTeamSplit : au moins une position porte un camp attribué (0/1). */
export function hasTeamSplit(positions: readonly MatchPlayerPosition[]): boolean {
  return positions.some((p) => p.team !== TEAM_UNKNOWN)
}

/**
 * coveredShare — la part des positions que le fond de carte contient réellement.
 *
 * C'est la note de couverture du bloc : une carte dont un tiers des points tombe hors cadre
 * ne montre pas le match entier, et le lecteur doit le savoir plutôt que le deviner.
 */
export function coveredShare(
  positions: readonly MatchPlayerPosition[],
  frame: MapFrame,
): number {
  if (positions.length === 0) return 0
  let inside = 0
  for (const p of positions) {
    const dx = p.x - frame.originX
    const dy = frame.originY - p.y
    if (dx >= 0 && dx < frame.widthM && dy >= 0 && dy < frame.heightM) inside++
  }
  return inside / positions.length
}
