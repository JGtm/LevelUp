import { describe, expect, it } from 'vitest'

import { chatterFeedEntries } from './chatterFeedLogic'
import type { ChatterLine } from './chatterLines'
import { withEventLines } from './feedEventLines'
import { availableFeedFilters, effectiveFeedFilter, filterFeedEntries, matchesFeedFilter } from './feedFilterLogic'
import { feedAt, type ReplayFeedEntry } from './killFeedLogic'
import { playerKeys } from './feedFollowed'
import { concernsFollowed } from './feedFollowed'

function line(frame: number, xuid: string, text = 'Ennemi aperçu !', voice: string | null = null): ChatterLine {
  return { key: `chatter-${frame}`, frame, xuid, sound: '0000000a', family: 'enemy', text, voice, count: 1 }
}

const DOC = { frameIntervalMs: 100 }

describe('chatterFeedEntries', () => {
  it("fait une ligne d'événement par réplique, à son instant, avec le joueur, le texte et la voix", () => {
    const [a, b] = chatterFeedEntries([line(10, 'A', 'Un', 'repliques/fr/0000000a'), line(20, 'B', 'Deux')], DOC)
    expect(a).toMatchObject({ key: 'chatter-10', replayMs: 1000, event: { family: 'replies', xuid: 'A', text: 'Un', voice: true, chatterFamily: 'enemy' } })
    expect(b.event).toMatchObject({ xuid: 'B', voice: false })
    expect(a.kill).toBeNull()
  })

  it("rejoint le fil trié et se coupe à l'instant lu comme les autres lignes", () => {
    const events = chatterFeedEntries([line(10, 'A'), line(30, 'A')], DOC)
    const feed = withEventLines([], events)
    expect(feedAt(feed, 1500).map((e) => e.key)).toEqual(['chatter-10'])
  })

  it('concerne le joueur suivi par la caméra : sa réplique porte le fond léger de son suivi', () => {
    const [e] = chatterFeedEntries([line(10, 'A')], DOC)
    expect(concernsFollowed(e, playerKeys({ xuid: 'A', board: undefined } as never))).toBe(true)
    expect(concernsFollowed(e, playerKeys({ xuid: 'B', board: undefined } as never))).toBe(false)
  })
})

describe('le filtre « Répliques » du fil', () => {
  const kill = { key: 'k', replayMs: 500, kill: { xuid: 'A', medals: [] }, medal: null, death: null } as unknown as ReplayFeedEntry
  const replies = chatterFeedEntries([line(10, 'A'), line(20, 'B')], DOC)

  it('ne garde que les répliques', () => {
    expect(filterFeedEntries([kill, ...replies], 'replies').map((e) => e.key)).toEqual(['chatter-10', 'chatter-20'])
    expect(matchesFeedFilter(kill, 'replies')).toBe(false)
  })

  it('« Tout » les montre avec les autres lignes', () => {
    expect(filterFeedEntries([kill, ...replies], 'tout')).toHaveLength(3)
  })

  it("n'est proposé que si le fil a au moins une réplique, avec son nombre de lignes", () => {
    expect(availableFeedFilters([kill]).map((o) => o.id)).not.toContain('replies')
    const opts = availableFeedFilters([kill, ...replies])
    expect(opts.find((o) => o.id === 'replies')?.count).toBe(2)
  })

  it('retombe sur Tout quand le filtre mémorisé n’est plus proposé (réglage du fil éteint)', () => {
    expect(effectiveFeedFilter('replies', availableFeedFilters([kill]))).toBe('tout')
    expect(effectiveFeedFilter('replies', availableFeedFilters([kill, ...replies]))).toBe('replies')
  })
})
