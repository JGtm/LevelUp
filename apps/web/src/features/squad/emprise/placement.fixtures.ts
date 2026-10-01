/**
 * placement.fixtures.ts — le bloc `squad_emprise.placement` d'une soirée à trois joueurs (JGtm,
 * Chocoboflor, Madina97294) au format du contrat (lot V3 du plan PLAN_EMPRISE_VIES_2026-09-28) :
 * des vies dans les quatre quarts, une vie au-delà des plafonds d'affichage (X > 2, frags > 5),
 * une vie non mesurée. Données figées (aucune lecture de base) ; les parts et les médianes sont
 * écrites comme le Go les publie (seuils du bloc : isolé à 1,0 portée, rentable à 1 frag).
 */
import type {
  SquadEmprisePlacement,
  SquadEmprisePlacementLife,
  SquadEmprisePlacementPlayer,
  SquadEmprisePlacementQuadrant,
} from '@/lib/api/types'

import { XUID } from './emprise.fixtures'

/** [portées de radar, frags, durée en s, part hors radar] : une vie mesurée. */
type LifeSpec = readonly [number, number, number, number]

const QUADRANTS: readonly SquadEmprisePlacementQuadrant[] = [
  'in_range_productive',
  'isolated_productive',
  'in_range_costly',
  'isolated_costly',
]

function quadrantOf(ratio: number, kills: number): SquadEmprisePlacementQuadrant {
  const isolated = ratio >= 1
  const productive = kills >= 1
  if (isolated) return productive ? 'isolated_productive' : 'isolated_costly'
  return productive ? 'in_range_productive' : 'in_range_costly'
}

function median(values: number[]): number {
  const v = [...values].sort((a, b) => a - b)
  const mid = v.length / 2
  return v.length % 2 ? v[(v.length - 1) / 2] : (v[mid - 1] + v[mid]) / 2
}

function player(xuid: string, gamertag: string, specs: readonly LifeSpec[], unmeasured = 0): SquadEmprisePlacementPlayer {
  const lives: SquadEmprisePlacementLife[] = specs.map(([ratio, kills, seconds, out], i) => ({
    match_id: `match-${i % 3}`,
    start_ms: 1000 + i * 90_000,
    duration_ms: seconds * 1000,
    radar_ratio: ratio,
    out_of_radar_share: out,
    kills,
    quadrant: quadrantOf(ratio, kills),
  }))
  const n = lives.length
  return {
    xuid,
    gamertag,
    lives_total: n + unmeasured,
    lives_measured: n,
    median_radar_ratio: n > 0 ? median(lives.map((l) => l.radar_ratio)) : undefined,
    median_kills: n > 0 ? median(lives.map((l) => l.kills)) : undefined,
    quadrants: QUADRANTS.map((q) => {
      const count = lives.filter((l) => l.quadrant === q).length
      return { quadrant: q, lives: count, ...(n > 0 ? { share: count / n } : {}) }
    }),
    lives,
  }
}

export const PLACEMENT_2209: SquadEmprisePlacement = {
  isolated_from_ratio: 1,
  productive_from_kills: 1,
  players: [
    player(
      XUID.jgtm,
      'JGtm',
      [
        [0.3, 2, 62, 0],
        [0.62, 1, 41, 0],
        [0.9, 0, 18, 0.1],
        [1.2, 3, 75, 0.6],
        [1.6, 0, 12, 0.9],
        // Au-delà des plafonds d'affichage : posée à 2 portées et à 5 frags, la vraie valeur reste en infobulle.
        [2.8, 7, 150, 1],
      ],
      1,
    ),
    player(XUID.choco, 'Chocoboflor', [
      [0.4, 0, 30, 0],
      [0.5, 1, 55, 0.05],
      [1.1, 0, 20, 0.7],
      [1.4, 2, 48, 0.5],
    ]),
    player(XUID.madina, 'Madina97294', [
      [0.2, 1, 90, 0],
      [0.8, 3, 33, 0.2],
      [1.9, 0, 9, 1],
      [0.7, 0, 25, 0],
    ]),
  ],
  coverage: {
    matches_total: 7,
    matches_with_placement: 7,
    matches_without_range: 1,
    stale_lives: 0,
    lives_total: 15,
    lives_measured: 14,
    lives_unmeasured: 1,
    measured_ms: 2_400_000,
    carrier_ms: 12_000,
    team_down_ms: 30_000,
    unplaced_ms: 90_000,
    teammate_unplaced_ms: 45_000,
  },
}
