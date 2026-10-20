import { describe, expect, it } from 'vitest'

import type { MatchViewHeader } from '@/lib/api/types'

import {
  normalizeReplayMatchContext,
  replayContextStaleTime,
  type ReplayMatchContextWire,
} from './replayMatchContext'

const header = { match_id: 'm', t0_ms: 12 } as unknown as MatchViewHeader

describe('normalizeReplayMatchContext', () => {
  it('reprend les quatre morceaux PAR IDENTITÉ quand ils sont là', () => {
    const scoreboard = [{ xuid: 'a', gamertag: 'A', is_me: true }]
    const events = [{ event_type: 'kill', actor_xuid: 'a', event_time_ms: 1 }]
    const media = { media_items: [] }
    const raw = { header, scoreboard, highlight_events: events, media } as unknown as ReplayMatchContextWire
    const ctx = normalizeReplayMatchContext(raw)
    expect(ctx.header).toBe(header)
    expect(ctx.scoreboard).toBe(scoreboard)
    expect(ctx.highlight_events).toBe(events)
    expect(ctx.media).toBe(media)
  })

  it('comble les tableaux null du transport par des vides', () => {
    const raw: ReplayMatchContextWire = { header, scoreboard: null, highlight_events: null, media: null }
    expect(normalizeReplayMatchContext(raw)).toEqual({
      header,
      scoreboard: [],
      highlight_events: [],
      media: { media_items: [] },
    })
  })
})

describe('replayContextStaleTime', () => {
  it('nulle pendant l’analyse du film, 10 min sinon', () => {
    const ctx = normalizeReplayMatchContext({ header, scoreboard: [], highlight_events: [], media: null })
    expect(replayContextStaleTime(ctx)).toBe(600_000)
    expect(replayContextStaleTime(undefined)).toBe(600_000)
    const pending = { ...ctx, header: { ...header, analysis_pending: true } }
    expect(replayContextStaleTime(pending)).toBe(0)
  })
})
