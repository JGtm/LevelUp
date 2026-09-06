/**
 * replayModel.test.ts — la jointure de l'artefact et de la vue match, sur oracles écrits à
 * la main.
 *
 * CE QUE CES CAS PROTÈGENT, ET QUE RIEN NE PROTÉGEAIT (registre 2026-09-05, W1) : la jointure
 * vivait dans une douzaine de `useMemo` d'une route, donc n'était atteignable qu'en montant
 * React et un routeur — aucun test ne la touchait. Elle est ici exercée sur un artefact
 * MINIMAL et une vue match MINIMALE, champ par champ.
 *
 * LE TÉMOIN, ET SES CHIFFRES. Film de 3 000 images à 100 ms (299,9 s), image zéro à 5 s du
 * début du match, countdown de 20 s, 200 s de jeu déclarées. Il s'ensuit, calculé à part :
 *
 *   coup d'envoi sur l'axe du film = 20 000 − 5 000 = 15 000 ms, soit l'image 150 ;
 *   fin de gameplay                = 20 000 + 200 000 − 5 000 = 215 000 ms, soit l'image 2 150 ;
 *   un kill servi à `event_time_ms` E se pose à E + 20 000 − 5 000 = E + 15 000 ;
 *   une capture prise à T ms après le début du match se pose à T − 5 000.
 */
import { describe, expect, it } from 'vitest'

import type { MatchViewResponse } from '@/lib/api/types'

import { testReplayDoc } from '../test/testDoc'
import { buildReplayModel } from './replayModel'

const ORIGIN_MS = 5_000
const T0_MS = 20_000
const MATCH_START = '2026-09-01T12:00:00Z'

/** Deux joueurs du film : `me` (allié, 3 vies fictives réduites à une) et `adv` (adverse). */
function doc() {
  return testReplayDoc({
    frameCount: 3_000,
    frameIntervalMs: 100,
    originMs: ORIGIN_MS,
    roster: [
      { xuid: 'me', name: 'Moi', filmIndex: 0 },
      { xuid: 'adv', name: 'Autre', filmIndex: 1 },
    ],
    tracks: [
      {
        slot: 1, team: -1, xuid: 'me', startFrame: 0, endFrame: 2_000,
        points: [{ t: 0, x: 0, y: 0 }, { t: 2_000, x: 5, y: 5 }],
      },
      {
        slot: 2, team: -1, xuid: 'adv', startFrame: 0, endFrame: 900,
        points: [{ t: 0, x: 1, y: 1 }, { t: 900, x: 2, y: 2 }],
      },
    ],
  } as never)
}

/** La vue match minimale : en-tête, scoreboard, un frag, une capture. */
function matchView(over: Record<string, unknown> = {}): MatchViewResponse {
  return {
    header: {
      t0_ms: T0_MS,
      playable_duration_seconds: 200,
      start_time: MATCH_START,
      score_mine: 3,
      score_theirs: 1,
    },
    team_tab: {
      scoreboard: [
        { xuid: 'me', gamertag: 'Moi', team_side: 't0', is_me: true },
        { xuid: 'adv', gamertag: 'Autre', team_side: 't1', is_me: false },
      ],
    },
    combat_tab: {
      highlight_events: [
        { event_type: 'kill', actor_xuid: 'me', event_time_ms: 30_000, victim_xuid: 'adv' },
      ],
    },
    media_tab: {
      media_items: [
        {
          file_id: '7',
          file_name: 'capture.png',
          file_path: '/media/capture.png',
          kind: 'image',
          liked: false,
          capture_start_time: '2026-09-01T12:01:00Z', // 60 s après le début du match
        },
      ],
    },
    ...over,
  } as unknown as MatchViewResponse
}

describe('buildReplayModel — sans donnée, rien n’est inventé', () => {
  it('rend un modèle VIDE sans artefact, même avec une vue match complète', () => {
    const m = buildReplayModel(null, matchView())
    expect(m).toEqual({
      scoreboard: [],
      identity: new Map(),
      marks: new Map(),
      players: [],
      clock: null,
      window: null,
      feed: [],
      media: [],
      score: null,
      t0Ms: 0,
    })
  })

  it('accepte une vue match ABSENTE : le film reste lisible, sans noms ni fil', () => {
    const m = buildReplayModel(doc(), undefined)
    expect(m.scoreboard).toEqual([])
    expect(m.score).toBeNull()
    // LE FIL N'EST PAS VIDE POUR AUTANT, et c'est une mesure : sans kill de la base, il ne
    // reste que les morts que le FILM date lui-même — les deux vies se ferment bien avant
    // l'horizon du document (images 900 et 2 000 sur 2 999), donc deux lignes neutres.
    expect(m.feed.filter((e) => e.kill)).toEqual([])
    expect(m.feed.map((e) => [e.death?.xuid, e.replayMs])).toEqual([
      ['adv', 90_000],
      ['me', 200_000],
    ])
    expect(m.t0Ms).toBe(0)
    // Le roster du film, lui, existe : il ne demande rien à la base.
    expect(m.players.map((p) => p.xuid)).toEqual(['me', 'adv'])
    // Sans countdown, l'horloge s'établit quand même — les deux axes retombent sur le match.
    expect(m.clock?.originMs).toBe(ORIGIN_MS)
    expect(m.clock?.t0Ms).toBe(0)
    // La fenêtre, elle, exige la durée jouable de l'en-tête : pas d'en-tête, pas de cadrage.
    expect(m.window).toBeNull()
  })
})

