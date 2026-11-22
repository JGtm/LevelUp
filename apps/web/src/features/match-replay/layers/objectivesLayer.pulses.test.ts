/**
 * Tests — objectivesLayer : LA LISTE BLANCHE DES ACTIONS QUI OUVRENT UN ANNEAU.
 *
 * Une élimination liée à l'objectif (défenseur ou attaquant de base, VIP adverse, joueur qui rend ou porte le drapeau)
 * porte le préfixe de sa famille mais ne se pose pas sur la zone : sans liste blanche, chaque élimination de base
 * ouvrirait un anneau de capture sur l'élément le plus proche de son auteur.
 */
import { describe, expect, it } from 'vitest'

import { buildObjectivePulses, normalizeMapObjectives } from './objectivesLayer'
import { testReplayDoc } from '../test/testDoc'

const elements = normalizeMapObjectives({
  markers: [{ role: 'flag_spawn', team: 0, x: 1, y: 9, z: 1 }],
})

const docAvec = (names: string[]) =>
  testReplayDoc({
    frameIntervalMs: 100,
    tracks: [
      { slot: 1, team: -1, xuid: 'A', points: [{ t: 0, x: 1, y: 9 }, { t: 100, x: 1, y: 9 }], startFrame: 0, endFrame: 100 },
    ],
    objectives: names.map((stat, i) => ({ t: 10 + i, xuid: 'A', stat, timeMs: 1_000 + i })),
  })

describe("buildObjectivePulses — liste blanche des actions sur l'objectif", () => {
  it("une élimination liée à l'objectif, une assistance, une désignation de VIP ou un nom inconnu n'ouvrent aucun anneau", () => {
    const names = [
      'zone_offensive_kills', 'zone_defensive_kills', 'flag_returners_killed', 'flag_kills_as_returner',
      'flag_kills_as_carrier', 'flag_carriers_killed', 'flag_capture_assists', 'vip_kills', 'vip_assists',
      'vip_kills_as_vip', 'vip_selected', 'kills', 'assists', 'zone_futur_inconnu',
    ]
    expect(buildObjectivePulses(docAvec(names), elements)).toEqual([])
  })

  it("les captures, sécurisations, prises, retours, vols et détonations ouvrent chacun un anneau", () => {
    const names = [
      'flag_grabs', 'flag_captures', 'flag_secures', 'flag_returns', 'flag_steals',
      'zone_captures', 'zone_secures', 'bomb_detonations',
    ]
    expect(buildObjectivePulses(docAvec(names), elements)).toHaveLength(names.length)
  })
})
