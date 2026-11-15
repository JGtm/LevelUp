/**
 * Tests — hillFeedLogic : une ligne « prend la colline » par CHANGEMENT DE TENANT de la colline,
 * lu dans `zoneStates[].spans`, à son instant sur l'axe du rejeu ; rien sans horloge sûre, rien
 * pour une base.
 */
import { describe, expect, it } from 'vitest'

import { testReplayDoc } from '../test/testDoc'
import { hillFeedEntries } from './hillFeedLogic'

const FRAME_MS = 100

interface Span {
  t0: number
  t1: number
  owner: number | null
  active?: boolean
}

/** Une zone dont les intervalles sont des colline (`active`) sauf demande contraire. */
function zone(zoneRef: number, spans: Span[], active = true) {
  return {
    zoneRef,
    spans: spans.map((s) => ({ active, ...s })),
    gauge: [],
    gaugeRamps: [],
  }
}

function docWith(zoneStates: unknown[], over: Record<string, unknown> = {}) {
  return testReplayDoc({ frameIntervalMs: FRAME_MS, zoneStates, ...over } as never)
}

/** Les lignes rendues : [instant, camp]. */
function lines(doc: ReturnType<typeof docWith>) {
  return hillFeedEntries(doc).map((e) => [e.replayMs, (e.event as { teamId: number }).teamId])
}

describe('hillFeedEntries', () => {
  it('un camp qui prend la colline fait une ligne, à l’image où son intervalle s’ouvre', () => {
    const doc = docWith([zone(3, [{ t0: 0, t1: 49, owner: null }, { t0: 50, t1: 200, owner: 1 }])])
    expect(lines(doc)).toEqual([[5_000, 1]])
    const [entry] = hillFeedEntries(doc)
    expect(entry.event).toMatchObject({ family: 'objectives', kind: 'hill', teamId: 1 })
  })

  it('un état neutre ou disputé ne fait pas de ligne, et le même camp qui reprend non plus', () => {
    const doc = docWith([
      zone(3, [
        { t0: 0, t1: 49, owner: null },
        { t0: 50, t1: 99, owner: 1 },
        { t0: 100, t1: 129, owner: null },
        { t0: 130, t1: 199, owner: 1 },
        { t0: 200, t1: 219, owner: null },
        { t0: 220, t1: 299, owner: 1 },
      ]),
    ])
    expect(lines(doc)).toEqual([[5_000, 1]])
  })

  it('le passage à l’autre camp fait une ligne, par un neutre ou directement', () => {
    const doc = docWith([
      zone(3, [
        { t0: 0, t1: 49, owner: 1 },
        { t0: 50, t1: 79, owner: null },
        { t0: 80, t1: 149, owner: 0 },
        { t0: 150, t1: 199, owner: 1 },
      ]),
    ])
    expect(lines(doc)).toEqual([
      [0, 1],
      [8_000, 0],
      [15_000, 1],
    ])
  })

  it('une colline qui revient après une rotation repart sans tenant : sa première prise s’écrit', () => {
    const doc = docWith([
      zone(3, [
        { t0: 0, t1: 99, owner: 1 },
        { t0: 1_000, t1: 1_099, owner: 1 },
      ]),
    ])
    expect(lines(doc)).toEqual([
      [0, 1],
      [100_000, 1],
    ])
  })

  it('les collines se lisent chacune pour elle-même, triées par instant', () => {
    const doc = docWith([
      zone(4, [{ t0: 300, t1: 399, owner: 0 }]),
      zone(3, [{ t0: 100, t1: 199, owner: 1 }]),
    ])
    expect(lines(doc)).toEqual([
      [10_000, 1],
      [30_000, 0],
    ])
  })

  it('une base n’a pas de ligne de colline : ses intervalles ne sont pas `active`', () => {
    const doc = docWith([zone(0, [{ t0: 0, t1: 99, owner: 1 }, { t0: 100, t1: 199, owner: 0 }], false)])
    expect(hillFeedEntries(doc)).toEqual([])
  })

  it('sans horloge sûre : aucune ligne', () => {
    const doc = docWith([zone(3, [{ t0: 0, t1: 99, owner: 1 }])], { coverage: { originResolved: false } })
    expect(hillFeedEntries(doc)).toEqual([])
  })

  it('un mode sans zone n’a aucune ligne', () => {
    expect(hillFeedEntries(docWith([]))).toEqual([])
  })
})
