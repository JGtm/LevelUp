/**
 * Tests — buildFragMarks (les repères d'éliminations de la frise compacte).
 */
import { describe, expect, it } from 'vitest'

import { buildFragMarks } from './fragMarks'
import { trackScale } from './replayTimelineTracksLogic'

/** 10 Hz, images 0 à 200 : une milliseconde vaut 1/10 000 de frise. */
const SCALE = trackScale({ startFrame: 0, leadInFrame: 0, endFrame: 200, startMs: 0, endMs: 20_000 }, 201)

describe('buildFragMarks', () => {
  it('un repère par frag, à sa place sur l’échelle de la frise, au camp du tueur', () => {
    const marks = buildFragMarks(
      [{ replayMs: 5_000, teamId: 0 }, { replayMs: 10_000, teamId: 1 }],
      100,
      SCALE,
    )
    expect(marks.map((m) => [m.teamId, m.ratio])).toEqual([[0, 0.25], [1, 0.5]])
    expect(new Set(marks.map((m) => m.key)).size).toBe(2)
  })

  it('un frag hors de la fenêtre de gameplay n’est pas posé', () => {
    expect(buildFragMarks([{ replayMs: -1_000, teamId: 0 }, { replayMs: 60_000, teamId: 0 }], 100, SCALE)).toEqual([])
  })

  it('une frise sans étendue, ou sans cadence, ne pose rien', () => {
    expect(buildFragMarks([{ replayMs: 1_000, teamId: 0 }], 100, { from: 0, span: 0 })).toEqual([])
    expect(buildFragMarks([{ replayMs: 1_000, teamId: 0 }], 0, SCALE)).toEqual([])
  })
})
