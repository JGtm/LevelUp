/**
 * Tests — feedFilterLogic : le filtre du fil ne porte que sur la liste AFFICHÉE.
 *
 * Ce qu'ils protègent : la sémantique des sept filtres, le fait qu'une entrée de menu sans
 * aucune ligne possible n'est pas proposée (Objectifs seul), le repli sur « Tout » sans
 * écraser la préférence, et la pureté (le fil entier n'est jamais modifié).
 */
import { describe, expect, it } from 'vitest'

import type { MedalEvent, ReplayFeedEntry, ReplayKill } from './killFeedLogic'
import {
  DEFAULT_FEED_FILTER,
  FEED_FILTER_IDS,
  availableFeedFilters,
  countByFeedFilter,
  effectiveFeedFilter,
  filterFeedEntries,
  matchesFeedFilter,
} from './feedFilterLogic'

function kill(over: Partial<ReplayKill> = {}): ReplayKill {
  return {
    tMs: 0,
    xuid: 'a',
    ally: true,
    teamID: 0,
    weaponKey: '',
    weaponLabel: '',
    weaponImageUrl: '',
    weaponTinted: false,
    assistState: '',
    assistGamertag: '',
    assistTeamID: null,
    killerDamagePct: null,
    assistDamagePct: null,
    victimXuid: 'b',
    victimGamertag: 'B',
    replayMs: 0,
    medals: [],
    ...over,
  } as ReplayKill
}

const MEDAL = { tMs: 0, xuid: 'a', gamertag: 'A', name: 'm', label: 'M', description: '', imageUrl: '' } as MedalEvent

function entry(key: string, over: Partial<ReplayFeedEntry>): ReplayFeedEntry {
  return { key, replayMs: 0, kill: null, medal: null, death: null, ...over }
}

const KILL = entry('k', { kill: kill() })
const KILL_MEDAL = entry('km', { kill: kill({ medals: [MEDAL] }) })
const KILL_ASSIST = entry('ka', { kill: kill({ assistState: 'named', assistGamertag: 'C' }) })
const MEDAL_ALONE = entry('m', { medal: MEDAL })
const DEATH = entry('d', { death: { replayMs: 0, xuid: 'a', kind: '', img: '', tinted: false } })
const OBJECTIVE = entry('o', { event: { family: 'objectives', xuid: 'a', stat: 'flag_captures' } })
const PICKUP = entry('p', { event: { family: 'pickups', xuid: 'a', name: 'S7 Sniper', nature: 'power' } })
const VEHICLE = entry('v', { event: { family: 'vehicles', xuid: 'a', name: 'Warthog' } })
const ALL = [KILL, KILL_MEDAL, KILL_ASSIST, MEDAL_ALONE, DEATH, OBJECTIVE, PICKUP, VEHICLE]

describe('matchesFeedFilter — sémantique des filtres', () => {
  it('Tout laisse passer chaque forme de ligne', () => {
    expect(ALL.every((e) => matchesFeedFilter(e, 'tout'))).toBe(true)
  })

  it('Éliminations : les seuls frags', () => {
    expect(ALL.filter((e) => matchesFeedFilter(e, 'kills')).map((e) => e.key)).toEqual(['k', 'km', 'ka'])
  })

  it('Médailles : les médailles seules et les frags qui en portent', () => {
    expect(ALL.filter((e) => matchesFeedFilter(e, 'medals')).map((e) => e.key)).toEqual(['km', 'm'])
  })

  it("Assistances : les frags dont l'assistance est nommée", () => {
    expect(ALL.filter((e) => matchesFeedFilter(e, 'assists')).map((e) => e.key)).toEqual(['ka'])
  })

  it('Objectifs, Armes et bonus, Véhicules : leur famille de ligne', () => {
    expect(ALL.filter((e) => matchesFeedFilter(e, 'objectives')).map((e) => e.key)).toEqual(['o'])
    expect(ALL.filter((e) => matchesFeedFilter(e, 'pickups')).map((e) => e.key)).toEqual(['p'])
    expect(ALL.filter((e) => matchesFeedFilter(e, 'vehicles')).map((e) => e.key)).toEqual(['v'])
  })
})

describe('availableFeedFilters — entrées proposées au menu', () => {
  it('toutes les entrées sont là sur un match qui a des objectifs, dans l’ordre de la maquette', () => {
    expect(availableFeedFilters(ALL).map((f) => f.id)).toEqual([
      'tout', 'kills', 'objectives', 'pickups', 'vehicles', 'medals', 'assists',
    ])
    expect([...FEED_FILTER_IDS]).toEqual(availableFeedFilters(ALL).map((f) => f.id))
  })

  it("Objectifs n'apparaît pas sur un match Assassin (aucune ligne d'objectif)", () => {
    const assassin = [KILL, KILL_MEDAL, DEATH, PICKUP]
    const ids = availableFeedFilters(assassin).map((f) => f.id)
    expect(ids).not.toContain('objectives')
    expect(ids).toEqual(['tout', 'kills', 'pickups', 'vehicles', 'medals', 'assists'])
  })

  it('Médailles, Assistances, Armes et bonus, Véhicules restent proposés même sans aucune ligne', () => {
    const ids = availableFeedFilters([KILL]).map((f) => f.id)
    for (const id of ['tout', 'kills', 'pickups', 'vehicles', 'medals', 'assists'] as const) {
      expect(ids).toContain(id)
    }
  })

  it('chaque entrée porte son nombre de lignes dans tout le match', () => {
    const counts = Object.fromEntries(availableFeedFilters(ALL).map((f) => [f.id, f.count]))
    expect(counts).toEqual({
      tout: 8, kills: 3, objectives: 1, pickups: 1, vehicles: 1, medals: 2, assists: 1,
    })
    expect(countByFeedFilter([]).tout).toBe(0)
  })
})

describe('effectiveFeedFilter — repli sans écraser la préférence', () => {
  it('rend le filtre mémorisé quand le menu le propose', () => {
    const available = availableFeedFilters(ALL)
    expect(effectiveFeedFilter('objectives', available)).toBe('objectives')
  })

  it("retombe sur Tout quand Objectifs n'existe pas pour ce match", () => {
    const available = availableFeedFilters([KILL])
    expect(effectiveFeedFilter('objectives', available)).toBe(DEFAULT_FEED_FILTER)
  })

  it('un filtre toujours présent reste appliqué tel quel même sans aucune ligne', () => {
    const available = availableFeedFilters([KILL])
    expect(effectiveFeedFilter('vehicles', available)).toBe('vehicles')
    expect(effectiveFeedFilter('medals', available)).toBe('medals')
  })
})

describe('filterFeedEntries — le fil entier reste entier', () => {
  it('ne modifie pas la liste reçue et rend une liste neuve', () => {
    const feed = Object.freeze([...ALL]) as ReplayFeedEntry[]
    const out = filterFeedEntries(feed, 'kills')
    expect(out).not.toBe(feed)
    expect(feed).toHaveLength(ALL.length)
    expect(out).toHaveLength(3)
  })

  it('Tout rend toutes les lignes', () => {
    expect(filterFeedEntries(ALL, 'tout')).toHaveLength(ALL.length)
  })
})
