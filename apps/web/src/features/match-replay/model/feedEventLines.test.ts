/**
 * Tests — withEventLines : le fil complet plus les lignes d'événement, avec les repliages.
 *
 * Ce qu'ils protègent : la liste est neuve et triée, le fil reçu reste entier, et l'action
 * « porteur éliminé » se replie sur son frag (badge) au lieu de faire une ligne de plus.
 */
import { describe, expect, it } from 'vitest'

import { eventFeedEntry, isObjectiveStateLine, withEventLines } from './feedEventLines'
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

describe('withEventLines', () => {
  it('replie « porteur éliminé » sur le frag et trie le reste', () => {
    const feed = [frag('k', 'a', 5_000)]
    const carrier = eventFeedEntry('c', 5_000, { family: 'objectives', xuid: 'a', stat: 'flag_carriers_killed' })
    const grab = eventFeedEntry('g', 2_000, { family: 'objectives', xuid: 'a', stat: 'flag_grabs' })
    const out = withEventLines(feed, [carrier, grab])
    expect(out.map((e) => e.key)).toEqual(['g', 'k'])
    expect(out[1].flagCarrierKill).toBe(true)
    expect(feed[0].flagCarrierKill).toBeUndefined()
  })

  it('une action sans frag apparié garde sa ligne', () => {
    const carrier = eventFeedEntry('c', 5_000, { family: 'objectives', xuid: 'a', stat: 'flag_carriers_killed' })
    expect(withEventLines([], [carrier]).map((e) => e.key)).toEqual(['c'])
  })

  it('sans ligne d’événement, rend une copie du fil', () => {
    const feed = [frag('k', 'a', 1)]
    const out = withEventLines(feed, [])
    expect(out).toEqual(feed)
    expect(out).not.toBe(feed)
  })
})

describe('isObjectiveStateLine', () => {
  it('reconnaît manche, crâne et colline, pas une action nommée ni une autre famille', () => {
    expect(isObjectiveStateLine({ family: 'objectives', kind: 'hill', teamId: 0 })).toBe(true)
    expect(isObjectiveStateLine({ family: 'objectives', xuid: 'a', stat: 'flag_grabs' })).toBe(false)
    expect(isObjectiveStateLine({ family: 'vehicles', xuid: 'a', name: 'Ghost' })).toBe(false)
  })
})
