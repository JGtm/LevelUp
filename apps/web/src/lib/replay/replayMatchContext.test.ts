import { describe, expect, it } from 'vitest'

import type { MatchViewHeader } from '@/lib/api/types'

import {
  normalizeReplayMatchContext,
  replayContextRefetchInterval,
  replayContextStaleTime,
  type ReplayMatchContext,
  type ReplayMatchContextWire,
} from './replayMatchContext'

const header = { match_id: 'm', t0_ms: 12 } as unknown as MatchViewHeader

describe('normalizeReplayMatchContext', () => {
  it('reprend les quatre morceaux PAR IDENTITÉ quand ils sont là', () => {
    const scoreboard = [{ xuid: 'a', gamertag: 'A', is_me: true }]
    const events = [{ event_type: 'kill', actor_xuid: 'a', event_time_ms: 1 }]
    const media = { media_items: [] }
    const raw = {
      header, scoreboard, highlight_events: events, media, source_unavailable: true,
    } as unknown as ReplayMatchContextWire
    const ctx = normalizeReplayMatchContext(raw)
    expect(ctx.header).toBe(header)
    expect(ctx.scoreboard).toBe(scoreboard)
    expect(ctx.highlight_events).toBe(events)
    expect(ctx.media).toBe(media)
    expect(ctx.source_unavailable).toBe(true)
  })

  it('comble les tableaux null du transport par des vides', () => {
    const raw: ReplayMatchContextWire = {
      header, scoreboard: null, highlight_events: null, media: null, source_unavailable: false,
    }
    expect(normalizeReplayMatchContext(raw)).toEqual({
      header,
      scoreboard: [],
      highlight_events: [],
      media: { media_items: [] },
      source_unavailable: false,
    })
  })
})

/** Un contexte normalisé, en-tête et drapeau de panne au choix. */
function ctx(over: { analysisPending?: boolean; sourceUnavailable?: boolean } = {}): ReplayMatchContext {
  return normalizeReplayMatchContext({
    header: { ...header, analysis_pending: over.analysisPending ?? false },
    scoreboard: [],
    highlight_events: [],
    media: null,
    source_unavailable: over.sourceUnavailable ?? false,
  })
}

describe('politique de rafraîchissement du contexte (celle de la vue complète)', () => {
  it('nouvel essai toutes les 30 s et fraîcheur nulle sur lecture en échec', () => {
    expect(replayContextRefetchInterval(ctx({ sourceUnavailable: true }))).toBe(30_000)
    expect(replayContextRefetchInterval(ctx({ sourceUnavailable: true, analysisPending: true }))).toBe(30_000)
    expect(replayContextStaleTime(ctx({ sourceUnavailable: true }))).toBe(0)
  })

  it('rechargement toutes les 60 s et fraîcheur nulle pendant l’analyse du film', () => {
    expect(replayContextRefetchInterval(ctx({ analysisPending: true }))).toBe(60_000)
    expect(replayContextStaleTime(ctx({ analysisPending: true }))).toBe(0)
  })

  it('arrêt et 10 min de fraîcheur sinon, y compris avant la première réponse', () => {
    expect(replayContextRefetchInterval(ctx())).toBe(false)
    expect(replayContextRefetchInterval(undefined)).toBe(false)
    expect(replayContextStaleTime(ctx())).toBe(600_000)
    expect(replayContextStaleTime(undefined)).toBe(600_000)
  })
})
