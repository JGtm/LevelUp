/**
 * Tests — objectiveFeedLogic : les lignes d'objectif du fil, tirées de `doc.objectives[]`.
 *
 * Ce qu'ils protègent : la liste fermée des actions (celle de `named.go`, tenue par un
 * garde-rail qui relit le Go), l'exclusion des frags et assistances, la garde d'horloge
 * (sans origine sûre, ni ligne ni entrée de menu), et le refus d'une action sans libellé
 * connu : écartée et COMPTÉE, jamais affichée sous un nom de repli.
 */
import { describe, expect, it } from 'vitest'

import { testReplayDoc } from '../test/testDoc'
import {
  CAPTURE_ASSIST_TOL_MS,
  OBJECTIVE_FEED_STATS,
  foldCaptureAssists,
  objectiveFamilyOfFeedStat,
  objectiveFeedEntries,
} from './objectiveFeedLogic'

const FRAME_MS = 100

function docWith(objectives: unknown[], over: Record<string, unknown> = {}) {
  return testReplayDoc({
    frameIntervalMs: FRAME_MS,
    objectives: objectives as never,
    ...over,
  } as never)
}

function action(stat: string, t = 50, xuid = 'x1') {
  return { stat, t, xuid, timeMs: t * FRAME_MS }
}

describe('objectiveFeedEntries — liste fermée', () => {
  it('une ligne par action nommée, à son instant sur l’axe du rejeu, triée', () => {
    const { entries, unlabelled } = objectiveFeedEntries(
      docWith([action('flag_returns', 80), action('flag_captures', 50)]),
    )
    expect(unlabelled).toBe(0)
    expect(entries.map((e) => e.replayMs)).toEqual([5_000, 8_000])
    expect(entries.map((e) => e.event)).toEqual([
      { family: 'objectives', xuid: 'x1', stat: 'flag_captures' },
      { family: 'objectives', xuid: 'x1', stat: 'flag_returns' },
    ])
    expect(entries.every((e) => e.kill === null && e.medal === null && e.death === null)).toBe(true)
  })

  it('couvre chaque action de la liste fermée, sans en écarter aucune', () => {
    const all = OBJECTIVE_FEED_STATS.map((s, i) => action(s, 10 + i))
    const { entries, unlabelled } = objectiveFeedEntries(docWith(all))
    expect(entries).toHaveLength(OBJECTIVE_FEED_STATS.length)
    expect(unlabelled).toBe(0)
  })

  it('exclut les frags et les assistances, publiés dans le même calque', () => {
    const { entries, unlabelled } = objectiveFeedEntries(
      docWith([action('kills'), action('assists'), action('flag_grabs')]),
    )
    expect(entries.map((e) => e.event)).toEqual([
      { family: 'objectives', xuid: 'x1', stat: 'flag_grabs' },
    ])
    expect(unlabelled).toBe(0)
  })

  it('une action d’une famille d’objectif sans libellé connu est écartée ET comptée', () => {
    const { entries, unlabelled } = objectiveFeedEntries(
      docWith([action('flag_teleports'), action('zone_unknown'), action('zone_captures')]),
    )
    expect(entries).toHaveLength(1)
    expect(unlabelled).toBe(2)
  })

  it('une statistique hors de toute famille d’objectif n’est ni affichée ni comptée', () => {
    const { entries, unlabelled } = objectiveFeedEntries(docWith([action('score_total')]))
    expect(entries).toEqual([])
    expect(unlabelled).toBe(0)
  })
})

describe('objectiveFeedEntries — garde d’horloge', () => {
  it('sans horloge sûre : aucune ligne', () => {
    const doc = docWith([action('flag_captures')], { coverage: { originResolved: false } })
    expect(objectiveFeedEntries(doc).entries).toEqual([])
  })

  it('une origine publiée rend l’horloge sûre même si le drapeau de couverture est faux', () => {
    const doc = docWith([action('flag_captures')], {
      coverage: { originResolved: false },
      originMs: 1_200,
    })
    expect(objectiveFeedEntries(doc).entries).toHaveLength(1)
  })

  it('un document sans objectif rend une liste vide', () => {
    expect(objectiveFeedEntries(docWith([])).entries).toEqual([])
  })
})

describe('objectiveFamilyOfFeedStat', () => {
  it('rend la famille de chaque action de la liste fermée', () => {
    expect(objectiveFamilyOfFeedStat('flag_secures')).toBe('flag')
    expect(objectiveFamilyOfFeedStat('zone_captures')).toBe('zone')
    expect(objectiveFamilyOfFeedStat('vip_selected')).toBe('vip')
    expect(objectiveFamilyOfFeedStat('bomb_detonations')).toBe('bomb')
  })
})

