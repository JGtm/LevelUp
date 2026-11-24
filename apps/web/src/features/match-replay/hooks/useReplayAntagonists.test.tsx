import { renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'
import type { ReplayFeedEntry, ReplayKill } from '../model/killFeedLogic'
import { useReplayAntagonists } from './useReplayAntagonists'

const players = [
  { xuid: 'a', team: 0, lives: [] },
  { xuid: 'b', team: 1, lives: [] },
] as unknown as ReplayPlayer[]

const feed: ReplayFeedEntry[] = [
  {
    key: 'k1',
    replayMs: 1000,
    kill: { xuid: 'b', victimXuid: 'a', victimGamertag: '', replayMs: 1000, medals: [] } as unknown as ReplayKill,
    medal: null,
    death: null,
  },
]

describe('useReplayAntagonists', () => {
  it('rend la fabrique des index, stable tant que le fil et le roster ne changent pas', () => {
    const { result, rerender } = renderHook(({ f, p }) => useReplayAntagonists(f, p, undefined), {
      initialProps: { f: feed, p: players },
    })
    const first = result.current
    expect(first.forSubject('a').at(1000).nemesis?.holder).toBe('b')
    rerender({ f: feed, p: players })
    expect(result.current).toBe(first)
    expect(result.current.forSubject('a')).toBe(first.forSubject('a'))
  })

  it('reconstruit la fabrique (donc les index) quand le fil change', () => {
    const { result, rerender } = renderHook(({ f, p }) => useReplayAntagonists(f, p, undefined), {
      initialProps: { f: feed, p: players },
    })
    const first = result.current
    rerender({ f: [...feed], p: players })
    expect(result.current).not.toBe(first)
  })
})
