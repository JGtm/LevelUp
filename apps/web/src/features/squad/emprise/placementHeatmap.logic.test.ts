/**
 * placementHeatmap.logic.test.ts — la grille de la carte de chaleur « Distance au coéquipier le
 * plus proche et frags, par vie » : cases de 0,25, dernière colonne qui absorbe le plafond, ligne
 * « 5 et plus », part sur toutes les vies de l'escouade, intensité en racine carrée de la part sur
 * la part maximale, seuil d'étiquette à 1,5 %, gros point d'un joueur.
 */
import { describe, expect, it } from 'vitest'

import type { SquadEmprisePlacementLife, SquadEmprisePlacementPlayer } from '@/lib/api/types'

import {
  binLives,
  CELL_INTENSITY_MIN,
  cellShowsLabel,
  columnOf,
  playerSpot,
  rowOf,
  X_BINS,
  Y_ROWS,
} from './placementHeatmap.logic'

const life = (radar: number, kills: number): SquadEmprisePlacementLife => ({
  match_id: 'm',
  start_ms: 0,
  duration_ms: 1000,
  radar_ratio: radar,
  out_of_radar_share: 0,
  kills,
  quadrant: 'in_range_costly',
})

const player = (lives: SquadEmprisePlacementLife[], extra: Partial<SquadEmprisePlacementPlayer> = {}): SquadEmprisePlacementPlayer => ({
  xuid: 'x',
  gamertag: 'G',
  lives_total: lives.length,
  lives_measured: lives.length,
  lives,
  quadrants: [],
  ...extra,
})

describe('la grille', () => {
  it('huit colonnes de 0,25 et six lignes', () => {
    expect(X_BINS).toBe(8)
    expect(Y_ROWS).toBe(6)
  })

  it('colonne : cases de 0,25, bord inférieur inclus, la dernière absorbe tout ce qui dépasse 1,75', () => {
    expect(columnOf(0)).toBe(0)
    expect(columnOf(0.2499)).toBe(0)
    expect(columnOf(0.25)).toBe(1)
    expect(columnOf(1.74)).toBe(6)
    expect(columnOf(1.75)).toBe(7)
    expect(columnOf(2)).toBe(7)
    expect(columnOf(9.3)).toBe(7)
    expect(columnOf(-0.1)).toBe(0)
  })

  it('ligne : les frags, la dernière ligne vaut 5 et plus', () => {
    expect([0, 1, 2, 3, 4, 5, 6, 12].map(rowOf)).toEqual([0, 1, 2, 3, 4, 5, 5, 5])
  })
})

describe('binLives', () => {
  it('sans vie : aucune case, total nul', () => {
    expect(binLives([])).toEqual({ cells: [], total: 0 })
    expect(binLives([player([])])).toEqual({ cells: [], total: 0 })
  })

  it('la part d’une case est rapportée aux vies de TOUS les joueurs du bloc', () => {
    const a = player([life(0.1, 0), life(0.1, 0), life(1.9, 5)])
    const b = player([life(0.2, 0), life(1.0, 2)], { xuid: 'y', gamertag: 'H' })
    const grid = binLives([a, b])
    expect(grid.total).toBe(5)
    const first = grid.cells.find((c) => c.col === 0 && c.row === 0)!
    expect(first.lives).toBe(3)
    expect(first.share).toBeCloseTo(3 / 5, 10)
    expect(first.x0).toBe(0)
    expect(first.x1).toBe(0.25)
    expect(grid.cells.reduce((s, c) => s + c.share, 0)).toBeCloseTo(1, 10)
  })

  it('bornes ouvertes : la dernière colonne et la dernière ligne', () => {
    const grid = binLives([player([life(2.8, 7), life(0.3, 1)])])
    const far = grid.cells.find((c) => c.col === 7 && c.row === 5)!
    expect(far.openRight).toBe(true)
    expect(far.openTop).toBe(true)
    const near = grid.cells.find((c) => c.col === 1 && c.row === 1)!
    expect(near.openRight).toBe(false)
    expect(near.openTop).toBe(false)
  })

  it('intensité : racine carrée de la part sur la part maximale, plancher pour la plus faible, 1 pour la plus forte', () => {
    const lives = [...Array(4).fill(0).map(() => life(0.1, 0)), life(1.9, 5)]
    const grid = binLives([player(lives)])
    const strong = grid.cells.find((c) => c.col === 0)!
    const weak = grid.cells.find((c) => c.col === 7)!
    expect(strong.intensity).toBe(1)
    expect(weak.intensity).toBeCloseTo(CELL_INTENSITY_MIN + (1 - CELL_INTENSITY_MIN) * Math.sqrt(1 / 4), 10)
    expect(weak.intensity).toBeGreaterThan(CELL_INTENSITY_MIN)
  })

  it('étiquette : à partir de 1,5 % des vies de l’escouade', () => {
    const many = Array.from({ length: 100 }, () => life(0.1, 0))
    const grid = binLives([player([...many, life(1.9, 5), life(1.9, 5)])]) // 2 / 102 = 1,96 %
    expect(cellShowsLabel(grid.cells.find((c) => c.col === 7)!)).toBe(true)
    const grid2 = binLives([player([...many, life(1.9, 5)])]) // 1 / 101 = 0,99 %
    expect(cellShowsLabel(grid2.cells.find((c) => c.col === 7)!)).toBe(false)
  })
})

describe('playerSpot', () => {
  it('distance médiane du contrat, moyenne des frags des vies, plafonnées à 2 et à 5', () => {
    const p = player([life(0.4, 0), life(0.5, 3)], { median_radar_ratio: 0.45, median_kills: 1.5 })
    expect(playerSpot(p)).toEqual({ x: 0.45, y: 1.5, meanKills: 1.5 })
    const far = player([life(2.5, 8), life(2.5, 6)], { median_radar_ratio: 2.5 })
    expect(playerSpot(far)).toEqual({ x: 2, y: 5, meanKills: 7 })
  })

  it('sans liste de vies : la médiane de frags du contrat ; sans distance médiane : aucun point', () => {
    expect(playerSpot(player([], { median_radar_ratio: 0.6, median_kills: 2 }))).toEqual({ x: 0.6, y: 2, meanKills: 2 })
    expect(playerSpot(player([life(0.4, 1)]))).toBeNull()
    expect(playerSpot(player([], { median_radar_ratio: 0.6 }))).toBeNull()
  })
})
