/**
 * Tests — skullFeedLogic : « prend le crâne » à chaque portage, « perd le crâne » à la fin d'un
 * portage FERMÉ seulement, et rien sans horloge sûre.
 */
import { describe, expect, it } from 'vitest'

import { testReplayDoc } from '../test/testDoc'
import { skullFeedEntries } from './skullFeedLogic'

const FRAME_MS = 100

function docWith(skullCarries: unknown[], over: Record<string, unknown> = {}) {
  return testReplayDoc({ frameIntervalMs: FRAME_MS, skullCarries: skullCarries as never, ...over } as never)
}

/** Ce qu'une ligne du crâne dit : instant, joueur, prise ou perte. */
function said(entries: ReturnType<typeof skullFeedEntries>) {
  return entries.map((e) => {
    const line = e.event as { xuid: string; change: string }
    return [e.replayMs, line.xuid, line.change]
  })
}

describe('skullFeedEntries', () => {
  it('une prise au début de chaque portage et une perte à la fin d’un portage fermé', () => {
    const entries = skullFeedEntries(
      docWith([
        { xuid: 'a', t0: 10, t1: 40, closed: true },
        { xuid: 'b', t0: 60, t1: 90, closed: true },
      ]),
    )
    expect(said(entries)).toEqual([
      [1_000, 'a', 'taken'],
      [4_000, 'a', 'lost'],
      [6_000, 'b', 'taken'],
      [9_000, 'b', 'lost'],
    ])
    expect(entries.every((e) => e.event?.family === 'objectives')).toBe(true)
  })

  it('un portage ouvert n’a pas de ligne de fin : sa fin n’est qu’une borne haute', () => {
    const entries = skullFeedEntries(docWith([{ xuid: 'a', t0: 10, t1: 150, closed: false }]))
    expect(said(entries)).toEqual([[1_000, 'a', 'taken']])
  })

  it('à instant égal, la perte précède la prise : le crâne change de main dans le bon sens', () => {
    const entries = skullFeedEntries(
      docWith([
        { xuid: 'b', t0: 40, t1: 80, closed: true },
        { xuid: 'a', t0: 10, t1: 40, closed: true },
      ]),
    )
    expect(said(entries).filter(([ms]) => ms === 4_000)).toEqual([
      [4_000, 'a', 'lost'],
      [4_000, 'b', 'taken'],
    ])
  })

  it('les clés sont uniques', () => {
    const entries = skullFeedEntries(
      docWith([
        { xuid: 'a', t0: 10, t1: 40, closed: true },
        { xuid: 'a', t0: 10, t1: 40, closed: true },
      ]),
    )
    expect(new Set(entries.map((e) => e.key)).size).toBe(entries.length)
  })

  it('sans horloge sûre : aucune ligne', () => {
    const doc = docWith([{ xuid: 'a', t0: 10, t1: 40, closed: true }], { coverage: { originResolved: false } })
    expect(skullFeedEntries(doc)).toEqual([])
  })

  it('un mode sans crâne n’a aucune ligne', () => {
    expect(skullFeedEntries(docWith([]))).toEqual([])
  })
})
