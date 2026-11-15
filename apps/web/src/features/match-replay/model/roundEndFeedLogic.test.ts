/**
 * Tests — roundEndFeedLogic : la ligne « fin de manche » avec le score de manche de chaque camp.
 *
 * Ce qu'ils protègent : seulement un mode à plusieurs manches, seulement une horloge sûre, le score
 * DANS la manche (jamais le cumul), un camp muet à zéro, et aucune ligne sans deux camps.
 */
import { describe, expect, it } from 'vitest'

import { testReplayDoc } from '../test/testDoc'
import { roundEndFeedEntries } from './roundEndFeedLogic'

const FRAME_MS = 100

/** Le témoin Oddball : manches 100/78 (fin f50) puis 100/43 (fin f150), total 200/121. */
const TEAMS = [
  {
    teamId: 0,
    rounds: [
      { round: 0, points: [{ t: 0, v: 0 }, { t: 50, v: 100 }] },
      { round: 1, points: [{ t: 100, v: 0 }, { t: 150, v: 100 }] },
    ],
    total: [{ t: 50, v: 100 }, { t: 150, v: 200 }],
  },
  {
    teamId: 1,
    rounds: [
      { round: 0, points: [{ t: 0, v: 0 }, { t: 50, v: 78 }] },
      { round: 1, points: [{ t: 100, v: 0 }, { t: 150, v: 43 }] },
    ],
    total: [{ t: 50, v: 78 }, { t: 150, v: 121 }],
  },
]

function docWith(teams: unknown[], over: Record<string, unknown> = {}) {
  return testReplayDoc({
    frameIntervalMs: FRAME_MS,
    scoreTimeline: { teams, players: [] } as never,
    ...over,
  } as never)
}

describe('roundEndFeedEntries', () => {
  it('une ligne par manche, à sa clôture, avec le score DE LA MANCHE de chaque camp', () => {
    const entries = roundEndFeedEntries(docWith(TEAMS), [0, 1])
    expect(entries.map((e) => e.replayMs)).toEqual([5_000, 15_000])
    expect(entries.map((e) => e.event)).toEqual([
      {
        family: 'objectives',
        kind: 'roundEnd',
        index: 1,
        scores: [
          { teamId: 0, score: 100 },
          { teamId: 1, score: 78 },
        ],
      },
      {
        family: 'objectives',
        kind: 'roundEnd',
        index: 2,
        scores: [
          { teamId: 0, score: 100 },
          { teamId: 1, score: 43 },
        ],
      },
    ])
  })

  it('un camp sans série vaut zéro, et les camps viennent de l’appelant, pas du calque', () => {
    const entries = roundEndFeedEntries(docWith([TEAMS[0]]), [0, 1])
    expect((entries[0].event as unknown as { scores: unknown[] }).scores).toEqual([
      { teamId: 0, score: 100 },
      { teamId: 1, score: 0 },
    ])
  })

  it('un mode à manche unique n’a aucune ligne', () => {
    const single = TEAMS.map((t) => ({ ...t, rounds: t.rounds.slice(0, 1) }))
    expect(roundEndFeedEntries(docWith(single), [0, 1])).toEqual([])
  })

  it('moins de deux camps connus : aucune ligne', () => {
    expect(roundEndFeedEntries(docWith(TEAMS), [0])).toEqual([])
    expect(roundEndFeedEntries(docWith(TEAMS), [])).toEqual([])
  })

  it('sans horloge sûre : aucune ligne', () => {
    const doc = docWith(TEAMS, { coverage: { originResolved: false } })
    expect(roundEndFeedEntries(doc, [0, 1])).toEqual([])
  })

  it('sans calque de score : aucune ligne', () => {
    expect(roundEndFeedEntries(testReplayDoc({ frameIntervalMs: FRAME_MS }), [0, 1])).toEqual([])
  })
})
