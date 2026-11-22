/**
 * Tests — carrierKillFold : « a éliminé un porteur de drapeau » et les autres éliminations liées à l'objectif se
 * replient en badge sur le frag.
 *
 * Ce qu'ils protègent : l'appariement contraint (MÊME tueur, instant le plus proche dans la
 * tolérance, un frag ne sert qu'une fois), le repli intégral quand rien ne s'apparie (on ne perd
 * jamais une information), et la pureté (le fil reçu n'est jamais modifié).
 */
import { describe, expect, it } from 'vitest'

import { CARRIER_KILL_FRAG_TOL_MS, foldCarrierKills } from './carrierKillFold'
import { eventFeedEntry } from './feedEventLines'
import type { ObjectiveFeedStat } from './objectiveFeedLogic'
import type { ReplayFeedEntry, ReplayKill } from './killFeedLogic'

function frag(key: string, xuid: string, replayMs: number): ReplayFeedEntry {
  return {
    key,
    replayMs,
    kill: { xuid, victimXuid: 'v', victimGamertag: 'V', replayMs, medals: [] } as unknown as ReplayKill,
    medal: null,
    death: null,
  }
}

function objective(xuid: string, replayMs: number, stat: ObjectiveFeedStat): ReplayFeedEntry {
  return eventFeedEntry(`o-${xuid}-${replayMs}-${stat}`, replayMs, { family: 'objectives', xuid, stat })
}

const carrierKilled = (xuid: string, replayMs: number) => objective(xuid, replayMs, 'flag_carriers_killed')

describe('foldCarrierKills — l’appariement', () => {
  it('replie l’action sur le frag du même tueur : la ligne disparaît, le frag porte le badge', () => {
    const feed = [frag('k1', 'a', 10_000), frag('k2', 'b', 10_000)]
    const out = foldCarrierKills(feed, [carrierKilled('a', 10_030)])
    expect(out.events).toEqual([])
    expect(out.feed.map((e) => e.flagCarrierKill)).toEqual([true, undefined])
  })

  it('ne touche pas au fil reçu : le frag badgé est une copie', () => {
    const feed = [frag('k1', 'a', 10_000)]
    const out = foldCarrierKills(feed, [carrierKilled('a', 10_000)])
    expect(feed[0].flagCarrierKill).toBeUndefined()
    expect(out.feed[0]).not.toBe(feed[0])
    expect(out.feed[0].kill).toBe(feed[0].kill)
  })

  it('un autre tueur ne s’apparie pas : la ligne d’objectif reste', () => {
    const action = carrierKilled('a', 10_000)
    const out = foldCarrierKills([frag('k', 'b', 10_000)], [action])
    expect(out.events).toEqual([action])
    expect(out.feed[0].flagCarrierKill).toBeUndefined()
  })

  it('tient dans la tolérance, pas au-delà : la ligne reste quand le frag est trop loin', () => {
    const inside = foldCarrierKills([frag('k', 'a', 10_000)], [carrierKilled('a', 10_000 + CARRIER_KILL_FRAG_TOL_MS)])
    expect(inside.events).toEqual([])
    const action = carrierKilled('a', 10_000 + CARRIER_KILL_FRAG_TOL_MS + 1)
    const outside = foldCarrierKills([frag('k', 'a', 10_000)], [action])
    expect(outside.events).toEqual([action])
    expect(outside.feed[0].flagCarrierKill).toBeUndefined()
  })

  it('prend le frag le plus proche d’un tueur qui en a deux dans la tolérance', () => {
    const feed = [frag('far', 'a', 9_700), frag('near', 'a', 10_050)]
    const out = foldCarrierKills(feed, [carrierKilled('a', 10_000)])
    expect(out.feed.map((e) => e.flagCarrierKill)).toEqual([undefined, true])
  })

  it('un frag ne sert qu’une fois : la seconde action garde sa ligne', () => {
    const second = carrierKilled('a', 10_020)
    const out = foldCarrierKills([frag('k', 'a', 10_000)], [carrierKilled('a', 10_000), second])
    expect(out.events).toEqual([second])
    expect(out.feed.filter((e) => e.flagCarrierKill)).toHaveLength(1)
  })

  it('deux actions du même tueur se répartissent sur ses deux frags', () => {
    const feed = [frag('k1', 'a', 10_000), frag('k2', 'a', 10_300)]
    const out = foldCarrierKills(feed, [carrierKilled('a', 10_000), carrierKilled('a', 10_300)])
    expect(out.events).toEqual([])
    expect(out.feed.map((e) => e.flagCarrierKill)).toEqual([true, true])
  })

  it('une action sans xuid garde sa ligne : aucun frag ne lui est supposé', () => {
    const action = carrierKilled('', 10_000)
    const out = foldCarrierKills([frag('k', '', 10_000)], [action])
    expect(out.events).toEqual([action])
  })

  it('ne replie que cette statistique : les autres lignes d’objectif passent telles quelles', () => {
    const grab = objective('a', 10_000, 'flag_grabs')
    const out = foldCarrierKills([frag('k', 'a', 10_000)], [grab])
    expect(out.events).toEqual([grab])
    expect(out.feed[0].flagCarrierKill).toBeUndefined()
  })
})

