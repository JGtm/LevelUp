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
  OBJECTIVE_FEED_STATS,
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
