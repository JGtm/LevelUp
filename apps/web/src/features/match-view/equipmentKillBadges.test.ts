/**
 * Tests — equipmentKillBadges (le badge « temps fort », LOT F.3).
 *
 * CE QU'ILS PROTÈGENT :
 *   - le SEUIL écrit d'avance (2 -> rien, 3 -> badge) ;
 *   - le MEILLEUR épisode gagne, jamais une somme sur le joueur ;
 *   - un badge par famille AU PLUS ;
 *   - un épisode sans propriétaire mesuré n'ouvre aucun badge — jamais un nom deviné.
 */
import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'
import { testReplayDoc } from '@/features/match-replay/test/testDoc'

import { computeEquipmentKillBadges, EQUIPMENT_KILL_BADGE_THRESHOLD } from './equipmentKillBadges'

/** Une vie : le slot, son propriétaire, et deux points pour que la fenêtre existe. */
function vie(slot: number, xuid: string) {
  return {
    slot,
    xuid,
    team: -1,
    startFrame: 0,
    endFrame: 100,
    points: [
      { t: 0, x: 0, y: 0 },
      { t: 100, x: 1, y: 1 },
    ],
  }
}

const SB: MatchScoreboardRow[] = [
  { xuid: 'a1', gamertag: 'Alpha', team_side: 't0' },
  { xuid: 'a2', gamertag: 'Bravo', team_side: 't0' },
] as MatchScoreboardRow[]

function temoin(over: Partial<ReplayDocument> = {}) {
  return testReplayDoc({
    frameCount: 200,
    frameIntervalMs: 100,
    roster: [
      { filmIndex: 0, xuid: 'a1', name: 'Alpha' },
      { filmIndex: 1, xuid: 'a2', name: 'Bravo' },
    ],
    tracks: [vie(1, 'a1'), vie(2, 'a2')],
    ...over,
  } as Partial<ReplayDocument>)
}

describe('computeEquipmentKillBadges — le seuil écrit d’avance', () => {
  it(`${EQUIPMENT_KILL_BADGE_THRESHOLD - 1} frags dans un épisode : aucun badge`, () => {
    const doc = temoin({
      equipmentEpisodes: [
        { slot: 1, fam: 'camo', t0: 10, t1: 60, k: EQUIPMENT_KILL_BADGE_THRESHOLD - 1 },
      ],
    } as Partial<ReplayDocument>)
    expect(computeEquipmentKillBadges(doc, SB)).toEqual([])
  })

  it(`${EQUIPMENT_KILL_BADGE_THRESHOLD} frags dans un épisode : un badge, le compte RÉEL`, () => {
    const doc = temoin({
      equipmentEpisodes: [
        { slot: 1, fam: 'camo', t0: 10, t1: 60, k: EQUIPMENT_KILL_BADGE_THRESHOLD },
      ],
    } as Partial<ReplayDocument>)
    const badges = computeEquipmentKillBadges(doc, SB)
    expect(badges).toEqual([
      { family: 'camo', kills: EQUIPMENT_KILL_BADGE_THRESHOLD, xuid: 'a1', playerName: 'Alpha' },
    ])
  })

  it('un épisode sans `k` (omitempty) ne franchit jamais le seuil', () => {
    const doc = temoin({
      equipmentEpisodes: [{ slot: 1, fam: 'camo', t0: 10, t1: 60 }],
    } as Partial<ReplayDocument>)
    expect(computeEquipmentKillBadges(doc, SB)).toEqual([])
  })
})

describe('computeEquipmentKillBadges — le meilleur épisode gagne', () => {
  it('deux épisodes de la même famille, même joueur : seul le meilleur ouvre le badge', () => {
    const doc = temoin({
      equipmentEpisodes: [
        { slot: 1, fam: 'camo', t0: 10, t1: 20, k: 3 },
        { slot: 1, fam: 'camo', t0: 30, t1: 90, k: 7 },
      ],
    } as Partial<ReplayDocument>)
    expect(computeEquipmentKillBadges(doc, SB)).toEqual([
      { family: 'camo', kills: 7, xuid: 'a1', playerName: 'Alpha' },
    ])
  })

  it('deux joueurs qualifiés sur la même famille : un seul badge, le plus fort gagne', () => {
    const doc = temoin({
      equipmentEpisodes: [
        { slot: 1, fam: 'camo', t0: 10, t1: 20, k: 3 },
        { slot: 2, fam: 'camo', t0: 10, t1: 20, k: 5 },
      ],
    } as Partial<ReplayDocument>)
    expect(computeEquipmentKillBadges(doc, SB)).toEqual([
      { family: 'camo', kills: 5, xuid: 'a2', playerName: 'Bravo' },
    ])
  })

  it('une somme SANS un seul épisode qualifié n’ouvre aucun badge', () => {
    // Deux épisodes de 2 frags chacun (total 4 sur le joueur) : ni l'un ni l'autre ne
    // franchit le seuil PAR ÉPISODE. Le badge ne regarde jamais la somme du joueur.
    const doc = temoin({
      equipmentEpisodes: [
        { slot: 1, fam: 'camo', t0: 10, t1: 20, k: 2 },
        { slot: 1, fam: 'camo', t0: 30, t1: 40, k: 2 },
      ],
    } as Partial<ReplayDocument>)
    expect(computeEquipmentKillBadges(doc, SB)).toEqual([])
  })
})

describe('computeEquipmentKillBadges — un badge par famille au plus', () => {
  it('camo ET surbouclier qualifiés : deux badges distincts', () => {
    const doc = temoin({
      equipmentEpisodes: [
        { slot: 1, fam: 'camo', t0: 10, t1: 20, k: 3 },
        { slot: 2, fam: 'overshield', t0: 10, t1: 20, k: 4 },
      ],
    } as Partial<ReplayDocument>)
    const badges = computeEquipmentKillBadges(doc, SB)
    expect(badges).toHaveLength(2)
    expect(badges.find((b) => b.family === 'camo')?.kills).toBe(3)
    expect(badges.find((b) => b.family === 'overshield')?.kills).toBe(4)
  })

  it('une famille hors des deux mesurées n’ouvre jamais de badge', () => {
    const doc = temoin({
      equipmentEpisodes: [{ slot: 1, fam: 'inconnue', t0: 10, t1: 20, k: 99 }],
    } as Partial<ReplayDocument>)
    expect(computeEquipmentKillBadges(doc, SB)).toEqual([])
  })
})

describe('computeEquipmentKillBadges — sans propriétaire mesuré, pas de badge', () => {
  it('un épisode sur un slot que le pont ne nomme à aucun joueur reste sans badge', () => {
    const doc = temoin({
      equipmentEpisodes: [{ slot: 9, fam: 'camo', t0: 10, t1: 20, k: 5 }],
    } as Partial<ReplayDocument>)
    expect(computeEquipmentKillBadges(doc, SB)).toEqual([])
  })
})