const ROSTER = [
  { xuid: 'a1', filmIndex: 0, name: 'A1', team: 0 },
  { xuid: 'a2', filmIndex: 1, name: 'A2', team: 0 },
  { xuid: 'b1', filmIndex: 2, name: 'B1', team: 1 },
]

/** Le champ `assistXuids` d'une ligne d'objectif, ou `undefined`. */
const creditedOn = (e: { event?: unknown }) => (e.event as { assistXuids?: string[] }).assistXuids

describe('objectiveFeedEntries — les assistances de capture se replient sur la capture', () => {
  it('les joueurs crédités rejoignent la capture de leur équipe, la ligne d’assistance disparaît', () => {
    const doc = docWith(
      [action('flag_captures', 50, 'a1'), action('flag_capture_assists', 50, 'a2'), action('flag_grabs', 40, 'b1')],
      { roster: ROSTER },
    )
    const { entries } = objectiveFeedEntries(doc)
    expect(entries.map((e) => e.event)).toEqual([
      { family: 'objectives', xuid: 'b1', stat: 'flag_grabs' },
      { family: 'objectives', xuid: 'a1', stat: 'flag_captures', assistXuids: ['a2'] },
    ])
  })

  it('une assistance du camp adverse au même instant ne s’attache pas à la capture : elle garde sa ligne', () => {
    const doc = docWith(
      [action('flag_captures', 50, 'a1'), action('flag_capture_assists', 50, 'b1')],
      { roster: ROSTER },
    )
    const { entries } = objectiveFeedEntries(doc)
    expect(entries.map((e) => (e.event as { stat: string }).stat).sort()).toEqual([
      'flag_capture_assists',
      'flag_captures',
    ])
    expect(entries.some((e) => creditedOn(e))).toBe(false)
  })

  it('une assistance d’un joueur absent du roster garde sa ligne', () => {
    const doc = docWith([action('flag_captures', 50, 'a1'), action('flag_capture_assists', 50, 'ghost')], {
      roster: ROSTER,
    })
    expect(objectiveFeedEntries(doc).entries).toHaveLength(2)
  })
})

describe('foldCaptureAssists — l’appariement', () => {
  const teamOf = (xuid: string) => ({ a1: 0, a2: 0, a3: 0, b1: 1 })[xuid as 'a1'] ?? null
  const line = (key: string, ms: number, stat: string, xuid: string) =>
    ({
      key,
      replayMs: ms,
      kill: null,
      medal: null,
      death: null,
      event: { family: 'objectives', xuid, stat },
    }) as never

  it('prend la capture de la même équipe la plus proche, dans la tolérance', () => {
    const out = foldCaptureAssists(
      [
        line('c1', 10_000, 'flag_captures', 'a1'),
        line('c2', 30_000, 'flag_captures', 'a1'),
        line('as', 29_900, 'flag_capture_assists', 'a2'),
      ],
      teamOf,
    )
    expect(out.map((e) => e.key)).toEqual(['c1', 'c2'])
    expect(creditedOn(out[1])).toEqual(['a2'])
    expect(creditedOn(out[0])).toBeUndefined()
  })

  it('tient dans la tolérance, pas au-delà', () => {
    const near = foldCaptureAssists(
      [
        line('c', 10_000, 'flag_captures', 'a1'),
        line('as', 10_000 + CAPTURE_ASSIST_TOL_MS, 'flag_capture_assists', 'a2'),
      ],
      teamOf,
    )
    expect(near.map((e) => e.key)).toEqual(['c'])
    const far = foldCaptureAssists(
      [
        line('c', 10_000, 'flag_captures', 'a1'),
        line('as', 10_000 + CAPTURE_ASSIST_TOL_MS + 1, 'flag_capture_assists', 'a2'),
      ],
      teamOf,
    )
    expect(far.map((e) => e.key)).toEqual(['c', 'as'])
  })

  it('plusieurs crédités sur une capture, sans doublon', () => {
    const out = foldCaptureAssists(
      [
        line('c', 10_000, 'flag_captures', 'a1'),
        line('as1', 10_000, 'flag_capture_assists', 'a2'),
        line('as2', 10_000, 'flag_capture_assists', 'a3'),
        line('as3', 10_000, 'flag_capture_assists', 'a2'),
      ],
      teamOf,
    )
    expect(out).toHaveLength(1)
    expect(creditedOn(out[0])).toEqual(['a2', 'a3'])
  })

  it('ne modifie pas la liste reçue', () => {
    const input = [line('c', 10_000, 'flag_captures', 'a1'), line('as', 10_000, 'flag_capture_assists', 'a2')]
    foldCaptureAssists(input, teamOf)
    expect(input).toHaveLength(2)
    expect(creditedOn(input[0])).toBeUndefined()
  })
})