describe('foldCarrierKills — les éliminations liées à l’objectif', () => {
  const KILL_STATS: ObjectiveFeedStat[] = [
    'flag_returners_killed', 'flag_kills_as_returner', 'flag_kills_as_carrier', 'zone_offensive_kills',
    'zone_defensive_kills', 'vip_kills', 'vip_kills_as_vip',
  ]

  it.each(KILL_STATS)('%s : la ligne disparaît et le frag du même tueur la porte en infobulle', (stat) => {
    const out = foldCarrierKills([frag('k1', 'a', 10_000), frag('k2', 'b', 10_000)], [objective('a', 10_030, stat)])
    expect(out.events).toEqual([])
    expect(out.feed[0].objectiveKills).toEqual([stat])
    expect(out.feed[0].flagCarrierKill).toBeUndefined()
    expect(out.feed[1].objectiveKills).toBeUndefined()
  })

  it('une élimination sans frag apparié garde sa ligne : on ne perd aucune information', () => {
    const action = objective('a', 10_000, 'zone_offensive_kills')
    const out = foldCarrierKills([frag('k', 'b', 10_000)], [action])
    expect(out.events).toEqual([action])
    expect(out.feed[0].objectiveKills).toBeUndefined()
  })

  it('l’assistance au VIP n’est pas une élimination : elle garde sa ligne', () => {
    const action = objective('a', 10_000, 'vip_assists')
    const out = foldCarrierKills([frag('k', 'a', 10_000)], [action])
    expect(out.events).toEqual([action])
    expect(out.feed[0].objectiveKills).toBeUndefined()
  })

  it('deux éliminations de natures différentes se replient sur le même frag, jamais deux de la même nature', () => {
    const feed = [frag('k', 'a', 10_000)]
    const deux = foldCarrierKills(feed, [objective('a', 10_000, 'flag_returners_killed'), objective('a', 10_010, 'flag_kills_as_carrier')])
    expect(deux.events).toEqual([])
    expect(deux.feed[0].objectiveKills).toEqual(['flag_returners_killed', 'flag_kills_as_carrier'])
    const second = objective('a', 10_020, 'flag_returners_killed')
    const memeNature = foldCarrierKills(feed, [objective('a', 10_000, 'flag_returners_killed'), second])
    expect(memeNature.events).toEqual([second])
  })

  it('un porteur éliminé garde son badge propre, et les autres natures vont en infobulle', () => {
    const out = foldCarrierKills([frag('k', 'a', 10_000)], [carrierKilled('a', 10_000), objective('a', 10_000, 'flag_kills_as_returner')])
    expect(out.feed[0].flagCarrierKill).toBe(true)
    expect(out.feed[0].objectiveKills).toEqual(['flag_kills_as_returner'])
  })
})