describe('buildReplayModel — l’identité et les marques', () => {
  it('résout le camp de chaque xuid depuis la ligne « moi » du scoreboard', () => {
    const { identity } = buildReplayModel(doc(), matchView())
    expect(identity.get('me')).toEqual({ gamertag: 'Moi', ally: true })
    expect(identity.get('adv')).toEqual({ gamertag: 'Autre', ally: false })
  })

  it('marque « moi » depuis le scoreboard et « ami » depuis les réglages', () => {
    const { marks } = buildReplayModel(doc(), matchView(), { friend_gamertags: ['Autre'] })
    expect(marks.get('me')).toBe('me')
    expect(marks.get('adv')).toBe('friend')
  })

  it('sans réglages, personne n’est ami — jamais une liste devinée', () => {
    const { marks } = buildReplayModel(doc(), matchView())
    expect(marks.get('me')).toBe('me')
    expect(marks.get('adv')).toBeUndefined()
  })
})

describe('buildReplayModel — l’horloge et la fenêtre de gameplay', () => {
  it('porte l’origine du film et le countdown de l’API', () => {
    const m = buildReplayModel(doc(), matchView())
    expect(m.t0Ms).toBe(T0_MS)
    expect(m.clock?.originMs).toBe(ORIGIN_MS)
    expect(m.clock?.t0Ms).toBe(T0_MS)
  })

  it('cadre le gameplay entre le coup d’envoi et la fin déclarée', () => {
    const { window } = buildReplayModel(doc(), matchView())
    // 20 000 − 5 000 = 15 000 ms de film, soit l'image 150 ; le préambule se pose 1 s avant.
    expect(window?.startMs).toBe(15_000)
    expect(window?.startFrame).toBe(150)
    expect(window?.leadInFrame).toBe(140)
    // 20 000 + 200 × 1 000 − 5 000 = 215 000 ms, soit l'image 2 150.
    expect(window?.endMs).toBe(215_000)
    expect(window?.endFrame).toBe(2_150)
  })

  it('rend une fenêtre NULLE sans durée jouable — un cadrage inventé amputerait le rejeu', () => {
    const sansDuree = matchView({ header: { t0_ms: T0_MS, start_time: MATCH_START } })
    expect(buildReplayModel(doc(), sansDuree).window).toBeNull()
  })

  it('rend une horloge NULLE sans origine publiée, et la fenêtre suit', () => {
    const sansOrigine = testReplayDoc({
      frameCount: 3_000, frameIntervalMs: 100,
      tracks: [{ slot: 1, team: -1, xuid: 'me', startFrame: 0, endFrame: 10, points: [{ t: 0, x: 0, y: 0 }] }],
    } as never)
    const m = buildReplayModel(sansOrigine, matchView())
    expect(m.clock).toBeNull()
    expect(m.window).toBeNull()
    // Et la piste médias se tait : elle ne pose rien sur une origine qu'on n'a pas.
    expect(m.media).toEqual([])
  })
})

describe('buildReplayModel — le fil, recalé une seule fois', () => {
  it('pose le frag sur l’axe du rejeu : event_time_ms + t0_ms − origine', () => {
    const { feed } = buildReplayModel(doc(), matchView())
    const kill = feed.find((e) => e.kill)
    // 30 000 + 20 000 − 5 000 = 45 000.
    expect(kill?.replayMs).toBe(45_000)
    expect(kill?.kill?.xuid).toBe('me')
    expect(kill?.kill?.victimXuid).toBe('adv')
  })

  it('TRIE le fil sur l’axe du rejeu, lignes de présence comprises', () => {
    const { feed } = buildReplayModel(doc(), matchView())
    const instants = feed.map((e) => e.replayMs)
    expect([...instants].sort((a, b) => a - b)).toEqual(instants)
  })

  it('ne rend AUCUNE ligne quand la vue match n’a pas d’événement', () => {
    const sansEvents = matchView({ combat_tab: { highlight_events: [] } })
    expect(buildReplayModel(doc(), sansEvents).feed.filter((e) => e.kill)).toEqual([])
  })
})

describe('buildReplayModel — les médias et le score', () => {
  it('pose la capture à son instant absolu, moins l’origine du film', () => {
    const { media } = buildReplayModel(doc(), matchView())
    // 60 s après le début du match, image zéro à 5 s : 60 000 − 5 000 = 55 000.
    expect(media).toHaveLength(1)
    expect(media[0]).toMatchObject({ id: '7', kind: 'image', replayMs: 55_000 })
  })

  it('reporte le score FINAL de l’en-tête, jamais celui déduit du film', () => {
    expect(buildReplayModel(doc(), matchView()).score).toEqual({ ally: 3, enemy: 1 })
  })

  it('rend un score NUL quand l’en-tête ne le publie pas', () => {
    const sansScore = matchView({
      header: { t0_ms: T0_MS, playable_duration_seconds: 200, start_time: MATCH_START },
    })
    expect(buildReplayModel(doc(), sansScore).score).toBeNull()
  })
})

describe('buildReplayModel — le roster', () => {
  it('joint le roster du film au scoreboard, dans l’ordre du film', () => {
    const { players } = buildReplayModel(doc(), matchView())
    expect(players.map((p) => p.xuid)).toEqual(['me', 'adv'])
    expect(players.map((p) => p.board?.gamertag)).toEqual(['Moi', 'Autre'])
    // Chaque joueur porte ses vies, triées : une seule ici, mais l'index est bien peuplé.
    expect(players.map((p) => p.lives.length)).toEqual([1, 1])
  })

  it('laisse SANS ligne de base un joueur que le scoreboard ne connaît pas', () => {
    const inconnu = matchView({
      team_tab: { scoreboard: [{ xuid: 'me', gamertag: 'Moi', team_side: 't0', is_me: true }] },
    })
    const { players } = buildReplayModel(doc(), inconnu)
    expect(players.find((p) => p.xuid === 'adv')?.board).toBeUndefined()
  })
})
