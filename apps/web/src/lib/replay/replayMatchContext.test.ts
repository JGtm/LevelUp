import { describe, expect, it } from 'vitest'

import type { MatchViewResponse } from '@/lib/api/types'

import { replayContextFromMatchView } from './replayMatchContext'

describe('replayContextFromMatchView', () => {
  it('rend undefined sans vue', () => {
    expect(replayContextFromMatchView(undefined)).toBeUndefined()
    expect(replayContextFromMatchView(null)).toBeUndefined()
  })

  it('reprend les quatre morceaux PAR IDENTITÉ, sans les copier', () => {
    const header = { match_id: 'm', t0_ms: 12 }
    const scoreboard = [{ xuid: 'a', gamertag: 'A', is_me: true }]
    const events = [{ event_type: 'kill', actor_xuid: 'a', event_time_ms: 1 }]
    const media = { media_items: [] }
    const view = {
      header,
      team_tab: { scoreboard, roster: [], nemesis: [], encounters: [] },
      combat_tab: { highlight_events: events },
      media_tab: media,
    } as unknown as MatchViewResponse
    const ctx = replayContextFromMatchView(view)
    expect(ctx?.header).toBe(header)
    expect(ctx?.scoreboard).toBe(scoreboard)
    expect(ctx?.highlight_events).toBe(events)
    expect(ctx?.media).toBe(media)
  })

  it('comble les morceaux absents par des vides', () => {
    const view = { header: { match_id: 'm' } } as unknown as MatchViewResponse
    expect(replayContextFromMatchView(view)).toEqual({
      header: { match_id: 'm' },
      scoreboard: [],
      highlight_events: [],
      media: { media_items: [] },
    })
  })
})
