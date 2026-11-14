/**
 * useReplayLives.test.tsx — LES TRACES DES VIES, câblées : la disponibilité de chaque calque suit
 * ses marques, les trois peintres portent leur id et peignent sur le filtre des réglages, et
 * « joueur suivi » sans joueur suivi retombe sur tous (cette partie seulement).
 */
import { renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { buildFilmAllegiance } from '@/lib/replay/filmAllegiance'

import type { ReplayFeedEntry, ReplayKill } from '../model/killFeedLogic'
import type { CanvasView } from '../model/replayView'
import { count, recordingContext } from '../test/recordingContext'
import { testReplayDoc } from '../test/testDoc'
import { useReplayLives, type LivesOptions } from './useReplayLives'

const DOC = testReplayDoc({
  frameIntervalMs: 100,
  bounds: { minX: 0, minY: 0, maxX: 100, maxY: 100 },
  tracks: [
    { slot: 1, team: 0, xuid: 'k', points: [{ t: 0, x: 25, y: 75, h: 0 }, { t: 100, x: 25, y: 75 }], startFrame: 0, endFrame: 100 },
    { slot: 2, team: 1, xuid: 'v', points: [{ t: 0, x: 75, y: 25, h: 90 }, { t: 100, x: 75, y: 25 }], startFrame: 0, endFrame: 100 },
  ] as never,
})

const VIEW: CanvasView = { bounds: DOC.bounds as CanvasView['bounds'], width: 400, height: 400, pad: 0 }

const KILL = {
  replayMs: 2_000, medals: [], tMs: 1_000, xuid: 'k', teamID: 0, weaponKey: '', weaponLabel: '',
  weaponImageUrl: '', weaponTinted: false, assistState: '', assistGamertag: '', assistTeamID: null,
  killerDamagePct: null, assistDamagePct: null, victimXuid: 'v', victimGamertag: 'V',
} as ReplayKill

const FEED: ReplayFeedEntry[] = [{ key: 'e1', replayMs: 2_000, kill: KILL, medal: null, death: null }]

const NO_ALLEGIANCE = buildFilmAllegiance([], null)

function options(over: Partial<LivesOptions> = {}): LivesOptions {
  return {
    doc: DOC,
    view: VIEW,
    feedEntries: FEED,
    settings: { showSpawns: true, showDeaths: true, showKillers: true, livesWho: 'all', livesWhen: 'all' },
    allegiance: NO_ALLEGIANCE,
    marks: new Map([['k', 'me' as const]]),
    identity: {
      colorOfSlot: (slot) => (slot === 1 ? 'ink-a' : 'ink-b'),
      colorOfXuid: (xuid) => (xuid === 'k' ? 'ink-a' : 'ink-b'),
      campOfSlot: () => null,
      markOfSlot: (slot) => (slot === 1 ? 'me' : undefined),
      seatedOfXuid: () => true,
    },
    ink: 'under',
    ...over,
  }
}

describe('useReplayLives', () => {
  it('chaque calque est disponible dès qu’il a une marque ; un joueur suivi est connu par la marque « moi »', () => {
    const { result } = renderHook(() => useReplayLives(options()))
    expect(result.current.available).toEqual({ spawns: true, deaths: true, killers: true, followedKnown: true })
    expect(result.current.show).toEqual({ spawns: true, deaths: true, killers: true })
  })

  it('sans fil, il ne reste que les apparitions : morts et tueurs ne sont pas proposés', () => {
    const { result } = renderHook(() => useReplayLives(options({ feedEntries: [] })))
    expect(result.current.available).toMatchObject({ spawns: true, deaths: false, killers: false })
  })

  it('les trois peintres portent leur id, dans l’ordre apparitions, morts, tueurs', () => {
    const { result } = renderHook(() => useReplayLives(options()))
    expect(result.current.layers.map((l) => l.id)).toEqual(['vies-apparitions', 'vies-morts', 'vies-tueurs'])
  })

  it('les peintres dessinent les marques du document : deux apparitions, une croix, un triangle et son trait', () => {
    const { result } = renderHook(() => useReplayLives(options()))
    const [spawns, deaths, killers] = result.current.layers
    const s = recordingContext()
    spawns.paint(s.ctx, 0, 1)
    expect(count(s.ops, 'moveTo')).toBe(2) // deux gouttes à cap lu (k : 0, v : 90)
    const d = recordingContext()
    deaths.paint(d.ctx, 0, 1)
    expect(count(d.ops, 'moveTo')).toBe(2)
    const k = recordingContext()
    killers.paint(k.ctx, 0, 1)
    expect(count(k.ops, 'closePath')).toBe(1)
    expect(count(k.ops, 'stroke')).toBe(4)
  })

  it('« joueur suivi » filtre sur la marque « moi » ; sans joueur suivi, il retombe sur tous', () => {
    const suivi = renderHook(() =>
      useReplayLives(options({ settings: { ...options().settings, livesWho: 'followed' } })),
    )
    const a = recordingContext()
    suivi.result.current.layers[1].paint(a.ctx, 0, 1)
    expect(count(a.ops, 'moveTo')).toBe(0) // la victime n'est pas le joueur suivi : aucune croix
    const personne = renderHook(() =>
      useReplayLives(options({ marks: new Map(), settings: { ...options().settings, livesWho: 'followed' }, identity: { ...options().identity, markOfSlot: () => undefined } })),
    )
    expect(personne.result.current.available.followedKnown).toBe(false)
    const b = recordingContext()
    personne.result.current.layers[1].paint(b.ctx, 0, 1)
    expect(count(b.ops, 'moveTo')).toBe(2) // retombé sur « tous »
  })

  it('« mon équipe » suit l’allégeance du film : sans camp connu, aucune marque n’est des miennes', () => {
    const { result } = renderHook(() =>
      useReplayLives(options({ settings: { ...options().settings, livesWho: 'team' } })),
    )
    const s = recordingContext()
    result.current.layers[0].paint(s.ctx, 0, 1)
    expect(count(s.ops, 'moveTo')).toBe(0)
  })

  it('« jusqu’ici » à l’image 0 : pas de mort ni de frag à venir', () => {
    const { result } = renderHook(() =>
      useReplayLives(options({ settings: { ...options().settings, livesWhen: 'sofar' } })),
    )
    const d = recordingContext()
    result.current.layers[1].paint(d.ctx, 0, 1)
    expect(count(d.ops, 'moveTo')).toBe(0)
    const d2 = recordingContext()
    result.current.layers[1].paint(d2.ctx, 30, 1)
    expect(count(d2.ops, 'moveTo')).toBe(2)
  })
})
